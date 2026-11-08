# Sistema de Facturación Digital (Microservicios)

Proyecto de Programación Avanzada 2026_2. Sistema de facturación basado en arquitectura de microservicios desarrollado en Go utilizando exclusivamente la librería estándar `net/http`, base de datos persistente en archivos JSON y una interfaz de usuario web multipágina desarrollada en JavaScript.

> **Autenticación:** el login emite un **token opaco de sesión** (32 bytes aleatorios en base64url, guardados en memoria del servicio de autenticación, sin JWT). El frontend lo envía como `Authorization: Bearer <token>` y los demás microservicios lo validan contra `GET /auth/session`. El backend aplica las reglas por rol: productos, clientes y la emisión de facturas exigen `admin`; un `cliente` solo puede ver y descargar las facturas emitidas a su correo, y recibe `403` en cualquier otro recurso o al intentar abrir una factura ajena. Se mantienen dos simplificaciones académicas: **contraseñas en texto plano** y **CORS `*`**; no es un diseño apto para producción.

---

## Requisitos

- **Go 1.22** o superior instalado en el sistema.
- Navegador web moderno (Chrome, Edge, Firefox, Safari) con conexión a Internet para cargar Tailwind CDN y las fuentes.

---

## Puertos y Servicios

| Servicio | Puerto | Descripción | Archivo de Persistencia |
| :--- | :---: | :--- | :--- |
| **Autenticación** | `8081` | Registro con rol, login con token de sesión, logout y validación de sesión | `data/usuarios.json` |
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
- `alejo123@gmail.com` / `alejo123` — rol `cliente` (agregado durante pruebas manuales)

En `registro.html` se elige el tipo de cuenta (rol) al crearla: por defecto `cliente`, con la opción `admin`.

- `cliente`: solo pide **correo** y **contraseña**. El correo debe estar ya registrado en Clientes (creado por el admin) y el **nombre no se pide**: Auth lo copia desde Clientes.
- `admin`: pide **nombre**, **correo** y **contraseña**; el registro público de admin se conserva.

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
| `AUTH_URL` | Productos, Clientes, Facturación | `http://localhost:8081` |
| `PRODUCTOS_URL` | Facturación | `http://localhost:8082` |
| `CLIENTES_URL` | Facturación y Auth | `http://localhost:8083` |
| `INTERNAL_TOKEN` | Auth y Clientes | `facturacion-interno` |
| `SESSION_TTL` | Auth | `8h` (acepta formatos de `time.ParseDuration`, por ejemplo `30m`) |
| `FRONTEND_DIR` | Web | `frontend` |

> Las sesiones viven en memoria: al reiniciar el servicio de autenticación todos los tokens dejan de ser válidos y hay que iniciar sesión de nuevo.

---

## Endpoints

Todas las rutas aceptan CORS desde cualquier origen. Son públicas `POST /auth/register`, `POST /auth/login` y `GET /auth/usuarios/{id}`; `POST /auth/logout` acepta el token pero responde `204` incluso sin él (es idempotente). **El resto exige la cabecera `Authorization: Bearer <token>`**; sin token o con un token inválido responden `401`. **Productos y Clientes son exclusivos del rol `admin`**; un `cliente` recibe `403` en todas sus rutas. Un `cliente` solo puede consultar `GET /facturas` y `GET /facturas/{id}` de sus propias facturas.

### Autenticación (`:8081`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Rol `admin`: exige `nombre`, `email`, `password`. Rol `cliente`: solo `email` y `password`; Auth consulta Clientes (`GET /clientes/por-email`) y copia el nombre que registró el admin; si el email no existe en Clientes responde `400`. Valida formato de email, email en usuarios único, contraseña de al menos 6 caracteres y rol válido. Guarda la contraseña en texto plano. | `201` usuario sin password · `400` datos inválidos o email no registrado en Clientes · `409` email duplicado |
| `POST` | `/auth/login` | Valida `email` y `password` y retorna `{ usuario, token, expira }`. El token es opaco y se guarda en memoria. | `200` sesión · `400` campos vacíos · `401` credenciales incorrectas |
| `POST` | `/auth/logout` | Invalida el token enviado en `Authorization`. Es idempotente. | `204` |
| `GET` | `/auth/session` | Devuelve el usuario (`id`, `nombre`, `email`, `rol`) asociado al token. Es el endpoint que consultan los demás microservicios. | `200` · `401` token ausente, inválido o expirado |
| `GET` | `/auth/usuarios/{id}` | Retorna `id`, `nombre`, `email` y `rol` del usuario. Heredado del flujo anterior; Facturación ya no lo usa (toma el emisor de la sesión). | `200` · `400` id inválido · `404` no encontrado |

