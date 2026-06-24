const AVATAR_HUES = [210, 168, 280, 32, 145, 12];

export function escapeHtml(s) {
  return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function userAvatarHue(name) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h + name.charCodeAt(i) * (i + 1)) % AVATAR_HUES.length;
  return AVATAR_HUES[h];
}

export function renderUserCell(user) {
  const safe = escapeHtml(user);
  const initial = escapeHtml((user || '?').charAt(0).toUpperCase());
  const hue = userAvatarHue(user || '');
  return (
    '<div class="user-report-user-cell">' +
    '<span class="user-report-avatar" style="--avatar-hue:' + hue + '" aria-hidden="true">' + initial + '</span>' +
    '<span class="user-report-username" title="' + safe + '">' + safe + '</span>' +
    '</div>'
  );
}

export function renderInstanceCell(instance) {
  const safe = escapeHtml(instance);
  return '<strong class="user-report-instance">' + safe + '</strong>';
}

export function renderDatabasesCell(label) {
  const safe = escapeHtml(label || '-');
  return '<span class="user-report-databases" title="' + safe + '">' + safe + '</span>';
}

export function renderHostCell(host) {
  const inst = String(host || '').includes('/') ? String(host).replace(/\/[^/]+$/, '') : host;
  return renderInstanceCell(inst);
}

export function renderStatusBadge(status) {
  const safe = escapeHtml(status || '—');
  const lower = safe.toLowerCase();
  let tone = 'neutral';
  if (lower.includes('inactive') || lower.includes('never')) tone = 'warn';
  if (lower.includes('active') || lower.includes('pass')) tone = 'ok';
  const cls = tone === 'warn' ? 'badge-posture-fail' : tone === 'ok' ? 'badge-success' : 'badge-muted';
  return '<span class="badge user-report-status ' + cls + '">' + safe + '</span>';
}

export function rowInstance(row) {
  return row?.instance || row?.host || '';
}

export function renderEmptyRow(colspan, message, className = 'user-report-empty') {
  return '<tr><td colspan="' + colspan + '" class="' + className + '">' + escapeHtml(message) + '</td></tr>';
}
