// Cliente HTTP centralizado. No usa tokens ni cabecera Authorization.

export class ApiError extends Error {
  constructor(message, status = 0) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/**
 * Realiza una petición a un microservicio.
 * @param {string} baseUrl URL base del servicio.
 * @param {string} path Ruta del endpoint.
 * @param {RequestInit} options Opciones de fetch.
 */
export async function apiFetch(baseUrl, path, options = {}) {
  const headers = {
    Accept: "application/json",
    ...(options.body ? { "Content-Type": "application/json" } : {}),
    ...(options.headers || {}),
  };

  let response;
  try {
    response = await fetch(baseUrl + path, { ...options, headers });
  } catch {
    throw new ApiError(
      "No se pudo conectar con el servidor. Verifica que los microservicios estén en ejecución.",
      0
    );
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
