package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"facturacion-microservicios/internal/jsondb"
)

func nuevoHandlerPrueba(t *testing.T) *Handler {
	t.Helper()

	path := filepath.Join(t.TempDir(), "usuarios.json")
	store, err := jsondb.New(path, 1,
		func(u *Usuario) int { return u.ID },
		func(u *Usuario, id int) { u.ID = id },
	)
	if err != nil {
		t.Fatalf("no se pudo crear el store: %v", err)
	}

	if _, err := store.Create(Usuario{
		Nombre:   "Ricardo Gómez",
		Email:    "ricardo@example.com",
		Password: "admin123",
		Rol:      RolAdmin,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := NewHandler(store)
	return handler
}

func muxDe(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func solicitar(t *testing.T, mux *http.ServeMux, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var decoded map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("respuesta no es JSON: %v (%s)", err, rec.Body.String())
		}
	}

	return rec, decoded
}

func TestRegistroCreaUsuarioConRolSinPassword(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, body := solicitar(t, mux, http.MethodPost, "/auth/register", RegisterRequest{
		Nombre:   "  Laura Díaz  ",
		Email:    "Laura.Diaz@Example.com",
		Password: "secreta123",
		Rol:      " cliente ",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if _, existe := body["password"]; existe {
		t.Fatalf("la respuesta no debe incluir password: %s", rec.Body.String())
	}
	if body["rol"] != RolCliente {
		t.Fatalf("la respuesta debe incluir el rol normalizado: %s", rec.Body.String())
	}
	if body["nombre"] != "Laura Díaz" {
		t.Fatalf("nombre inesperado: %v", body["nombre"])
	}
	if body["email"] != "laura.diaz@example.com" {
		t.Fatalf("el email debe normalizarse a minúsculas: %v", body["email"])
	}

	usuarios, err := handler.store.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(usuarios) != 2 {
		t.Fatalf("se esperaban 2 usuarios, hay %d", len(usuarios))
	}
	if usuarios[1].Password != "secreta123" {
		t.Fatalf("la contraseña debe guardarse tal cual (texto plano): %q", usuarios[1].Password)
	}
	if usuarios[1].Rol != RolCliente {
		t.Fatalf("el rol debe guardarse: %+v", usuarios[1])
	}
}

func TestRegistroNormalizaRolAdmin(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, _ := solicitar(t, mux, http.MethodPost, "/auth/register", RegisterRequest{
		Nombre: "Otro Admin", Email: "admin2@example.com", Password: "secreta123", Rol: " ADMIN ",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}

	guardado, err := handler.store.FindOne(func(u *Usuario) bool { return u.Email == "admin2@example.com" })
	if err != nil || guardado == nil {
		t.Fatalf("find: %v", err)
	}
	if guardado.Rol != RolAdmin {
		t.Fatalf("el rol debe normalizarse a minúsculas: %+v", guardado)
	}
}

func TestRegistroRechazaRolInvalido(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	casos := []struct {
		nombre string
		rol    string
	}{
		{"rol vacío", ""},
		{"rol desconocido", "superusuario"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec, _ := solicitar(t, mux, http.MethodPost, "/auth/register", RegisterRequest{
				Nombre: "Ana", Email: "ana@example.com", Password: "123456", Rol: caso.rol,
			})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba 400, se obtuvo %d: %s", rec.Code, rec.Body.String())
			}
		})
	}

	usuarios, err := handler.store.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(usuarios) != 1 {
		t.Fatalf("ningún usuario con rol inválido debe crearse, hay %d", len(usuarios))
	}
}

func TestRegistroRechazaEmailDuplicado(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, _ := solicitar(t, mux, http.MethodPost, "/auth/register", RegisterRequest{
		Nombre:   "Otro Ricardo",
		Email:    "RICARDO@example.com",
		Password: "otra123",
		Rol:      RolCliente,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegistroValidaDatos(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	casos := []struct {
		nombre string
		req    RegisterRequest
	}{
		{"nombre vacío", RegisterRequest{Nombre: "  ", Email: "a@b.com", Password: "123456", Rol: RolAdmin}},
		{"email inválido", RegisterRequest{Nombre: "Ana", Email: "sin-arroba", Password: "123456", Rol: RolAdmin}},
		{"password corta", RegisterRequest{Nombre: "Ana", Email: "ana@example.com", Password: "123", Rol: RolAdmin}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec, _ := solicitar(t, mux, http.MethodPost, "/auth/register", caso.req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba 400, se obtuvo %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLoginCorrectoDevuelveUsuarioSinPassword(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, body := solicitar(t, mux, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "Ricardo@Example.com",
		Password: "admin123",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if _, existe := body["password"]; existe {
		t.Fatalf("la respuesta no debe incluir password: %s", rec.Body.String())
	}
	if _, existe := body["token"]; existe {
		t.Fatalf("la respuesta no debe incluir token: %s", rec.Body.String())
	}
	if body["nombre"] != "Ricardo Gómez" {
		t.Fatalf("usuario inesperado: %v", body)
	}
	if body["rol"] != RolAdmin {
		t.Fatalf("el login debe retornar el rol: %s", rec.Body.String())
	}
}

func TestLoginRechazaCredencialesInvalidas(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	casos := []struct {
		nombre string
		req    LoginRequest
	}{
		{"password incorrecta", LoginRequest{Email: "ricardo@example.com", Password: "incorrecta"}},
		{"usuario inexistente", LoginRequest{Email: "nadie@example.com", Password: "admin123"}},
		{"campos vacíos", LoginRequest{Email: "", Password: ""}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec, _ := solicitar(t, mux, http.MethodPost, "/auth/login", caso.req)
			esperado := http.StatusUnauthorized
			if caso.req.Email == "" && caso.req.Password == "" {
				esperado = http.StatusBadRequest
			}
			if rec.Code != esperado {
				t.Fatalf("se esperaba %d, se obtuvo %d: %s", esperado, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetUsuarioPorID(t *testing.T) {
	handler := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, body := solicitar(t, mux, http.MethodGet, "/auth/usuarios/1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", rec.Code)
	}
	if body["email"] != "ricardo@example.com" {
		t.Fatalf("usuario inesperado: %v", body)
	}
	if body["rol"] != RolAdmin {
		t.Fatalf("debe incluir el rol: %v", body)
	}
	if _, existe := body["password"]; existe {
		t.Fatalf("no debe exponer password: %s", rec.Body.String())
	}

	rec, _ = solicitar(t, mux, http.MethodGet, "/auth/usuarios/99", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("se esperaba 404, se obtuvo %d", rec.Code)
	}

	rec, _ = solicitar(t, mux, http.MethodGet, "/auth/usuarios/abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400, se obtuvo %d", rec.Code)
	}
}
