// CRUD del catálogo de productos: listado, creación, edición y eliminación.

import { API } from "./config.js";
import { apiDelete, apiGet, apiPost, apiPut } from "./api.js";
import { initLayout } from "./layout.js";
import { formatCurrency, setButtonLoading, showToast } from "./ui.js";

let products = [];
let modalElement = null;
let editingProductId = null;
// El catálogo con CRUD es solo del rol admin. El rol cliente lo ve en lectura.
let canManageProducts = false;

function renderRows(list) {
  const tbody = document.getElementById("catalog-rows");
  const empty = document.getElementById("catalog-empty");
  const table = document.getElementById("catalog-table");
  if (!tbody) return;

  tbody.innerHTML = "";

  if (!list.length) {
    table?.classList.add("hidden");
    empty?.classList.remove("hidden");
    empty?.classList.add("flex");
    return;
  }

  table?.classList.remove("hidden");
  empty?.classList.add("hidden");
  empty?.classList.remove("flex");

  list.forEach((product, index) => {
    const row = document.createElement("tr");
    row.className = `h-14 ${
      index % 2 === 0 ? "bg-surface-container-lowest" : ""
    } hover:bg-surface-container-low/60 transition-colors`;
    row.innerHTML = `
      <td class="pl-space-lg pr-space-md py-space-sm font-tabular-amount-md text-tabular-amount-md text-secondary">${product.id}</td>
      <td class="px-space-md py-space-sm font-body-md text-body-md text-on-surface product-name"></td>
      <td class="px-space-md py-space-sm font-tabular-amount-md text-tabular-amount-md text-on-surface text-right">${formatCurrency(product.precio)}</td>
      <td class="pl-space-md pr-space-lg py-space-sm font-tabular-amount-md text-tabular-amount-md text-right">
        <span class="product-stock ${product.stock > 0 ? "text-on-surface" : "text-error font-semibold"}"></span>
      </td>
      <td class="px-space-md py-space-sm text-right whitespace-nowrap product-actions">
        <button class="h-8 px-3 inline-flex items-center rounded-lg font-label-md text-label-md bg-surface-container-low text-on-surface hover:bg-surface-container transition-colors" data-action="edit" type="button" ${canManageProducts ? "" : "hidden"}>Editar</button>
        <button class="h-8 px-3 inline-flex items-center rounded-lg font-label-md text-label-md bg-error-container/60 text-error hover:bg-error-container transition-colors" data-action="delete" type="button" ${canManageProducts ? "" : "hidden"}>Eliminar</button>
      </td>
    `;
    row.querySelector(".product-name").textContent = product.nombre;
    row.querySelector(".product-stock").textContent = product.stock;
    if (canManageProducts) {
      row
        .querySelector('[data-action="edit"]')
        .addEventListener("click", () => openModal(product));
      row
        .querySelector('[data-action="delete"]')
        .addEventListener("click", () => deleteProduct(product));
    }
    tbody.appendChild(row);
  });
}

async function loadProducts() {
  const tbody = document.getElementById("catalog-rows");
  if (tbody) {
    tbody.innerHTML = `<tr><td class="py-12 text-center text-on-surface-variant" colspan="5">Cargando productos...</td></tr>`;
  }

  try {
    products = await apiGet(API.productos, "/productos");
    renderRows(Array.isArray(products) ? products : []);
  } catch (error) {
    showToast(error.message, "error");
    if (tbody) {
      tbody.innerHTML = `<tr><td class="py-12 text-center text-error" colspan="5"></td></tr>`;
      tbody.querySelector("td").textContent = error.message;
    }
  }
}

/**
 * Abre el modal en modo creación (sin argumento) o edición (con el producto).
 * @param {object|null} product Producto a editar.
 */
