package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// SessionTTLPorDefecto es la duración de una sesión cuando no se configura otra.
const SessionTTLPorDefecto = 8 * time.Hour

// Sesion es el registro interno de un token activo. No se serializa: las
// sesiones viven únicamente en memoria del proceso de autenticación.
type Sesion struct {
	UsuarioID int
	Expira    time.Time
}

// SessionStore guarda los tokens opacos activos en memoria.
//
// El token es aleatorio (32 bytes de crypto/rand en base64url), no contiene
// información del usuario y no es un JWT. Al reiniciar el servicio se pierden
// todas las sesiones y los usuarios deben volver a iniciar sesión.
type SessionStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	tokens map[string]Sesion
}

// NewSessionStore crea un almacén de sesiones con el TTL indicado.
// Si ttl es menor o igual a cero se usa SessionTTLPorDefecto.
func NewSessionStore(ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = SessionTTLPorDefecto
	}
	return &SessionStore{
		ttl:    ttl,
		tokens: make(map[string]Sesion),
	}
}

// Create genera un token nuevo para el usuario y devuelve también su fecha de
// expiración. Reintenta si por azar el token ya existiera.
func (s *SessionStore) Create(usuarioID int) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ahora := time.Now()
	s.limpiarExpiradas(ahora)

	for intento := 0; intento < 3; intento++ {
		token, err := nuevoToken()
		if err != nil {
			return "", time.Time{}, err
		}
		if _, existe := s.tokens[token]; existe {
			continue
		}

		expira := ahora.Add(s.ttl).UTC()
		s.tokens[token] = Sesion{UsuarioID: usuarioID, Expira: expira}
		return token, expira, nil
	}

	return "", time.Time{}, errors.New("no se pudo generar un token de sesión único")
}

// Get devuelve la sesión asociada al token. Si el token no existe o ya expiró,
// retorna false (y elimina el registro expirado).
func (s *SessionStore) Get(token string) (Sesion, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sesion, existe := s.tokens[token]
	if !existe {
		return Sesion{}, false
	}

	if time.Now().After(sesion.Expira) {
		delete(s.tokens, token)
		return Sesion{}, false
	}

	return sesion, true
}

// Delete invalida un token. Es idempotente.
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

// Len informa cuántas sesiones activas hay. Se usa en pruebas y diagnóstico.
func (s *SessionStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tokens)
}

// limpiarExpiradas elimina los tokens vencidos. Debe llamarse con el lock tomado.
func (s *SessionStore) limpiarExpiradas(ahora time.Time) {
	for token, sesion := range s.tokens {
		if ahora.After(sesion.Expira) {
			delete(s.tokens, token)
		}
	}
}

// nuevoToken genera 32 bytes aleatorios codificados en base64url sin relleno.
func nuevoToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
