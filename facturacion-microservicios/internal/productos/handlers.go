package productos

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"facturacion-microservicios/internal/authclient"
	"facturacion-microservicios/internal/httpx"
	"facturacion-microservicios/internal/jsondb"
)

// Handler maneja las rutas del servicio de productos.
type Handler struct {
	store *jsondb.Store[Producto]
	auth  *authclient.Client
}

// NewHandler crea el handler de producción: todas las rutas exigen rol admin.
// El catálogo solo lo consume el administrador y el servicio de Facturación,
// que reenvía la sesión del admin que emite la factura.
func NewHandler(store *jsondb.Store[Producto], authURL string) *Handler {
	return &Handler{store: store, auth: authclient.New(authURL)}
}

// NewHandlerSinAuth crea un handler sin autenticación. Solo para pruebas.
func NewHandlerSinAuth(store *jsondb.Store[Producto]) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registra las rutas del servicio.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	authclient.Registrar(mux, h.auth, []authclient.Route{
		{Pattern: "GET /productos", Handler: h.handleList, AdminOnly: true},
		{Pattern: "GET /productos/{id}", Handler: h.handleGet, AdminOnly: true},
		{Pattern: "POST /productos", Handler: h.handleCreate, AdminOnly: true},
		{Pattern: "POST /productos/descontar-stock", Handler: h.handleDescontarStock, AdminOnly: true},
		{Pattern: "PUT /productos/{id}", Handler: h.handleUpdate, AdminOnly: true},
		{Pattern: "DELETE /productos/{id}", Handler: h.handleDelete, AdminOnly: true},
	})
}

// handleList maneja GET /productos.
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.GetAll()
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if items == nil {
		items = []Producto{}
	}
	httpx.RespondJSON(w, http.StatusOK, items)
}

// handleGet maneja GET /productos/{id}.
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de producto inválido")
		return
	}

	producto, err := h.store.GetByID(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if producto == nil {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Producto con ID %d no encontrado", id))
		return
	}

	httpx.RespondJSON(w, http.StatusOK, producto)
}

// handleCreate maneja POST /productos.
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	if req.Nombre == "" {
		httpx.RespondError(w, http.StatusBadRequest, "El nombre del producto es obligatorio")
		return
	}
	if req.Precio <= 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El precio debe ser mayor a 0")
		return
	}
	if req.Stock < 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El stock no puede ser negativo")
		return
	}

	producto := Producto{
		Nombre: req.Nombre,
		Precio: req.Precio,
		Stock:  req.Stock,
	}

	producto, err := h.store.Create(producto)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	httpx.RespondCreated(w, producto)
}

// productoNoEncontradoError indica que un producto del lote no existe.
type productoNoEncontradoError struct {
	ID int
}

func (e *productoNoEncontradoError) Error() string {
	return fmt.Sprintf("Producto con ID %d no encontrado", e.ID)
}

// stockInsuficienteError indica que no hay unidades suficientes para un producto.
type stockInsuficienteError struct {
	ID         int
	Nombre     string
	Disponible int
	Solicitado int
}

func (e *stockInsuficienteError) Error() string {
	return fmt.Sprintf("Stock insuficiente para \"%s\": disponible %d, solicitado %d",
		e.Nombre, e.Disponible, e.Solicitado)
}

// handleDescontarStock maneja POST /productos/descontar-stock.
// Descuenta el stock de un lote de productos de forma atómica: si algún
// producto no existe o no tiene stock suficiente, no se modifica ninguno.
func (h *Handler) handleDescontarStock(w http.ResponseWriter, r *http.Request) {
	var req DescontarStockRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if len(req.Items) == 0 {
		httpx.RespondError(w, http.StatusBadRequest, "Debe incluir al menos un producto")
		return
	}

	// Combinar cantidades repetidas del mismo producto.
	cantidades := make(map[int]int)
	orden := make([]int, 0, len(req.Items))
	for i, item := range req.Items {
		if item.ProductoID <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, fmt.Sprintf("El producto_id del item %d es inválido", i+1))
			return
		}
		if item.Cantidad <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, fmt.Sprintf("La cantidad del item %d debe ser mayor a 0", i+1))
			return
		}
		if _, existe := cantidades[item.ProductoID]; !existe {
			orden = append(orden, item.ProductoID)
		}
		cantidades[item.ProductoID] += item.Cantidad
	}

	err := h.store.Transact(func(items []Producto) error {
		indice := make(map[int]int, len(items))
		for i := range items {
			indice[items[i].ID] = i
		}

		for _, productoID := range orden {
			pos, existe := indice[productoID]
			if !existe {
				return &productoNoEncontradoError{ID: productoID}
			}

			cantidad := cantidades[productoID]
			if items[pos].Stock < cantidad {
				return &stockInsuficienteError{
					ID:         items[pos].ID,
					Nombre:     items[pos].Nombre,
					Disponible: items[pos].Stock,
					Solicitado: cantidad,
				}
			}
		}

		// Solo se aplican los cambios si todo el lote es válido.
		for _, productoID := range orden {
			pos := indice[productoID]
			items[pos].Stock -= cantidades[productoID]
		}

		return nil
	})

	if err != nil {
		var noEncontrado *productoNoEncontradoError
		if errors.As(err, &noEncontrado) {
			httpx.RespondError(w, http.StatusNotFound, noEncontrado.Error())
			return
		}

		var sinStock *stockInsuficienteError
		if errors.As(err, &sinStock) {
			httpx.RespondError(w, http.StatusConflict, sinStock.Error())
			return
		}

		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	httpx.RespondNoContent(w)
}

// handleUpdate maneja PUT /productos/{id}.
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de producto inválido")
		return
	}

	var req UpdateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	if req.Nombre == "" {
		httpx.RespondError(w, http.StatusBadRequest, "El nombre del producto es obligatorio")
		return
	}
	if req.Precio <= 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El precio debe ser mayor a 0")
		return
	}
	if req.Stock < 0 {
		httpx.RespondError(w, http.StatusBadRequest, "El stock no puede ser negativo")
		return
	}

	updated, ok, err := h.store.Update(id, func(p *Producto) {
		p.Nombre = req.Nombre
		p.Precio = req.Precio
		p.Stock = req.Stock
	})
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if !ok {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Producto con ID %d no encontrado", id))
		return
	}

	httpx.RespondJSON(w, http.StatusOK, updated)
}

// handleDelete maneja DELETE /productos/{id}.
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "ID de producto inválido")
		return
	}

	ok, err := h.store.Delete(id)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}
	if !ok {
		httpx.RespondError(w, http.StatusNotFound, fmt.Sprintf("Producto con ID %d no encontrado", id))
		return
	}

	httpx.RespondNoContent(w)
}
