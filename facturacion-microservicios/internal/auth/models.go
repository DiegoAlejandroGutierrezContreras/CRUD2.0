// Package auth implementa la lógica del servicio de autenticación.
package auth

// Roles permitidos para un usuario del sistema.
const (
	RolAdmin   = "admin"
	RolCliente = "cliente"
)

// RolValido indica si el rol recibido es uno de los valores permitidos.
func RolValido(rol string) bool {
	return rol == RolAdmin || rol == RolCliente
}

// Usuario representa un usuario del sistema.
// El rol (admin o cliente) solo controla la interfaz del frontend:
// la API no exige autorización por rol.
type Usuario struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Rol      string `json:"rol"`
}

// UsuarioSinPassword es la vista pública del usuario, sin el campo password.
type UsuarioSinPassword struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

// LoginRequest es el body de POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest es el body de POST /auth/register.
type RegisterRequest struct {
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Rol      string `json:"rol"`
}

// SinPassword convierte un Usuario a UsuarioSinPassword.
func (u *Usuario) SinPassword() UsuarioSinPassword {
	return UsuarioSinPassword{
		ID:     u.ID,
		Nombre: u.Nombre,
		Email:  u.Email,
		Rol:    u.Rol,
	}
}
