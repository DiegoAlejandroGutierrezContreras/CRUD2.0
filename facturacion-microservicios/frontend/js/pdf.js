// Generador de PDF mínimo para descargar una factura sin depender de
// librerías externas. Construye un PDF 1.4 con texto Helvetica y codificación
// WinAnsi, suficiente para el comprobante que muestra la interfaz.

const PAGE_W = 595.28; // A4 en puntos
const PAGE_H = 841.89;
const MARGIN = 50;
const CONTENT_W = PAGE_W - MARGIN * 2;
const RIGHT = PAGE_W - MARGIN;
const GRAY = "0.42 0.42 0.42";

// Caracteres Unicode fuera de Latin-1 que se traducen a WinAnsi o a ASCII.
const UNICODE_MAP = {
  "€": "\x80",
  "—": "-",
  "–": "-",
  "“": '"',
  "”": '"',
  "‘": "'",
  "’": "'",
  "…": "...",
};

/** Convierte un texto a una cadena de bytes WinAnsi (un carácter por byte). */
function toWinAnsi(value) {
  let out = "";
  for (const char of String(value ?? "")) {
    if (UNICODE_MAP[char] !== undefined) {
      out += UNICODE_MAP[char];
      continue;
    }
    const code = char.codePointAt(0);
    out += code <= 0xff ? char : "?";
  }
  return out;
}

/** Escapa un texto para incluirlo dentro de un literal PDF `( ... )`. */
function escapePDF(value) {
  return toWinAnsi(value)
    .replace(/\\/g, "\\\\")
    .replace(/\(/g, "\\(")
    .replace(/\)/g, "\\)")
    .replace(/[\r\n]+/g, " ");
}

/** Formatea un número para los operadores del PDF: 2 decimales como máximo. */
function num(value) {
  return String(Math.round(value * 100) / 100);
}

/** Ancho aproximado del texto en puntos (alineación de importes). */
function textWidth(text, size, bold) {
  let units = 0;
  for (const char of text) {
    if (" il.,:;'|!".includes(char)) units += 0.28;
    else if ("MW@%".includes(char)) units += 0.88;
    else units += 0.55;
  }
  return units * size * (bold ? 1.04 : 1);
}

function truncate(value, max) {
  const text = String(value ?? "");
  return text.length > max ? `${text.slice(0, max - 1)}…` : text;
}

function formatMoney(value) {
  const amount = Math.round(Number(value) || 0);
  return `$ ${String(amount).replace(/\B(?=(\d{3})+(?!\d))/g, ".")}`;
}

function formatDate(value) {
  if (!value) return "—";
  const parts = String(value).split("-");
  if (parts.length !== 3) return String(value);
  return `${parts[2]}/${parts[1]}/${parts[0]}`;
}

function rolLabel(rol) {
  if (rol === "admin") return "Administrador";
  if (rol === "cliente") return "Cliente";
  return "—";
}

/** Contenido de una página PDF. Las coordenadas crecen hacia arriba. */
class PdfPage {
  constructor() {
    this.ops = [];
  }

  text(x, y, size, bold, value, options = {}) {
    const content = String(value ?? "");
    const { align = "left", color = "0 0 0" } = options;
    let px = x;
    if (align === "right") px = x - textWidth(content, size, bold);
    else if (align === "center") px = x - textWidth(content, size, bold) / 2;

    this.ops.push(
      `BT ${color} rg /${bold ? "F2" : "F1"} ${num(size)} Tf ` +
        `${num(px)} ${num(y)} Td (${escapePDF(content)}) Tj ET`
    );
  }

  band(y, height, shade = 0.95) {
    this.ops.push(
      `${num(shade)} ${num(shade)} ${num(shade)} rg ` +
        `${num(MARGIN)} ${num(y)} ${num(CONTENT_W)} ${num(height)} re f 0 0 0 rg`
    );
  }

  line(y, x1 = MARGIN, x2 = RIGHT) {
    this.ops.push(
      `0.85 0.85 0.85 RG 0.6 w ${num(x1)} ${num(y)} m ${num(x2)} ${num(y)} l S`
    );
  }

  content() {
    return this.ops.join("\n");
  }
}

function drawBlock(page, x, y, title, lines) {
  page.text(x, y, 8, true, title, { color: GRAY });
  let cursor = y - 15;
  lines.forEach((line, index) => {
    page.text(x, cursor, index === 0 ? 11 : 9, index === 0, truncate(line, 42));
    cursor -= index === 0 ? 16 : 14;
  });
}

function drawTableHead(page, topY) {
  page.band(topY - 13, 16);
  const baseline = topY - 9;
  page.text(MARGIN + 4, baseline, 8, true, "Producto");
  page.text(360, baseline, 8, true, "Cantidad", { align: "right" });
  page.text(460, baseline, 8, true, "Precio unit.", { align: "right" });
  page.text(RIGHT, baseline, 8, true, "Subtotal", { align: "right" });
  return topY - 32;
}

