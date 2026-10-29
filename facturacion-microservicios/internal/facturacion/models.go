// Package facturacion implementa la lógica del servicio de facturación.
package facturacion

// Factura representa una factura generada con el esquema exacto del enunciado.
type Factura struct {
	FacturaID int            `json:"factura_id"`
	Cliente   FacturaCliente `json:"cliente"`
	Usuario   FacturaUsuario `json:"usuario"`
	Productos []FacturaItem  `json:"productos"`
	Total     float64        `json:"total"`
	Fecha     string         `json:"fecha"`
}

// FacturaCliente contiene únicamente id, nombre y email según el enunciado.
type FacturaCliente struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}

// FacturaUsuario es la información del usuario que emitió la factura.
// El rol se copia del usuario en el servicio de autenticación.
type FacturaUsuario struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Rol    string `json:"rol"`
}

// FacturaItem es un producto dentro de una factura.
type FacturaItem struct {
	ID             int     `json:"id"`
	Nombre         string  `json:"nombre"`
	Cantidad       int     `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Subtotal       float64 `json:"subtotal"`
}

// CreateFacturaRequest es el body de POST /facturas.
type CreateFacturaRequest struct {
	UsuarioID int                 `json:"usuario_id"`
	ClienteID int                 `json:"cliente_id"`
	Items     []CreateFacturaItem `json:"items"`
}

// CreateFacturaItem representa un producto y cantidad solicitada.
type CreateFacturaItem struct {
	ProductoID int `json:"producto_id"`
	Cantidad   int `json:"cantidad"`
}

// --- Modelos para consumir servicios externos ---

// UsuarioExterno es la respuesta de GET /auth/usuarios/{id}.
type UsuarioExterno struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

// ClienteExterno es la respuesta de GET /clientes/{id}.
type ClienteExterno struct {
	ID        int    `json:"id"`
	Nombre    string `json:"nombre"`
	CC        string `json:"cc"`
	Email     string `json:"email"`
	Direccion string `json:"direccion"`
}

// ProductoExterno es la respuesta de GET /productos/{id}.
type ProductoExterno struct {
	ID     int     `json:"id"`
	Nombre string  `json:"nombre"`
	Precio float64 `json:"precio"`
	Stock  int     `json:"stock"`
}

// DescontarStockRequest es el body de POST /productos/descontar-stock.
type DescontarStockRequest struct {
	Items []DescontarStockItem `json:"items"`
}

// DescontarStockItem representa un producto y la cantidad a descontar.
type DescontarStockItem struct {
	ProductoID int `json:"producto_id"`
	Cantidad   int `json:"cantidad"`
}

// ErrorExterno es la estructura de error de los servicios externos.
type ErrorExterno struct {
	Error string `json:"error"`
}
