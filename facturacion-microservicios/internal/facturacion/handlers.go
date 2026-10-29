package facturacion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"facturacion-microservicios/internal/httpx"
	"facturacion-microservicios/internal/jsondb"
)

// ServiceURLs contiene las URLs de los servicios externos.
type ServiceURLs struct {
	AuthURL      string
	ProductosURL string
	ClientesURL  string
}

// Handler maneja las rutas del servicio de facturación.
type Handler struct {
	store  *jsondb.Store[Factura]
	urls   ServiceURLs
	client *http.Client
}

// NewHandler crea un nuevo handler de facturación.
func NewHandler(store *jsondb.Store[Factura], urls ServiceURLs) *Handler {
	return &Handler{
		store: store,
		urls:  urls,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RegisterRoutes registra las rutas del servicio.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /facturas", h.handleCreate)
	mux.HandleFunc("GET /facturas", h.handleList)
	mux.HandleFunc("GET /facturas/{id}", h.handleGet)
}

// handleCreate maneja POST /facturas.
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateFacturaRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	// Validar request básico
	if req.UsuarioID <= 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El usuario_id es obligatorio y debe ser mayor a 0")
		return
	}
	if req.ClienteID <= 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El cliente_id es obligatorio y debe ser mayor a 0")
		return
	}
	if len(req.Items) == 0 {
		httpx.RespondError(w, http.StatusBadRequest, "Debe incluir al menos un item en la factura")
		return
	}

	for i, item := range req.Items {
		if item.ProductoID <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, fmt.Sprintf("El producto_id del item %d es inválido", i+1))
			return
		}
		if item.Cantidad <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, fmt.Sprintf("La cantidad del item %d debe ser mayor a 0", i+1))
			return
		}
	}

	// Combinar items del mismo producto si hubiese repetidos
	combinedItems := combineItems(req.Items)

	// 1. Obtener el usuario que emite la factura
	usuario, err := h.getUsuario(req.UsuarioID)
	if err != nil {
		handleServiceError(w, err, "autenticación")
		return
	}

	// 2. Obtener cliente desde el servicio de Clientes
	cliente, err := h.getCliente(req.ClienteID)
	if err != nil {
		handleServiceError(w, err, "clientes")
		return
	}

	// 3. Obtener productos y validar stock antes de crear la factura
	var productosFactura []FacturaItem

	for _, item := range combinedItems {
		producto, err := h.getProducto(item.ProductoID)
		if err != nil {
			handleServiceError(w, err, "productos")
			return
		}

		if producto.Stock < item.Cantidad {
			httpx.RespondError(w, http.StatusConflict, fmt.Sprintf(
				"Stock insuficiente para \"%s\": disponible %d, solicitado %d",
				producto.Nombre, producto.Stock, item.Cantidad))
			return
		}

		subtotal := roundMoney(float64(item.Cantidad) * producto.Precio)
		productosFactura = append(productosFactura, FacturaItem{
			ID:             producto.ID,
			Nombre:         producto.Nombre,
			Cantidad:       item.Cantidad,
			PrecioUnitario: producto.Precio,
			Subtotal:       subtotal,
		})
	}

	// Calcular total
	total := CalculateTotal(productosFactura)

	// 4. Crear la factura con el esquema exacto (cliente SOLO id, nombre, email)
	factura := Factura{
		Cliente: FacturaCliente{
			ID:     cliente.ID,
			Nombre: cliente.Nombre,
			Email:  cliente.Email,
		},
		Usuario: FacturaUsuario{
			ID:     usuario.ID,
			Nombre: usuario.Nombre,
			Rol:    usuario.Rol,
		},
		Productos: productosFactura,
		Total:     total,
		Fecha:     time.Now().Format("2006-01-02"),
	}

	factura, err = h.store.Create(factura)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error guardando la factura")
		return
	}

	// 5. Descontar stock de forma atómica. Si falla, se elimina la factura
	// recién creada (compensación; no hay transacción distribuida real).
	if err := h.descontarStock(combinedItems); err != nil {
		if _, deleteErr := h.store.Delete(factura.FacturaID); deleteErr != nil {
			// La factura queda creada pero sin descuento de stock; se reporta
			// el error original de todas formas.
			fmt.Printf("Error eliminando factura %d tras fallo de stock: %v\n", factura.FacturaID, deleteErr)
		}
		handleServiceError(w, err, "productos")
		return
	}

	httpx.RespondCreated(w, factura)
}

// emailClienteQuery es el parámetro de query con el correo del cliente.
// Lo usa el rol cliente del frontend para ver solo sus facturas; la API no
// exige autorización, es una restricción visual del proyecto académico.
const emailClienteQuery = "email"

// handleList maneja GET /facturas.
// Con ?email=... filtra las facturas cuyo factura.cliente.email coincida.
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	facturas, err := h.store.GetAll()
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error leyendo las facturas")
		return
	}

	if facturas == nil {
		facturas = []Factura{}
	}

	if email := strings.TrimSpace(r.URL.Query().Get(emailClienteQuery)); email != "" {
		filtradas := make([]Factura, 0, len(facturas))
		for _, factura := range facturas {
			if strings.EqualFold(factura.Cliente.Email, email) {
				filtradas = append(filtradas, factura)
			}
		}
		facturas = filtradas
	}

	sort.Slice(facturas, func(i, j int) bool {
		return facturas[i].FacturaID > facturas[j].FacturaID
	})

	httpx.RespondJSON(w, http.StatusOK, facturas)
}

