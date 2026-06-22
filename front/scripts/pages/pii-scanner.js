import { getPiiScanner } from '../api/services/scanner.js';
import { hostsApi } from '../api/index.js';
import { getHostInstance } from '../api/services/host-report.js';
import { mapHostsResponse } from '../api/mappers.js';
import { paginateSlice, mountTablePagination } from '../utils/pagination.js';
import {
  buildPiiDisplayRows,
  countPiiTables,
  renderPiiDataTableRows,
  renderPiiEmptyState,
  renderPiiLowConfChips,
  renderPiiMetaTableRows,
  renderPiiSummaryBar,
} from '../utils/pii-ui.js';

const piiDataPager = { page: 1, pageSize: 15 };
const piiMetaPager = { page: 1, pageSize: 15 };
const piiLowConfPager = { page: 1, pageSize: 24 };
const piiLowConfFilter = { search: '' };
let piiLowConfToolbarBound = false;

function escapeHtml(s) {
  return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function gval(id, fallback = '') {
  const el = document.getElementById(id);
  return el ? el.value : fallback;
}

function setDataTableVisible(visible) {
  const tableWrap = document.querySelector('#pii-data-scan-block .pii-table-wrap');
  if (tableWrap) tableWrap.style.display = visible ? '' : 'none';
}

function hidePager(id) {
  const el = document.getElementById(id);
  if (el) {
    el.hidden = true;
    el.innerHTML = '';
  }
}

function mountEmptyPager(id, pageSize, onChange) {
  mountTablePagination(document.getElementById(id), {
    page: 1, totalPages: 1, total: 0, start: 0, end: 0, pageSize,
    onPage: onChange || (() => {}),
  });
}

function filterLowConfTables(tables) {
  const q = (piiLowConfFilter.search || '').trim().toLowerCase();
  if (!q) return tables || [];
  return (tables || []).filter((raw) => String(raw || '').toLowerCase().includes(q));
}

function renderLowConfSection(schema) {
  const lowBlock = document.getElementById('pii-low-conf-block');
  const lowContent = document.getElementById('pii-low-conf-content');
  const lowBadge = document.getElementById('pii-low-conf-badge');
  const pagerEl = document.getElementById('pii-low-conf-pagination');
  if (!lowBlock || !lowContent) return;

  const all = piiCatalog.lowConf || [];
  const filtered = filterLowConfTables(all);
  const pg = paginateSlice(filtered, piiLowConfPager.page, piiLowConfPager.pageSize);
  piiLowConfPager.page = pg.page;

  if (!all.length) {
    lowBlock.style.display = 'none';
    lowContent.innerHTML = '';
    hidePager('pii-low-conf-pagination');
    return;
  }

  lowBlock.style.display = 'block';
  if (lowBadge) {
    lowBadge.textContent = all.length + ' table' + (all.length === 1 ? '' : 's');
  }

  const showingText = filtered.length !== all.length
    ? pg.total + ' of ' + all.length + ' tables'
    : (pg.total === all.length ? all.length + ' tables' : pg.start + '–' + pg.end + ' of ' + all.length);

  lowContent.innerHTML =
    '<div class="pii-low-conf-panel">' +
    '<p class="pii-low-conf-intro">' +
    'These tables had <strong>medium or low</strong> confidence PII signals during the value scan. ' +
    'They are not shown in the Data Scan table above because matches did not meet the high-confidence threshold.' +
    '</p>' +
    '<div class="pii-low-conf-toolbar">' +
    '<label class="pii-low-conf-search-wrap">' +
    '<span class="pii-low-conf-search-icon" aria-hidden="true">⌕</span>' +
    '<input type="search" id="pii-low-conf-search" class="pii-low-conf-search" ' +
    'placeholder="Filter tables…" autocomplete="off" aria-label="Filter low-confidence tables" ' +
    'value="' + escapeHtml(piiLowConfFilter.search) + '" />' +
    '</label>' +
    '<span class="pii-low-conf-showing" id="pii-low-conf-showing">' + escapeHtml(showingText) + '</span>' +
    '</div>' +
    '<div class="pii-low-conf-grid" id="pii-low-conf-grid">' +
    (pg.items.length ? renderPiiLowConfChips(pg.items, schema) :
      '<p class="pii-empty-hint" style="margin:0;">No tables match this filter.</p>') +
    '</div></div>';

  bindPiiLowConfToolbarOnce();

  mountTablePagination(pagerEl, {
    page: pg.page,
    totalPages: pg.totalPages,
    total: pg.total,
    start: pg.start,
    end: pg.end,
    pageSize: pg.pageSize,
    pageSizes: [24, 48, 96],
    onPage: (p) => {
      piiLowConfPager.page = p;
      renderLowConfSection(schema);
    },
    onPageSize: (size) => {
      piiLowConfPager.pageSize = size;
      piiLowConfPager.page = 1;
      renderLowConfSection(schema);
    },
  });
}

function bindPiiLowConfToolbarOnce() {
  if (piiLowConfToolbarBound) return;
  const block = document.getElementById('pii-low-conf-block');
  if (!block) return;
  piiLowConfToolbarBound = true;
  block.addEventListener('input', (e) => {
    if (e.target?.id !== 'pii-low-conf-search') return;
    piiLowConfFilter.search = e.target.value;
    piiLowConfPager.page = 1;
    renderLowConfSection(piiCatalog.schema || 'public');
  });
}

let piiCatalog = {
  host: '',
  instance: '',
  database: '',
  rows: [],
  meta: [],
  lowConf: [],
  defaultDatabase: '',
  schema: 'public',
  runOption: '',
  status: '',
  available: false,
  message: '',
};

function piiStatusHint(status) {
  switch (status) {
    case 'no_tables':
      return 'Check <code>[piiscanner].database</code>, <code>schema</code>, and <code>include_tables</code> / <code>exclude_tables</code> in <code>kshieldconfig.toml</code>.';
    case 'no_data':
      return 'Try <code>run_option = "metascan"</code> or <code>"deepscan"</code> in <code>[piiscanner]</code>.';
    case 'error':
      return 'Review collector logs and database connectivity, then re-run the PII scan.';
    default:
      return '';
  }
}

function piiStatusTitle(status, message) {
  if (message) return message;
  switch (status) {
    case 'no_tables':
      return 'No tables found for PII scan';
    case 'no_data':
      return 'No PII data found in database';
    case 'error':
      return 'PII scan failed';
    default:
      return 'No PII scan stored for this host.';
  }
}

function renderPiiResults() {
  const instance = piiCatalog.instance || gval('pii-instance-select', '');
  const database = piiCatalog.database || piiCatalog.defaultDatabase || gval('pii-database-select', '');
  const host = piiCatalog.host || piiHostKey(instance, database);
  const schema = piiCatalog.schema || 'public';
  const dataDisplay = buildPiiDisplayRows(piiCatalog.rows, schema);

  const targetEl = document.getElementById('pii-scan-target');
  if (targetEl) {
    const instPart = instance
      ? '<strong>' + escapeHtml(instance) + '</strong>'
      : '<strong>' + escapeHtml(host || '—') + '</strong>';
    const dbPart = database ? ' · database <code class="pii-db-badge">' + escapeHtml(database) + '</code>' : '';
    const runPart = piiCatalog.runOption
      ? ' · <code>' + escapeHtml(piiCatalog.runOption) + '</code>'
      : '';
    const status = piiCatalog.available
      ? 'Results from <code>scan_results.pii_report_json</code>'
      : escapeHtml(piiCatalog.message || 'No PII scan for this database yet');
    targetEl.innerHTML = instPart + dbPart + runPart + ' — ' + status;
  }

  const dataBody = document.getElementById('pii-data-tbody');
  const dataPagerEl = document.getElementById('pii-data-pagination');
  const noData = document.getElementById('pii-no-data');
  const metaBlock = document.getElementById('pii-meta-scan-block');
  const metaBody = document.getElementById('pii-meta-tbody');
  const metaPagerEl = document.getElementById('pii-meta-pagination');
  const lowBlock = document.getElementById('pii-low-conf-block');
  const resultsPanel = document.getElementById('pii-results-panel');
  const summaryEl = document.getElementById('pii-results-summary');

  if (resultsPanel) resultsPanel.style.display = 'block';

  if (!piiCatalog.available) {
    if (summaryEl) summaryEl.innerHTML = '';
    setDataTableVisible(false);
    if (dataBody) dataBody.innerHTML = '';
    hidePager('pii-data-pagination');
    hidePager('pii-meta-pagination');
    hidePager('pii-low-conf-pagination');
    if (noData) {
      noData.style.display = 'block';
      noData.innerHTML = renderPiiEmptyState(
        piiCatalog.message || 'No PII scan stored for this host.',
        'Configure <code>[piiscanner]</code> in <code>kshieldconfig.toml</code>, add <code>pii_scanner</code> to <code>scan_commands</code>, and optionally set <code>[piiscanner].schedule</code> for a separate cron.',
      );
    }
    if (metaBlock) metaBlock.style.display = 'none';
    if (lowBlock) lowBlock.style.display = 'none';
    return;
  }

  const terminalStatus = piiCatalog.status === 'no_tables' ||
    piiCatalog.status === 'no_data' ||
    piiCatalog.status === 'error';

  if (terminalStatus) {
    if (summaryEl) summaryEl.innerHTML = '';
    setDataTableVisible(false);
    if (dataBody) dataBody.innerHTML = '';
    hidePager('pii-data-pagination');
    hidePager('pii-meta-pagination');
    hidePager('pii-low-conf-pagination');
    if (noData) {
      noData.style.display = 'block';
      noData.innerHTML = renderPiiEmptyState(
        piiStatusTitle(piiCatalog.status, piiCatalog.message),
        piiStatusHint(piiCatalog.status),
      );
    }
    if (metaBlock) metaBlock.style.display = 'none';
    if (lowBlock) lowBlock.style.display = 'none';
    return;
  }

  if (summaryEl) {
    summaryEl.innerHTML = renderPiiSummaryBar({
      tables: countPiiTables(piiCatalog.rows, piiCatalog.meta),
      dataFindings: piiCatalog.rows.length,
      metaFindings: piiCatalog.meta.length,
      lowConf: piiCatalog.lowConf.length,
    });
  }

  if (dataDisplay.length) {
    const dataPg = paginateSlice(dataDisplay, piiDataPager.page, piiDataPager.pageSize);
    piiDataPager.page = dataPg.page;
    setDataTableVisible(true);
    if (dataBody) dataBody.innerHTML = renderPiiDataTableRows(dataPg.items);
    if (noData) noData.style.display = 'none';
    mountTablePagination(dataPagerEl, {
      page: dataPg.page,
      totalPages: dataPg.totalPages,
      total: dataPg.total,
      start: dataPg.start,
      end: dataPg.end,
      pageSize: dataPg.pageSize,
      pageSizes: [15, 25, 50],
      onPage: (p) => {
        piiDataPager.page = p;
        renderPiiResults();
      },
      onPageSize: (size) => {
        piiDataPager.pageSize = size;
        piiDataPager.page = 1;
        renderPiiResults();
      },
    });
  } else {
    setDataTableVisible(false);
    if (dataBody) dataBody.innerHTML = '';
    mountEmptyPager('pii-data-pagination', piiDataPager.pageSize);
    if (noData) {
      noData.style.display = 'block';
      noData.innerHTML = renderPiiEmptyState(
        'No high-confidence PII in data scan',
        'Try <code>run_option = "metascan"</code> or <code>"deepscan"</code> in <code>[piiscanner]</code>.',
      );
    }
  }

  if (metaBlock && metaBody) {
    if (piiCatalog.meta.length) {
      const metaPg = paginateSlice(piiCatalog.meta, piiMetaPager.page, piiMetaPager.pageSize);
      piiMetaPager.page = metaPg.page;
      metaBlock.style.display = 'block';
      metaBody.innerHTML = renderPiiMetaTableRows(metaPg.items, schema);
      mountTablePagination(metaPagerEl, {
        page: metaPg.page,
        totalPages: metaPg.totalPages,
        total: metaPg.total,
        start: metaPg.start,
        end: metaPg.end,
        pageSize: metaPg.pageSize,
        pageSizes: [15, 25, 50],
        onPage: (p) => {
          piiMetaPager.page = p;
          renderPiiResults();
        },
        onPageSize: (size) => {
          piiMetaPager.pageSize = size;
          piiMetaPager.page = 1;
          renderPiiResults();
        },
      });
    } else {
      metaBlock.style.display = 'none';
      hidePager('pii-meta-pagination');
    }
  }

  if (lowBlock) {
    if (piiCatalog.lowConf.length) {
      renderLowConfSection(schema);
    } else if (!dataDisplay.length && !piiCatalog.meta.length) {
      lowBlock.style.display = 'block';
      const lowBadge = document.getElementById('pii-low-conf-badge');
      const lowContent = document.getElementById('pii-low-conf-content');
      if (lowBadge) lowBadge.textContent = 'None';
      if (lowContent) {
        lowContent.innerHTML = renderPiiEmptyState(
          'No PII data found in database',
          'Try a different <code>run_option</code> in <code>[piiscanner]</code>.',
        );
      }
      hidePager('pii-low-conf-pagination');
    } else {
      lowBlock.style.display = 'none';
      hidePager('pii-low-conf-pagination');
    }
  }
}

function ingestPiiData(data) {
  piiDataPager.page = 1;
  piiMetaPager.page = 1;
  piiLowConfPager.page = 1;
  piiLowConfFilter.search = '';
  const instance = data?.instance || '';
  const database = data?.database || data?.defaultDatabase || '';
  piiCatalog = {
    host: data?.host || piiHostKey(instance, database),
    instance,
    database,
    rows: data?.rows || [],
    meta: data?.meta || [],
    lowConf: data?.lowConfTables || [],
    defaultDatabase: database,
    schema: data?.schema || 'public',
    runOption: data?.runOption || '',
    status: data?.status || '',
    available: !!data?.available,
    message: data?.message || data?.scanMessage || '',
  };
  renderPiiResults();
}

function showPiiLoading(instance, database) {
  const targetEl = document.getElementById('pii-scan-target');
  if (targetEl) {
    const label = database ? instance + ' / ' + database : instance;
    targetEl.innerHTML = '<strong>' + escapeHtml(label) + '</strong> — Loading PII report…';
  }
  const resultsPanel = document.getElementById('pii-results-panel');
  const dataBody = document.getElementById('pii-data-tbody');
  const noData = document.getElementById('pii-no-data');
  const metaBlock = document.getElementById('pii-meta-scan-block');
  if (resultsPanel) resultsPanel.style.display = 'block';
  setDataTableVisible(false);
  if (dataBody) dataBody.innerHTML = '';
  hidePager('pii-data-pagination');
  hidePager('pii-meta-pagination');
  hidePager('pii-low-conf-pagination');
  if (noData) {
    noData.style.display = 'block';
    noData.innerHTML = renderPiiEmptyState('Loading PII report…', '');
  }
  if (metaBlock) metaBlock.style.display = 'none';
}

function piiHostKey(instance, database) {
  if (!instance) return '';
  if (!database) return instance;
  return instance + '/' + database;
}

function parsePiiHostKey(hostKey) {
  const raw = String(hostKey || '').trim();
  const slash = raw.indexOf('/');
  if (slash > 0 && raw.includes(':')) {
    return { instance: raw.slice(0, slash), database: raw.slice(slash + 1).split('/')[0] };
  }
  return { instance: raw, database: '' };
}

function databasesFromLabel(label, instance) {
  const m = String(label || '').match(/^\d+\s*\(([^)]+)\)/);
  if (!m) return [];
  return m[1].split(',').map((name) => {
    const n = name.trim();
    return { name: n, hostKey: piiHostKey(instance, n) };
  }).filter((db) => db.name);
}

