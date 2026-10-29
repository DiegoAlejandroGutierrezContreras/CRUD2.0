package facturacion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"facturacion-microservicios/internal/auth"
	"facturacion-microservicios/internal/clientes"
	"facturacion-microservicios/internal/jsondb"
	"facturacion-microservicios/internal/productos"
)

type entorno struct {
	authSrv   *httptest.Server
	prodSrv   *httptest.Server
	cliSrv    *httptest.Server
	factSrv   *httptest.Server
	usuarios  *jsondb.Store[auth.Usuario]
	productos *jsondb.Store[productos.Producto]
	clientes  *jsondb.Store[clientes.Cliente]
	facturas  *jsondb.Store[Factura]
}

func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()

	dir := t.TempDir()

	usuarios, err := jsondb.New(filepath.Join(dir, "usuarios.json"), 1,
		func(u *auth.Usuario) int { return u.ID },
		func(u *auth.Usuario, id int) { u.ID = id },
	)
	if err != nil {
		t.Fatalf("store usuarios: %v", err)
	}
	if _, err := usuarios.Create(auth.Usuario{
		Nombre: "Ricardo Gómez", Email: "ricardo@example.com", Password: "admin123", Rol: auth.RolAdmin,
	}); err != nil {
		t.Fatalf("seed usuario: %v", err)
	}

	productosStore, err := jsondb.New(filepath.Join(dir, "productos.json"), 101,
		func(p *productos.Producto) int { return p.ID },
		func(p *productos.Producto, id int) { p.ID = id },
	)
	if err != nil {
		t.Fatalf("store productos: %v", err)
	}
	for _, p := range []productos.Producto{
		{Nombre: "Laptop Lenovo ThinkPad", Precio: 3500000, Stock: 14},
		{Nombre: "Mouse inalámbrico Logitech", Precio: 12000, Stock: 47},
		{Nombre: "Monitor Samsung 24''", Precio: 80000, Stock: 18},
	} {
		if _, err := productosStore.Create(p); err != nil {
			t.Fatalf("seed producto: %v", err)
		}
	}

	clientesStore, err := jsondb.New(filepath.Join(dir, "clientes.json"), 201,
		func(c *clientes.Cliente) int { return c.ID },
		func(c *clientes.Cliente, id int) { c.ID = id },
	)
	if err != nil {
		t.Fatalf("store clientes: %v", err)
	}
	for _, c := range []clientes.Cliente{
		{Nombre: "Carlos Pérez", CC: "1234567", Email: "carlos@example.com", Direccion: "Calle 123"},
		{Nombre: "Ana Torres", CC: "56789101", Email: "ana@example.com", Direccion: "Carrera 10"},
	} {
		if _, err := clientesStore.Create(c); err != nil {
			t.Fatalf("seed cliente: %v", err)
		}
	}

	facturasStore, err := jsondb.New(filepath.Join(dir, "facturas.json"), 5001,
		func(f *Factura) int { return f.FacturaID },
		func(f *Factura, id int) { f.FacturaID = id },
	)
	if err != nil {
		t.Fatalf("store facturas: %v", err)
	}

	authMux := http.NewServeMux()
	auth.NewHandler(usuarios).RegisterRoutes(authMux)
	authSrv := httptest.NewServer(authMux)

	prodMux := http.NewServeMux()
	productos.NewHandler(productosStore).RegisterRoutes(prodMux)
	prodSrv := httptest.NewServer(prodMux)

	cliMux := http.NewServeMux()
	clientes.NewHandler(clientesStore).RegisterRoutes(cliMux)
	cliSrv := httptest.NewServer(cliMux)

	factMux := http.NewServeMux()
	NewHandler(facturasStore, ServiceURLs{
		AuthURL:      authSrv.URL,
		ProductosURL: prodSrv.URL,
		ClientesURL:  cliSrv.URL,
	}).RegisterRoutes(factMux)
	factSrv := httptest.NewServer(factMux)

	t.Cleanup(func() {
		factSrv.Close()
		cliSrv.Close()
		prodSrv.Close()
		authSrv.Close()
	})

	return &entorno{
		authSrv:   authSrv,
		prodSrv:   prodSrv,
		cliSrv:    cliSrv,
		factSrv:   factSrv,
		usuarios:  usuarios,
		productos: productosStore,
		clientes:  clientesStore,
		facturas:  facturasStore,
	}
}

