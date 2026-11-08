// Consulta del servicio de Clientes para completar los datos que el
// usuario no introduce: en el registro de un cliente, el nombre viene del
// directorio que administra el administrador.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClienteRegistro es la vista mínima de un cliente que devuelve el servicio
// de Clientes. Solo se exponen los datos necesarios para crear el usuario.
type ClienteRegistro struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}

// ClienteLookup busca un cliente existente a partir de su email. Es la interfaz
// que Auth usa para no poder crear cuentas de clientes con correos que el
// administrador no haya dado de alta en el directorio de clientes.
type ClienteLookup interface {
	BuscarPorEmail(email string) (*ClienteRegistro, error)
}

// Errores que el registro traduce directamente a respuestas HTTP.
var (
	// ErrClienteNoEncontrado indica que el email no está en Clientes.
	ErrClienteNoEncontrado = errors.New("no existe un cliente con ese email")
	// ErrClientesNoDisponible indica que la llamada al servicio de Clientes falló.
	ErrClientesNoDisponible = errors.New("servicio de clientes no disponible")
)

// ClientesRegistroHTTP implementa ClienteLookup contra el servicio de Clientes.
// El token interno evita exponer públicamente la consulta de correos.
type ClientesRegistroHTTP struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClientesRegistro crea un cliente HTTP para el servicio de Clientes.
// El token se envía como cabecera X-Internal-Token.
func NewClientesRegistro(baseURL, token string) *ClientesRegistroHTTP {
	return &ClientesRegistroHTTP{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// BuscarPorEmail consulta GET {baseURL}/clientes/por-email?email=....
func (c *ClientesRegistroHTTP) BuscarPorEmail(email string) (*ClienteRegistro, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		c.baseURL+"/clientes/por-email?email="+url.QueryEscape(email),
		nil,
	)
	if err != nil {
		return nil, ErrClientesNoDisponible
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrClientesNoDisponible
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrClienteNoEncontrado
	case resp.StatusCode != http.StatusOK:
		return nil, ErrClientesNoDisponible
	}

	var cliente ClienteRegistro
	if err := json.NewDecoder(resp.Body).Decode(&cliente); err != nil {
		return nil, ErrClientesNoDisponible
	}
	if strings.TrimSpace(cliente.Nombre) == "" {
		return nil, ErrClientesNoDisponible
	}
	return &cliente, nil
}