/** Distribuye la factura en una o varias páginas. */
function renderPages(factura) {
  const pages = [new PdfPage()];
  let page = pages[0];
  let y = PAGE_H - MARGIN;

  page.text(MARGIN, y, 20, true, `FACTURA #${factura.factura_id}`);
  y -= 16;
  page.text(MARGIN, y, 9, false, "Sistema de Facturación Digital", { color: GRAY });
  y -= 14;
  page.text(MARGIN, y, 9, false, `Fecha de emisión: ${formatDate(factura.fecha)}`, {
    color: GRAY,
  });
  y -= 12;
  page.line(y);
  y -= 26;

  drawBlock(page, MARGIN, y, "CLIENTE", [
    factura.cliente?.nombre || "—",
    factura.cliente?.email || "—",
    `ID: ${factura.cliente?.id ?? "—"}`,
  ]);
  drawBlock(page, 320, y, "ATENDIDO POR", [
    factura.usuario?.nombre || "—",
    `Rol: ${rolLabel(factura.usuario?.rol)}`,
  ]);
  y -= 74;
  page.line(y);
  y -= 22;

  page.text(MARGIN, y, 10, true, "DETALLE DE LA FACTURA");
  y -= 12;
  y = drawTableHead(page, y);

  (factura.productos || []).forEach((item) => {
    if (y < 110) {
      page = new PdfPage();
      pages.push(page);
      y = PAGE_H - MARGIN;
      page.text(
        MARGIN,
        y,
        14,
        true,
        `FACTURA #${factura.factura_id} (continuación)`,
        { color: GRAY }
      );
      y -= 26;
      y = drawTableHead(page, y);
    }

    page.text(MARGIN + 4, y, 9, false, truncate(item.nombre, 52));
    page.text(360, y, 9, false, String(item.cantidad ?? ""), { align: "right" });
    page.text(460, y, 9, false, formatMoney(item.precio_unitario), {
      align: "right",
    });
    page.text(RIGHT, y, 9, true, formatMoney(item.subtotal), { align: "right" });
    page.line(y - 5);
    y -= 18;
  });

  y -= 10;
  page.line(y);
  y -= 26;
  page.text(RIGHT - 130, y, 11, true, "TOTAL", { align: "right" });
  page.text(RIGHT, y, 15, true, formatMoney(factura.total), { align: "right" });

  pages.forEach((item, index) => {
    item.text(MARGIN, 36, 8, false, "Documento generado por el Sistema de Facturación Digital", {
      color: GRAY,
    });
    item.text(RIGHT, 36, 8, false, `Página ${index + 1} de ${pages.length}`, {
      align: "right",
      color: GRAY,
    });
  });

  return pages;
}

/**
 * Construye los bytes de un PDF con el detalle de la factura.
 * @param {object} factura Factura devuelta por el servicio de facturación.
 * @returns {Uint8Array}
 */
export function buildFacturaPDF(factura) {
  const pages = renderPages(factura);
  const firstPageObject = 6; // 1 catálogo, 2 páginas, 3/4 fuentes, 5 info

  const objects = [];
  const kids = pages.map((_, index) => `${firstPageObject + index * 2} 0 R`).join(" ");
  objects[1] = "<< /Type /Catalog /Pages 2 0 R >>";
  objects[2] = `<< /Type /Pages /Kids [${kids}] /Count ${pages.length} >>`;
  objects[3] =
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>";
  objects[4] =
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>";
  objects[5] =
    `<< /Title (${escapePDF(`Factura #${factura.factura_id}`)}) ` +
    "/Producer (Sistema de Facturacion Digital) >>";

  pages.forEach((page, index) => {
    const content = page.content();
    const pageObject = firstPageObject + index * 2;
    objects[pageObject] =
      `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${PAGE_W} ${PAGE_H}] ` +
      `/Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> ` +
      `/Contents ${pageObject + 1} 0 R >>`;
    objects[pageObject + 1] =
      `<< /Length ${content.length} >>\nstream\n${content}\nendstream`;
  });

  const parts = [];
  const offsets = [];
  let offset = 0;
  const push = (chunk) => {
    parts.push(chunk);
    offset += chunk.length;
  };

  push("%PDF-1.4\n%âãÏÓ\n");
  for (let index = 1; index < objects.length; index += 1) {
    offsets[index] = offset;
    push(`${index} 0 obj\n${objects[index]}\nendobj\n`);
  }

  const startxref = offset;
  push(`xref\n0 ${objects.length}\n`);
  push("0000000000 65535 f \n");
  for (let index = 1; index < objects.length; index += 1) {
    push(`${String(offsets[index]).padStart(10, "0")} 00000 n \n`);
  }
  push(
    `trailer\n<< /Size ${objects.length} /Root 1 0 R /Info 5 0 R >>\n` +
      `startxref\n${startxref}\n%%EOF\n`
  );

  // Cada carácter de `parts` es un byte WinAnsi, así que la longitud de la
  // cadena coincide con el desplazamiento real del archivo.
  const bytes = new Uint8Array(offset);
  let position = 0;
  for (const part of parts) {
    for (let index = 0; index < part.length; index += 1) {
      bytes[position] = part.charCodeAt(index) & 0xff;
      position += 1;
    }
  }
  return bytes;
}

/** Descarga la factura como PDF con el nombre `factura-<id>.pdf`. */
export function downloadFacturaPDF(factura) {
  const bytes = buildFacturaPDF(factura);
  const blob = new Blob([bytes], { type: "application/pdf" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `factura-${factura.factura_id}.pdf`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1500);
}
