import { getGucDrift, getGucBaseline, getGucSnapshots, putGucBaseline } from '../api/services/guc.js';
import { paginateSlice, mountTablePagination } from '../utils/pagination.js';

const gucDriftPager = { page: 1, pageSize: 15 };
const gucHostPager = { page: 1, pageSize: 12 };
const gucSnapshotsPager = { page: 1, pageSize: 15 };
const gucHostFilter = { search: '', status: 'all' };
let gucDriftCache = null;
let gucSnapshotsCache = null;
let gucHostToolbarBound = false;

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function statusBadge(status) {
  if (status === 'matched') return 'badge-success';
  if (status === 'drifted' || status === 'drift') return 'badge-danger';
  if (status === 'missing') return 'badge-warning';
  if (status === 'no_snapshot') return 'badge-muted';
  return 'badge-warning';
}

function statusLabel(status) {
  const map = {
    matched: 'Matched',
    drifted: 'Drifting',
    missing: 'Missing keys',
    no_snapshot: 'No snapshot',
    drift: 'Drift',
  };
  return map[status] || status;
}

function formatTimestamp(iso) {
  if (!iso) return '—';
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

/**
 * Parse postgresql.conf-style lines into a GUC map (key -> value).
 * @param {string} text
 * @returns {Record<string, string>}
 */
export function parsePostgresqlConf(text) {
  const settings = {};
  for (const rawLine of text.split(/\r?\n/)) {
    let line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const hash = line.indexOf('#');
    if (hash > 0) {
      line = line.slice(0, hash).trim();
    }
    const eq = line.indexOf('=');
    if (eq < 1) continue;
    const key = line.slice(0, eq).trim();
    let value = line.slice(eq + 1).trim();
    if (
      (value.startsWith("'") && value.endsWith("'")) ||
      (value.startsWith('"') && value.endsWith('"'))
    ) {
      value = value.slice(1, -1);
    }
    if (key) settings[key] = value;
  }
  return settings;
}

/**
 * @param {string} text
 * @param {string} [filename]
 * @returns {{ settings: Record<string, string> }}
 */
export function parseBaselineFileContent(text, filename = '') {
  const trimmed = text.trim();
  const isJson = filename.toLowerCase().endsWith('.json') || trimmed.startsWith('{');
  if (isJson) {
    const parsed = JSON.parse(trimmed);
    const settings = parsed.settings || parsed;
    if (!settings || typeof settings !== 'object' || Array.isArray(settings)) {
      throw new Error('JSON must be an object or { "settings": { ... } }');
    }
    const out = {};
    for (const [k, v] of Object.entries(settings)) {
      if (v != null && String(k).trim()) out[String(k).trim()] = String(v);
    }
    if (!Object.keys(out).length) {
      throw new Error('No GUC settings found in JSON file');
    }
    return { settings: out };
  }
  const settings = parsePostgresqlConf(text);
  if (!Object.keys(settings).length) {
    throw new Error('No GUC settings found — use postgresql.conf (key = value) lines');
  }
  return { settings };
}

function setBaselineUploadStatus(message, isError = false) {
  const el = document.getElementById('guc-baseline-upload-status');
  if (!el) return;
  el.textContent = message || '';
  el.classList.toggle('guc-baseline-upload-status--error', Boolean(isError && message));
  el.classList.toggle('guc-baseline-upload-status--ok', Boolean(!isError && message));
}

function setBaselineUploadOpen(open) {
  const details = document.getElementById('guc-baseline-upload-details');
  if (details) details.open = open;
}

function renderBaselinePanel(baseline) {
  const title = document.getElementById('guc-baseline-title');
  const metaRow = document.getElementById('guc-baseline-meta-row');
  const keysEl = document.getElementById('guc-baseline-keys');
  if (!title || !metaRow || !keysEl) return;

  if (!baseline?.key_count) {
    title.textContent = 'Not configured';
    metaRow.innerHTML = '<span class="guc-meta-chip guc-meta-chip--warn">No baseline uploaded yet</span>';
    keysEl.innerHTML = '<p class="guc-empty-inline">Upload a postgresql.conf below as the reference baseline. Drift is computed when you open this page.</p>';
    setBaselineUploadOpen(true);
    return;
  }

  setBaselineUploadOpen(false);
  const baselineLabel = String(baseline.label || 'global').trim() || 'global';
  title.textContent = baselineLabel.charAt(0).toUpperCase() + baselineLabel.slice(1);
  metaRow.innerHTML =
    '<span class="guc-meta-chip">' + baseline.key_count + ' tracked keys</span>' +
    '<span class="guc-meta-chip">Updated ' + escapeHtml(formatTimestamp(baseline.updated_at)) + '</span>';

  const settings = baseline.settings || {};
  const keys = Object.keys(settings).sort();
  keysEl.innerHTML = keys.map((k) =>
    '<span class="guc-key-pill" title="' + escapeHtml(settings[k]) + '">' +
    escapeHtml(k) + ' = ' + escapeHtml(settings[k]) + '</span>'
  ).join('');
}

function updateStats(data) {
  const sub = document.getElementById('guc-drift-subtitle');
  const banner = document.getElementById('guc-fleet-banner');
  const stats = data?.stats || {};

  if (sub) {
    sub.textContent = 'Golden baseline vs collector SHOW ALL · ' + (stats.hosts_compared ?? 0) + ' server(s)';
  }

  const set = (id, val) => {
    const el = document.getElementById(id);
    if (el) el.textContent = String(val ?? '—');
  };
  set('guc-stat-hosts', stats.hosts_compared);
  set('guc-stat-matched', stats.matched_servers);
  set('guc-stat-drifting', stats.drifting_servers);
  set('guc-stat-missing', stats.missing_servers);
  set('guc-stat-total', stats.total_drifted);

  if (banner) {
    const compared = stats.hosts_compared ?? 0;
    const drifting = stats.drifting_servers ?? 0;
    const missing = stats.missing_servers ?? 0;
    if (compared === 0) {
      banner.hidden = true;
    } else if (drifting === 0 && missing === 0) {
      banner.hidden = false;
      banner.className = 'guc-fleet-banner guc-fleet-banner--ok';
      banner.textContent = 'All ' + compared + ' host(s) match the golden baseline';
    } else {
      banner.hidden = false;
      banner.className = 'guc-fleet-banner guc-fleet-banner--warn';
      banner.textContent = drifting + ' drifting · ' + missing + ' with missing keys';
    }
  }
}

export function filterHostSummaries(summaries, filter) {
  let out = summaries || [];
  const q = (filter?.search || '').trim().toLowerCase();
  if (q) {
    out = out.filter((h) =>
      String(h.host || '').toLowerCase().includes(q) ||
      String(h.target_id || '').toLowerCase().includes(q),
    );
  }
  const status = filter?.status || 'all';
  if (status !== 'all') {
    out = out.filter((h) => h.status === status);
  }
  return out;
}

function renderHostSummaries(data) {
  const tbody = document.getElementById('guc-host-summaries');
  const hint = document.getElementById('guc-host-status-hint');
  const pagerEl = document.getElementById('guc-host-pagination');
  if (!tbody) return;

  const all = data?.host_summaries || [];
  const filtered = filterHostSummaries(all, gucHostFilter);
  const pg = paginateSlice(filtered, gucHostPager.page, gucHostPager.pageSize);
  gucHostPager.page = pg.page;

  if (hint) {
    if (!all.length) {
      hint.textContent = 'No data';
    } else if (filtered.length !== all.length) {
      hint.textContent = filtered.length + ' of ' + all.length + ' hosts';
    } else {
      hint.textContent = all.length + ' host' + (all.length === 1 ? '' : 's') + ' in fleet';
    }
  }

  if (!all.length) {
    tbody.innerHTML =
      '<tr><td colspan="6" class="guc-table-empty">' +
      'No live config yet. Collectors include SHOW ALL with each scan push when mainserver is enabled.</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: gucHostPager.pageSize,
      onPage: () => {}, onPageSize: () => {},
    });
    return;
  }

  if (!filtered.length) {
    tbody.innerHTML =
      '<tr><td colspan="6" class="guc-table-empty">No hosts match your search or filter.</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: gucHostPager.pageSize,
      onPage: () => {}, onPageSize: () => {},
    });
    return;
  }

  tbody.innerHTML = pg.items.map((h) =>
    '<tr class="clickable guc-host-row guc-host-row--' + escapeHtml(h.status) + '" ' +
    'data-goto="host-detail" data-host="' + escapeHtml(h.host) + '" data-section="sub-guc-drift">' +
    '<td><strong>' + escapeHtml(h.host) + '</strong></td>' +
    '<td><span class="badge ' + statusBadge(h.status) + '">' + escapeHtml(statusLabel(h.status)) + '</span></td>' +
    '<td>' + escapeHtml(h.drift_count ?? 0) + '</td>' +
    '<td>' + escapeHtml(h.missing_count ?? 0) + '</td>' +
    '<td class="guc-host-target-cell">' + escapeHtml(h.target_id || '—') + '</td>' +
    '<td><button type="button" class="btn btn-row" data-goto="host-detail" data-host="' +
    escapeHtml(h.host) + '" data-section="sub-guc-drift">Open</button></td></tr>',
  ).join('');

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pg.pageSize,
    pageSizes: [12, 24, 50],
    onPage: (p) => {
      gucHostPager.page = p;
      renderHostSummaries(gucDriftCache);
    },
    onPageSize: (size) => {
      gucHostPager.pageSize = size;
      gucHostPager.page = 1;
      renderHostSummaries(gucDriftCache);
    },
  });
}

