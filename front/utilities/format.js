/** Shared formatting helpers. */
export function formatPercent(value) {
  if (typeof value === 'string' && value.endsWith('%')) return value;
  return `${Math.round(Number(value) || 0)}%`;
}

export function hostLabel(name, ip) {
  return ip ? `${name} (${ip})` : name;
}
