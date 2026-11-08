// Nueva factura: carrito con validación de stock en la interfaz.

import { API } from "./config.js";
import { apiGet, apiPost } from "./api.js";
import { initLayout } from "./layout.js";
import { formatCurrency, setButtonLoading, showToast } from "./ui.js";

const state = {
  usuario: null,
  clientes: [],
  productos: [],
  cart: new Map(),
  draft: new Map(),
  clienteId: null,
  submitting: false,
};

const elements = {};

function cacheElements() {
  elements.clientSelect = document.getElementById("client-select");
  elements.clientEmail = document.getElementById("client-email");
  elements.clientAddress = document.getElementById("client-address");
  elements.clientInfo = document.getElementById("client-info");
  elements.productsList = document.getElementById("products-list");
  elements.cartContainer = document.getElementById("cart-container");
  elements.emptyCart = document.getElementById("empty-cart-message");
  elements.cartTotal = document.getElementById("cart-total");
  elements.generateButton = document.getElementById("btn-generar");
  elements.billingUser = document.getElementById("billing-user");
}

function availableFor(product) {
  return product.stock - (state.cart.get(product.id) || 0);
}

function findProduct(id) {
  return state.productos.find((product) => product.id === id);
}

function renderClientSelect() {
  const select = elements.clientSelect;
  if (!select) return;

  const previous = state.clienteId;
  select.innerHTML = `<option value="">Selecciona un cliente</option>`;

  if (!state.clientes.length) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = "No hay clientes registrados";
    select.appendChild(option);
    select.disabled = true;
    updateClientInfo();
    return;
  }

  select.disabled = false;
  // Cada opción muestra el correo: el admin asocia la factura de forma inequívoca.
  state.clientes.forEach((client) => {
    const option = document.createElement("option");
    option.value = String(client.id);
    option.textContent = `${client.id} — ${client.nombre} — ${client.email}`;
    select.appendChild(option);
  });

  if (previous && state.clientes.some((client) => client.id === previous)) {
    select.value = String(previous);
  } else {
    state.clienteId = null;
    select.value = "";
  }

  updateClientInfo();
}

function updateClientInfo() {
  const client = state.clientes.find((item) => item.id === state.clienteId);
  if (!client) {
    elements.clientInfo?.classList.add("hidden");
    elements.clientEmail.textContent = "—";
    elements.clientAddress.textContent = "—";
    updateGenerateState();
    return;
  }

  elements.clientInfo?.classList.remove("hidden");
  // La factura queda asociada a este email exacto: conviene que sea visible.
  elements.clientEmail.textContent = client.email;
  elements.clientAddress.textContent = client.direccion;
  updateGenerateState();
}

