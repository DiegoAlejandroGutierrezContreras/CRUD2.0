// Login y registro sin tokens.

import { API } from "./config.js";
import { apiPost } from "./api.js";
import { redirectIfLoggedIn, setUsuario } from "./session.js";
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
      const usuario = await apiPost(API.auth, "/auth/login", { email, password });
      setUsuario(usuario);
      window.location.replace("productos.html");
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

  const nameInput = document.getElementById("name");
  const emailInput = document.getElementById("email");
  const passwordInput = document.getElementById("password");
  const rolSelect = document.getElementById("rol");
  const submitButton = document.getElementById("submit-button");
  const errorBanner = document.getElementById("inline-error-banner");

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    hideBanner(errorBanner);

    const nombre = nameInput.value.trim();
    const email = emailInput.value.trim();
    const password = passwordInput.value;
    const rol = rolSelect?.value || "";

    if (!nombre) {
      showBanner(errorBanner, "El nombre es obligatorio.");
      return;
    }
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

    setButtonLoading(submitButton, true, "Creando cuenta...");
    try {
      await apiPost(API.auth, "/auth/register", { nombre, email, password, rol });
      window.location.replace(
        `index.html?registrado=1&email=${encodeURIComponent(email)}`
      );
    } catch (error) {
      showBanner(errorBanner, error.message || "No se pudo crear la cuenta.");
      setButtonLoading(submitButton, false);
    }
  });
}
