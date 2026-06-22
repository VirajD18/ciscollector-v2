import { getSslScanner } from '../api/services/scanner.js';
import {
  formatScannerTargetLabel,
  initScannerTargetSelects,
  readScannerTarget,
} from '../utils/scanner-host-select.js';
import {
  renderSslAuditCheckList,
  renderSslParamsPanel,
} from '../utils/ssl-ui.js';

function updateStats(stats) {
  const failed = (stats.fail ?? 0) + (stats.critical ?? 0);
  const map = {
    'ssl-stat-pass': stats.pass,
    'ssl-stat-warning': stats.warning,
    'ssl-stat-failed': failed,
    'ssl-stat-total': stats.total,
  };
  Object.entries(map).forEach(([id, val]) => {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
  });
}

function renderParams(params) {
  const panel = document.getElementById('ssl-params-panel');
  if (!panel) return;
  panel.innerHTML = params && Object.keys(params).length
    ? renderSslParamsPanel(params, { layout: 'report-block' })
    : '';
}

function updateMeta(label, hostKey) {
  const hostEl = document.getElementById('ssl-callout-host');
  if (hostEl) hostEl.textContent = label || '—';
  const auditLink = document.getElementById('ssl-host-audit-link');
  if (auditLink && hostKey) auditLink.dataset.host = hostKey;
  const metaEl = document.getElementById('ssl-scan-meta');
  if (metaEl) {
    metaEl.textContent = label
      ? 'Live data from latest scan for ' + label +
        ' (SSL settings are shared across all databases on this instance).'
      : '';
  }
}

function renderSslChecks(data, target) {
  const list = document.getElementById('ssl-check-list');
  const checks = data?.cells || [];
  const label = formatScannerTargetLabel(target?.instance, target?.database) || data?.host || '';

  if (!checks.length) {
    if (list) {
      list.innerHTML = renderSslAuditCheckList([], [], {
        emptyMsg: data?.message,
        wrap: false,
      });
    }
    updateStats({ pass: 0, fail: 0, warning: 0, critical: 0, total: 0 });
    renderParams(null);
    updateMeta(label, target?.hostKey);
    return;
  }

  if (list) {
    list.innerHTML = renderSslAuditCheckList(
      checks.map((c) => ({
        title: c.title,
        status: c.status,
        message: c.desc || c.message || '',
      })),
      data.hbaLines || [],
      { wrap: false },
    );
  }

  updateStats({
    pass: data.pass ?? 0,
    fail: data.fail ?? 0,
    warning: data.warning ?? 0,
    critical: data.critical ?? 0,
    total: checks.length,
  });
  renderParams(data.sslParams || {});
  updateMeta(label, target?.hostKey);
}

async function loadSslForTarget(target) {
  const list = document.getElementById('ssl-check-list');
  if (!target?.hostKey) {
    if (list) {
      list.innerHTML = '<p style="color:var(--muted);padding:16px;">Select instance and database.</p>';
    }
    return;
  }
  if (list) list.innerHTML = '<p style="color:var(--muted);padding:16px;">Loading SSL scan…</p>';
  try {
    renderSslChecks(await getSslScanner(target.hostKey), target);
  } catch (err) {
    if (list) list.innerHTML = '<p style="color:var(--danger);padding:16px;">' + err.message + '</p>';
  }
}

export async function initSslScannerPage() {
  const prev = readScannerTarget('ssl-instance-select', 'ssl-database-select');
  const target = await initScannerTargetSelects({
    instanceSelectId: 'ssl-instance-select',
    databaseSelectId: 'ssl-database-select',
    keepInstance: prev.instance,
    keepDatabase: prev.database,
    onTargetChange: (t) => { void loadSslForTarget(t); },
  });
  await loadSslForTarget(target);
}
