package clientes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"facturacion-microservicios/internal/jsondb"
)

func nuevoHandlerPrueba(t *testing.T) (*Handler, *jsondb.Store[Cliente]) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "clientes.json")
	store, err := jsondb.New(path, 201,
		func(c *Cliente) int { return c.ID },
		func(c *Cliente, id int) { c.ID = id },
	)
	if err != nil {
		t.Fatalf("no se pudo crear el store: %v", err)
	}

	seed := []Cliente{
		{Nombre: "Carlos Pérez", CC: "1234567", Email: "carlos@example.com", Direccion: "Calle 123"},
		{Nombre: "Ana Torres", CC: "56789101", Email: "ana@example.com", Direccion: "Carrera 10"},
	}
	for _, c := range seed {
		if _, err := store.Create(c); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	return NewHandler(store), store
}

func muxDe(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func solicitar(t *testing.T, mux *http.ServeMux, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var decoded map[string]any
	if rec.Body.Len() > 0 && bytes.HasPrefix(bytes.TrimSpace(rec.Body.Bytes()), []byte("{")) {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("respuesta no es JSON: %v (%s)", err, rec.Body.String())
		}
	}

	return rec, decoded
}

func contarClientes(t *testing.T, store *jsondb.Store[Cliente]) int {
	t.Helper()

	items, err := store.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	return len(items)
}

func TestListarYConsultarClientes(t *testing.T) {
	handler, _ := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, _ := solicitar(t, mux, http.MethodGet, "/clientes", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", rec.Code)
	}

	var lista []Cliente
	if err := json.Unmarshal(rec.Body.Bytes(), &lista); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("se esperaban 2 clientes, hay %d", len(lista))
	}
	if lista[0].Email != "carlos@example.com" || lista[1].Email != "ana@example.com" {
		t.Fatalf("clientes inesperados: %+v", lista)
	}

	rec, body := solicitar(t, mux, http.MethodGet, "/clientes/201", nil)
	if rec.Code != http.StatusOK || body["nombre"] != "Carlos Pérez" || body["cc"] != "1234567" {
		t.Fatalf("detalle inesperado (%d): %s", rec.Code, rec.Body.String())
	}

	rec, _ = solicitar(t, mux, http.MethodGet, "/clientes/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("se esperaba 404, se obtuvo %d", rec.Code)
	}

	rec, _ = solicitar(t, mux, http.MethodGet, "/clientes/abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400, se obtuvo %d", rec.Code)
	}
}