function bindGucHostToolbar() {
  if (gucHostToolbarBound) return;
  const searchEl = document.getElementById('guc-host-search');
  const statusEl = document.getElementById('guc-host-status-filter');
  if (!searchEl && !statusEl) return;
  gucHostToolbarBound = true;

  const applyFilter = () => {
    gucHostFilter.search = searchEl?.value || '';
    gucHostFilter.status = statusEl?.value || 'all';
    gucHostPager.page = 1;
    renderHostSummaries(gucDriftCache);
  };

  if (searchEl) {
    searchEl.addEventListener('input', applyFilter);
  }
  if (statusEl) {
    statusEl.addEventListener('change', applyFilter);
  }
}

function renderSnapshotsTable(snapshots) {
  const tbody = document.getElementById('guc-snapshots-tbody');
  const pagerEl = document.getElementById('guc-snapshots-pagination');
  if (!tbody) return;

  gucSnapshotsCache = snapshots;
  const rows = snapshots?.snapshots || [];
  const pg = paginateSlice(rows, gucSnapshotsPager.page, gucSnapshotsPager.pageSize);
  gucSnapshotsPager.page = pg.page;

  if (!rows.length) {
    tbody.innerHTML =
      '<tr><td colspan="4" class="guc-table-empty">' +
      'No snapshots yet — collectors include SHOW ALL with each scan push.</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: gucSnapshotsPager.pageSize,
      onPage: () => {}, onPageSize: () => {},
    });
    return;
  }

  tbody.innerHTML = pg.items.map((r) =>
    '<tr><td><strong>' + escapeHtml(r.host) + '</strong></td>' +
    '<td>' + escapeHtml(formatTimestamp(r.collected_at)) + '</td>' +
    '<td><span class="badge badge-info">' + escapeHtml(r.key_count) + ' keys</span></td>' +
    '<td style="font-size:11px;color:var(--muted);">' + escapeHtml(r.node_id) + '</td></tr>',
  ).join('');

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pg.pageSize,
    pageSizes: [15, 25, 50],
    onPage: (p) => {
      gucSnapshotsPager.page = p;
      renderSnapshotsTable(gucSnapshotsCache);
    },
    onPageSize: (size) => {
      gucSnapshotsPager.pageSize = size;
      gucSnapshotsPager.page = 1;
      renderSnapshotsTable(gucSnapshotsCache);
    },
  });
}

