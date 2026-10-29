# Sistema de Facturación Digital (Microservicios)

Proyecto de Programación Avanzada 2026_2. Sistema de facturación basado en arquitectura de microservicios desarrollado en Go utilizando exclusivamente la librería estándar `net/http`, base de datos persistente en archivos JSON y una interfaz de usuario web multipágina desarrollada en JavaScript.

> **Alcance académico:** no se usan JWT, tokens, sesiones, cookies, OAuth ni roles. El login valida las credenciales y el frontend guarda solo el objeto usuario en `localStorage` como referencia visual. Las contraseñas se almacenan en texto plano por decisión del equipo; **no es un diseño apto para producción**.

---

## Requisitos

- **Go 1.22** o superior instalado en el sistema.
- Navegador web moderno (Chrome, Edge, Firefox, Safari) con conexión a Internet para cargar Tailwind CDN y las fuentes.

---

## Puertos y Servicios

| Servicio | Puerto | Descripción | Archivo de Persistencia |
| :--- | :---: | :--- | :--- |
| **Autenticación** | `8081` | Registro, login y consulta de usuarios (sin tokens ni roles) | `data/usuarios.json` |
| **Productos** | `8082` | CRUD de catálogo de productos y descuento atómico de stock | `data/productos.json` |
| **Clientes** | `8083` | CRUD de clientes con nombre, cédula, correo y dirección | `data/clientes.json` |
| **Facturación** | `8084` | Orquestador que consume los 3 anteriores y emite facturas | `data/facturas.json` |
| **Frontend Web** | `3000` | Servidor web estático que sirve la interfaz de usuario | Carpeta `frontend/` |

---

## Credenciales de Prueba

Usuarios registrados en `data/usuarios.json` (contraseña en texto plano, sin campo `rol`):

- `ricardo@example.com` / `admin123`
- `maria@example.com` / `cliente123`

Ambos usuarios tienen exactamente las mismas capacidades dentro del sistema.

---

## Ejecución

Abrir una terminal por servicio desde la carpeta `facturacion-microservicios/`:

```bash
# Terminal 1 - Autenticación
go run ./cmd/auth

# Terminal 2 - Productos
go run ./cmd/productos

# Terminal 3 - Clientes
go run ./cmd/clientes

# Terminal 4 - Facturación
go run ./cmd/facturacion

# Terminal 5 - Frontend Web
go run ./cmd/web
```

Una vez iniciados los cinco procesos, abrir en el navegador: **http://localhost:3000**

Cada servicio acepta variables de entorno para cambiar puerto y rutas:

| Variable | Servicio | Valor por defecto |
| :--- | :--- | :--- |
| `PORT` | Todos | `8081`, `8082`, `8083`, `8084`, `3000` |
| `DATA_FILE` | Auth, Productos, Clientes, Facturación | `data/*.json` |
| `AUTH_URL` | Facturación | `http://localhost:8081` |
| `PRODUCTOS_URL` | Facturación | `http://localhost:8082` |
| `CLIENTES_URL` | Facturación | `http://localhost:8083` |
| `FRONTEND_DIR` | Web | `frontend` |

---

## Endpoints

Todas las rutas aceptan CORS desde cualquier origen. Ninguna requiere cabecera `Authorization`.

### Autenticación (`:8081`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Registra un usuario (`nombre`, `email`, `password`). Valida nombre, formato de email, email único y contraseña de al menos 6 caracteres. Guarda la contraseña en texto plano. | `201` usuario sin password · `400` datos inválidos · `409` email duplicado |
| `POST` | `/auth/login` | Valida `email` y `password` y retorna el usuario sin password. No emite token. | `200` usuario · `400` campos vacíos · `401` credenciales incorrectas |
| `GET` | `/auth/usuarios/{id}` | Retorna `id`, `nombre` y `email` del usuario. Lo usa Facturación para registrar quién emitió la factura. | `200` · `400` id inválido · `404` no encontrado |

### Productos (`:8082`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `GET` | `/productos` | Lista completa del catálogo. | `200` |
| `GET` | `/productos/{id}` | Detalle de un producto. | `200` · `400` · `404` |
| `POST` | `/productos` | Crea un producto (`nombre`, `precio`, `stock`). | `201` · `400` |
| `POST` | `/productos/descontar-stock` | Descuenta un lote de stock de forma atómica (ver abajo). | `204` · `400` · `404` · `409` |
| `PUT` | `/productos/{id}` | Actualiza un producto. | `200` · `400` · `404` |
| `DELETE` | `/productos/{id}` | Elimina un producto. | `204` · `404` |

Body de `POST /productos/descontar-stock`:

```json
{
  "items": [
    { "producto_id": 101, "cantidad": 2 },
    { "producto_id": 102, "cantidad": 3 }
  ]
}
```

La operación valida **todo el lote** antes de escribir: si un producto no existe (`404`) o no tiene stock suficiente (`409`), ningún producto se modifica. Los ítems repetidos del mismo producto se combinan sumando cantidades.

### Clientes (`:8083`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `GET` | `/clientes` | Lista completa de clientes. | `200` |
| `GET` | `/clientes/{id}` | Detalle de un cliente. | `200` · `400` · `404` |
| `POST` | `/clientes` | Crea un cliente (`nombre`, `cc`, `email`, `direccion`). Valida cédula numérica, email y unicidad. | `201` · `400` · `409` |
| `PUT` | `/clientes/{id}` | Actualiza un cliente. | `200` · `400` · `404` · `409` |
| `DELETE` | `/clientes/{id}` | Elimina un cliente. | `204` · `404` |

