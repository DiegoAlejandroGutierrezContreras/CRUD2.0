// Guardia temprana de rol. Se ejecuta antes de que la página se renderice:
// si el rol no corresponde, el usuario es redirigido sin ver la pantalla.
// La autoridad real sigue siendo el backend; esto solo evita ketes visuales.
(function () {
  "use strict";

  const STORAGE_KEY = "facturacion_sesion";

  const PAGE_ROLES = {
    productos: ["admin"],
    clientes: ["admin"],
    "nueva-factura": ["admin"],
    facturas: ["admin", "cliente"],
    factura: ["admin", "cliente"],
  };

  function redirigir(url) {
    window.location.replace(url);
  }

  function homeFor(rol) {
    return rol === "admin" ? "productos.html" : "facturas.html";
  }

  let sesion;
  try {
    sesion = JSON.parse(localStorage.getItem(STORAGE_KEY) || "null");
  } catch {
    sesion = null;
  }

  const pagina = String(window.location.pathname)
    .split("/")
    .pop()
    .replace(/\.html$/i, "");
  const roles = PAGE_ROLES[pagina] || ["admin", "cliente"];

  if (!sesion || !sesion.usuario || typeof sesion.token !== "string" || !sesion.token) {
    redirigir("index.html");
    return;
  }

  if (Array.prototype.indexOf.call(["admin", "cliente"], sesion.usuario.rol) === -1) {
    localStorage.removeItem(STORAGE_KEY);
    redirigir("index.html");
    return;
  }

  if (sesion.expira && Date.parse(sesion.expira) <= Date.now()) {
    localStorage.removeItem(STORAGE_KEY);
    redirigir("index.html");
    return;
  }

  if (roles.indexOf(sesion.usuario.rol) === -1) {
    redirigir(homeFor(sesion.usuario.rol));
  }
})();
