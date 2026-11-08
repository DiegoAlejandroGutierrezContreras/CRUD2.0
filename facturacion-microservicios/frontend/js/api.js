// Cliente HTTP centralizado. Adjunta el token de la sesión y, si el backend
// responde 401 con un token enviado, cierra la sesión local y vuelve al login.

import { clearSession, getToken } from "./session.js";

const TIMEOUT_MS = 15000;

export class ApiError extends Error {
  constructor(message, status = 0) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** Redirige al login cuando la sesión ya no es válida en el servidor. */
function sesionExpirada() {
  clearSession();
  const path = window.location.pathname;
  if (!path.endsWith("index.html") && path !== "/") {
    window.location.replace("index.html?expirado=1");
  }
}

/**
 * Realiza una petición a un microservicio.
 * @param {string} baseUrl URL base del servicio.
 * @param {string} path Ruta del endpoint.
 * @param {RequestInit} options Opciones de fetch.
 */
export async function apiFetch(baseUrl, path, options = {}) {
  const token = getToken();
  const headers = {
    Accept: "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.body ? { "Content-Type": "application/json" } : {}),
    ...(options.headers || {}),
  };

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);

  let response;
  try {
    response = await fetch(baseUrl + path, {
      ...options,
      headers,
      signal: options.signal || controller.signal,
    });
  } catch (error) {
    if (error?.name === "AbortError") {
      throw new ApiError("El servidor tardó demasiado en responder.", 0);
    }
    throw new ApiError(
      "No se pudo conectar con el servidor. Verifica que los microservicios estén en ejecución.",
      0
    );
  } finally {
    clearTimeout(timer);
  }

  if (response.status === 401 && token) {
    sesionExpirada();
  }

  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  let data = null;

  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }

  if (!response.ok) {
    const message =
      data && typeof data === "object" && data.error
        ? data.error
        : typeof data === "string" && data
          ? data
          : `Error ${response.status}`;
    throw new ApiError(message, response.status);
  }

  return data;
}

export const apiGet = (baseUrl, path) => apiFetch(baseUrl, path);

export const apiPost = (baseUrl, path, body) =>
  apiFetch(baseUrl, path, { method: "POST", body: JSON.stringify(body) });

export const apiPut = (baseUrl, path, body) =>
  apiFetch(baseUrl, path, { method: "PUT", body: JSON.stringify(body) });

export const apiDelete = (baseUrl, path) =>
  apiFetch(baseUrl, path, { method: "DELETE" });