function renderProducts() {
  const list = elements.productsList;
  if (!list) return;

  list.innerHTML = "";

  if (!state.productos.length) {
    list.innerHTML = `
      <div class="py-10 text-center text-on-surface-variant">
        <span class="material-symbols-outlined text-[32px] text-outline-variant block mb-1">inventory_2</span>
        <p class="font-body-md text-body-md">No hay productos disponibles</p>
      </div>
    `;
    return;
  }

  state.productos.forEach((product) => {
    const available = Math.max(availableFor(product), 0);
    const inCart = state.cart.get(product.id) || 0;
    const draft = Math.min(state.draft.get(product.id) || 1, Math.max(available, 0));
    const soldOut = available <= 0;

    const row = document.createElement("div");
    row.className =
      "flex flex-col sm:flex-row sm:items-center justify-between p-space-md rounded-lg bg-surface-container-low hover:bg-surface-container transition-colors gap-3";
    row.dataset.productId = String(product.id);
    row.innerHTML = `
      <div class="flex flex-col min-w-0">
        <span class="font-headline-sm text-[15px] leading-tight text-on-surface product-name"></span>
        <div class="flex flex-wrap items-center gap-space-md mt-1">
          <span class="font-tabular-amount-md text-tabular-amount-md text-primary font-semibold">${formatCurrency(product.precio)}</span>
          <span class="text-outline-variant text-[12px]">•</span>
          <span class="font-body-sm text-body-sm ${soldOut ? "text-error font-semibold" : "text-on-surface-variant"}">${soldOut ? "Agotado" : `Stock: ${product.stock}`}</span>
          ${inCart > 0 ? `<span class="font-body-sm text-body-sm text-tertiary font-semibold">En carrito: ${inCart}</span>` : ""}
        </div>
      </div>
      <div class="flex items-center gap-space-sm self-end sm:self-center shrink-0">
        <div class="flex items-center bg-surface-container-lowest rounded-lg px-1 h-11 shadow-sm ${soldOut ? "opacity-50" : ""}">
          <button class="w-8 h-8 flex items-center justify-center text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high rounded transition-colors disabled:opacity-40 disabled:pointer-events-none" data-action="minus" type="button" aria-label="Quitar una unidad" ${soldOut || draft <= 1 ? "disabled" : ""}>
            <span class="material-symbols-outlined text-[16px]">remove</span>
          </button>
          <span class="w-8 text-center font-tabular-amount-md text-tabular-amount-md text-on-surface select-none">${soldOut ? 0 : draft}</span>
          <button class="w-8 h-8 flex items-center justify-center text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high rounded transition-colors disabled:opacity-40 disabled:pointer-events-none" data-action="plus" type="button" aria-label="Agregar una unidad" ${soldOut || draft >= available ? "disabled" : ""}>
            <span class="material-symbols-outlined text-[16px]">add</span>
          </button>
        </div>
        <button class="h-11 px-space-md bg-secondary-container text-on-secondary-container hover:bg-surface-variant active:scale-[0.98] font-label-md text-label-md rounded-lg flex items-center justify-center transition-all cursor-pointer disabled:opacity-40 disabled:pointer-events-none" data-action="add" type="button" ${soldOut ? "disabled" : ""}>
          ${soldOut ? "Agotado" : "Agregar"}
        </button>
      </div>
    `;
    row.querySelector(".product-name").textContent = product.nombre;

    row.querySelector('[data-action="minus"]')?.addEventListener("click", () => {
      const current = state.draft.get(product.id) || 1;
      state.draft.set(product.id, Math.max(1, current - 1));
      renderProducts();
    });

    row.querySelector('[data-action="plus"]')?.addEventListener("click", () => {
      const current = state.draft.get(product.id) || 1;
      const max = availableFor(product);
      if (current >= max) {
        showToast(`Solo hay ${max} unidad(es) disponibles de "${product.nombre}".`, "info");
        return;
      }
      state.draft.set(product.id, current + 1);
      renderProducts();
    });

    row.querySelector('[data-action="add"]')?.addEventListener("click", () => {
      addToCart(product);
    });

    list.appendChild(row);
  });
}

function addToCart(product) {
  const available = availableFor(product);
  const quantity = state.draft.get(product.id) || 1;

  if (available <= 0) {
    showToast(`"${product.nombre}" está agotado.`, "error");
    return;
  }

  if (quantity > available) {
    showToast(
      `Solo hay ${available} unidad(es) disponibles de "${product.nombre}".`,
      "error"
    );
    return;
  }

  state.cart.set(product.id, (state.cart.get(product.id) || 0) + quantity);
  state.draft.set(product.id, 1);
  renderProducts();
  renderCart();
  showToast(`"${product.nombre}" agregado al carrito.`, "success");
}

function removeFromCart(productId) {
  state.cart.delete(productId);
  const product = findProduct(productId);
  if (product && state.draft.get(productId) > availableFor(product)) {
    state.draft.set(productId, 1);
  }
  renderProducts();
  renderCart();
}

function cartTotal() {
  let total = 0;
  state.cart.forEach((quantity, productId) => {
    const product = findProduct(productId);
    if (product) total += product.precio * quantity;
  });
  return total;
}

function renderCart() {
  const container = elements.cartContainer;
  if (!container) return;

  container.innerHTML = "";

  if (state.cart.size === 0) {
    elements.emptyCart?.classList.remove("hidden");
    elements.emptyCart?.classList.add("flex");
  } else {
    elements.emptyCart?.classList.add("hidden");
    elements.emptyCart?.classList.remove("flex");
  }

  state.cart.forEach((quantity, productId) => {
    const product = findProduct(productId);
    if (!product) return;

    const subtotal = product.precio * quantity;
    const item = document.createElement("div");
    item.className =
      "flex items-center justify-between p-space-md rounded-lg bg-surface-container-low group";
    item.innerHTML = `
      <div class="flex flex-col pr-2 min-w-0">
        <span class="font-label-md text-label-md text-on-surface cart-name"></span>
        <span class="font-body-sm text-body-sm text-on-surface-variant">${quantity} x ${formatCurrency(product.precio)}</span>
        <span class="font-tabular-amount-md text-tabular-amount-md text-on-surface mt-1">Subtotal: ${formatCurrency(subtotal)}</span>
      </div>
      <button class="p-2 text-outline hover:text-error rounded-lg hover:bg-error-container transition-colors cursor-pointer flex items-center justify-center shrink-0" data-remove="${productId}" title="Quitar" type="button">
        <span class="material-symbols-outlined text-[20px]">delete</span>
      </button>
    `;
    item.querySelector(".cart-name").textContent = product.nombre;
    item.querySelector("[data-remove]").addEventListener("click", () => {
      removeFromCart(productId);
    });
    container.appendChild(item);
  });

  if (elements.cartTotal) {
    elements.cartTotal.textContent = formatCurrency(cartTotal());
  }

  updateGenerateState();
}