func TestCrearCliente(t *testing.T) {
	handler, store := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, body := solicitar(t, mux, http.MethodPost, "/clientes", CreateRequest{
		Nombre:    "  Laura Díaz  ",
		CC:        "  1029384756  ",
		Email:     "Laura.Diaz@Example.com",
		Direccion: "  Calle 9 #1-23  ",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if body["id"] != float64(203) {
		t.Fatalf("id esperado 203, obtenido %v", body["id"])
	}
	if body["nombre"] != "Laura Díaz" {
		t.Fatalf("el nombre debe recortarse: %v", body["nombre"])
	}
	if body["email"] != "laura.diaz@example.com" {
		t.Fatalf("el email debe normalizarse a minúsculas: %v", body["email"])
	}
	if body["cc"] != "1029384756" || body["direccion"] != "Calle 9 #1-23" {
		t.Fatalf("datos recortados inesperados: %v", body)
	}
	if got := contarClientes(t, store); got != 3 {
		t.Fatalf("se esperaban 3 clientes, hay %d", got)
	}

	// Email duplicado (insensible a mayúsculas)
	rec, _ = solicitar(t, mux, http.MethodPost, "/clientes", CreateRequest{
		Nombre: "Otro Carlos", CC: "111222333", Email: "CARLOS@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("email duplicado: se esperaba 409, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}

	// Cédula duplicada
	rec, _ = solicitar(t, mux, http.MethodPost, "/clientes", CreateRequest{
		Nombre: "Nueva Ana", CC: "56789101", Email: "nueva.ana@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("cc duplicada: se esperaba 409, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}

	if got := contarClientes(t, store); got != 3 {
		t.Fatalf("no deben crearse clientes duplicados, hay %d", got)
	}
}

func TestCrearClienteValidaDatos(t *testing.T) {
	handler, store := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	casos := []struct {
		nombre string
		req    CreateRequest
	}{
		{"nombre vacío", CreateRequest{Nombre: "  ", CC: "123", Email: "a@b.com", Direccion: "X"}},
		{"cc vacía", CreateRequest{Nombre: "Ana", CC: "", Email: "a@b.com", Direccion: "X"}},
		{"cc no numérica", CreateRequest{Nombre: "Ana", CC: "abc123", Email: "a@b.com", Direccion: "X"}},
		{"email inválido", CreateRequest{Nombre: "Ana", CC: "123", Email: "sin-arroba", Direccion: "X"}},
		{"dirección vacía", CreateRequest{Nombre: "Ana", CC: "123", Email: "a@b.com", Direccion: "  "}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec, _ := solicitar(t, mux, http.MethodPost, "/clientes", caso.req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba 400, se obtuvo %d: %s", rec.Code, rec.Body.String())
			}
		})
	}

	if got := contarClientes(t, store); got != 2 {
		t.Fatalf("ningún caso inválido debe crear clientes, hay %d", got)
	}
}

func TestActualizarCliente(t *testing.T) {
	handler, store := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, body := solicitar(t, mux, http.MethodPut, "/clientes/201", UpdateRequest{
		Nombre:    "Carlos Pérez Gómez",
		CC:        "1234567",
		Email:     "carlos.gomez@example.com",
		Direccion: "Calle 123 #45-67",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if body["nombre"] != "Carlos Pérez Gómez" || body["email"] != "carlos.gomez@example.com" {
		t.Fatalf("actualización inesperada: %v", body)
	}

	guardado, err := store.GetByID(201)
	if err != nil || guardado == nil {
		t.Fatalf("get: %v", err)
	}
	if guardado.Direccion != "Calle 123 #45-67" {
		t.Fatalf("debe persistir la actualización: %+v", guardado)
	}

	// Email de otro cliente
	rec, _ = solicitar(t, mux, http.MethodPut, "/clientes/201", UpdateRequest{
		Nombre: "Carlos", CC: "1234567", Email: "ana@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("email duplicado: se esperaba 409, se obtuvo %d", rec.Code)
	}

	// CC de otro cliente
	rec, _ = solicitar(t, mux, http.MethodPut, "/clientes/201", UpdateRequest{
		Nombre: "Carlos", CC: "56789101", Email: "carlos.gomez@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("cc duplicada: se esperaba 409, se obtuvo %d", rec.Code)
	}

	// Datos inválidos
	rec, _ = solicitar(t, mux, http.MethodPut, "/clientes/201", UpdateRequest{
		Nombre: "  ", CC: "1234567", Email: "carlos.gomez@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400, se obtuvo %d", rec.Code)
	}

	// Cliente inexistente
	rec, _ = solicitar(t, mux, http.MethodPut, "/clientes/999", UpdateRequest{
		Nombre: "Nadie", CC: "999", Email: "nadie@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("se esperaba 404, se obtuvo %d", rec.Code)
	}

	// ID inválido
	rec, _ = solicitar(t, mux, http.MethodPut, "/clientes/abc", UpdateRequest{
		Nombre: "Nadie", CC: "999", Email: "nadie@example.com", Direccion: "X",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400, se obtuvo %d", rec.Code)
	}
}

func TestEliminarCliente(t *testing.T) {
	handler, store := nuevoHandlerPrueba(t)
	mux := muxDe(handler)

	rec, _ := solicitar(t, mux, http.MethodDelete, "/clientes/202", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("se esperaba 204, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if got := contarClientes(t, store); got != 1 {
		t.Fatalf("se esperaba 1 cliente restante, hay %d", got)
	}

	rec, _ = solicitar(t, mux, http.MethodGet, "/clientes/202", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("el cliente eliminado no debe existir: se esperaba 404, se obtuvo %d", rec.Code)
	}

	// Eliminar de nuevo
	rec, _ = solicitar(t, mux, http.MethodDelete, "/clientes/202", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("segundo delete: se esperaba 404, se obtuvo %d", rec.Code)
	}

	// ID inválido
	rec, _ = solicitar(t, mux, http.MethodDelete, "/clientes/abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba 400, se obtuvo %d", rec.Code)
	}
}
