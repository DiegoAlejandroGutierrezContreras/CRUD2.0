// Login y registro. El login guarda el usuario y el token opaco de sesión.

import { API } from "./config.js";
import { apiPost } from "./api.js";
import { homeFor, redirectIfLoggedIn, setSession } from "./session.js";
import { setButtonLoading } from "./ui.js";

function showBanner(element, message, type = "error") {
  if (!element) return;
  const textTarget = element.querySelector("[data-banner-text]") || element.querySelector("span:last-child");
  if (textTarget) {
    textTarget.textContent = message || "";
  } else {
    element.textContent = message || "";
  }
  element.classList.remove("hidden");
  if (type === "success") {
    element.classList.remove("bg-error-container", "text-on-error-container");
    element.classList.add("bg-tertiary-container", "text-on-tertiary");
  } else {
    element.classList.remove("bg-tertiary-container", "text-on-tertiary");
    element.classList.add("bg-error-container", "text-on-error-container");
  }
}

function hideBanner(element) {
  element?.classList.add("hidden");
}

export function initLogin() {
  redirectIfLoggedIn();

  const form = document.getElementById("login-form");
  if (!form) return;

  const emailInput = document.getElementById("email");
  const passwordInput = document.getElementById("password");
  const submitButton = document.getElementById("submit-button");
  const errorBanner = document.getElementById("error-banner");
  const successBanner = document.getElementById("success-banner");
  const params = new URLSearchParams(window.location.search);

  if (params.get("registrado") === "1") {
    showBanner(successBanner, "Cuenta creada correctamente. Inicia sesión.", "success");
    const email = params.get("email");
    if (email) emailInput.value = email;
  } else if (params.get("expirado") === "1") {
    showBanner(errorBanner, "Tu sesión expiró. Inicia sesión nuevamente.");
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    hideBanner(errorBanner);

    const email = emailInput.value.trim();
    const password = passwordInput.value;

    if (!email || !password) {
      showBanner(errorBanner, "Ingresa tu correo y contraseña.");
      return;
    }

    setButtonLoading(submitButton, true, "Verificando...");
    try {
      const login = await apiPost(API.auth, "/auth/login", { email, password });
      setSession(login);
      // El cliente entra directo a sus facturas; el admin, al catálogo.
      window.location.replace(homeFor(login.usuario?.rol));
    } catch (error) {
      showBanner(errorBanner, error.message || "No se pudo iniciar sesión.");
      passwordInput.value = "";
      passwordInput.focus();
    } finally {
      setButtonLoading(submitButton, false);
    }
  });
}

export function initRegister() {
  redirectIfLoggedIn();

  const form = document.getElementById("signup-form");
  if (!form) return;

  const nameField = document.getElementById("name-field");
  const nameInput = document.getElementById("name");
  const emailInput = document.getElementById("email");
  const passwordInput = document.getElementById("password");
  const rolSelect = document.getElementById("rol");
  const submitButton = document.getElementById("submit-button");
  const errorBanner = document.getElementById("inline-error-banner");

  // El rol manda: el cliente no escribe nombre, solo correo y contraseña.
  function actualizarCampoNombre() {
    const rol = rolSelect?.value || "";
    if (nameField && nameInput) {
      nameField.hidden = rol !== "admin";
      if (rol !== "admin") nameInput.value = "";
    }
  }

  rolSelect?.addEventListener("change", actualizarCampoNombre);
  actualizarCampoNombre();

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    hideBanner(errorBanner);

    const rol = rolSelect?.value || "";
    const email = emailInput.value.trim();
    const password = passwordInput.value;
    const nombre = rol === "admin" ? nameInput.value.trim() : "";

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      showBanner(errorBanner, "Ingresa un correo electrónico válido.");
      return;
    }
    if (password.length < 6) {
      showBanner(errorBanner, "La contraseña debe tener al menos 6 caracteres.");
      return;
    }
    if (rol !== "admin" && rol !== "cliente") {
      showBanner(errorBanner, "Selecciona el tipo de cuenta (rol).");
      return;
    }
    if (rol === "admin" && !nombre) {
      showBanner(errorBanner, "El nombre es obligatorio para la cuenta de administrador.");
      return;
    }

    setButtonLoading(submitButton, true, "Creando cuenta...");
    try {
      const cuerpo = rol === "admin" ? { nombre, email, password, rol } : { email, password, rol };
      await apiPost(API.auth, "/auth/register", cuerpo);
      window.location.replace(
        `index.html?registrado=1&email=${encodeURIComponent(email)}`
      );
    } catch (error) {
      showBanner(errorBanner, error.message || "No se pudo crear la cuenta.");
      setButtonLoading(submitButton, false);
    }
  });
}
