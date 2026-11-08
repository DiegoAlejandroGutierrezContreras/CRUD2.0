package clientes

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"facturacion-microservicios/internal/authclient"
	"facturacion-microservicios/internal/httpx"
	"facturacion-microservicios/internal/jsondb"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var ccRegex = regexp.MustCompile(`^\d+$`)

// Errores de unicidad que devuelve la validación atómica del store.
var (
	errEmailDuplicado = errors.New("ya existe un cliente con ese email")
	errCCDuplicada    = errors.New("ya existe un cliente con esa cédula")
)

// Handler maneja las rutas del servicio de clientes.
type Handler struct {
	store         *jsondb.Store[Cliente]
	auth          *authclient.Client
	internalToken string
}

// NewHandler crea el handler de producción: todas las rutas exigen rol admin.
// Facturación consulta un cliente reenviando la sesión del admin que emite.
func NewHandler(store *jsondb.Store[Cliente], authURL string) *Handler {
	return &Handler{store: store, auth: authclient.New(authURL)}
}

// NewHandlerConInterno crea el handler habilitando además la ruta
// `GET /clientes/por-email`, reservada a Auth mediante un token de servicio.
func NewHandlerConInterno(store *jsondb.Store[Cliente], authURL, internalToken string) *Handler {
	return &Handler{
		store:         store,
		auth:          authclient.New(authURL),
		internalToken: internalToken,
	}
}

// NewHandlerSinAuth crea un handler sin autenticación. Solo para pruebas.
func NewHandlerSinAuth(store *jsondb.Store[Cliente]) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registra las rutas del servicio.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// La consulta por email es servicio-a-servicio: Auth la usa al registrar
	// cuentas de clientes. Va fuera del middleware de sesión de Auth.
	mux.HandleFunc("GET /clientes/por-email", h.handlePorEmail)
	authclient.Registrar(mux, h.auth, []authclient.Route{
		{Pattern: "GET /clientes", Handler: h.handleList, AdminOnly: true},
		{Pattern: "GET /clientes/{id}", Handler: h.handleGet, AdminOnly: true},
		{Pattern: "POST /clientes", Handler: h.handleCreate, AdminOnly: true},
		{Pattern: "PUT /clientes/{id}", Handler: h.handleUpdate, AdminOnly: true},
		{Pattern: "DELETE /clientes/{id}", Handler: h.handleDelete, AdminOnly: true},
	})
}

// handlePorEmail responde con la vista mínima de un cliente. Solo la usa Auth
// al crear cuentas de clientes; por eso exige el token de servicio cuando el
// handler está en modo producción.
func (h *Handler) handlePorEmail(w http.ResponseWriter, r *http.Request) {
	if h.internalToken != "" &&
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Token")), []byte(h.internalToken)) != 1 {
		httpx.RespondError(w, http.StatusForbidden, "Token interno inválido")
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" || !emailRegex.MatchString(email) {
		httpx.RespondError(w, http.StatusBadRequest, "El formato del email es inválido")
		return
	}

	cliente, err := h.store.FindOne(func(c *Cliente) bool {
		return strings.EqualFold(c.Email, email)
	})
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if cliente == nil {
		httpx.RespondError(w, http.StatusNotFound, "No existe un cliente con ese email")
		return
	}

	httpx.RespondJSON(w, http.StatusOK, ClienteEmail{
		ID:     cliente.ID,
		Nombre: cliente.Nombre,
		Email:  cliente.Email,
	})
}

// handleList maneja GET /clientes.
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.GetAll()
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if items == nil {
		items = []Cliente{}
	}
	httpx.RespondJSON(w, http.StatusOK, items)
}

// handleGet maneja GET /clientes/{id}.
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de cliente inválido")
		return
	}

	cliente, err := h.store.GetByID(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if cliente == nil {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Cliente con ID %d no encontrado", id))
		return
	}

	httpx.RespondJSON(w, http.StatusOK, cliente)
}

// validateCliente valida los campos de un cliente.
func validateCliente(nombre, cc, email, direccion string) string {
	if nombre == "" {
		return "El nombre es obligatorio"
	}
	if cc == "" {
		return "La cédula (cc) es obligatoria"
	}
	if !ccRegex.MatchString(cc) {
		return "La cédula (cc) debe ser numérica"
	}
	if !emailRegex.MatchString(email) {
		return "El formato del email es inválido"
	}
	if direccion == "" {
		return "La dirección es obligatoria"
	}
	return ""
}

// handleCreate maneja POST /clientes.
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	req.CC = strings.TrimSpace(req.CC)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Direccion = strings.TrimSpace(req.Direccion)

	if msg := validateCliente(req.Nombre, req.CC, req.Email, req.Direccion); msg != "" {
		httpx.RespondError(w, http.StatusBadRequest, msg)
		return
	}

	// Verificar email y cédula únicos de forma atómica: la comprobación y la
	// escritura ocurren bajo el mismo lock del store.
	cliente := Cliente{
		Nombre:    req.Nombre,
		CC:        req.CC,
		Email:     req.Email,
		Direccion: req.Direccion,
	}

	cliente, err := h.store.CreateUnique(cliente, func(items []Cliente) error {
		for i := range items {
			if strings.EqualFold(items[i].Email, req.Email) {
				return errEmailDuplicado
			}
			if items[i].CC == req.CC {
				return errCCDuplicada
			}
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errEmailDuplicado):
			httpx.RespondError(w, http.StatusConflict, "Ya existe un cliente con ese email")
		case errors.Is(err, errCCDuplicada):
			httpx.RespondError(w, http.StatusConflict, "Ya existe un cliente con esa cédula")
		default:
			httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		}
		return
	}

	httpx.RespondCreated(w, cliente)
}

// handleUpdate maneja PUT /clientes/{id}.
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de cliente inválido")
		return
	}

	var req UpdateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	req.CC = strings.TrimSpace(req.CC)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Direccion = strings.TrimSpace(req.Direccion)

	if msg := validateCliente(req.Nombre, req.CC, req.Email, req.Direccion); msg != "" {
		httpx.RespondError(w, http.StatusBadRequest, msg)
		return
	}

	// Verificar email y cédula únicos (excluyendo el cliente actual) y escribir
	// en la misma operación atómica.
	updated, ok, err := h.store.UpdateUnique(id, func(c *Cliente) {
		c.Nombre = req.Nombre
		c.CC = req.CC
		c.Email = req.Email
		c.Direccion = req.Direccion
	}, func(items []Cliente, index int) error {
		for i := range items {
			if i == index {
				continue
			}
			if strings.EqualFold(items[i].Email, req.Email) {
				return errEmailDuplicado
			}
			if items[i].CC == req.CC {
				return errCCDuplicada
			}
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errEmailDuplicado):
			httpx.RespondError(w, http.StatusConflict, "Ya existe un cliente con ese email")
		case errors.Is(err, errCCDuplicada):
			httpx.RespondError(w, http.StatusConflict, "Ya existe un cliente con esa cédula")
		default:
			httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		}
		return
	}
	if !ok {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Cliente con ID %d no encontrado", id))
		return
	}

	httpx.RespondJSON(w, http.StatusOK, updated)
}

// handleDelete maneja DELETE /clientes/{id}.
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de cliente inválido")
		return
	}

	ok, err := h.store.Delete(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if !ok {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Cliente con ID %d no encontrado", id))
		return
	}

	httpx.RespondNoContent(w)
}