function normalizePiiInstances(hostsData) {
  const rows = mapHostsResponse(hostsData) || [];
  return rows.map((h) => {
    if (Array.isArray(h)) {
      const parsed = parsePiiHostKey(h[0]);
      const instance = parsed.instance || h[0] || '';
      let databases = parsed.database ? [{ name: parsed.database, hostKey: piiHostKey(instance, parsed.database) }] : [];
      if (!databases.length) {
        databases = databasesFromLabel(h[2], instance);
      }
      return { instance, ip: h[1] || '', databases };
    }
    const instance = h.instance || h[0] || '';
    let databases = Array.isArray(h.databases) ? h.databases.map((db) => ({
      name: db.name || db,
      hostKey: db.host_key || piiHostKey(instance, db.name || db),
    })) : [];
    if (!databases.length) {
      databases = databasesFromLabel(h.databases_label, instance);
    }
    return { instance, ip: h.ip || h[1] || '', databases };
  }).filter((h) => h.instance);
}

async function enrichInstanceDatabases(instance) {
  const inst = piiInstances.find((x) => x.instance === instance);
  if (!inst || inst.databases?.length) return inst;
  try {
    const overview = await getHostInstance(instance);
    const dbs = (overview?.databases || []).map((db) => ({
      name: db.name || db,
      hostKey: db.host_key || piiHostKey(instance, db.name || db),
    }));
    if (dbs.length) {
      inst.databases = dbs;
    }
  } catch {
    /* keep empty */
  }
  return inst;
}