### Productos (`:8082`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `GET` | `/productos` | Lista completa del catálogo. **Solo admin.** | `200` · `403` |
| `GET` | `/productos/{id}` | Detalle de un producto. **Solo admin.** | `200` · `400` · `403` · `404` |
| `POST` | `/productos` | Crea un producto (`nombre`, `precio`, `stock`). **Solo admin.** | `201` · `400` · `403` |
| `POST` | `/productos/descontar-stock` | Descuenta un lote de stock de forma atómica (ver abajo). **Solo admin**; Facturación reenvía la sesión del administrador que emite. | `204` · `400` · `403` · `404` · `409` |
| `PUT` | `/productos/{id}` | Actualiza un producto. **Solo admin.** | `200` · `400` · `403` · `404` |
| `DELETE` | `/productos/{id}` | Elimina un producto. **Solo admin.** | `204` · `403` · `404` |

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
| `GET` | `/clientes` | Lista completa de clientes. **Solo admin.** | `200` · `403` |
| `GET` | `/clientes/{id}` | Detalle de un cliente. **Solo admin.** | `200` · `400` · `403` · `404` |
| `GET` | `/clientes/por-email?email=` | **Interna:** devuelve `{ id, nombre, email }` sin cédula ni dirección. Exige `X-Internal-Token`; la usa Auth al registrar clientes. | `200` · `400` · `403`	token wrong · `404` no existe |
| `POST` | `/clientes` | Crea un cliente (`nombre`, `cc`, `email`, `direccion`). Valida cédula numérica, email y unicidad atómica. **Solo admin.** | `201` · `400` · `403` · `409` |
| `PUT` | `/clientes/{id}` | Actualiza un cliente. **Solo admin.** | `200` · `400` · `403` · `404` · `409` |
| `DELETE` | `/clientes/{id}` | Elimina un cliente. **Solo admin.** | `204` · `403` · `404` |

### Facturación (`:8084`)

| Método | Ruta | Descripción | Respuestas |
| :--- | :--- | :--- | :--- |
| `POST` | `/facturas` | Crea una factura con `cliente_id` e `items`. El emisor se toma de la sesión (un `usuario_id` en el body se ignora). **Solo admin.** | `201` factura · `400` datos inválidos · `403` rol cliente · `404` cliente/producto inexistente · `409` stock insuficiente · `502` servicio no disponible |
| `GET` | `/facturas` | Admin: todas las facturas, de la más reciente a la más antigua. Cliente: solo las que tienen su correo en `cliente.email`. El query `?email=` se ignora en modo autenticado. | `200` |
| `GET` | `/facturas/{id}` | Detalle de una factura. Admin: cualquiera. Cliente: solo si `cliente.email` coincide con el de su sesión; si no, `403` con "No tienes acceso a esta factura". | `200` · `400` · `403` acceso denegado · `404` |

Body de `POST /facturas` (el emisor sale del token; ya no se envía `usuario_id`):

```json
{
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

El rol se elige al registrar la cuenta y se guarda en `data/usuarios.json` y en el campo `usuario.rol` de cada factura emitida. **El backend aplica el rol en cada petición**, no solo la interfaz.

| Capacidad | Admin | Cliente |
| :--- | :---: | :---: |
| Catálogo de productos con CRUD | Sí | No (`403`) |
| Clientes con CRUD (crear, editar, eliminar) | Sí | No (`403`) |
| Nueva factura | Sí | No (`403`) |
| Facturas | Todas | Solo las de su correo: `factura.cliente.email == sesión.email` |
| Ver y descargar sus facturas en PDF | Sí (las ve todas) | Sí |
| Detalle de una factura ajena | Sí | `403` |

Implementación de la restricción:

- `POST /auth/login` devuelve un token opaco; el frontend lo guarda junto al usuario y la expiración (`session.js`) y `api.js` lo envía en `Authorization` en cada petición.
- Cada microservicio llama a `GET /auth/session` (`internal/authclient`) para obtener el usuario autenticado. El middleware `Authenticate` exige sesión y `RequireAdmin` exige rol admin; Productos y Clientes usan `RequireAdmin` en todas sus rutas.
- El login redirige al admin al catálogo y al cliente a `facturas.html`. Las páginas de productos, clientes y nueva factura solo aceptan rol admin; un cliente que las abra vuelve a sus facturas.
- Facturación toma el emisor de la sesión, filtra `GET /facturas` por el correo de la sesión y devuelve `403` en `GET /facturas/{id}` cuando la factura es ajena. El cliente puede descargar en PDF las facturas que ve.
- **Limitación conocida:** el vínculo usuario↔factura es el **email**. Si se cambia el correo de un cliente después de emitir la factura, la factura conserva el correo anterior y deja de coincidir con la sesión. En un sistema real se usaría el id del cliente o del usuario.
- Las sesiones se guardan en memoria: un reinicio de Auth invalida todos los tokens.

---

## Flujo de Emisión y Control de Stock

1. Solo el rol admin accede a "Nueva factura"; un cliente que llame `POST /facturas` recibe `403`.
2. El frontend carga clientes y productos y limita el carrito al stock disponible de cada producto.
3. Antes de enviar la factura, el frontend vuelve a consultar el catálogo y ajusta el carrito si el stock cambió.
4. Facturación valida la sesión, toma el emisor del token, consulta el cliente (`GET /clientes/{id}`) y cada producto (`GET /productos/{id}`) reenviando el mismo token.
5. Si algún producto no tiene stock suficiente, responde `409` **sin crear la factura**.
6. Si todo es válido, crea la factura en `data/facturas.json` con el emisor de la sesión.
7. Luego solicita el descuento atómico a `POST /productos/descontar-stock` con la sesión del admin.
8. Si el descuento falla (por ejemplo, por una carrera con otra factura), Facturación elimina la factura recién creada y propaga el error. No existe una transacción distribuida real: se usa compensación.

### Ejemplo completo con `curl`

```bash
# 1. Login del admin (guarda el token)
TOKEN=$(curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"ricardo@example.com","password":"admin123"}' | jq -r .token)

