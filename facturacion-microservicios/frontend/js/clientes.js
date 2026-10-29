// CRUD de clientes: listado, creación, edición y eliminación.

import { API } from "./config.js";
import { apiDelete, apiGet, apiPost, apiPut } from "./api.js";
import { initLayout } from "./layout.js";
import { escapeHTML, initials, setButtonLoading, showToast } from "./ui.js";

let clients = [];
let modalElement = null;
let editingClientId = null;

function renderClients(list) {
  const tbody = document.getElementById("clients-table-body");
  const table = document.getElementById("clients-table");
  const empty = document.getElementById("empty-state");
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

  list.forEach((client) => {
    const row = document.createElement("tr");
    row.className =
      "h-14 transition-colors duration-150 hover:bg-surface-container-low/60";
    row.innerHTML = `
      <td class="px-6 py-4 font-tabular-amount-md text-tabular-amount-md text-on-surface-variant">${client.id}</td>
      <td class="px-6 py-4 font-body-md text-body-md text-on-surface font-medium">
        <span class="inline-flex items-center gap-2">
          <span class="w-7 h-7 rounded-full bg-secondary-container text-on-secondary-container inline-flex items-center justify-center font-label-sm text-label-sm">${escapeHTML(initials(client.nombre))}</span>
          <span class="client-name"></span>
        </span>
      </td>
      <td class="px-6 py-4 font-tabular-amount-md text-tabular-amount-md text-on-surface"></td>
      <td class="px-6 py-4 font-body-md text-body-md text-on-surface-variant"></td>
      <td class="px-6 py-4 font-body-md text-body-md text-on-surface-variant"></td>
      <td class="px-6 py-4 text-right whitespace-nowrap client-actions">
        <button class="h-8 px-3 inline-flex items-center rounded-lg font-label-md text-label-md bg-surface-container-low text-on-surface hover:bg-surface-container transition-colors" data-action="edit" type="button">Editar</button>
        <button class="h-8 px-3 inline-flex items-center rounded-lg font-label-md text-label-md bg-error-container/60 text-error hover:bg-error-container transition-colors" data-action="delete" type="button">Eliminar</button>
      </td>
    `;
    row.querySelector(".client-name").textContent = client.nombre;
    row.querySelector("td:nth-child(3)").textContent = client.cc;
    row.querySelector("td:nth-child(4)").textContent = client.email;
    row.querySelector("td:nth-child(5)").textContent = client.direccion;
    row
      .querySelector('[data-action="edit"]')
      .addEventListener("click", () => openModal(client));
    row
      .querySelector('[data-action="delete"]')
      .addEventListener("click", () => deleteClient(client));
    tbody.appendChild(row);
  });
}

async function loadClients() {
  try {
    clients = await apiGet(API.clientes, "/clientes");
    renderClients(Array.isArray(clients) ? clients : []);
  } catch (error) {
    showToast(error.message, "error");
  }
}

/**
 * Abre el modal en modo creación (sin argumento) o edición (con el cliente).
 * @param {object|null} client Cliente a editar.
 */
function openModal(client = null) {
  if (!modalElement) return;

  editingClientId = client ? client.id : null;

  const title = document.getElementById("modal-title");
  const subtitle = document.getElementById("modal-subtitle");
  const saveText = document.getElementById("save-client-btn-text");

  if (title) title.textContent = client ? "Editar cliente" : "Nuevo cliente";
  if (subtitle) {
    subtitle.textContent = client
      ? "Actualiza los datos del cliente seleccionado."
      : "Ingresa los datos para incorporar el cliente al directorio de facturación.";
  }
  if (saveText) saveText.textContent = client ? "Guardar cambios" : "Guardar cliente";

  document.getElementById("client-name").value = client?.nombre ?? "";
  document.getElementById("client-cedula").value = client?.cc ?? "";
  document.getElementById("client-email").value = client?.email ?? "";
  document.getElementById("client-address").value = client?.direccion ?? "";

  showFormError("");
  modalElement.classList.remove("hidden");
  modalElement.classList.add("flex");
  document.body.classList.add("overflow-hidden");
  document.getElementById("client-name")?.focus();
}

function closeModal() {
  if (!modalElement) return;
  modalElement.classList.add("hidden");
  modalElement.classList.remove("flex");
  document.body.classList.remove("overflow-hidden");
  editingClientId = null;
}

async function deleteClient(client) {
  const confirmed = window.confirm(
    `¿Eliminar el cliente "${client.nombre}" (ID ${client.id})? Esta acción no se puede deshacer.`
  );
  if (!confirmed) return;

  try {
    await apiDelete(API.clientes, `/clientes/${client.id}`);
    showToast("Cliente eliminado correctamente.", "success");
    await loadClients();
  } catch (error) {
    showToast(error.message || "No se pudo eliminar el cliente.", "error");
  }
}

function showFormError(message) {
  const banner = document.getElementById("client-form-error");
  const text = document.getElementById("client-form-error-text");
  if (!banner) return;
  if (!message) {
    banner.classList.add("hidden");
    return;
  }
  banner.classList.remove("hidden");
  if (text) text.textContent = message;
}

function initModal() {
  modalElement = document.getElementById("client-modal");
  if (!modalElement) return;

  document.getElementById("new-client-button")?.addEventListener("click", () => openModal());
  document.getElementById("empty-new-client-button")?.addEventListener("click", () => openModal());
  document.getElementById("close-modal-btn")?.addEventListener("click", closeModal);
  document.getElementById("cancel-btn")?.addEventListener("click", closeModal);

  modalElement.addEventListener("click", (event) => {
    if (event.target === modalElement) closeModal();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !modalElement.classList.contains("hidden")) {
      closeModal();
    }
  });

  const form = document.getElementById("new-client-form");
  form?.addEventListener("submit", async (event) => {
    event.preventDefault();
    showFormError("");

    const nombre = document.getElementById("client-name").value.trim();
    const cc = document.getElementById("client-cedula").value.trim();
    const email = document.getElementById("client-email").value.trim();
    const direccion = document.getElementById("client-address").value.trim();
    const submitButton = document.getElementById("save-client-btn");

    if (!nombre) {
      showFormError("El nombre es obligatorio.");
      return;
    }
    if (!/^\d+$/.test(cc)) {
      showFormError("La cédula debe ser numérica.");
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      showFormError("Ingresa un correo electrónico válido.");
      return;
    }
    if (!direccion) {
      showFormError("La dirección es obligatoria.");
      return;
    }

    const payload = { nombre, cc, email, direccion };

    setButtonLoading(submitButton, true, "Guardando...");
    try {
      if (editingClientId) {
        await apiPut(API.clientes, `/clientes/${editingClientId}`, payload);
        showToast("Cliente actualizado correctamente.", "success");
      } else {
        await apiPost(API.clientes, "/clientes", payload);
        showToast("Cliente creado correctamente.", "success");
      }
      closeModal();
      form.reset();
      await loadClients();
    } catch (error) {
      showFormError(error.message);
    } finally {
      setButtonLoading(submitButton, false);
    }
  });

  if (new URLSearchParams(window.location.search).get("nuevo") === "1") {
    openModal();
  }
}

initLayout("clientes");
initModal();
loadClients();