function populatePiiInstanceSelect(select, instances, selectedInstance) {
  if (!select) return '';
  if (!instances.length) {
    select.innerHTML = '<option value="">No hosts in DB</option>';
    return '';
  }
  const want = selectedInstance || select.value || instances[0].instance;
  select.innerHTML = instances.map((inst) => {
    const sel = inst.instance === want ? ' selected' : '';
    const ip = inst.ip && inst.ip !== '-' ? ' (' + inst.ip + ')' : '';
    return '<option value="' + escapeHtml(inst.instance) + '"' + sel + '>' +
      escapeHtml(inst.instance) + escapeHtml(ip) + '</option>';
  }).join('');
  if (want && [...select.options].some((o) => o.value === want)) {
    select.value = want;
  }
  return select.value || want;
}

function populatePiiDatabaseSelect(select, databases, selectedDatabase) {
  if (!select) return '';
  if (!databases?.length) {
    select.innerHTML = '<option value="">No databases</option>';
    select.disabled = true;
    return '';
  }
  select.disabled = false;
  const want = selectedDatabase || select.value || databases[0].name;
  select.innerHTML = databases.map((db) => {
    const name = db.name || db;
    const sel = name === want ? ' selected' : '';
    return '<option value="' + escapeHtml(name) + '"' + sel + '>' + escapeHtml(name) + '</option>';
  }).join('');
  if (want && [...select.options].some((o) => o.value === want)) {
    select.value = want;
  }
  return select.value || want;
}

