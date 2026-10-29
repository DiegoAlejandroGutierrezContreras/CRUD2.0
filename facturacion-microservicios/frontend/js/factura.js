// Detalle de factura, modal JSON e impresión. El rol cliente solo puede ver
// facturas cuyo cliente tenga el mismo correo que el usuario de la sesión.

import { API } from "./config.js";
import { apiGet } from "./api.js";
import { initLayout } from "./layout.js";
import { formatCurrency, formatDate, showToast } from "./ui.js";

const ACCESO_DENEGADO = "No tienes acceso a esta factura";

let facturaActual = null;

function emailsIguales(a, b) {
  return String(a || "").trim().toLowerCase() === String(b || "").trim().toLowerCase();
}

function renderFactura(factura) {
  facturaActual = factura;

  document.title = `Factura #${factura.factura_id} — Factura`;
  document.getElementById("invoice-number").textContent = `Factura #${factura.factura_id}`;
  document.getElementById("invoice-date").textContent = formatDate(factura.fecha);
  document.getElementById("invoice-client-name").textContent = factura.cliente?.nombre || "—";
  document.getElementById("invoice-client-email").textContent = factura.cliente?.email || "—";
  document.getElementById("invoice-client-id").textContent = `ID: ${factura.cliente?.id ?? "—"}`;
  document.getElementById("invoice-user-name").textContent = factura.usuario?.nombre || "—";
  document.getElementById("invoice-total").textContent = formatCurrency(factura.total);
  document.getElementById("json-title").textContent = `Factura #${factura.factura_id} JSON`;
  document.getElementById("json-code").textContent = JSON.stringify(factura, null, 2);

  const rol = factura.usuario?.rol;
  const rolLabel = rol === "admin" ? "Administrador" : rol === "cliente" ? "Cliente" : "—";
  document.getElementById("invoice-user-rol").textContent = rolLabel;

  const tbody = document.getElementById("invoice-items");
  tbody.innerHTML = "";

  (factura.productos || []).forEach((item) => {
    const row = document.createElement("tr");
    row.className = "hover:bg-surface-container-low/40 transition-colors";
    row.innerHTML = `
      <td class="py-space-md px-space-md font-label-md text-label-md text-on-surface item-name"></td>
      <td class="py-space-md px-space-md text-center tabular-nums">${item.cantidad}</td>
      <td class="py-space-md px-space-md text-right font-tabular-amount-md text-tabular-amount-md text-on-surface-variant">${formatCurrency(item.precio_unitario)}</td>
      <td class="py-space-md px-space-md text-right font-tabular-amount-md text-tabular-amount-md text-on-surface font-semibold">${formatCurrency(item.subtotal)}</td>
    `;
    row.querySelector(".item-name").textContent = item.nombre;
    tbody.appendChild(row);
  });
}

function showError(message) {
  document.getElementById("invoice-sheet")?.classList.add("hidden");
  const errorBox = document.getElementById("invoice-error");
  if (errorBox) {
    errorBox.classList.remove("hidden");
    document.getElementById("invoice-error-text").textContent = message;
  }
}

function initJsonModal() {
  const modal = document.getElementById("json-modal");
  const openButton = document.getElementById("btn-view-json");
  const closeButton = document.getElementById("close-json-modal");
  const closeFooterButton = document.getElementById("btn-close-footer");

  openButton?.addEventListener("click", () => {
    if (facturaActual) modal?.showModal();
  });
  closeButton?.addEventListener("click", () => modal?.close());
  closeFooterButton?.addEventListener("click", () => modal?.close());

  modal?.addEventListener("click", (event) => {
    const rect = modal.getBoundingClientRect();
    const inside =
      event.clientY >= rect.top &&
      event.clientY <= rect.top + rect.height &&
      event.clientX >= rect.left &&
      event.clientX <= rect.left + rect.width;
    if (!inside) modal.close();
  });
}

async function loadFactura(usuario) {
  const id = Number(new URLSearchParams(window.location.search).get("id"));
  if (!id || !Number.isFinite(id) || id <= 0) {
    showError("No se indicó una factura válida.");
    return;
  }

  const esAdmin = usuario?.rol === "admin";
  let path = `/facturas/${id}`;
  if (!esAdmin && usuario?.email) {
    path += `?email=${encodeURIComponent(usuario.email)}`;
  }

  try {
    const factura = await apiGet(API.facturacion, path);

    // Doble verificación en el navegador (la respuesta del backend ya filtró).
    if (!esAdmin && !emailsIguales(factura?.cliente?.email, usuario?.email)) {
      showError(ACCESO_DENEGADO);
      showToast(ACCESO_DENEGADO, "error");
      return;
    }

    renderFactura(factura);
  } catch (error) {
    const message = error.status === 403 ? ACCESO_DENEGADO : error.message;
    showError(message);
    showToast(message, "error");
  }
}

const usuario = initLayout("facturas");
initJsonModal();
document.getElementById("btn-print")?.addEventListener("click", () => window.print());
loadFactura(usuario);
