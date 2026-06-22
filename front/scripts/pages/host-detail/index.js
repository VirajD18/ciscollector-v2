import { getHostReport, getHostInstance } from '../../api/services/host-report.js';
import {
  renderTable,
  renderCisAuditModule,
  renderPiiModule,
  renderLogParserModule,
  renderFullReportPanel,
  renderCriticalViolationsModule,
  initHostAuditTable,
  initCriticalViolationsTable,
  resetHostDetailTablePagers,
  resolveCisAuditRows,
  bindHostCisInteractions,
} from './render-utils.js';

function reportPageUrl(hostKey) {
  return '/report.html?host=' + encodeURIComponent(hostKey);
}

const MODULE_SLOTS = {
  cis_audit: 'host-mod-cis',
  pii_results: 'host-mod-pii',
  log_parser: 'host-mod-logparser',
};

let currentInstance = '';
let currentDatabase = '';
let dbSelectBound = false;

function escapeHtml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function dbPostureBadgeClass(posture) {
  if (posture === 'Failing') return 'badge-posture-fail';
  if (posture === 'Passing') return 'badge-success';
  return 'badge-warning';
}

function setText(id, text) {
  const el = document.getElementById(id);
  if (el) el.textContent = text;
}

function setHtml(id, html) {
  const el = document.getElementById(id);
  if (el) el.innerHTML = html;
}

function splitHostRoute(hostId, opts = {}) {
  let instance = String(opts.instance || '').trim();
  let database = String(opts.database || '').trim();
  const raw = String(hostId || '').trim();
  if (!instance && raw.includes('/') && raw.includes(':')) {
    const slash = raw.indexOf('/');
    instance = raw.slice(0, slash);
    database = database || raw.slice(slash + 1).split('/')[0];
  } else if (!instance) {
    instance = raw;
  }
  return { instance, database };
}

function hostKeyFor(instance, database) {
  if (!database) return instance;
  return instance + '/' + database;
}

function renderKpis(host, mods, criticalFailed) {
  const root = document.getElementById('host-detail-kpis');
  if (!root) return;
  const drift = host.guc_drift !== '-' ? host.guc_drift + ' GUCs' : '-';
  const hostKey = host.name || host.id || '';
  const exportLink = hostKey
    ? '<a class="link" href="' + reportPageUrl(hostKey) + '" target="_blank" rel="noopener noreferrer">Open HTML Report</a>'
    : '<span style="color:var(--muted);">No export</span>';
  const failed = criticalFailed ?? 0;
  root.innerHTML =
    '<div class="stat-card"><div class="stat-label">Critical Failures</div>' +
    '<div class="stat-value ' + (failed > 0 ? 'danger' : '') + '">' + failed + '</div></div>' +
    '<div class="stat-card"><div class="stat-label">CIS Score</div>' +
    '<div class="stat-value ' + (parseInt(host.cis_pct, 10) < 70 ? 'warning' : '') + '">' + host.cis_pct + '</div></div>' +
    '<div class="stat-card"><div class="stat-label">Failed Controls</div><div class="stat-value">' + host.failed_controls + '</div></div>' +
    '<div class="stat-card"><div class="stat-label">GUC Drift</div><div class="stat-value danger">' + drift + '</div></div>' +
    '<div class="stat-card"><div class="stat-label">Audit Report</div><div class="stat-value" style="font-size:16px">' + exportLink + '</div></div>';
  root.style.gridTemplateColumns = 'repeat(5,1fr)';
}

export function activateHostSubTab(panelId) {
  if (!panelId || !String(panelId).startsWith('sub-')) return;
  const panel = document.getElementById(panelId);
  if (!panel) return;
  const block = panel.closest('.report-block');
  if (!block) return;
  const subName = String(panelId).slice(4);
  block.querySelectorAll('.sub-tab').forEach((t) => {
    t.classList.toggle('active', t.dataset.sub === subName);
  });
  block.querySelectorAll('.host-subpanel').forEach((p) => p.classList.remove('active'));
  panel.classList.add('active');
}

function renderModules(modules, cisResponses) {
  Object.entries(MODULE_SLOTS).forEach(([key, slotId]) => {
    const mod = modules[key];
    const el = document.getElementById(slotId);
    if (!el) return;
    if (key === 'cis_audit') {
      const cisRows = resolveCisAuditRows(cisResponses, mod);
      el.innerHTML = renderCisAuditModule(cisResponses, mod);
      initHostAuditTable(el, cisRows, { idPrefix: 'host-cis' });
      bindHostCisInteractions(el);
    } else if (key === 'pii_results') {
      el.innerHTML = renderPiiModule(mod);
    } else if (key === 'log_parser') {
      el.innerHTML = renderLogParserModule(mod);
    } else {
      el.innerHTML = renderTable(mod, 'No data for this module.');
    }
  });
}

function showState(state, message) {
  const banner = document.getElementById('host-detail-banner');
  if (banner) {
    banner.style.display = state === 'ok' ? 'none' : 'block';
    banner.className = 'callout ' + (state === 'error' ? 'banner-error' : '');
    banner.textContent = message || '';
  }
}

