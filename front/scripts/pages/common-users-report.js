import { getCommonUsersReport } from '../api/services/user-reports.js';
import { paginateSlice, mountTablePagination } from '../utils/pagination.js';
import {
  escapeHtml,
  renderUserCell,
  renderInstanceCell,
  renderDatabasesCell,
  renderEmptyRow,
  rowInstance,
} from '../utils/user-report-ui.js';

const pager = { page: 1, pageSize: 15 };
const filter = { search: '' };
let cache = { rows: [], message: '', userCount: 0, hostCount: 0 };
let toolbarBound = false;

function updateStats(all, filtered) {
  const usersEl = document.getElementById('common-users-stat-users');
  const hostsEl = document.getElementById('common-users-stat-hosts');
  if (usersEl) {
    usersEl.textContent = String(all.length ? (filtered.length !== all.length ? filtered.length : (cache.userCount || all.length)) : 0);
  }
  if (hostsEl) {
    const hostSet = new Set((filter.search ? filtered : all).map((r) => rowInstance(r)));
    hostsEl.textContent = String(hostSet.size || cache.hostCount || 0);
  }
}

function rowDatabasesLabel(row) {
  return row?.databases_label || row?.databasesLabel || '-';
}

function filterRows(rows) {
  const q = (filter.search || '').trim().toLowerCase();
  if (!q) return rows || [];
  return (rows || []).filter((row) => {
    const blob = [row.user, rowInstance(row), rowDatabasesLabel(row)].join(' ').toLowerCase();
    return blob.includes(q);
  });
}

function renderTable() {
  const tbody = document.getElementById('common-users-tbody');
  const hint = document.getElementById('common-users-hint');
  const pagerEl = document.getElementById('common-users-pagination');
  if (!tbody) return;

  const all = cache.rows || [];
  const filtered = filterRows(all);
  const pg = paginateSlice(filtered, pager.page, pager.pageSize);
  pager.page = pg.page;

  updateStats(all, filtered);

  if (hint) {
    if (!all.length) {
      hint.textContent = 'No data yet';
    } else {
      hint.textContent = filtered.length + ' role' + (filtered.length === 1 ? '' : 's') +
        (filtered.length !== all.length ? ' (filtered)' : '') + ' across fleet';
    }
  }

  if (!all.length) {
    tbody.innerHTML = renderEmptyRow(4, cache.message || 'No login-capable users in latest scans.');
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: pager.pageSize,
      onPage: () => {},
    });
    return;
  }

  if (!filtered.length) {
    tbody.innerHTML = renderEmptyRow(4, 'No rows match this search.');
    mountTablePagination(pagerEl, {
      page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize: pager.pageSize,
      onPage: () => {},
    });
    return;
  }

  tbody.innerHTML = pg.items.map((row) => {
    const inst = rowInstance(row);
    return '<tr class="user-report-row">' +
    '<td class="user-report-col-user">' + renderUserCell(row.user) + '</td>' +
    '<td class="user-report-col-host">' + renderInstanceCell(inst) + '</td>' +
    '<td class="user-report-col-databases">' + renderDatabasesCell(rowDatabasesLabel(row)) + '</td>' +
    '<td class="user-report-col-action">' +
    '<button type="button" class="btn btn-row user-report-action" data-goto="host-detail" data-host-instance="' +
    escapeHtml(inst) + '">View Host <span aria-hidden="true">→</span></button>' +
    '</td>' +
    '</tr>';
  }).join('');

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pager.pageSize,
    pageSizes: [15, 25, 50],
    onPage: (p) => {
      pager.page = p;
      renderTable();
    },
    onPageSize: (size) => {
      pager.pageSize = size;
      pager.page = 1;
      renderTable();
    },
  });
}

function bindToolbar() {
  if (toolbarBound) return;
  toolbarBound = true;
  const searchEl = document.getElementById('common-users-search');
  if (searchEl) {
    searchEl.addEventListener('input', () => {
      filter.search = searchEl.value;
      pager.page = 1;
      renderTable();
    });
  }
}

let loaded = false;

export async function initCommonUsersReportPage({ force = false } = {}) {
  bindToolbar();
  const tbody = document.getElementById('common-users-tbody');
  if (tbody && (!loaded || force)) {
    tbody.innerHTML = renderEmptyRow(4, 'Loading common users…');
  }
  try {
    const data = await getCommonUsersReport();
    cache = {
      rows: data?.rows || [],
      message: data?.message || '',
      userCount: data?.userCount ?? 0,
      hostCount: data?.hostCount ?? 0,
    };
    pager.page = 1;
    renderTable();
    loaded = true;
  } catch (err) {
    cache = { rows: [], message: '', userCount: 0, hostCount: 0 };
    if (tbody) {
      tbody.innerHTML = renderEmptyRow(4, err.message, 'user-report-empty user-report-empty--error');
    }
  }
}