function openModal(product = null) {
  if (!modalElement) return;

  editingProductId = product ? product.id : null;

  const title = document.getElementById("product-modal-title");
  const subtitle = document.getElementById("product-modal-subtitle");
  const saveText = document.getElementById("save-product-btn-text");

  if (title) title.textContent = product ? "Editar producto" : "Nuevo producto";
  if (subtitle) {
    subtitle.textContent = product
      ? "Actualiza el nombre, el precio o el stock del producto."
      : "Ingresa los datos del producto para el catálogo.";
  }
  if (saveText) saveText.textContent = product ? "Guardar cambios" : "Guardar producto";

  document.getElementById("product-name").value = product?.nombre ?? "";
  document.getElementById("product-precio").value = product ? String(product.precio) : "";
  document.getElementById("product-stock").value = product ? String(product.stock) : "";

  showFormError("");
  modalElement.classList.remove("hidden");
  modalElement.classList.add("flex");
  document.body.classList.add("overflow-hidden");
  document.getElementById("product-name")?.focus();
}

function closeModal() {
  if (!modalElement) return;
  modalElement.classList.add("hidden");
  modalElement.classList.remove("flex");
  document.body.classList.remove("overflow-hidden");
  editingProductId = null;
}

async function deleteProduct(product) {
  const confirmed = window.confirm(
    `¿Eliminar el producto "${product.nombre}" (ID ${product.id})? Esta acción no se puede deshacer.`
  );
  if (!confirmed) return;

  try {
    await apiDelete(API.productos, `/productos/${product.id}`);
    showToast("Producto eliminado correctamente.", "success");
    await loadProducts();
  } catch (error) {
    showToast(error.message || "No se pudo eliminar el producto.", "error");
  }
}

function showFormError(message) {
  const banner = document.getElementById("product-form-error");
  const text = document.getElementById("product-form-error-text");
  if (!banner) return;
  if (!message) {
    banner.classList.add("hidden");
    return;
  }
  banner.classList.remove("hidden");
  if (text) text.textContent = message;
}

function initModal() {
  modalElement = document.getElementById("product-modal");
  if (!modalElement) return;

  document.getElementById("new-product-button")?.addEventListener("click", () => openModal());
  document.getElementById("empty-new-product-button")?.addEventListener("click", () => openModal());
  document.getElementById("close-product-modal-btn")?.addEventListener("click", closeModal);
  document.getElementById("cancel-product-btn")?.addEventListener("click", closeModal);

  modalElement.addEventListener("click", (event) => {
    if (event.target === modalElement) closeModal();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !modalElement.classList.contains("hidden")) {
      closeModal();
    }
  });

  const form = document.getElementById("product-form");
  form?.addEventListener("submit", async (event) => {
    event.preventDefault();
    showFormError("");

    const nombre = document.getElementById("product-name").value.trim();
    const precio = Number(document.getElementById("product-precio").value);
    const stockInput = document.getElementById("product-stock").value.trim();
    const stock = stockInput === "" ? NaN : Number(stockInput);
    const submitButton = document.getElementById("save-product-btn");

    if (!nombre) {
      showFormError("El nombre es obligatorio.");
      return;
    }
    if (!Number.isFinite(precio) || precio <= 0) {
      showFormError("El precio debe ser un número mayor a 0.");
      return;
    }
    if (!Number.isInteger(stock) || stock < 0) {
      showFormError("El stock debe ser un número entero mayor o igual a 0.");
      return;
    }

    const payload = { nombre, precio, stock };

    setButtonLoading(submitButton, true, "Guardando...");
    try {
      if (editingProductId) {
        await apiPut(API.productos, `/productos/${editingProductId}`, payload);
        showToast("Producto actualizado correctamente.", "success");
      } else {
        await apiPost(API.productos, "/productos", payload);
        showToast("Producto creado correctamente.", "success");
      }
      closeModal();
      form.reset();
      await loadProducts();
    } catch (error) {
      showFormError(error.message);
    } finally {
      setButtonLoading(submitButton, false);
    }
  });
}

// initLayout valida sesión y rol (cliente y admin pueden entrar aquí).
const usuarioActual = initLayout("productos");

canManageProducts = usuarioActual?.rol === "admin";

if (canManageProducts) {
  initModal();
} else {
  document.getElementById("new-product-button")?.classList.add("hidden");
  document.getElementById("empty-new-product-button")?.classList.add("hidden");
  document.getElementById("catalog-actions-heading")?.classList.add("hidden");
}
loadProducts();
