// Sesión del usuario: guarda en localStorage el usuario público, el token
// opaco que emite el servicio de autenticación y su fecha de expiración.
// El token se envía como "Authorization: Bearer <token>" en cada petición; el
// backend lo valida y aplica las reglas por rol (no es solo seguridad visual).

const STORAGE_KEY = "facturacion_sesion";

const ROLES_VALIDOS = ["admin", "cliente"];

/** Indica si el rol recibido es uno de los válidos ("admin" o "cliente"). */
export function rolValido(rol) {
  return ROLES_VALIDOS.includes(rol);
}

/**
 * Devuelve la sesión guardada, o null si no existe, está incompleta o expiró.
 * Al detectar una sesión inválida la elimina del almacenamiento.
 */
export function getSession() {
  let sesion;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    sesion = raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }

  if (
    !sesion ||
    !sesion.usuario ||
    typeof sesion.token !== "string" ||
    !sesion.token ||
    !rolValido(sesion.usuario.rol)
  ) {
    clearSession();
    return null;
  }

  if (sesion.expira && Date.parse(sesion.expira) <= Date.now()) {
    clearSession();
    return null;
  }

  return sesion;
}

/** Guarda la respuesta del login: { usuario, token, expira }. */
export function setSession({ usuario, token, expira }) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ usuario, token, expira }));
}

export function getUsuario() {
  return getSession()?.usuario ?? null;
}

/** Token opaco de la sesión actual, o cadena vacía si no hay sesión. */
export function getToken() {
  return getSession()?.token ?? "";
}

export function clearSession() {
  localStorage.removeItem(STORAGE_KEY);
}

/**
 * Pantalla principal según el rol: el admin administra el catálogo y el
 * cliente solo accede a sus facturas.
 * @param {string} rol "admin" o "cliente".
 */
export function homeFor(rol) {
  return rol === "admin" ? "productos.html" : "facturas.html";
}

/**
 * Guardia de navegación. Verifica que exista una sesión con token y rol
 * válidos; si no, redirige al login. El backend sigue siendo la autoridad:
 * esta guardia solo evita mostrar pantallas sin sesión.
 */
export function requireSession() {
  const sesion = getSession();
  if (!sesion) {
    window.location.replace("index.html");
    return null;
  }
  return sesion.usuario;
}

/**
 * Guardia por rol: si el rol del usuario no está en la lista permitida,
 * redirige a la pantalla principal de ese rol.
 * @param {string[]} roles Roles autorizados para la página.
 */
export function requireRoles(roles) {
  const usuario = requireSession();
  if (!usuario) return null;

  if (!roles.includes(usuario.rol)) {
    // Si el rol no puede ver esta página, vuelve a su pantalla principal
    // (el cliente a sus facturas; el admin al catálogo).
    window.location.replace(homeFor(usuario.rol));
    return null;
  }
  return usuario;
}

export function redirectIfLoggedIn() {
  const sesion = getSession();
  if (sesion) {
    window.location.replace(homeFor(sesion.usuario.rol));
  }
}
