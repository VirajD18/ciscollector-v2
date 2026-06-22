/** DOM helpers used across dashboard modules. */
export function gid(id) {
  return document.getElementById(id);
}

export function gval(id, def) {
  const n = gid(id);
  if (!n || n.value === undefined || n.value === null) {
    return def === undefined ? '' : def;
  }
  return n.value;
}

export function gon(id, type, fn) {
  const n = gid(id);
  if (n) n.addEventListener(type, fn);
}

export function gq(sel) {
  return document.querySelector(sel);
}

/** True when element id has className (e.g. active page check). */
export function gcls(id, className) {
  const el = gid(id);
  return !!el?.classList.contains(className);
}

export async function loadHtml(url) {
  const res = await fetch(url, { credentials: 'same-origin' });
  if (!res.ok) throw new Error(`Failed to load ${url}: ${res.status}`);
  return res.text();
}