function renderGucDriftTable(data) {
  const tbody = document.getElementById('guc-drift-tbody');
  const pagerEl = document.getElementById('guc-drift-pagination');
  const tableHint = document.getElementById('guc-drift-table-hint');
  if (!tbody) return;

  const rows = data?.rows || [];
  updateStats(data);
  renderHostSummaries(data);

  if (tableHint) {
    tableHint.textContent = rows.length
      ? rows.length + ' difference' + (rows.length === 1 ? '' : 's')
      : 'All clear';
  }

  const pg = paginateSlice(rows, gucDriftPager.page, gucDriftPager.pageSize);
  gucDriftPager.page = pg.page;

  if (!pg.total) {
    tbody.innerHTML =
      '<tr><td colspan="5" class="guc-table-empty guc-table-empty--ok">' +
      'No drift or missing keys — all compared hosts match the baseline.</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: gucDriftPager.pageSize,
      onPage: () => {}, onPageSize: () => {},
    });
    return;
  }

  tbody.innerHTML = pg.items.map((row) => {
    const badge = statusBadge(row.status);
    return '<tr class="clickable" data-goto="host-detail" data-host="' + escapeHtml(row.host) + '" data-section="sub-guc-drift">' +
      '<td><strong>' + escapeHtml(row.host) + '</strong></td>' +
      '<td><code>' + escapeHtml(row.guc) + '</code></td>' +
      '<td><span class="guc-val-live">' + escapeHtml(row.live) + '</span></td>' +
      '<td><span class="guc-val-baseline">' + escapeHtml(row.baseline) + '</span></td>' +
      '<td><span class="badge ' + badge + '">' + escapeHtml(statusLabel(row.status)) + '</span></td></tr>';
  }).join('');

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pg.pageSize,
    onPage: (p) => {
      gucDriftPager.page = p;
      renderGucDriftTable(gucDriftCache);
    },
    onPageSize: (size) => {
      gucDriftPager.pageSize = size;
      gucDriftPager.page = 1;
      renderGucDriftTable(gucDriftCache);
    },
  });
}

