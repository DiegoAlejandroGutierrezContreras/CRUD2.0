# Sistema de Facturación Digital (Microservicios)

Proyecto de Programación Avanzada 2026_2. Sistema de facturación basado en arquitectura de microservicios desarrollado en Go utilizando exclusivamente la librería estándar `net/http`, base de datos persistente en archivos JSON y una interfaz de usuario web multipágina desarrollada en JavaScript.

> **Alcance académico:** no se usan JWT, tokens, sesiones, cookies ni OAuth. El campo `rol` (`admin` o `cliente`) se guarda con el usuario y en el emisor de cada factura, pero solo sirve para adaptar el menú, el CRUD y el listado de facturas del frontend; **la API no exige autorización por rol**. Las contraseñas se almacenan en texto plano por decisión del equipo; **no es un diseño apto para producción**.

---

## Requisitos

- **Go 1.22** o superior instalado en el sistema.
- Navegador web moderno (Chrome, Edge, Firefox, Safari) con conexión a Internet para cargar Tailwind CDN y las fuentes.

---

## Puertos y Servicios

| Servicio | Puerto | Descripción | Archivo de Persistencia |
| :--- | :---: | :--- | :--- |
| **Autenticación** | `8081` | Registro con rol, login y consulta de usuarios (sin tokens) | `data/usuarios.json` |
| **Productos** | `8082` | CRUD de catálogo de productos y descuento atómico de stock | `data/productos.json` |
| **Clientes** | `8083` | CRUD de clientes con nombre, cédula, correo y dirección | `data/clientes.json` |
| **Facturación** | `8084` | Orquestador que consume los 3 anteriores y emite facturas | `data/facturas.json` |
| **Frontend Web** | `3000` | Servidor web estático que sirve la interfaz de usuario | Carpeta `frontend/` |

---

## Credenciales de Prueba

Usuarios registrados en `data/usuarios.json` (contraseña en texto plano):

- `ricardo@example.com` / `admin123` — rol `admin`
- `maria@example.com` / `cliente123` — rol `cliente`
- `diego@gmail.com` / `diego123` — rol `admin`

En `registro.html` se elige el tipo de cuenta (rol) al crearla: `admin` o `cliente`.

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
| `POST` | `/auth/register` | Registra un usuario (`nombre`, `email`, `password`, `rol`). Valida nombre, formato de email, email único, contraseña de al menos 6 caracteres y rol `admin`/`cliente` (400 si falta o es inválido). Guarda la contraseña en texto plano. | `201` usuario sin password · `400` datos inválidos · `409` email duplicado |
| `POST` | `/auth/login` | Valida `email` y `password` y retorna el usuario con su rol, sin password. No emite token. | `200` usuario · `400` campos vacíos · `401` credenciales incorrectas |
| `GET` | `/auth/usuarios/{id}` | Retorna `id`, `nombre`, `email` y `rol` del usuario. Lo usa Facturación para registrar quién emitió la factura. | `200` · `400` id inválido · `404` no encontrado |

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
| `GET` | `/facturas` | Lista las facturas ordenadas de la más reciente a la más antigua. Admite `?email=...` para filtrar solo las facturas cuyo `cliente.email` coincida (lo usa el rol cliente). | `200` |
| `GET` | `/facturas/{id}` | Detalle de una factura. Admite `?email=...`; si el correo no coincide con `factura.cliente.email` responde `403`. | `200` · `400` · `403` acceso denegado · `404` |

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

Ejemplo de factura guardada (el emisor incluye su rol):

```json
{
  "factura_id": 5002,
  "cliente": {
    "id": 202,
    "nombre": "Ana Torres",
    "email": "ana@example.com"
  },
  "usuario": {
    "id": 1,
    "nombre": "Ricardo Gómez",
    "rol": "admin"
  },
  "productos": [
    {
      "id": 103,
      "nombre": "Monitor Samsung 24''",
      "cantidad": 2,
      "precio_unitario": 80000,
      "subtotal": 160000
    },
    {
      "id": 102,
      "nombre": "Mouse inalámbrico Logitech",
      "cantidad": 1,
      "precio_unitario": 12000,
      "subtotal": 12000
    }
  ],
  "total": 172000,
  "fecha": "2026-10-19"
}
```

