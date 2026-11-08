// Package auth implementa la lógica del servicio de autenticación.
package auth

import "time"

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
// El rol (admin o cliente) lo aplica el backend en cada microservicio: las
// mutaciones de catálogo y clientes, y la emisión de facturas, exigen admin.
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

// LoginResponse es la respuesta de POST /auth/login. Incluye el usuario sin
// contraseña y el token opaco que el frontend debe enviar como
// "Authorization: Bearer <token>" en cada petición autenticada.
type LoginResponse struct {
	Usuario UsuarioSinPassword `json:"usuario"`
	Token   string             `json:"token"`
	Expira  time.Time          `json:"expira"`
}

// RegisterRequest es el body de POST /auth/register.
// En rol "cliente" no se envía nombre: Auth lo obtiene del directorio de
// Clientes a partir del email.
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