let gucBaselineFormBound = false;
let gucBaselineSource = 'file';

function setGucBaselineSource(source) {
  gucBaselineSource = source === 'paste' ? 'paste' : 'file';
  document.querySelectorAll('[data-baseline-source]').forEach((btn) => {
    const active = btn.getAttribute('data-baseline-source') === gucBaselineSource;
    btn.classList.toggle('active', active);
    btn.setAttribute('aria-selected', active ? 'true' : 'false');
  });
  document.querySelectorAll('[data-baseline-panel]').forEach((panel) => {
    const show = panel.getAttribute('data-baseline-panel') === gucBaselineSource;
    panel.classList.toggle('active', show);
    panel.hidden = !show;
  });
}

async function readBaselineInput() {
  if (gucBaselineSource === 'paste') {
    const pasteEl = document.getElementById('guc-baseline-paste');
    const text = (pasteEl?.value || '').trim();
    if (!text) {
      throw new Error('Paste your postgresql.conf content first');
    }
    return parseBaselineFileContent(text);
  }
  const fileInput = document.getElementById('guc-baseline-file');
  const file = fileInput?.files?.[0];
  if (!file) {
    throw new Error('Choose a config file first');
  }
  const text = await file.text();
  return parseBaselineFileContent(text, file.name);
}

async function refreshGucDriftPage() {
  const [baseline, snapshots, data] = await Promise.all([
    getGucBaseline().catch(() => ({ key_count: 0, settings: {} })),
    getGucSnapshots().catch(() => ({ snapshots: [] })),
    getGucDrift(),
  ]);
  renderBaselinePanel(baseline);
  gucSnapshotsPager.page = 1;
  renderSnapshotsTable(snapshots);
  gucDriftCache = data;
  gucDriftPager.page = 1;
  gucHostPager.page = 1;
  gucHostFilter.search = '';
  gucHostFilter.status = 'all';
  const searchEl = document.getElementById('guc-host-search');
  const statusEl = document.getElementById('guc-host-status-filter');
  if (searchEl) searchEl.value = '';
  if (statusEl) statusEl.value = 'all';
  renderGucDriftTable(data);
  return baseline;
}

function bindGucBaselineUploadForm() {
  if (gucBaselineFormBound) return;
  const form = document.getElementById('guc-baseline-form');
  if (!form) return;
  gucBaselineFormBound = true;

  document.querySelectorAll('[data-baseline-source]').forEach((btn) => {
    btn.addEventListener('click', () => {
      setGucBaselineSource(btn.getAttribute('data-baseline-source'));
      setBaselineUploadStatus('');
    });
  });

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const fileInput = document.getElementById('guc-baseline-file');
    const pasteInput = document.getElementById('guc-baseline-paste');
    const submitBtn = document.getElementById('guc-baseline-submit');

    setBaselineUploadStatus('Saving…');
    if (submitBtn) submitBtn.disabled = true;

    try {
      const parsed = await readBaselineInput();
      const result = await putGucBaseline({ settings: parsed.settings });
      setBaselineUploadStatus(
        'Baseline saved — ' + (result.key_count ?? Object.keys(parsed.settings).length) + ' keys tracked',
        false
      );
      setBaselineUploadOpen(false);
      if (fileInput) fileInput.value = '';
      if (pasteInput) pasteInput.value = '';
      await refreshGucDriftPage();
    } catch (err) {
      setBaselineUploadStatus(err.message || 'Save failed', true);
    } finally {
      if (submitBtn) submitBtn.disabled = false;
    }
  });
}

export async function initGucDriftPage() {
  const tbody = document.getElementById('guc-drift-tbody');
  if (!tbody) return;
  bindGucBaselineUploadForm();
  bindGucHostToolbar();
  setBaselineUploadStatus('');
  tbody.innerHTML = '<tr><td colspan="5" class="guc-table-empty">Loading…</td></tr>';
  try {
    await refreshGucDriftPage();
  } catch (err) {
    tbody.innerHTML = '<tr><td colspan="5" class="guc-table-empty" style="color:var(--danger);">Failed to load: ' +
      escapeHtml(err.message) + '</td></tr>';
  }
}