---

## Roles y Permisos

El rol se elige al registrar la cuenta y se guarda en `data/usuarios.json` y en el campo `usuario.rol` de cada factura emitida.

| Capacidad | Admin | Cliente |
| :--- | :---: | :---: |
| Catálogo de productos con CRUD | Sí | Solo lectura |
| Clientes con CRUD (crear, editar, eliminar) | Sí | No (la página redirige) |
| Nueva factura | Sí | No (la página redirige) |
| Facturas | Todas | Solo las de su correo: `factura.cliente.email == usuario.email` |

Implementación de la restricción:

- El menú (`layout.js`) y las guardias de página (`session.js`) dependen del rol del usuario en `localStorage`.
- El listado y el detalle de facturas se filtran en el backend con `GET /facturas?email=...` y `GET /facturas/{id}?email=...` (403 con mensaje "No tienes acceso a esta factura").
- Es una **simplificación académica**: la API no exige autorización, así que una llamada directa sin filtro vería todos los datos. Es una decisión equivalente a almacenar las contraseñas en texto plano.

---

## Flujo de Emisión y Control de Stock

1. Solo el rol admin accede a "Nueva factura".
2. El frontend carga clientes y productos y limita el carrito al stock disponible de cada producto.
3. Antes de enviar la factura, el frontend vuelve a consultar el catálogo y ajusta el carrito si el stock cambió.
4. Facturación consulta el usuario (`GET /auth/usuarios/{id}`), el cliente (`GET /clientes/{id}`) y cada producto (`GET /productos/{id}`).
5. Si algún producto no tiene stock suficiente, responde `409` **sin crear la factura**.
6. Si todo es válido, crea la factura en `data/facturas.json` copiando el rol del usuario emisor.
7. Luego solicita el descuento atómico a `POST /productos/descontar-stock`.
8. Si el descuento falla (por ejemplo, por una carrera con otra factura), Facturación elimina la factura recién creada y propaga el error. No existe una transacción distribuida real: se usa compensación.

---

## Pantallas del Frontend

| Página | Descripción |
| :--- | :--- |
| `index.html` | Inicio de sesión |
| `registro.html` | Creación de cuenta con selector de rol (admin / cliente) |
| `productos.html` | Catálogo de productos con CRUD (acciones visibles solo para admin) |
| `clientes.html` | CRUD de clientes: crear, editar y eliminar con confirmación |
| `nueva-factura.html` | Solo admin: selección de cliente, carrito y validación de stock |
| `facturas.html` | Admin ve todas; cliente solo las facturas de su correo |
| `factura.html` | Detalle de factura con rol del emisor, modal JSON e impresión / PDF |

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
│   ├── usuarios.json   # Usuarios con rol (password en texto plano)
│   ├── productos.json  # Catálogo y stock
│   ├── clientes.json   # Clientes
│   └── facturas.json   # Facturas emitidas (el usuario incluye rol)
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
│       ├── api.js            # Cliente HTTP sin tokens (GET/POST/PUT/DELETE)
│       ├── auth.js           # Login y registro con rol
│       ├── clientes.js       # CRUD de clientes
│       ├── config.js         # URLs de los servicios
│       ├── factura.js        # Detalle, rol del emisor, modal JSON e impresión
│       ├── facturas.js       # Historial filtrado por correo del cliente
│       ├── layout.js         # Header/footer, menú por rol y guardias de página
│       ├── nueva-factura.js  # Carrito y validación de stock
│       ├── productos.js      # CRUD del catálogo (rol admin)
│       ├── session.js        # Usuario y rol en localStorage
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
- `internal/auth`: registro con rol (`admin`/`cliente`), rechazo de rol ausente o inválido, normalización del rol, login y consulta de usuarios sin password.
- `internal/clientes`: CRUD completo (listar, consultar, crear, actualizar y eliminar), validaciones de cédula/correo y unicidad de email y CC.
- `internal/productos`: descuento atómico de stock, lote inválido sin cambios, ítems repetidos y validaciones.
- `internal/facturacion`: emisión con rol del emisor, descuento de stock, rechazo por stock insuficiente, compensación si falla el descuento, listado filtrado por `?email=` del cliente, detalle con `403` si el correo no coincide y validación de referencias.
