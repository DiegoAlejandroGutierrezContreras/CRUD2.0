// Package authclient valida tokens de sesión contra el servicio de
// autenticación y expone middlewares HTTP para proteger rutas.
//
// Cada microservicio que necesita saber quién llama consulta
// GET {AUTH_URL}/auth/session con la cabecera Authorization del request
// original. No hay JWT: el token es opaco y el servicio de autenticación es la
// única fuente de verdad sobre si sigue activo.
package authclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"facturacion-microservicios/internal/httpx"
)

// Usuario es la vista pública del usuario autenticado que devuelve
// GET /auth/session.
type Usuario struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

// RolAdmin es el rol con permisos de escritura.
const RolAdmin = "admin"

var (
	// ErrNoAutorizado indica token ausente, inválido o expirado.
	ErrNoAutorizado = errors.New("token de sesión ausente o inválido")
	// ErrNoDisponible indica que el servicio de autenticación no respondió.
	ErrNoDisponible = errors.New("servicio de autenticación no disponible")
)

// Client consulta el servicio de autenticación.
type Client struct {
	baseURL string
	http    *http.Client
}

// New crea un cliente para el servicio de autenticación indicado.
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Validar obtiene el usuario asociado al token del request.
func (c *Client) Validar(r *http.Request) (*Usuario, error) {
	token := httpx.BearerToken(r)
	if token == "" {
		return nil, ErrNoAutorizado
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, c.baseURL+"/auth/session", nil)
	if err != nil {
		return nil, ErrNoDisponible
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrNoDisponible
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrNoAutorizado
	default:
		return nil, ErrNoDisponible
	}

	var usuario Usuario
	if err := json.NewDecoder(resp.Body).Decode(&usuario); err != nil {
		return nil, ErrNoDisponible
	}

	return &usuario, nil
}

// Authenticate exige un token válido y guarda el usuario en el contexto.
func (c *Client) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usuario, err := c.Validar(r)
		if err != nil {
			if errors.Is(err, ErrNoAutorizado) {
				httpx.RespondError(w, http.StatusUnauthorized,
					"Sesión inválida o expirada. Inicia sesión nuevamente.")
				return
			}
			httpx.RespondError(w, http.StatusBadGateway,
				"No se pudo validar la sesión con el servicio de autenticación.")
			return
		}

		ctx := context.WithValue(r.Context(), usuarioCtxKey{}, usuario)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin exige sesión válida y rol admin.
func (c *Client) RequireAdmin(next http.Handler) http.Handler {
	return c.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usuario, ok := UsuarioFromContext(r.Context())
		if !ok || usuario.Rol != RolAdmin {
			httpx.RespondError(w, http.StatusForbidden, "Se requieren permisos de administrador")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// Route describe una ruta y si requiere rol admin.
type Route struct {
	Pattern   string
	Handler   http.HandlerFunc
	AdminOnly bool
}

// Registrar inscribe las rutas en el mux aplicando autenticación. Si el
// cliente es nil, las rutas quedan abiertas: ese modo solo se usa en pruebas
// unitarias que verifican la lógica de negocio sin montar el servicio de auth.
func Registrar(mux *http.ServeMux, c *Client, routes []Route) {
	for _, route := range routes {
		var handler http.Handler = route.Handler
		if c != nil {
			if route.AdminOnly {
				handler = c.RequireAdmin(handler)
			} else {
				handler = c.Authenticate(handler)
			}
		}
		mux.Handle(route.Pattern, handler)
	}
}

// UsuarioFromContext recupera el usuario que dejó el middleware.
func UsuarioFromContext(ctx context.Context) (*Usuario, bool) {
	usuario, ok := ctx.Value(usuarioCtxKey{}).(*Usuario)
	return usuario, ok
}

type usuarioCtxKey struct{}