function populateDatabaseSelect(databases, selected) {
  const row = document.getElementById('host-db-select-row');
  const picker = document.getElementById('host-database-picker');
  if (!row || !picker) return;
  if (!databases?.length || databases.length < 2) {
    row.hidden = true;
    picker.innerHTML = '';
    return;
  }
  row.hidden = false;
  picker.innerHTML = databases.map((db) => {
    const name = db.name || db;
    const cis = db.cis_pct ? escapeHtml(db.cis_pct) + ' CIS' : 'No CIS score';
    const posture = db.posture || '—';
    const active = name === selected;
    return '<button type="button" class="host-db-pill' + (active ? ' active' : '') + '" role="tab"' +
      ' aria-selected="' + (active ? 'true' : 'false') + '" data-db="' + escapeHtml(name) + '">' +
      '<span class="host-db-pill-name">' + escapeHtml(name) + '</span>' +
      '<span class="host-db-pill-stats">' +
      '<span class="host-db-pill-cis">' + cis + '</span>' +
      '<span class="badge ' + dbPostureBadgeClass(posture) + ' host-db-pill-posture">' + escapeHtml(posture) + '</span>' +
      '</span></button>';
  }).join('');
}

function bindDatabaseSelect(onChange) {
  if (dbSelectBound) return;
  const picker = document.getElementById('host-database-picker');
  if (!picker) return;
  dbSelectBound = true;
  picker.addEventListener('click', (e) => {
    const pill = e.target.closest('.host-db-pill');
    if (!pill || pill.classList.contains('active')) return;
    const db = pill.dataset.db;
    if (!db) return;
    picker.querySelectorAll('.host-db-pill').forEach((el) => {
      const isActive = el === pill;
      el.classList.toggle('active', isActive);
      el.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });
    onChange(db);
  });
}

async function loadDatabaseReport(instance, database, opts = {}) {
  const reportKey = hostKeyFor(instance, database);
  currentInstance = instance;
  currentDatabase = database;
  showState('loading', 'Loading report for database ' + database + '…');
  Object.values(MODULE_SLOTS).forEach((id) => {
    setHtml(id, '<p style="color:var(--muted);">Loading…</p>');
  });
  setHtml('host-mod-critical-violations', '<p style="color:var(--muted);">Loading…</p>');

  const data = await getHostReport(reportKey);
  if (!data?.host) {
    showState('error', 'No scan data found for ' + reportKey + '. Run ciscollector and persist to SQLite.');
    return;
  }
  const h = data.host;
  if (data.html_export) h._htmlExport = data.html_export;
  showState('ok');
  setText('host-detail-title', instance);
  setText('host-detail-subtitle',
    'Database: ' + database + ' · ' + (h.ip || '-') + ' · PostgreSQL ' + (h.postgres_version || '-') +
    ' · Agent ' + (h.agent || 'Online') + ' · Last audit ' + (h.last_audit || '-'));
  renderKpis(h, data.modules, data.critical_failed);
  const critEl = document.getElementById('host-mod-critical-violations');
  if (critEl) {
    critEl.innerHTML = renderCriticalViolationsModule(data.critical_checks, data.critical_failed);
    initCriticalViolationsTable(critEl, data.critical_checks);
  }
  renderModules(data.modules || {}, data.postgres_cis_responses);

  if (opts.section) {
    requestAnimationFrame(() => {
      activateHostSubTab(opts.section);
      const el = document.getElementById(opts.section);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  }
  const full = document.getElementById('host-mod-full-report');
  if (full) {
    const reportHost = h.id || h.name || reportKey;
    full.innerHTML = renderFullReportPanel({
      hostLabel: reportHost,
      openUrl: reportPageUrl(reportHost),
      downloadUrl: data.html_export?.download_url || '',
    });
  }
}

export async function loadHostDetail(hostId, opts = {}) {
  if (!hostId) return;
  resetHostDetailTablePagers();
  const { instance, database: initialDb } = splitHostRoute(hostId, opts);
  if (!instance) return;

  setText('host-detail-title', instance);
  setText('host-detail-subtitle', 'Loading databases…');
  showState('loading', 'Loading host overview…');

  try {
    const overview = await getHostInstance(instance);
    if (!overview?.databases?.length) {
      showState('error', 'No scan data found for instance ' + instance + '.');
      return;
    }
    const selectedDb = initialDb || overview.default_database || overview.databases[0].name;
    populateDatabaseSelect(overview.databases, selectedDb);
    bindDatabaseSelect(async (db) => {
      try {
        await loadDatabaseReport(instance, db, {});
        if (typeof window !== 'undefined' && window.history) {
          const hash = '#host/' + encodeURIComponent(instance) + '/db/' + encodeURIComponent(db);
          if (location.hash !== hash) history.replaceState(null, '', hash);
        }
      } catch (err) {
        console.error(err);
        showState('error', 'Failed to load database report: ' + (err.message || 'API error'));
      }
    });
    await loadDatabaseReport(instance, selectedDb, opts);
  } catch (err) {
    console.error(err);
    showState('error', 'Failed to load host report: ' + (err.message || 'API error'));
  }
}

export function getCurrentHostDetailTarget() {
  return hostKeyFor(currentInstance, currentDatabase);
}