let piiBindingsDone = false;
let piiInstances = [];
let piiLoadSeq = 0;
let piiLastLoadedHost = '';
let piiLastLoadAt = 0;
let piiInflightHost = '';
let piiAbort = null;
let piiInitPromise = null;

function currentPiiTarget() {
  const instance = gval('pii-instance-select', '');
  const database = gval('pii-database-select', '');
  return { instance, database, hostKey: piiHostKey(instance, database) };
}

function bindPiiPageOnce() {
  if (piiBindingsDone) return;
  piiBindingsDone = true;

  const instSelect = document.getElementById('pii-instance-select');
  const dbSelect = document.getElementById('pii-database-select');

  if (instSelect) {
    instSelect.addEventListener('change', async () => {
      const inst = await enrichInstanceDatabases(instSelect.value);
      populatePiiDatabaseSelect(dbSelect, inst?.databases || [], '');
      const { instance, database } = currentPiiTarget();
      if (instance && database) void loadPiiForTarget(instance, database, true);
    });
  }
  if (dbSelect) {
    dbSelect.addEventListener('change', () => {
      const { instance, database } = currentPiiTarget();
      if (instance && database) void loadPiiForTarget(instance, database, true);
    });
  }
}

async function loadPiiForTarget(instance, database, force = false) {
  const hostKey = piiHostKey(instance, database);
  if (!hostKey) return;

  const now = Date.now();
  if (!force) {
    if (hostKey === piiInflightHost) return;
    if (hostKey === piiLastLoadedHost && now - piiLastLoadAt < 800) return;
  }

  piiInflightHost = hostKey;
  const seq = ++piiLoadSeq;
  if (piiAbort) piiAbort.abort();
  piiAbort = new AbortController();
  showPiiLoading(instance, database);

  try {
    const data = await getPiiScanner(hostKey, { signal: piiAbort.signal });
    if (seq !== piiLoadSeq) return;
    piiLastLoadedHost = hostKey;
    piiLastLoadAt = Date.now();
    ingestPiiData(data);
    return data;
  } catch (err) {
    if (err?.name === 'AbortError') return;
    if (seq !== piiLoadSeq) return;
    piiCatalog = {
      ...piiCatalog,
      host: hostKey,
      instance,
      database,
      available: false,
      message: err.message,
    };
    renderPiiResults();
    throw err;
  } finally {
    if (piiInflightHost === hostKey) piiInflightHost = '';
  }
}

