/** Shared SSL audit rendering for scanner page + host detail tab. */

const SSL_PARAM_KEYS = [
  'ssl_ciphers', 'ssl_cert_file', 'ssl_key_file', 'ssl_ca_file', 'ssl_prefer_server_ciphers',
];

function escapeHtml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function normalizeSslAuditStatus(raw) {
  const s = String(raw || '').toLowerCase().trim();
  if (s === 'pass') return 'pass';
  if (s === 'warning' || s === 'warn') return 'warning';
  if (s === 'critical') return 'critical';
  return 'fail';
}

export function sslBadgeLabel(status) {
  const s = normalizeSslAuditStatus(status);
  if (s === 'pass') return 'Pass';
  if (s === 'warning') return 'Warning';
  return 'Failed';
}

function sslBadgeClass(status) {
  const s = normalizeSslAuditStatus(status);
  if (s === 'pass') return 'pass';
  if (s === 'warning') return 'warning';
  return 'fail';
}

function rowStatusClass(status) {
  const s = normalizeSslAuditStatus(status);
  if (s === 'pass') return 'ssl-check-row--pass';
  if (s === 'warning') return 'ssl-check-row--warning';
  return 'ssl-check-row--fail';
}

function rowBarClass(status) {
  return sslBadgeClass(status);
}

function renderHbaLinesBlock(lines) {
  if (!lines?.length) return '';
  return '<div class="ssl-hba-lines">' +
    '<div class="ssl-hba-lines-label">pg_hba.conf — lines without <code>hostssl</code></div>' +
    '<ul class="ssl-fail-lines">' +
    lines.map(line => '<li><code>' + escapeHtml(String(line).trim()) + '</code></li>').join('') +
    '</ul></div>';
}

export function renderSslAuditSummary(cells) {
  const pass = cells.filter(c => normalizeSslAuditStatus(c.status) === 'pass').length;
  const fail = cells.filter(c => normalizeSslAuditStatus(c.status) === 'fail').length;
  const warning = cells.filter(c => normalizeSslAuditStatus(c.status) === 'warning').length;
  const critical = cells.filter(c => normalizeSslAuditStatus(c.status) === 'critical').length;
  const failed = fail + critical;
  const total = cells.length;
  const passPct = total ? Math.round((pass / total) * 100) : 0;
  const overall = failed > 0 ? 'fail' : (warning > 0 ? 'warning' : 'pass');

  let pills = '<span class="ssl-summary-pill ssl-summary-pill--pass">' + pass + ' pass</span>';
  if (warning) pills += '<span class="ssl-summary-pill ssl-summary-pill--warn">' + warning + ' warning</span>';
  if (failed) {
    pills += '<span class="ssl-summary-pill ssl-summary-pill--fail">' + failed + ' failed</span>';
  }

  return '<div class="ssl-summary">' +
    '<div class="ssl-summary-stats">' + pills +
    '<span class="ssl-summary-total">' + total + ' SSL audit checks</span></div>' +
    '<div class="ssl-summary-track">' +
    '<div class="ssl-summary-fill ssl-summary-fill--' + overall + '" style="width:' + passPct + '%"></div>' +
    '</div></div>';
}

export function renderSslAuditCheckList(cells, hbaLines, options = {}) {
  const emptyMsg = options.emptyMsg ||
    'No SSL audit data. Add ssl_audit to scan_commands or run: ciscollector -r --ssl-check --json';
  const wrap = options.wrap !== false;

  if (!cells?.length && !hbaLines?.length) {
    const empty = '<p class="ssl-empty-msg">' + escapeHtml(emptyMsg) + '</p>';
    return wrap ? empty : empty;
  }

  const hbaAttached = new Set();
  let html = '';

  cells.forEach(c => {
    const status = normalizeSslAuditStatus(c.status);
    const title = c.title || '';
    const message = c.message || c.desc || '';
    const isHbaCheck = /hba/i.test(title);
    const attachHba = isHbaCheck && hbaLines?.length && !hbaAttached.size;
    if (attachHba) hbaAttached.add(1);

    let detail = '';
    if (message) {
      detail += '<div class="hba-check-desc">' + escapeHtml(message) + '</div>';
    }
    if (attachHba) {
      detail += renderHbaLinesBlock(hbaLines);
    }

    html += '<div class="hba-check-row ssl-check-row ' + rowStatusClass(status) + '">' +
      '<div class="hba-check-bar ' + rowBarClass(status) + '"></div>' +
      '<div class="hba-check-main">' +
      '<div class="hba-check-title">' + escapeHtml(title) + '</div>' +
      detail +
      '</div>' +
      '<div class="hba-check-status"><span class="hba-badge ' + sslBadgeClass(status) + '">' +
      sslBadgeLabel(status) + '</span></div></div>';
  });

  if (hbaLines?.length && !hbaAttached.size) {
    html += '<div class="ssl-hba-lines ssl-hba-lines--standalone">' + renderHbaLinesBlock(hbaLines) + '</div>';
  }

  if (!wrap) return html;
  return '<div class="hba-check-list ssl-check-list">' + html + '</div>';
}

export function renderSslParamsPanel(params, options = {}) {
  const keys = SSL_PARAM_KEYS.filter(k => Object.prototype.hasOwnProperty.call(params || {}, k));
  if (!keys.length) return '';

  const title = options.title || 'SSL Parameters';
  const subtitle = options.subtitle || 'PostgreSQL GUC settings from latest SSL audit scan';
  const layout = options.layout || 'report-block';

  let rows = '';
  keys.forEach(k => {
    const raw = params[k];
    const empty = raw == null || String(raw).trim() === '';
    const display = empty ? '—' : String(raw);
    rows += '<tr><td><code>' + escapeHtml(k) + '</code></td><td>' + escapeHtml(display) + '</td></tr>';
  });

  if (layout === 'compact') {
    return '<h4 class="ssl-section-title">' + escapeHtml(title) + '</h4>' +
      '<div class="host-module-table"><table><thead><tr><th>Parameter</th><th>Value</th></tr></thead><tbody>' +
      rows + '</tbody></table></div>';
  }

  return '<article class="report-block">' +
    '<div class="report-block-header"><h2>' + escapeHtml(title) + '</h2>' +
    '<p>' + escapeHtml(subtitle) + '</p></div>' +
    '<div class="report-block-body"><table><thead><tr><th>Parameter</th><th>Value</th></tr></thead><tbody>' +
    rows + '</tbody></table></div></article>';
}

export function renderSslStatsBar(stats) {
  const total = stats.total ?? 0;
  return '<div class="ssl-stats-bar">' +
    '<div class="ssl-stat"><span class="ssl-stat-label">Checks</span><span class="ssl-stat-value" data-stat="total">' + total + '</span></div>' +
    '<div class="ssl-stat ssl-stat--pass"><span class="ssl-stat-label">Pass</span><span class="ssl-stat-value" data-stat="pass">' + (stats.pass ?? 0) + '</span></div>' +
    '<div class="ssl-stat ssl-stat--warn"><span class="ssl-stat-label">Warning</span><span class="ssl-stat-value" data-stat="warning">' + (stats.warning ?? 0) + '</span></div>' +
    '<div class="ssl-stat ssl-stat--fail"><span class="ssl-stat-label">Fail</span><span class="ssl-stat-value" data-stat="fail">' + (stats.fail ?? 0) + '</span></div>' +
    '<div class="ssl-stat ssl-stat--critical"><span class="ssl-stat-label">Critical</span><span class="ssl-stat-value" data-stat="critical">' + (stats.critical ?? 0) + '</span></div>' +
    '</div>';
}