function updateGenerateState() {
  const button = elements.generateButton;
  if (!button) return;

  const disabled = state.submitting || !state.clienteId || state.cart.size === 0;
  button.disabled = disabled;
  button.classList.toggle("opacity-50", disabled);
  button.classList.toggle("pointer-events-none", disabled);
}

async function loadData() {
  try {
    const [clientes, productos] = await Promise.all([
      apiGet(API.clientes, "/clientes"),
      apiGet(API.productos, "/productos"),
    ]);
    state.clientes = Array.isArray(clientes) ? clientes : [];
    state.productos = Array.isArray(productos) ? productos : [];
  } catch (error) {
    showToast(error.message, "error");
    state.clientes = [];
    state.productos = [];
  }

  renderClientSelect();
  renderProducts();
  renderCart();
}

async function refreshProducts() {
  const productos = await apiGet(API.productos, "/productos");
  state.productos = Array.isArray(productos) ? productos : [];
  renderProducts();
  renderCart();
}

function adjustedCartAgainstStock() {
  let changed = false;
  const mensajes = [];

  state.cart.forEach((quantity, productId) => {
    const product = state.productos.find((item) => item.id === productId);
    if (!product) {
      state.cart.delete(productId);
      changed = true;
      return;
    }
    if (quantity > product.stock) {
      if (product.stock <= 0) {
        state.cart.delete(productId);
      } else {
        state.cart.set(productId, product.stock);
      }
      changed = true;
      mensajes.push(`"${product.nombre}": ahora hay ${Math.max(product.stock, 0)} disponible(s).`);
    }
  });

  return { changed, mensajes };
}

async function generateInvoice() {
  if (state.submitting) return;

  if (!state.clienteId) {
    showToast("Selecciona un cliente para facturar.", "error");
    return;
  }
  if (state.cart.size === 0) {
    showToast("Agrega al menos un producto al carrito.", "error");
    return;
  }

  state.submitting = true;
  updateGenerateState();
  const button = elements.generateButton;
  setButtonLoading(button, true, "Generando...");

  try {
    // Revalidar stock justo antes de enviar la factura.
    await refreshProducts();
    const { changed, mensajes } = adjustedCartAgainstStock();

    if (changed) {
      renderProducts();
      renderCart();
      showToast(
        `El stock cambió y el carrito se ajustó. ${mensajes.join(" ")}`,
        "info"
      );
      return;
    }

    const items = Array.from(state.cart.entries()).map(([productoId, cantidad]) => ({
      producto_id: productoId,
      cantidad,
    }));

    const factura = await apiPost(API.facturacion, "/facturas", {
      cliente_id: state.clienteId,
      items,
    });

    state.cart.clear();
    showToast(`Factura #${factura.factura_id} generada correctamente.`, "success");
    window.location.href = `factura.html?id=${factura.factura_id}`;
  } catch (error) {
    showToast(error.message, "error");
    if (error.status === 409) {
      try {
        await refreshProducts();
      } catch {
        /* si falla la recarga, se conserva el estado actual */
      }
    }
  } finally {
    state.submitting = false;
    setButtonLoading(button, false);
    updateGenerateState();
  }
}

function bindEvents() {
  elements.clientSelect?.addEventListener("change", (event) => {
    const value = Number(event.target.value);
    state.clienteId = Number.isFinite(value) && value > 0 ? value : null;
    updateClientInfo();
  });

  elements.generateButton?.addEventListener("click", generateInvoice);

  document.getElementById("new-client-link")?.addEventListener("click", () => {
    window.location.href = "clientes.html?nuevo=1";
  });
}

state.usuario = initLayout("nueva-factura");
if (state.usuario) {
  cacheElements();
  if (elements.billingUser) {
    elements.billingUser.textContent = state.usuario.nombre;
  }
  bindEvents();
  loadData();
}