### Facturación (`:8084`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `POST` | `/facturas` | Crea una factura con `usuario_id`, `cliente_id` e `items`. | `201` factura · `400` datos inválidos · `404` usuario/cliente/producto inexistente · `409` stock insuficiente · `502` servicio no disponible |
| `GET` | `/facturas` | Lista las facturas ordenadas de la más reciente a la más antigua. | `200` |
| `GET` | `/facturas/{id}` | Detalle de una factura. | `200` · `400` · `404` |

Body de `POST /facturas`:

```json
{
  "usuario_id": 1,
  "cliente_id": 201,
  "items": [
    { "producto_id": 101, "cantidad": 1 },
    { "producto_id": 102, "cantidad": 2 }
  ]
}
```

---

## Flujo de Emisión y Control de Stock

1. El frontend carga clientes y productos y limita el carrito al stock disponible de cada producto.
2. Antes de enviar la factura, el frontend vuelve a consultar el catálogo y ajusta el carrito si el stock cambió.
3. Facturación consulta el usuario (`GET /auth/usuarios/{id}`), el cliente (`GET /clientes/{id}`) y cada producto (`GET /productos/{id}`).
4. Si algún producto no tiene stock suficiente, responde `409` **sin crear la factura**.
5. Si todo es válido, crea la factura en `data/facturas.json`.
6. Luego solicita el descuento atómico a `POST /productos/descontar-stock`.
7. Si el descuento falla (por ejemplo, por una carrera con otra factura), Facturación elimina la factura recién creada y propaga el error. No existe una transacción distribuida real: se usa compensación.

---

## Pantallas del Frontend

| Página | Descripción |
| :--- | :--- |
| `index.html` | Inicio de sesión |
| `registro.html` | Creación de cuenta |
| `productos.html` | Catálogo de productos con CRUD (crear, editar y eliminar) |
| `clientes.html` | Directorio de clientes con CRUD completo (crear, editar y eliminar) |
| `nueva-factura.html` | Selección de cliente, carrito y validación de stock |
| `facturas.html` | Historial de facturas |
| `factura.html` | Detalle de factura, modal JSON e impresión / PDF |

El diseño usa Tailwind CSS por CDN, tipografía Inter y Material Symbols. Los importes se formatean con `Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 })`.

---

## Estructura del Proyecto

```text
facturacion-microservicios/
├── cmd/
│   ├── auth/         # Punto de entrada del microservicio de autenticación (:8081)
│   ├── productos/    # Punto de entrada del microservicio de productos (:8082)
│   ├── clientes/     # Punto de entrada del microservicio de clientes (:8083)
│   ├── facturacion/  # Punto de entrada del microservicio de facturación (:8084)
│   └── web/          # Servidor HTTP para archivos estáticos del frontend (:3000)
├── data/
│   ├── usuarios.json   # Usuarios (password en texto plano, sin rol)
│   ├── productos.json  # Catálogo y stock
│   ├── clientes.json   # Clientes
│   └── facturas.json   # Facturas emitidas
├── docs/
│   └── DOCUMENTO_TECNICO.md   # Documento técnico de arquitectura y endpoints
├── frontend/
│   ├── index.html
│   ├── registro.html
│   ├── productos.html
│   ├── clientes.html
│   ├── nueva-factura.html
│   ├── facturas.html
│   ├── factura.html
│   ├── css/design.css        # Tokens visuales, foco accesible y reglas @media print
│   └── js/
│       ├── api.js            # Cliente HTTP sin tokens
│       ├── auth.js           # Login y registro
│       ├── clientes.js       # Listado y modal de clientes
│       ├── config.js         # URLs de los servicios
│       ├── factura.js        # Detalle, modal JSON e impresión
│       ├── facturas.js       # Historial
│       ├── layout.js         # Header/footer compartidos y guardia visual
│       ├── nueva-factura.js  # Carrito y validación de stock
│       ├── productos.js      # Catálogo
│       ├── session.js        # Usuario en localStorage
│       ├── tailwind-config.js
│       └── ui.js             # Formato de moneda, toasts y utilidades
├── internal/
│   ├── auth/         # Lógica, modelos y handlers de autenticación
│   ├── productos/    # Lógica, modelos y handlers de productos
│   ├── clientes/     # Lógica, modelos y handlers de clientes
│   ├── facturacion/  # Orquestación de facturas
│   ├── httpx/        # Utilidades HTTP (JSON decoder, respuestas y errores)
│   ├── jsondb/       # Almacén atómico y concurrente sobre archivos JSON
│   └── middleware/   # Middleware CORS
└── go.mod            # Definición del módulo Go
```

---

## Pruebas

```bash
go build ./...
go vet ./...
go test ./...
```

Las pruebas cubren:

- `internal/jsondb`: creación concurrente de IDs, `Transact` con confirmación y reversión, reapertura del archivo.
- `internal/auth`: registro, login y consulta de usuarios sin password ni rol.
- `internal/clientes`: CRUD completo (listar, consultar, crear, actualizar y eliminar), validaciones de cédula/correo y unicidad de email y CC.
- `internal/productos`: descuento atómico de stock, lote inválido sin cambios, ítems repetidos y validaciones.
- `internal/facturacion`: emisión con descuento de stock, rechazo por stock insuficiente, compensación si falla el descuento, listado, detalle y validación de referencias.
