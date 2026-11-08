// Historial de facturas. El backend decide el alcance: el rol admin ve todas
// y el rol cliente solo las facturas emitidas a su correo.

import { API } from "./config.js";
import { apiGet } from "./api.js";
import { initLayout } from "./layout.js";
import { downloadFacturaPDF } from "./pdf.js";
import { formatCurrency, formatDate, showToast } from "./ui.js";

function aplicarVistaPorRol(usuario) {
  const esAdmin = usuario?.rol === "admin";
  if (esAdmin) return;

  document.getElementById("new-invoice-button")?.classList.add("hidden");
  document.getElementById("empty-new-invoice-button")?.classList.add("hidden");

  const subtitle = document.getElementById("facturas-subtitle");
  if (subtitle) subtitle.textContent = "Facturas emitidas a tu correo";

  const emptyTitle = document.getElementById("facturas-empty-title");
  if (emptyTitle) emptyTitle.textContent = "No tienes facturas registradas";
}

function renderFacturas(facturas) {
  const tbody = document.getElementById("facturas-rows");
  const table = document.getElementById("facturas-table");
  const empty = document.getElementById("facturas-empty");
  if (!tbody) return;

  tbody.innerHTML = "";

  if (!facturas.length) {
    table?.classList.add("hidden");
    empty?.classList.remove("hidden");
    empty?.classList.add("flex");
    return;
  }

  table?.classList.remove("hidden");
  empty?.classList.add("hidden");
  empty?.classList.remove("flex");

  facturas.forEach((factura) => {
    const row = document.createElement("tr");
    row.className =
      "h-16 bg-surface-container-lowest hover:bg-surface-container-low/60 transition-colors duration-150 group";
    row.innerHTML = `
      <td class="py-space-sm px-space-lg font-label-md text-label-md text-on-surface">
        <span class="inline-flex items-center px-2 py-0.5 rounded-md bg-surface-container text-on-surface font-label-md text-label-md">#${factura.factura_id}</span>
      </td>
      <td class="py-space-sm px-space-lg font-body-md text-body-md text-on-surface-variant">${formatDate(factura.fecha)}</td>
      <td class="py-space-sm px-space-lg font-body-md text-body-md text-on-surface font-semibold factura-cliente"></td>
      <td class="py-space-sm px-space-lg font-tabular-amount-md text-tabular-amount-md text-on-surface text-right tracking-tight">${formatCurrency(factura.total)}</td>
      <td class="py-space-sm px-space-lg text-right whitespace-nowrap">
        <a class="inline-flex items-center justify-center h-8 px-space-md rounded-lg font-label-md text-label-md bg-surface-container-low text-on-surface hover:bg-primary hover:text-on-primary shadow-sm transition-all duration-150" href="factura.html?id=${factura.factura_id}">Ver</a>
        <button class="inline-flex items-center justify-center h-8 px-space-md ml-space-xs rounded-lg font-label-md text-label-md bg-secondary-container text-on-secondary-container hover:bg-surface-container-highest shadow-sm transition-all duration-150" data-action="download" type="button">Descargar</button>
      </td>
    `;
    row.querySelector(".factura-cliente").textContent = factura.cliente?.nombre || "—";
    row
      .querySelector('[data-action="download"]')
      .addEventListener("click", () => {
        try {
          downloadFacturaPDF(factura);
          showToast(`Factura #${factura.factura_id} descargada.`, "success");
        } catch {
          showToast("No se pudo generar el PDF de la factura.", "error");
        }
      });
    tbody.appendChild(row);
  });
}

async function loadFacturas() {
  const tbody = document.getElementById("facturas-rows");
  if (tbody) {
    tbody.innerHTML = `<tr><td class="py-12 text-center text-on-surface-variant" colspan="5">Cargando facturas...</td></tr>`;
  }

  try {
    // Sin filtros: el backend aplica el alcance según la sesión.
    const facturas = await apiGet(API.facturacion, "/facturas");
    const list = Array.isArray(facturas) ? facturas : [];
    list.sort((a, b) => b.factura_id - a.factura_id);
    renderFacturas(list);
    const count = document.getElementById("facturas-count");
    if (count) count.textContent = `${list.length} registro(s)`;
  } catch (error) {
    showToast(error.message, "error");
    if (tbody) {
      tbody.innerHTML = `<tr><td class="py-12 text-center text-error" colspan="5"></td></tr>`;
      tbody.querySelector("td").textContent = error.message;
    }
  }
}

const usuario = initLayout("facturas");
if (usuario) {
  aplicarVistaPorRol(usuario);
  loadFacturas();
}