// handleGet maneja GET /facturas/{id}.
// Con ?email=... valida que el correo del cliente de la factura coincida; si
// no, responde 403 "No tienes acceso a esta factura".
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de factura inválido")
		return
	}

	factura, err := h.store.GetByID(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error leyendo la factura")
		return
	}
	if factura == nil {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Factura con ID %d no encontrada", id))
		return
	}

	if email := strings.TrimSpace(r.URL.Query().Get(emailClienteQuery)); email != "" &&
		!strings.EqualFold(factura.Cliente.Email, email) {
		httpx.RespondError(w, http.StatusForbidden, "No tienes acceso a esta factura")
		return
	}

	httpx.RespondJSON(w, http.StatusOK, factura)
}

// --- Clientes HTTP para servicios externos ---

type serviceError struct {
	StatusCode int
	Message    string
	IsNetwork  bool
}

func (e *serviceError) Error() string {
	return e.Message
}

func handleServiceError(w http.ResponseWriter, err error, servicio string) {
	se, ok := err.(*serviceError)
	if !ok {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	if se.IsNetwork {
		httpx.RespondError(w, http.StatusBadGateway,
			fmt.Sprintf("No se pudo conectar con el servicio de %s. Intente más tarde.", servicio))
		return
	}

	switch se.StatusCode {
	case http.StatusNotFound:
		httpx.RespondError(w, http.StatusNotFound, se.Message)
	case http.StatusConflict:
		httpx.RespondError(w, http.StatusConflict, se.Message)
	case http.StatusUnprocessableEntity:
		httpx.RespondError(w, http.StatusUnprocessableEntity, se.Message)
	default:
		httpx.RespondError(w, http.StatusBadGateway,
			fmt.Sprintf("Error en el servicio de %s: %s", servicio, se.Message))
	}
}

func (h *Handler) doRequest(method, url string, body any) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, &serviceError{Message: "Error interno", IsNetwork: false}
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, &serviceError{Message: "Error creando solicitud", IsNetwork: true}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, &serviceError{Message: "Servicio no disponible", IsNetwork: true}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, &serviceError{Message: "Error leyendo respuesta", IsNetwork: false}
	}

	return respBody, resp.StatusCode, nil
}

func (h *Handler) getUsuario(id int) (*UsuarioExterno, error) {
	body, status, err := h.doRequest("GET", fmt.Sprintf("%s/auth/usuarios/%d", h.urls.AuthURL, id), nil)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, decodeServiceError(body, status, fmt.Sprintf("Usuario con ID %d no encontrado", id))
	}

	var usuario UsuarioExterno
	if err := json.Unmarshal(body, &usuario); err != nil {
		return nil, &serviceError{Message: "Error decodificando respuesta del servicio de autenticación"}
	}

	return &usuario, nil
}

func (h *Handler) getCliente(id int) (*ClienteExterno, error) {
	body, status, err := h.doRequest("GET", fmt.Sprintf("%s/clientes/%d", h.urls.ClientesURL, id), nil)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, decodeServiceError(body, status, fmt.Sprintf("Cliente con ID %d no encontrado", id))
	}

	var cliente ClienteExterno
	if err := json.Unmarshal(body, &cliente); err != nil {
		return nil, &serviceError{Message: "Error decodificando respuesta del servicio de clientes"}
	}

	return &cliente, nil
}

func (h *Handler) getProducto(id int) (*ProductoExterno, error) {
	body, status, err := h.doRequest("GET", fmt.Sprintf("%s/productos/%d", h.urls.ProductosURL, id), nil)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, decodeServiceError(body, status, fmt.Sprintf("Producto con ID %d no encontrado", id))
	}

	var producto ProductoExterno
	if err := json.Unmarshal(body, &producto); err != nil {
		return nil, &serviceError{Message: "Error decodificando respuesta del servicio de productos"}
	}

	return &producto, nil
}

// descontarStock solicita al servicio de productos descontar el stock del lote.
func (h *Handler) descontarStock(items []CreateFacturaItem) error {
	reqItems := make([]DescontarStockItem, 0, len(items))
	for _, item := range items {
		reqItems = append(reqItems, DescontarStockItem{
			ProductoID: item.ProductoID,
			Cantidad:   item.Cantidad,
		})
	}

	body, status, err := h.doRequest("POST", h.urls.ProductosURL+"/productos/descontar-stock",
		DescontarStockRequest{Items: reqItems})
	if err != nil {
		return err
	}

	if status != http.StatusNoContent && status != http.StatusOK {
		return decodeServiceError(body, status, "No se pudo descontar el stock")
	}

	return nil
}

// decodeServiceError construye un serviceError a partir del body de error.
func decodeServiceError(body []byte, status int, fallback string) *serviceError {
	var errResp ErrorExterno
	json.Unmarshal(body, &errResp)
	msg := errResp.Error
	if msg == "" {
		msg = fallback
	}
	return &serviceError{StatusCode: status, Message: msg}
}

// --- Funciones auxiliares ---

// combineItems combina items con el mismo producto_id sumando cantidades.
func combineItems(items []CreateFacturaItem) []CreateFacturaItem {
	combined := make(map[int]int)
	order := []int{}

	for _, item := range items {
		if _, exists := combined[item.ProductoID]; !exists {
			order = append(order, item.ProductoID)
		}
		combined[item.ProductoID] += item.Cantidad
	}

	var result []CreateFacturaItem
	for _, pid := range order {
		result = append(result, CreateFacturaItem{
			ProductoID: pid,
			Cantidad:   combined[pid],
		})
	}

	return result
}

// CalculateTotal calcula el total de una factura sumando los subtotales.
func CalculateTotal(items []FacturaItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Subtotal
	}
	return roundMoney(total)
}

// roundMoney redondea un valor monetario a 2 decimales.
func roundMoney(amount float64) float64 {
	return math.Round(amount*100) / 100
}