# 2. Crear un cliente con el correo que usará la persona
curl -s -X POST http://localhost:8083/clientes \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"nombre":"María López","cc":"1020304050","email":"maria@example.com","direccion":"Calle 45 #12-34"}'

# 3. Emitir una factura a ese cliente
curl -s -X POST http://localhost:8084/facturas \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"cliente_id":203,"items":[{"producto_id":102,"cantidad":2}]}'

# 4. La persona se registra como cliente con ese mismo correo
#    Solo envía email y password: Auth consulta Clientes y copia el nombre.
curl -s -X POST http://localhost:8081/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"maria@example.com","password":"cliente123","rol":"cliente"}'

# 5. Login de la persona: solo verá su factura
MARIA=$(curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"maria@example.com","password":"cliente123"}' | jq -r .token)

curl -s http://localhost:8084/facturas -H "Authorization: Bearer $MARIA"
# -> solo las facturas de maria@example.com

curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8084/facturas/5001 \
  -H "Authorization: Bearer $MARIA"
# -> 403 (factura de otro cliente)
```

---

## Pantallas del Frontend

| Página | Descripción |
| :--- | :--- |
| `index.html` | Inicio de sesión |
| `registro.html` | Cliente: solo correo y contraseña; admin: nombre, correo y contraseña |
| `productos.html` | Solo admin: catálogo de productos con CRUD |
| `clientes.html` | Solo admin: CRUD de clientes con confirmación |
| `nueva-factura.html` | Solo admin: selección de cliente, carrito y validación de stock |
| `facturas.html` | Admin ve todas; cliente solo las facturas de su correo y las puede descargar |
| `factura.html` | Detalle de factura con rol del emisor, modal JSON, impresión y descarga en PDF |

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
│   ├── DOCUMENTO_TECNICO.md       # Documento técnico de arquitectura y endpoints
│   └── GUIA_COMPLETA_CODIGO.md    # Guía exhaustiva: carpetas, archivos, funciones y flujos
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
│       ├── api.js            # Cliente HTTP con token, timeout y manejo global de 401
│       ├── auth.js           # Login (guarda usuario + token) y registro en dos modos
│       ├── clientes.js       # CRUD de clientes
│       ├── config.js         # URLs de los servicios
│       ├── factura.js        # Detalle, modal JSON e impresión
│       ├── facturas.js       # Historial; el backend aplica el alcance por rol
│       ├── head-guard.js     # Redirige antes de render si el rol no corresponde
│       ├── layout.js         # Header/footer, menú por rol y logout
│       ├── nueva-factura.js  # Carrito, correo del cliente y validación de stock
│       ├── pdf.js            # Generación y descarga de la factura en PDF
│       ├── productos.js      # CRUD del catálogo (rol admin)
│       ├── session.js        # Sesión {usuario, token, expira} en localStorage
│       ├── tailwind-config.js
│       └── ui.js             # Formato de moneda, toasts y utilidades
├── internal/
│   ├── auth/         # Registro (cliente validado en Clientes), login y sesiones
│   ├── authclient/   # Cliente de /auth/session y middlewares Authenticate/RequireAdmin
│   ├── productos/    # Modelos, handlers y reglas por rol del catálogo
│   ├── clientes/     # Modelos, handlers, unicidad atómica y consulta interna por email
│   ├── facturacion/  # Orquestación, aislamiento de facturas y compensación
│   ├── httpx/        # Utilidades HTTP (JSON, errores y BearerToken)
│   ├── jsondb/       # Almacén atómico y concurrente sobre archivos JSON
│   └── middleware/   # Middleware CORS
└── go.mod            # Definición del módulo Go
```

---

## Verificación

```bash
gofmt -l .        # verificar formateo de código
go build ./cmd/... # compilar todos los microservicios sin errores
go vet ./...      # análisis estático
```

La verificación del sistema se realiza ejecutando los microservicios (`go run ./cmd/...`) y navegando por la interfaz web en `http://localhost:3000` o mediante peticiones HTTP descritas en el ejemplo con `curl` de *Flujo de Emisión y Control de Stock*.
