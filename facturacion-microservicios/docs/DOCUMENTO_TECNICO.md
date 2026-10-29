# Documento Técnico: Proyecto de Programación Avanzada 2026_2

**Integrantes:**
- [INTEGRANTE 1]
- [INTEGRANTE 2]

---

## (a) Arquitectura del Sistema

El sistema implementa una arquitectura orientada a microservicios independientes desarrollada en Go utilizando exclusivamente la biblioteca estándar `net/http` para la comunicación REST y archivos JSON como almacén de base de datos persistente.

### Componentes:

1. **Frontend (JavaScript / HTML):** Interfaz multipágina desacoplada servida por un servidor web estático (`:3000`). Incluye inicio de sesión, registro, catálogo de productos con CRUD, directorio de clientes con CRUD completo, emisión de facturas con control de stock, historial y detalle de factura con modal JSON e impresión a PDF.
2. **Servicio 1 - Autenticación (`:8081`):** Microservicio para registro con rol (`admin`/`cliente`), login y consulta de usuarios. No emite tokens; las contraseñas se almacenan en texto plano como simplificación académica y el rol solo se usa en la interfaz del frontend. Persiste en `data/usuarios.json`.
3. **Servicio 2 - Productos (`:8082`):** Microservicio para operaciones CRUD del catálogo (precios y stock) y para el descuento atómico de stock solicitado por Facturación. Rutas abiertas, sin JWT ni autorización. Persiste en `data/productos.json`.
4. **Servicio 3 - Clientes (`:8083`):** Microservicio para operaciones CRUD de clientes (nombre, cédula cc, correo, dirección). Rutas abiertas, sin JWT ni autorización. Persiste en `data/clientes.json`.
5. **Servicio de Facturación (`:8084`):** Microservicio orquestador que consume a los tres microservicios anteriores para generar facturas, validar stock y descontarlo. Persiste en `data/facturas.json`.

### Diagrama de Arquitectura

```mermaid
graph TD
    subgraph Frontend ["Frontend Web (:3000)"]
        UI["Interfaz multipágina (HTML + JavaScript)"]
    end

    subgraph Microservicios ["Capa de Microservicios (Go net/http)"]
        AuthSvc["Servicio 1: Autenticación (:8081)"]
        ProdSvc["Servicio 2: Productos (:8082)"]
        CliSvc["Servicio 3: Clientes (:8083)"]
        FactSvc["Servicio de Facturación (:8084)"]
    end

    subgraph BaseDeDatos ["Base de Datos (Archivos JSON)"]
        DBUsers[("data/usuarios.json")]
        DBProd[("data/productos.json")]
        DBCli[("data/clientes.json")]
        DBFact[("data/facturas.json")]
    end

    UI -->|POST /auth/login, /auth/register (rol)| AuthSvc
    UI -->|GET /productos + CRUD| ProdSvc
    UI -->|CRUD /clientes| CliSvc
    UI -->|POST /facturas, GET /facturas?email=| FactSvc

    FactSvc -->|1. GET /auth/usuarios/{id}| AuthSvc
    FactSvc -->|2. GET /clientes/{id}| CliSvc
    FactSvc -->|3. GET /productos/{id}| ProdSvc
    FactSvc -->|4. POST /productos/descontar-stock| ProdSvc

    AuthSvc <-->|Lectura / Escritura| DBUsers
    ProdSvc <-->|Lectura / Escritura| DBProd
    CliSvc <-->|Lectura / Escritura| DBCli
    FactSvc <-->|Lectura / Escritura| DBFact
```

---

## (b) Endpoints por Servicio y Flujos

### 1. Endpoints Reales del Sistema

Ninguna ruta requiere autenticación ni cabecera `Authorization`. Todos los servicios aplican CORS.

#### Servicio de Autenticación (`puerto 8081`)

- `POST /auth/register`: Registra un usuario nuevo (`nombre`, `email`, `password`, `rol`). Valida nombre obligatorio, formato de email, email único, contraseña de al menos 6 caracteres y que el rol sea `admin` o `cliente` (responde `400` si falta o es inválido, aceptando cualquier combinación de mayúsculas). Guarda la contraseña en texto plano y responde `201` con `id`, `nombre`, `email` y `rol` (sin password).
- `POST /auth/login`: Recibe `email` y `password`, busca el usuario y compara la contraseña con `crypto/subtle.ConstantTimeCompare`. Responde `200` con los datos del usuario y su rol (sin password) o `401` si las credenciales son incorrectas. **No genera token.**
- `GET /auth/usuarios/{id}`: Retorna `id`, `nombre`, `email` y `rol`. Es consumido por Facturación para registrar quién emitió la factura. Responde `404` si el usuario no existe.

