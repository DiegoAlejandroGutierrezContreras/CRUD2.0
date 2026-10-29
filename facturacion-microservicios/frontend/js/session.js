// Sesión del usuario sin tokens: solo se guarda el objeto usuario (con su rol)
// en localStorage. El rol controla la navegación del frontend; no es seguridad.

const STORAGE_KEY = "facturacion_usuario";

const ROLES_VALIDOS = ["admin", "cliente"];

/** Indica si el rol recibido es uno de los válidos ("admin" o "cliente"). */
export function rolValido(rol) {
  return ROLES_VALIDOS.includes(rol);
}

/** Retorna true si el usuario (o el de la sesión) tiene rol admin. */
export function isAdmin(usuario = getUsuario()) {
  return usuario?.rol === "admin";
}

export function getUsuario() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function setUsuario(usuario) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(usuario));
}

export function clearUsuario() {
  localStorage.removeItem(STORAGE_KEY);
}

/**
 * Guardia de navegación. No es seguridad real: solo evita mostrar las
 * pantallas internas cuando no se ha iniciado sesión con un rol válido.
 * Las sesiones guardadas sin rol (anteriores a la función de roles) se
 * descartan para pedir un nuevo inicio de sesión.
 */
export function requireSession() {
  const usuario = getUsuario();
  if (!usuario) {
    window.location.replace("index.html");
    return null;
  }
  if (!rolValido(usuario.rol)) {
    clearUsuario();
    window.location.replace("index.html");
    return null;
  }
  return usuario;
}

/**
 * Guardia por rol: si el rol del usuario no está en la lista permitida,
 * redirige al catálogo. Tampoco es seguridad real; solo adapta la interfaz.
 * @param {string[]} roles Roles autorizados para la página.
 */
export function requireRoles(roles) {
  const usuario = requireSession();
  if (!usuario) return null;

  if (!roles.includes(usuario.rol)) {
    window.location.replace("productos.html");
    return null;
  }
  return usuario;
}

export function redirectIfLoggedIn() {
  if (getUsuario()) {
    window.location.replace("productos.html");
  }
}
