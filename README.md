# Sistema de Facturación Digital (Microservicios)

Proyecto de Programación Avanzada 2026_2: sistema de facturación con arquitectura de microservicios en Go (solo librería estándar `net/http`), persistencia en archivos JSON e interfaz web multipágina en JavaScript puro.

Todo el código vive en la carpeta [`facturacion-microservicios/`](facturacion-microservicios/).

> **Alcance académico:** no se usan JWT, tokens, sesiones, cookies ni OAuth. El campo `rol` (`admin` o `cliente`) adapta la interfaz del frontend y filtra las facturas del cliente, pero **la API no exige autorización por rol**. Las contraseñas se almacenan en texto plano por decisión del equipo; **no es un diseño apto para producción**.

## Opciones del sistema

| Rol | Puede |
| :--- | :--- |
| **Admin** | Catálogo de productos con CRUD, CRUD de clientes, emitir facturas y ver todas las facturas |
| **Cliente** | Catálogo en solo lectura y ver únicamente sus facturas (`factura.cliente.email == usuario.email`) |

## Ejecución rápida

Desde `facturacion-microservicios/`, abrir 5 terminales (o seguir los comandos del README interno):

```bash
go run ./cmd/auth         # 8081
go run ./cmd/productos    # 8082
go run ./cmd/clientes     # 8083
go run ./cmd/facturacion  # 8084
go run ./cmd/web          # 3000 -> http://localhost:3000
```

**Credenciales de prueba** (contraseñas en texto plano):

- `ricardo@example.com` / `admin123` — rol admin
- `maria@example.com` / `cliente123` — rol cliente
- `diego@gmail.com` / `diego123` — rol admin

## Estructura

```text
facturacion-microservicios/
├── cmd/                # 1 binario por proceso (auth, productos, clientes, facturacion, web)
├── internal/           # logica de cada servicio, jsondb, httpx y middleware CORS
├── data/               # persistencia JSON (usuarios, productos, clientes, facturas)
├── frontend/           # interfaz multipágina (HTML + JS modules + CSS)
└── docs/               # DOCUMENTO_TECNICO.md
```

## Documentación

- [`facturacion-microservicios/README.md`](facturacion-microservicios/README.md): endpoints de cada servicio, flujos, estructura y verificación.
- [`facturacion-microservicios/docs/DOCUMENTO_TECNICO.md`](facturacion-microservicios/docs/DOCUMENTO_TECNICO.md): arquitectura, decisiones de alcance y verificación del sistema.