#### Servicio de Productos (`puerto 8082`)

- `GET /productos`: Lista completa de productos disponibles.
- `GET /productos/{id}`: Detalle de un producto por su identificador.
- `POST /productos`: Crea un producto (`nombre`, `precio`, `stock`).
- `POST /productos/descontar-stock`: Descuenta el stock de un lote de productos de forma atómica. Body: `{ "items": [{ "producto_id", "cantidad" }] }`. Si un producto no existe responde `404`; si no hay stock suficiente responde `409`; en ambos casos **no se modifica ningún producto**. En éxito responde `204`.
- `PUT /productos/{id}`: Actualiza un producto existente (`nombre`, `precio`, `stock`).
- `DELETE /productos/{id}`: Elimina un producto por su identificador.

#### Servicio de Clientes (`puerto 8083`)

- `GET /clientes`: Lista completa de clientes registrados.
- `GET /clientes/{id}`: Detalle de un cliente por su identificador.
- `POST /clientes`: Registra un cliente (`nombre`, `cc`, `email`, `direccion`). Valida formato y unicidad de cédula y correo.
- `PUT /clientes/{id}`: Actualiza los datos de un cliente existente.
- `DELETE /clientes/{id}`: Elimina un cliente por su identificador.

#### Servicio de Facturación (`puerto 8084`)

- `POST /facturas`: Recibe `usuario_id`, `cliente_id` y una lista de `items` (`producto_id` y `cantidad`). Consulta los servicios de Auth, Clientes y Productos, valida stock, calcula subtotales y total, asigna `factura_id` autoincremental y `fecha` actual, guarda en `data/facturas.json` (copiando el rol del usuario emisor), descuenta el stock y retorna la factura con `201`.
- `GET /facturas`: Lista todas las facturas ordenadas de la más reciente a la más antigua. Con el parámetro `?email=...` retorna únicamente las facturas cuyo `factura.cliente.email` coincida con el correo dado (sin distinguir mayúsculas); el rol cliente del frontend lo usa para ver solo sus facturas.
- `GET /facturas/{id}`: Retorna el detalle de una factura o `404` si no existe. Con `?email=...` valida que el correo coincida con `factura.cliente.email`: si no coincide responde `403` con el mensaje "No tienes acceso a esta factura".

---

### 2. Flujos del Sistema

#### Flujo 1: Autenticación de Usuario (Registro y Login)

1. El usuario envía `POST /auth/register` con su nombre, correo, contraseña y el rol elegido en `registro.html` (`admin` o `cliente`). El servicio valida los datos, comprueba que el correo no exista y guarda el registro en `data/usuarios.json` con la contraseña en texto plano.
2. El usuario envía `POST /auth/login` con correo y contraseña. El servicio busca el usuario y compara la contraseña; si es correcta responde con `id`, `nombre`, `email` y `rol`.
3. El frontend guarda ese objeto usuario (con rol) en `localStorage` y redirige al catálogo. No se almacena ni envía ningún token. El rol controla el menú y las guardias de página: admin ve todo el CRUD, cliente solo lectura del catálogo y sus propias facturas. Si una sesión guardada no tiene rol válido, se descarta y se vuelve al login.

#### Flujo 2: Operaciones CRUD (Productos y Clientes)

1. **Productos:** Las operaciones `GET`, `POST`, `PUT` y `DELETE` sobre `/productos` son atendidas directamente por el microservicio del puerto `8082`, sin middleware de autorización. El frontend muestra las acciones de CRUD solo al rol admin. Las modificaciones actualizan `data/productos.json`.
2. **Clientes:** Las operaciones `GET`, `POST`, `PUT` y `DELETE` sobre `/clientes` son atendidas por el microservicio del puerto `8083`, sin middleware de autorización. El frontend administra clientes (crear, editar y eliminar con confirmación) desde su página reservada al rol admin. Las modificaciones actualizan `data/clientes.json`, y las facturas ya emitidas conservan la copia de los datos del cliente que guardan al momento de emitirse.

#### Flujo 3: Generación de Factura con Control de Stock