func enviarFactura(t *testing.T, url string, req CreateFacturaRequest) (*http.Response, []byte) {
	t.Helper()

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := http.Post(url+"/facturas", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("leer body: %v", err)
	}

	return resp, buf.Bytes()
}

func stockDe(t *testing.T, store *jsondb.Store[productos.Producto], id int) int {
	t.Helper()

	p, err := store.GetByID(id)
	if err != nil || p == nil {
		t.Fatalf("producto %d no encontrado: %v", id, err)
	}
	return p.Stock
}

func TestCrearFacturaDescuentaStock(t *testing.T) {
	e := nuevoEntorno(t)

	resp, body := enviarFactura(t, e.factSrv.URL, CreateFacturaRequest{
		UsuarioID: 1,
		ClienteID: 201,
		Items: []CreateFacturaItem{
			{ProductoID: 101, Cantidad: 1},
			{ProductoID: 102, Cantidad: 2},
		},
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d: %s", resp.StatusCode, body)
	}

	var factura Factura
	if err := json.Unmarshal(body, &factura); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if factura.FacturaID != 5001 {
		t.Fatalf("factura_id esperado 5001, obtenido %d", factura.FacturaID)
	}
	if factura.Cliente.ID != 201 || factura.Cliente.Nombre != "Carlos Pérez" || factura.Cliente.Email != "carlos@example.com" {
		t.Fatalf("cliente inesperado: %+v", factura.Cliente)
	}
	if factura.Usuario.ID != 1 || factura.Usuario.Nombre != "Ricardo Gómez" {
		t.Fatalf("usuario inesperado: %+v", factura.Usuario)
	}
	if factura.Usuario.Rol != auth.RolAdmin {
		t.Fatalf("la factura debe incluir el rol del usuario emisor: %+v", factura.Usuario)
	}
	if factura.Total != 3524000 {
		t.Fatalf("total esperado 3524000, obtenido %v", factura.Total)
	}
	if len(factura.Fecha) != len("2006-01-02") {
		t.Fatalf("fecha con formato inesperado: %q", factura.Fecha)
	}
	if factura.Fecha != time.Now().Format("2006-01-02") {
		t.Fatalf("fecha esperada %s, obtenida %s", time.Now().Format("2006-01-02"), factura.Fecha)
	}
	if len(factura.Productos) != 2 {
		t.Fatalf("se esperaban 2 items, hay %d", len(factura.Productos))
	}

	if got := stockDe(t, e.productos, 101); got != 13 {
		t.Fatalf("stock 101 esperado 13, obtenido %d", got)
	}
	if got := stockDe(t, e.productos, 102); got != 45 {
		t.Fatalf("stock 102 esperado 45, obtenido %d", got)
	}

	guardadas, err := e.facturas.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(guardadas) != 1 {
		t.Fatalf("se esperaba 1 factura guardada, hay %d", len(guardadas))
	}
}

func TestCrearFacturaRechazaStockInsuficiente(t *testing.T) {
	e := nuevoEntorno(t)

	resp, body := enviarFactura(t, e.factSrv.URL, CreateFacturaRequest{
		UsuarioID: 1,
		ClienteID: 201,
		Items: []CreateFacturaItem{
			{ProductoID: 101, Cantidad: 999},
		},
	})

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d: %s", resp.StatusCode, body)
	}

	guardadas, err := e.facturas.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(guardadas) != 0 {
		t.Fatalf("no debe quedar factura creada, hay %d", len(guardadas))
	}
	if got := stockDe(t, e.productos, 101); got != 14 {
		t.Fatalf("el stock no debió cambiar, está en %d", got)
	}
}

