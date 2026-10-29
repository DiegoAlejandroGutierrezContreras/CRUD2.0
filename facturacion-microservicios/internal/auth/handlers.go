package auth

import (
	"crypto/subtle"
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

// Handler maneja las rutas del servicio de autenticación.
type Handler struct {
	store *jsondb.Store[Usuario]
}

// NewHandler crea un nuevo handler de autenticación.
func NewHandler(store *jsondb.Store[Usuario]) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registra las rutas del servicio.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/login", h.handleLogin)
	mux.HandleFunc("GET /auth/usuarios/{id}", h.handleGetUsuario)
}

// handleRegister maneja POST /auth/register.
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	// Validaciones
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Nombre == "" {
		httpx.RespondError(w, http.StatusBadRequest, "El nombre es obligatorio")
		return
	}

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

	// Verificar email único
	existing, err := h.store.FindOne(func(u *Usuario) bool {
		return strings.EqualFold(u.Email, req.Email)
	})
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if existing != nil {
		httpx.RespondError(w, http.StatusConflict, "Ya existe un usuario con ese email")
		return
	}

	usuario := Usuario{
		Nombre:   req.Nombre,
		Email:    req.Email,
		Password: req.Password,
		Rol:      req.Rol,
	}

	usuario, err = h.store.Create(usuario)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	httpx.RespondCreated(w, usuario.SinPassword())
}

// handleLogin maneja POST /auth/login.
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

	// Comparar contraseña en texto plano (simplificación académica, sin tokens).
	if subtle.ConstantTimeCompare([]byte(usuario.Password), []byte(req.Password)) != 1 {
		httpx.RespondError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	httpx.RespondJSON(w, http.StatusOK, usuario.SinPassword())
}

// handleGetUsuario maneja GET /auth/usuarios/{id}.
// Es usado por el servicio de facturación para registrar quién emitió la factura.
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