async function initPiiScannerPageInner() {
  bindPiiPageOnce();

  const instSelect = document.getElementById('pii-instance-select');
  const dbSelect = document.getElementById('pii-database-select');
  const keepInstance = instSelect?.value || piiCatalog.instance || '';
  const keepDatabase = dbSelect?.value || piiCatalog.database || '';

  try {
    const hostsData = await hostsApi.getHosts();
    piiInstances = normalizePiiInstances(hostsData);
    const instance = populatePiiInstanceSelect(instSelect, piiInstances, keepInstance);
    const inst = await enrichInstanceDatabases(instance);
    const database = populatePiiDatabaseSelect(dbSelect, inst?.databases || [], keepDatabase);
    if (instance && database) {
      if (piiHostKey(instance, database) === piiLastLoadedHost && !piiInflightHost && piiCatalog.available) {
        renderPiiResults();
        return;
      }
      await loadPiiForTarget(instance, database);
    }
    return;
  } catch {
    /* keep existing options */
  }

  const { instance, database, hostKey } = currentPiiTarget();
  if (!hostKey) return;
  if (hostKey === piiLastLoadedHost && !piiInflightHost && piiCatalog.available) {
    renderPiiResults();
    return;
  }
  await loadPiiForTarget(instance, database);
}

export function initPiiScannerPage() {
  if (piiInitPromise) return piiInitPromise;
  piiInitPromise = initPiiScannerPageInner().finally(() => {
    piiInitPromise = null;
  });
  return piiInitPromise;
}
