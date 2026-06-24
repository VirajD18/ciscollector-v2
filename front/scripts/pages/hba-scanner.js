import { getHbaScanner } from '../api/services/scanner.js';
import {
  formatScannerTargetLabel,
  initScannerTargetSelects,
  readScannerTarget,
} from '../utils/scanner-host-select.js';

function renderHbaChecks(data, target) {
  const list = document.getElementById('hba-check-list');
  if (!list) return;
  const checks = data?.checks || [];
  const label = formatScannerTargetLabel(target?.instance, target?.database) || data?.host || '';

  if (!checks.length) {
    list.innerHTML = '<p style="color:var(--muted);padding:16px;">' +
      (data?.message || 'No HBA scan data. Run collector with hba_scanner or -r --hba-scanner --json') + '</p>';
    updateHbaMeta(label, target?.hostKey);
    return;
  }

  list.innerHTML = checks.map((c) => {
    const isPass = c.status === 'pass';
    const rowClass = isPass ? 'hba-check-row--pass' : 'hba-check-row--fail';
    return '<div class="hba-check-row ' + rowClass + '">' +
      '<div class="hba-check-bar ' + (isPass ? 'pass' : 'fail') + '"></div>' +
      '<div class="hba-check-main"><div class="hba-check-num">HBA Check ' + c.n + '</div>' +
      '<div class="hba-check-title">' + c.title + '</div>' +
      '<div class="hba-check-desc">' + (c.desc || '') + '</div></div>' +
      '<div class="hba-check-status"><span class="hba-badge ' + (isPass ? 'pass' : 'fail') + '">' +
      (isPass ? 'Pass' : 'Fail') + '</span></div></div>';
  }).join('');

  const passEl = document.getElementById('hba-stat-pass');
  const failEl = document.getElementById('hba-stat-fail');
  const totalEl = document.getElementById('hba-stat-total');
  if (passEl) passEl.textContent = data.pass ?? 0;
  if (failEl) failEl.textContent = data.fail ?? 0;
  if (totalEl) totalEl.textContent = checks.length;
  updateHbaMeta(label, target?.hostKey);
}

function updateHbaMeta(label, hostKey) {
  const hostEl = document.getElementById('hba-callout-host');
  if (hostEl) hostEl.textContent = label || '—';
  const auditLink = document.getElementById('hba-host-audit-link');
  if (auditLink && hostKey) auditLink.dataset.host = hostKey;
  const metaEl = document.getElementById('hba-scan-meta');
  if (metaEl) {
    metaEl.textContent = label
      ? 'Live data from latest scan for ' + label +
        ' (pg_hba.conf is shared across all databases on this instance).'
      : '';
  }
}

async function loadHbaForTarget(target) {
  const list = document.getElementById('hba-check-list');
  if (!target?.hostKey) {
    if (list) {
      list.innerHTML = '<p style="color:var(--muted);padding:16px;">Select instance and database.</p>';
    }
    return;
  }
  if (list) {
    list.innerHTML = '<p style="color:var(--muted);padding:16px;">Loading HBA scan…</p>';
  }
  try {
    const data = await getHbaScanner(target.hostKey);
    renderHbaChecks(data, target);
  } catch (err) {
    if (list) list.innerHTML = '<p style="color:var(--danger);">' + err.message + '</p>';
  }
}

export async function initHbaScannerPage() {
  const prev = readScannerTarget('hba-instance-select', 'hba-database-select');
  const target = await initScannerTargetSelects({
    instanceSelectId: 'hba-instance-select',
    databaseSelectId: 'hba-database-select',
    keepInstance: prev.instance,
    keepDatabase: prev.database,
    onTargetChange: (t) => { void loadHbaForTarget(t); },
  });
  await loadHbaForTarget(target);
}
