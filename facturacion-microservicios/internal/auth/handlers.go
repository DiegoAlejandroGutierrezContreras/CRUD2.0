package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"facturacion-microservicios/internal/httpx"
	"facturacion-microservicios/internal/jsondb"
)

// emailRegex valida el formato básico de un email.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// errEmailDuplicado lo devuelve la validación atómica de CreateUnique.
var errEmailDuplicado = errors.New("ya existe un usuario con ese email")

// Handler maneja las rutas del servicio de autenticación.
type Handler struct {
	store    *jsondb.Store[Usuario]
	sessions *SessionStore
	clientes ClienteLookup
}

// NewHandler crea un handler con un almacén de sesiones en memoria y el TTL
// por defecto. No incluye la consulta a Clientes: útil en pruebas del rol
// admin o para tests aislados.
func NewHandler(store *jsondb.Store[Usuario]) *Handler {
	return NewHandlerWithSessions(store, NewSessionStore(SessionTTLPorDefecto))
}

// NewHandlerWithSessions permite inyectar un almacén de sesiones con un TTL
// distinto (útil en pruebas).
func NewHandlerWithSessions(store *jsondb.Store[Usuario], sessions *SessionStore) *Handler {
	return &Handler{store: store, sessions: sessions}
}

// NewHandlerConClientes crea un handler que cuando el rol es "cliente" exige
// que el email exista en el directorio de Clientes. Ese servicio aporta el
// nombre que el administrador cargó, nunca el usuario.
func NewHandlerConClientes(store *jsondb.Store[Usuario], sessions *SessionStore, clientes ClienteLookup) *Handler {
	return &Handler{store: store, sessions: sessions, clientes: clientes}
}

// RegisterRoutes registra las rutas del servicio.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/login", h.handleLogin)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)
	mux.HandleFunc("GET /auth/session", h.handleSession)
	mux.HandleFunc("GET /auth/usuarios/{id}", h.handleGetUsuario)
}

// handleRegister maneja POST /auth/register.
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	// Validaciones. La contraseña no se recorta: los espacios forman parte del
	// secreto y el login compara el valor exacto.
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !emailRegex.MatchString(req.Email) {
		httpx.RespondError(w, http.StatusBadRequest, "El formato del email es inválido")
		return
	}

	if len(req.Password) < 6 {
		httpx.RespondError(w, http.StatusBadRequest, "La contraseña debe tener al menos 6 caracteres")
		return
	}

	// Validar el rol: obligatorio y debe ser "admin" o "cliente".
	req.Rol = strings.ToLower(strings.TrimSpace(req.Rol))
	if !RolValido(req.Rol) {
		httpx.RespondError(w, http.StatusBadRequest, "El rol debe ser \"admin\" o \"cliente\"")
		return
	}

	// En el rol "cliente" el nombre no lo escribe el usuario: lo toma del
	// cliente que cargó el administrador. Cualquier nombre enviado en el body
	// se descarta a propósito.
	if req.Rol == RolCliente {
		if h.clientes == nil {
			httpx.RespondError(w, http.StatusInternalServerError, "El servicio de clientes no está configurado")
			return
		}
		cliente, err := h.clientes.BuscarPorEmail(req.Email)
		switch {
		case err == nil:
			req.Nombre = cliente.Nombre
		case errors.Is(err, ErrClienteNoEncontrado):
			httpx.RespondError(w, http.StatusBadRequest,
				"No existe un cliente con ese email. Pídele al administrador que lo registre antes de crear la cuenta.")
			return
		case errors.Is(err, ErrClientesNoDisponible):
			httpx.RespondError(w, http.StatusBadGateway, "No se pudo validar el correo con el servicio de clientes")
			return
		default:
			httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
			return
		}
	} else {
		req.Nombre = strings.TrimSpace(req.Nombre)
		if req.Nombre == "" {
			httpx.RespondError(w, http.StatusBadRequest, "El nombre es obligatorio")
			return
		}
	}

	usuario := Usuario{
		Nombre:   req.Nombre,
		Email:    req.Email,
		Password: req.Password,
		Rol:      req.Rol,
	}

	// La verificación de email único y la escritura se hacen bajo el mismo
	// lock del store, así dos registros concurrentes no pueden duplicar el email.
	usuario, err := h.store.CreateUnique(usuario, func(items []Usuario) error {
		for i := range items {
			if strings.EqualFold(items[i].Email, req.Email) {
				return errEmailDuplicado
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errEmailDuplicado) {
			httpx.RespondError(w, http.StatusConflict, "Ya existe un usuario con ese email")
			return
		}
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	httpx.RespondCreated(w, usuario.SinPassword())
}

// clientePorEmail delega en el lookup de Clientes y devuelve la vista mínima
// del cliente. Si el lookup no está configurado, lo reporta para que el
// registro de clientes no se complete a ciegas.
func (h *Handler) clientePorEmail(email string) (*ClienteRegistro, error) {
	if h.clientes == nil {
		return nil, ErrClientesNoDisponible
	}
	return h.clientes.BuscarPorEmail(email)
}

// handleLogin maneja POST /auth/login. Devuelve el usuario y un token opaco de
// sesión que el cliente debe enviar en la cabecera Authorization.
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		httpx.RespondError(w, http.StatusBadRequest, "Email y contraseña son obligatorios")
		return
	}

	// Buscar usuario por email
	usuario, err := h.store.FindOne(func(u *Usuario) bool {
		return strings.EqualFold(u.Email, req.Email)
	})
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	if usuario == nil {
		httpx.RespondError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	// Comparar contraseña en texto plano (simplificación académica).
	if subtle.ConstantTimeCompare([]byte(usuario.Password), []byte(req.Password)) != 1 {
		httpx.RespondError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	token, expira, err := h.sessions.Create(usuario.ID)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "No se pudo iniciar la sesión")
		return
	}

	httpx.RespondJSON(w, http.StatusOK, LoginResponse{
		Usuario: usuario.SinPassword(),
		Token:   token,
		Expira:  expira,
	})
}

// handleLogout maneja POST /auth/logout. Invalida el token recibido.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := httpx.BearerToken(r); token != "" {
		h.sessions.Delete(token)
	}
	httpx.RespondNoContent(w)
}

// handleSession maneja GET /auth/session. Es el endpoint que usan los demás
// microservicios para validar un token y obtener el usuario autenticado.
func (h *Handler) handleSession(w http.ResponseWriter, r *http.Request) {
	token := httpx.BearerToken(r)
	if token == "" {
		httpx.RespondError(w, http.StatusUnauthorized, "Token de sesión ausente")
		return
	}

	sesion, ok := h.sessions.Get(token)
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, "Sesión inválida o expirada")
		return
	}

	usuario, err := h.store.GetByID(sesion.UsuarioID)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if usuario == nil {
		// El usuario fue eliminado: la sesión deja de ser válida.
		h.sessions.Delete(token)
		httpx.RespondError(w, http.StatusUnauthorized, "Sesión inválida o expirada")
		return
	}

	httpx.RespondJSON(w, http.StatusOK, usuario.SinPassword())
}

// handleGetUsuario maneja GET /auth/usuarios/{id}.
// Es un endpoint interno heredado del flujo anterior a las sesiones; se
// mantiene para compatibilidad con las pruebas de facturación. En producción la
// facturación toma el emisor de la sesión y no llama a esta ruta.
func (h *Handler) handleGetUsuario(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de usuario inválido")
		return
	}

	usuario, err := h.store.GetByID(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if usuario == nil {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Usuario con ID %d no encontrado", id))
		return
	}

	httpx.RespondJSON(w, http.StatusOK, usuario.SinPassword())
}