func TestCrearFacturaRevierteSiElDescuentoFalla(t *testing.T) {
	e := nuevoEntorno(t)

	// Servicio de productos inconsistente: informa stock suficiente en GET pero
	// rechaza el descuento, simulando una carrera con otra factura.
	prodMux := http.NewServeMux()
	prodMux.HandleFunc("GET /productos/101", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":101,"nombre":"Laptop Lenovo ThinkPad","precio":3500000,"stock":5}`)
	})
	prodMux.HandleFunc("POST /productos/descontar-stock", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"error":"Stock insuficiente"}`)
	})
	prodSrv := httptest.NewServer(prodMux)
	defer prodSrv.Close()

	factMux := http.NewServeMux()
	NewHandler(e.facturas, ServiceURLs{
		AuthURL:      e.authSrv.URL,
		ProductosURL: prodSrv.URL,
		ClientesURL:  e.cliSrv.URL,
	}).RegisterRoutes(factMux)
	factSrv := httptest.NewServer(factMux)
	defer factSrv.Close()

	resp, body := enviarFactura(t, factSrv.URL, CreateFacturaRequest{
		UsuarioID: 1,
		ClienteID: 201,
		Items:     []CreateFacturaItem{{ProductoID: 101, Cantidad: 3}},
	})

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d: %s", resp.StatusCode, body)
	}

	guardadas, err := e.facturas.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(guardadas) != 0 {
		t.Fatalf("la factura debió eliminarse tras fallar el stock, hay %d", len(guardadas))
	}
}

func TestCrearFacturaValidaReferencias(t *testing.T) {
	e := nuevoEntorno(t)

	casos := []struct {
		nombre   string
		req      CreateFacturaRequest
		esperado int
	}{
		{
			"sin usuario_id",
			CreateFacturaRequest{ClienteID: 201, Items: []CreateFacturaItem{{ProductoID: 101, Cantidad: 1}}},
			http.StatusBadRequest,
		},
		{
			"sin items",
			CreateFacturaRequest{UsuarioID: 1, ClienteID: 201},
			http.StatusBadRequest,
		},
		{
			"cantidad cero",
			CreateFacturaRequest{UsuarioID: 1, ClienteID: 201, Items: []CreateFacturaItem{{ProductoID: 101, Cantidad: 0}}},
			http.StatusBadRequest,
		},
		{
			"usuario inexistente",
			CreateFacturaRequest{UsuarioID: 99, ClienteID: 201, Items: []CreateFacturaItem{{ProductoID: 101, Cantidad: 1}}},
			http.StatusNotFound,
		},
		{
			"cliente inexistente",
			CreateFacturaRequest{UsuarioID: 1, ClienteID: 999, Items: []CreateFacturaItem{{ProductoID: 101, Cantidad: 1}}},
			http.StatusNotFound,
		},
		{
			"producto inexistente",
			CreateFacturaRequest{UsuarioID: 1, ClienteID: 201, Items: []CreateFacturaItem{{ProductoID: 999, Cantidad: 1}}},
			http.StatusNotFound,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resp, body := enviarFactura(t, e.factSrv.URL, caso.req)
			if resp.StatusCode != caso.esperado {
				t.Fatalf("se esperaba %d, se obtuvo %d: %s", caso.esperado, resp.StatusCode, body)
			}
		})
	}

	guardadas, err := e.facturas.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(guardadas) != 0 {
		t.Fatalf("no debe haber facturas guardadas, hay %d", len(guardadas))
	}
}

func TestListarYConsultarFacturas(t *testing.T) {
	e := nuevoEntorno(t)

	for _, clienteID := range []int{201, 202} {
		resp, body := enviarFactura(t, e.factSrv.URL, CreateFacturaRequest{
			UsuarioID: 1,
			ClienteID: clienteID,
			Items:     []CreateFacturaItem{{ProductoID: 102, Cantidad: 1}},
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("crear factura: %d %s", resp.StatusCode, body)
		}
	}

	resp, err := http.Get(e.factSrv.URL + "/facturas")
	if err != nil {
		t.Fatalf("get lista: %v", err)
	}
	defer resp.Body.Close()

	var lista []Factura
	if err := json.NewDecoder(resp.Body).Decode(&lista); err != nil {
		t.Fatalf("decode lista: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("se esperaban 2 facturas, hay %d", len(lista))
	}
	if lista[0].FacturaID != 5002 || lista[1].FacturaID != 5001 {
		t.Fatalf("orden inesperado: %d, %d", lista[0].FacturaID, lista[1].FacturaID)
	}

	respDetalle, err := http.Get(e.factSrv.URL + "/facturas/5002")
	if err != nil {
		t.Fatalf("get detalle: %v", err)
	}
	defer respDetalle.Body.Close()
	if respDetalle.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", respDetalle.StatusCode)
	}

	respNoExiste, err := http.Get(e.factSrv.URL + "/facturas/9999")
	if err != nil {
		t.Fatalf("get inexistente: %v", err)
	}
	defer respNoExiste.Body.Close()
	if respNoExiste.StatusCode != http.StatusNotFound {
		t.Fatalf("se esperaba 404, se obtuvo %d", respNoExiste.StatusCode)
	}
}

func TestCalcularTotalRedondea(t *testing.T) {
	items := []FacturaItem{
		{Cantidad: 3, PrecioUnitario: 0.1, Subtotal: 0.3},
		{Cantidad: 1, PrecioUnitario: 0.2, Subtotal: 0.2},
	}
	if got := CalculateTotal(items); got != 0.5 {
		t.Fatalf("total esperado 0.5, obtenido %v", got)
	}
}

func TestListarFacturasFiltraPorEmailCliente(t *testing.T) {
	e := nuevoEntorno(t)

	for _, clienteID := range []int{201, 202} {
		resp, body := enviarFactura(t, e.factSrv.URL, CreateFacturaRequest{
			UsuarioID: 1,
			ClienteID: clienteID,
			Items:     []CreateFacturaItem{{ProductoID: 102, Cantidad: 1}},
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("crear factura: %d %s", resp.StatusCode, body)
		}
	}

	casos := []struct {
		nombre   string
		email    string
		esperado int
		factura  int
	}{
		{"sin filtro ve todas", "", 2, 0},
		{"correo en mayúsculas", "CARLOS@EXAMPLE.COM", 1, 5001},
		{"correo del otro cliente", "ana@example.com", 1, 5002},
		{"correo sin facturas", "nadie@example.com", 0, 0},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			url := e.factSrv.URL + "/facturas"
			if caso.email != "" {
				url += "?email=" + caso.email
			}
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
			}

			var lista []Factura
			if err := json.NewDecoder(resp.Body).Decode(&lista); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(lista) != caso.esperado {
				t.Fatalf("se esperaban %d facturas, hay %d", caso.esperado, len(lista))
			}
			if caso.factura != 0 && (len(lista) == 0 || lista[0].FacturaID != caso.factura) {
				t.Fatalf("factura inesperada: %+v", lista)
			}
		})
	}
}

func TestDetalleFacturaValidaEmailCliente(t *testing.T) {
	e := nuevoEntorno(t)

	resp, body := enviarFactura(t, e.factSrv.URL, CreateFacturaRequest{
		UsuarioID: 1,
		ClienteID: 201,
		Items:     []CreateFacturaItem{{ProductoID: 102, Cantidad: 1}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("crear factura: %d %s", resp.StatusCode, body)
	}

	// El correo coincide (en cualquier combinación de mayúsculas): 200.
	for _, email := range []string{"carlos@example.com", "CARLOS@example.com"} {
		detalle, err := http.Get(e.factSrv.URL + "/facturas/5001?email=" + email)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if detalle.StatusCode != http.StatusOK {
			t.Fatalf("dueño de la factura: se esperaba 200, se obtuvo %d", detalle.StatusCode)
		}
		detalle.Body.Close()
	}

	// El correo es de otro cliente: 403 con el mensaje de acceso.
	detalle, err := http.Get(e.factSrv.URL + "/facturas/5001?email=ana@example.com")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer detalle.Body.Close()

	if detalle.StatusCode != http.StatusForbidden {
		t.Fatalf("otro cliente: se esperaba 403, se obtuvo %d", detalle.StatusCode)
	}
	data, err := io.ReadAll(detalle.Body)
	if err != nil {
		t.Fatalf("leer body: %v", err)
	}
	if !strings.Contains(string(data), "No tienes acceso a esta factura") {
		t.Fatalf("mensaje inesperado: %s", data)
	}

	// Sin filtro el detalle sigue disponible (la API es visual, no exige rol).
	completo, err := http.Get(e.factSrv.URL + "/facturas/5001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer completo.Body.Close()
	if completo.StatusCode != http.StatusOK {
		t.Fatalf("sin filtro: se esperaba 200, se obtuvo %d", completo.StatusCode)
	}

	// Una factura inexistente sigue respondiendo 404 con el filtro puesto.
	inexistente, err := http.Get(e.factSrv.URL + "/facturas/9999?email=carlos@example.com")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer inexistente.Body.Close()
	if inexistente.StatusCode != http.StatusNotFound {
		t.Fatalf("se esperaba 404, se obtuvo %d", inexistente.StatusCode)
	}
}
