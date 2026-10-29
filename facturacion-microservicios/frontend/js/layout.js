// Header y footer compartidos. El menú se adapta al rol del usuario:
// admin ve todas las secciones, cliente solo Productos y Facturas.
// No es seguridad real: la API sigue abierta sin autorización.

import { clearUsuario, requireRoles } from "./session.js";
import { escapeHTML } from "./ui.js";

const NAV_ITEMS = [
  { id: "productos", label: "Productos", href: "productos.html", roles: ["admin", "cliente"] },
  { id: "clientes", label: "Clientes", href: "clientes.html", roles: ["admin"] },
  { id: "nueva-factura", label: "Nueva factura", href: "nueva-factura.html", roles: ["admin"] },
  { id: "facturas", label: "Facturas", href: "facturas.html", roles: ["admin", "cliente"] },
];

// Roles permitidos por página. Cada página interna se valida al inicializarse.
const PAGE_ROLES = {
  productos: ["admin", "cliente"],
  clientes: ["admin"],
  "nueva-factura": ["admin"],
  facturas: ["admin", "cliente"],
};

function navLink(item, active) {
  const classes =
    item.id === active
      ? "bg-secondary-container text-on-secondary-container"
      : "text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface";
  return `<a class="px-space-md py-space-sm font-label-md text-label-md rounded-lg transition-colors ${classes}" href="${item.href}">${item.label}</a>`;
}

function renderHeader(usuario, active) {
  const container = document.getElementById("app-header");
  if (!container) return;

  const items = NAV_ITEMS.filter((item) => item.roles.includes(usuario.rol));
  const links = items.map((item) => navLink(item, active)).join("");
  const mobileLinks = items.map(
    (item) =>
      `<a class="block px-space-md py-2.5 rounded-lg font-label-md text-label-md ${
        item.id === active
          ? "bg-secondary-container text-on-secondary-container"
          : "text-on-surface-variant hover:bg-surface-container"
      }" href="${item.href}">${item.label}</a>`
  ).join("");

  const rolLabel = usuario.rol === "admin" ? "Administrador" : "Cliente";

  container.innerHTML = `
    <header class="no-print fixed top-0 left-0 right-0 z-50 bg-surface/90 backdrop-blur-md shadow-[0_1px_8px_rgba(0,0,0,0.04)]">
      <div class="h-16 max-w-7xl mx-auto px-margin lg:px-margin-desktop flex items-center justify-between gap-space-md">
        <div class="flex items-center gap-space-xl min-w-0">
          <a class="font-headline-sm text-headline-sm tracking-tight text-on-surface select-none" href="productos.html">Factura</a>
          <nav class="hidden md:flex items-center gap-space-xs">${links}</nav>
        </div>
        <div class="flex items-center gap-space-md shrink-0">
          <span class="inline-flex items-center px-2 py-0.5 rounded-full bg-surface-container text-on-surface-variant font-label-sm text-label-sm">${rolLabel}</span>
          <span class="hidden sm:inline font-label-md text-label-md text-on-surface max-w-[180px] truncate" title="${escapeHTML(usuario.nombre)}">${escapeHTML(usuario.nombre)}</span>
          <span class="hidden sm:inline text-outline-variant select-none">|</span>
          <button class="hidden sm:inline font-label-md text-label-md text-on-surface-variant hover:text-error transition-colors" data-logout type="button">Cerrar sesión</button>
          <button class="md:hidden w-10 h-10 inline-flex items-center justify-center rounded-lg text-on-surface-variant hover:bg-surface-container" id="mobile-menu-button" type="button" aria-label="Abrir menú" aria-expanded="false">
            <span class="material-symbols-outlined">menu</span>
          </button>
        </div>
      </div>
      <div class="hidden md:hidden border-t border-surface-container-high bg-surface" id="mobile-menu">
        <nav class="px-margin py-space-sm flex flex-col gap-1">
          ${mobileLinks}
          <button class="mt-1 text-left px-space-md py-2.5 rounded-lg font-label-md text-label-md text-on-surface-variant hover:bg-error-container hover:text-error transition-colors" data-logout type="button">Cerrar sesión</button>
        </nav>
      </div>
    </header>
  `;

  const toggle = container.querySelector("#mobile-menu-button");
  const menu = container.querySelector("#mobile-menu");
  toggle?.addEventListener("click", () => {
    const isHidden = menu.classList.contains("hidden");
    menu.classList.toggle("hidden");
    toggle.setAttribute("aria-expanded", String(isHidden));
    toggle.querySelector(".material-symbols-outlined").textContent = isHidden ? "close" : "menu";
  });

  container.querySelectorAll("[data-logout]").forEach((button) => {
    button.addEventListener("click", () => {
      clearUsuario();
      window.location.replace("index.html");
    });
  });
}

function renderFooter() {
  const container = document.getElementById("app-footer");
  if (!container) return;

  container.innerHTML = `
    <footer class="no-print w-full bg-surface-container-lowest shadow-[0_1px_8px_rgba(0,0,0,0.04)]">
      <div class="max-w-7xl mx-auto px-margin lg:px-margin-desktop py-space-md flex items-center justify-between gap-space-md">
        <p class="font-body-sm text-body-sm text-on-surface-variant">Factura — Sistema de Facturación Digital</p>
        <p class="font-body-sm text-body-sm text-outline">© Factura</p>
      </div>
    </footer>
  `;
}

/**
 * Inicializa la sesión visual de una página interna y valida el rol según
 * PAGE_ROLES. Si el rol no está permitido, redirige al catálogo.
 * @param {"productos"|"clientes"|"nueva-factura"|"facturas"} active
 * @returns {object|null} usuario de la sesión
 */
export function initLayout(active) {
  const usuario = requireRoles(PAGE_ROLES[active] || ["admin"]);
  if (!usuario) return null;

  renderHeader(usuario, active);
  renderFooter();
  return usuario;
}
