import { getLogReadinessFleet } from '../api/services/scanner.js';
import { paginateSlice, mountTablePagination } from '../utils/pagination.js';

const logGucPager = { page: 1, pageSize: 15 };
const logGucFilter = { search: '' };
let logGucCache = { rows: [], message: '' };
let logGucToolbarBound = false;

function escapeHtml(s) {
  return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function connCellClass(ok, value) {
  if (value === '—') return '';
  return ok ? 'icon-pass' : 'icon-fail';
}

function prefixCellClass(cisStatus) {
  switch (cisStatus) {
    case 'pass':
      return 'icon-pass';
    case 'partial':
      return 'icon-warn';
    default:
      return 'icon-fail';
  }
}

function readinessBadge(status) {
  const label = String(status || '').toUpperCase() === 'PASS' ? 'PASS' : 'FAIL';
  const cls = label === 'PASS' ? 'badge-success' : 'badge-danger';
  return '<span class="badge ' + cls + '">' + label + '</span>';
}

function filterLogGucRows(rows) {
  const q = (logGucFilter.search || '').trim().toLowerCase();
  if (!q) return rows || [];
  return (rows || []).filter((row) => String(row.host || '').toLowerCase().includes(q));
}

function renderGucReadinessTable() {
  const tbody = document.getElementById('log-guc-tbody');
  const hint = document.getElementById('log-guc-hint');
  const pagerEl = document.getElementById('log-guc-pagination');
  if (!tbody) return;

  const all = logGucCache.rows || [];
  const filtered = filterLogGucRows(all);
  const pg = paginateSlice(filtered, logGucPager.page, logGucPager.pageSize);
  logGucPager.page = pg.page;

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
    tbody.innerHTML = '<tr><td colspan="5" style="color:var(--muted);">' +
      escapeHtml(logGucCache.message || 'No GUC readiness data. Run collector scans with log parser or postgres_cis.') +
      '</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: logGucPager.pageSize,
      onPage: () => {},
    });
    return;
  }

  if (!filtered.length) {
    tbody.innerHTML = '<tr><td colspan="5" style="color:var(--muted);">No hosts match this search.</td></tr>';
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: logGucPager.pageSize,
      onPage: () => {},
    });
    return;
  }

  tbody.innerHTML = pg.items.map((row) => {
    const connVal = row.logConnections || (row.logConnectionsOk ? 'on' : 'off');
    const connCls = connCellClass(row.logConnectionsOk, connVal);
    const prefixCls = prefixCellClass(row.logLinePrefixCis);
    return '<tr>' +
      '<td>' + escapeHtml(row.host) + '</td>' +
      '<td class="' + connCls + '">' + escapeHtml(connVal) + '</td>' +
      '<td class="' + prefixCls + '">' + escapeHtml(row.logLinePrefix || '—') + '</td>' +
      '<td>' + readinessBadge(row.logparserReadiness) + '</td>' +
      '<td><button type="button" class="btn btn-row" data-goto="host-detail" data-host="' +
      escapeHtml(row.host) + '">Open Host</button></td>' +
      '</tr>';
  }).join('');

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pg.pageSize,
    pageSizes: [15, 25, 50],
    onPage: (p) => {
      logGucPager.page = p;
      renderGucReadinessTable();
    },
    onPageSize: (size) => {
      logGucPager.pageSize = size;
      logGucPager.page = 1;
      renderGucReadinessTable();
    },
  });
}

function bindLogGucToolbar() {
  if (logGucToolbarBound) return;
  logGucToolbarBound = true;
  const searchEl = document.getElementById('log-guc-search');
  if (searchEl) {
    searchEl.addEventListener('input', () => {
      logGucFilter.search = searchEl.value;
      logGucPager.page = 1;
      renderGucReadinessTable();
    });
  }
}

let logReadinessLoaded = false;

export async function initLogReadinessPage() {
  bindLogGucToolbar();
  const tbody = document.getElementById('log-guc-tbody');
  if (tbody && !logReadinessLoaded) {
    tbody.innerHTML = '<tr><td colspan="5" style="color:var(--muted);">Loading GUC readiness…</td></tr>';
  }
  try {
    const data = await getLogReadinessFleet();
    logGucCache = { rows: data?.rows || [], message: data?.message || '' };
    logGucPager.page = 1;
    renderGucReadinessTable();
    logReadinessLoaded = true;
  } catch (err) {
    logGucCache = { rows: [], message: '' };
    if (tbody) {
      tbody.innerHTML = '<tr><td colspan="5" style="color:var(--danger);">' +
        escapeHtml(err.message) + '</td></tr>';
    }
  }
}