1. **Selección en el frontend:** El usuario elige un cliente y agrega productos al carrito. La interfaz impide superar el stock disponible de cada producto y muestra el estado "Agotado" cuando corresponde.
2. **Revalidación previa:** Antes de enviar la factura, el frontend vuelve a consultar `GET /productos` y ajusta el carrito si el stock cambió mientras el usuario lo editaba.
3. **Petición:** El frontend envía `POST /facturas` con el usuario de `localStorage` y el carrito:
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
4. **Consulta de usuario:** Facturación realiza `GET /auth/usuarios/{usuario_id}` para obtener `id`, `nombre` y `rol`.
5. **Consulta de cliente:** Facturación realiza `GET /clientes/{cliente_id}` para obtener `id`, `nombre` y `email`.
6. **Consulta y validación de productos:** Por cada item, consulta `GET /productos/{producto_id}` para obtener nombre y precio vigente. Si algún producto no tiene stock suficiente, responde `409` sin crear la factura.
7. **Cálculo y persistencia:** Calcula subtotales y total, asigna `factura_id` y `fecha`, y guarda la factura en `data/facturas.json`.
8. **Descuento atómico:** Solicita `POST /productos/descontar-stock` con el lote validado. El servicio de Productos verifica todo el lote bajo un lock y solo entonces escribe los nuevos stocks.
9. **Compensación:** Si el descuento falla (por ejemplo, por una carrera con otra factura), Facturación elimina la factura recién creada y responde con el error correspondiente (`409` o `502`). No existe una transacción distribuida real; se usa compensación.
10. **Visualización e impresión:** El frontend redirige al detalle de la factura, muestra los datos del cliente y del usuario emisor, permite ver el JSON y ofrece impresión o guardado en PDF mediante `window.print()` y reglas `@media print`.

---

## (c) Ejemplo de Factura con Esquema Exacto

Esquema: `{ "factura_id", "cliente": {"id","nombre","email"}, "usuario": {"id","nombre","rol"}, "productos": [{"id","nombre","cantidad","precio_unitario","subtotal"}], "total", "fecha": "YYYY-MM-DD" }`.

```json
{
  "factura_id": 5001,
  "cliente": {
    "id": 201,
    "nombre": "Carlos Pérez",
    "email": "carlos@example.com"
  },
  "usuario": {
    "id": 1,
    "nombre": "Ricardo Gómez",
    "rol": "admin"
  },
  "productos": [
    {
      "id": 101,
      "nombre": "Laptop Lenovo ThinkPad",
      "cantidad": 1,
      "precio_unitario": 3500000.00,
      "subtotal": 3500000.00
    },
    {
      "id": 102,
      "nombre": "Mouse inalámbrico Logitech",
      "cantidad": 2,
      "precio_unitario": 12000.00,
      "subtotal": 24000.00
    }
  ],
  "total": 3524000.00,
  "fecha": "2026-10-18"
}
```

---

## (d) Simplificaciones Académicas y Verificación

### Decisiones de alcance

- **Sin JWT, tokens, sesiones, cookies ni OAuth.** El curso no cubrió esos mecanismos y no forman parte del alcance del proyecto.
- **Roles como simplificación académica.** El campo `rol` (`admin` o `cliente`) se guarda en `data/usuarios.json` y en el emisor (`usuario.rol`) de cada factura. El frontend lo usa para filtrar el menú, bloquear páginas por URL, ocultar el CRUD del catálogo a los clientes y pedir al backend solo las facturas de su correo (`GET /facturas?email=...` y `GET /facturas/{id}?email=...` con `403`). La API no exige autorización: una llamada directa sin filtros puede leer todo, igual que ocurre con las contraseñas.
- **Contraseñas en texto plano.** Es una simplificación acordada por el equipo para el alcance académico. Un sistema real debe usar un hash con sal (por ejemplo, Argon2id o bcrypt) y nunca almacenar contraseñas legibles.
- **Endpoints abiertos.** La "sesión" del frontend es solo visual: `localStorage` guarda el usuario con su rol para adaptar las páginas, pero no autentica las llamadas HTTP.
- **Sin edición de facturas en la interfaz.** Las facturas son inmutables una vez emitidas; el CRUD de productos y clientes se administra desde la interfaz con acciones visibles para el rol admin.
- **Sin generación de PDF en el servidor.** La impresión y el guardado en PDF se delegan al navegador.

### Verificación del sistema

Durante el desarrollo el sistema se validó con pruebas automatizadas (`go test`, `httptest`) y una prueba de extremo a extremo que levanta los 5 procesos y recorre, como admin y como cliente, los flujos de login/registro con rol, CRUD de productos y clientes (incluida eliminación con confirmación), emisión de factura con descuento de stock, filtro de facturas por correo del cliente y `403` al intentar ver una factura ajena.

Para la entrega los archivos de pruebas se retiraron del repositorio para dejar únicamente el sistema ejecutable y los entregables. La verificación reproducible es:

```bash
go build ./...
go vet ./...
```

y la verificación manual descrita en *Flujo de Emisión y Control de Stock*, *Roles y Permisos* y la sección *Verificación* del README.
