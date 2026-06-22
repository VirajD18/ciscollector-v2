import { getHostReport } from '../api/services/host-report.js';

import { getHbaScanner, getSslScanner } from '../api/services/scanner.js';



const SECTIONS = [

  { id: 0, name: 'Overall Score', color: '#373854' },

  { id: 1, name: 'Section 1 - Installation and Patches', color: '#EA4335' },

  { id: 2, name: 'Section 2 - Directory and File Permissions', color: '#FBBC05' },

  { id: 3, name: 'Section 3 - Logging Monitoring and Auditing', color: '#34A853' },

  { id: 4, name: 'Section 4 - User Access and Authorization', color: '#673AB7' },

  { id: 5, name: 'Section 5 - Connection and Login', color: '#4285F4' },

  { id: 6, name: 'Section 6 - Postgres Settings', color: '#9E379F' },

  { id: 7, name: 'Section 7 - Replication', color: '#7BB3FF' },

  { id: 8, name: 'Section 8 - Special Configuration Considerations', color: '#FF6F69' },

];



const ICON_TICK = '<td class="icon_column"><svg style="color:#20d908" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z" fill="#20d908"/><path d="M10.97 4.97a.235.235 0 0 0-.02.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-1.071-1.05z" fill="#20d908"/></svg></td>';

const ICON_FAIL = '<td class="icon_column"><svg style="color:red" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z" fill="red"/><path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z" fill="red"/></svg></td>';

const ICON_MANUAL = '<td class="icon_column"><svg style="color:#FFD700" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" aria-hidden="true"><path d="M16 8s-3-5.5-8-5.5S0 8 0 8s3 5.5 8 5.5S16 8 16 8zM8 4.5c1.932 0 3.5 1.568 3.5 3.5S9.932 11.5 8 11.5 4.5 9.932 4.5 8 6.068 4.5 8 4.5z" fill="#FFD700"/><path d="M8 5.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5z" fill="#FFD700"/></svg></td>';

const ICON_INFO = '<td class="icon_column"><svg class="infoIcon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="16" height="16" aria-label="Show details"><path d="M464 256A208 208 0 1 0 48 256a208 208 0 1 0 416 0zM0 256a256 256 0 1 1 512 0A256 256 0 1 1 0 256zm169.8-90.7c7.9-22.3 29.1-37.3 52.8-37.3h58.3c34.9 0 63.1 28.3 63.1 63.1c0 22.6-12.1 43.5-31.7 54.8L280 264.4c-.2 13-10.9 23.6-24 23.6c-13.3 0-24-10.7-24-24V250.5c0-8.6 4.6-16.5 12.1-20.8l44.3-25.4c4.7-2.7 7.6-7.7 7.6-13.1c0-8.4-6.8-15.1-15.1-15.1H222.6c-3.4 0-6.4 2.1-7.5 5.3l-.4 1.2c-4.4 12.5-18.2 19-30.6 14.6s-19-18.2-14.6-30.6l.4-1.2zM224 352a32 32 0 1 1 64 0 32 32 0 1 1 -64 0z"/></svg></td>';



function escapeHtml(v) {

  return String(v ?? '')

    .replace(/&/g, '&amp;')

    .replace(/</g, '&lt;')

    .replace(/>/g, '&gt;')

    .replace(/"/g, '&quot;');

}



function nl2br(text) {

  return escapeHtml(text).replace(/\n/g, '<br>');

}



function rowAnchorId(control, title) {

  return String(control ?? '') + String(title ?? '');

}



function normalizeCis(raw) {

  if (!raw || typeof raw !== 'object') return null;

  return {

    Control: raw.Control ?? raw.control ?? '',

    Title: raw.Title ?? raw.title ?? '',

    Status: raw.Status ?? raw.status ?? '',

    Description: raw.Description ?? raw.description ?? '',

    FailReason: raw.FailReason ?? raw.fail_reason ?? '',

    Rationale: raw.Rationale ?? raw.rationale ?? '',

    Procedure: raw.Procedure ?? raw.procedure ?? '',

    References: raw.References ?? raw.references ?? '',

    Critical: !!(raw.Critical ?? raw.critical),

    ManualCheckData: raw.ManualCheckData ?? raw.manual_check_data ?? null,

  };

}



function cisFromHostData(data) {

  const fromTop = (data.postgres_cis_responses || []).map(normalizeCis).filter(Boolean);

  if (fromTop.length) return fromTop;

  const rows = data.modules?.cis_audit?.rows || [];

  return rows.map(row => normalizeCis({

    Control: row.cells?.[0],

    Title: row.cells?.[1],

    Status: row.status || row.cells?.[2],

  })).filter(r => r.Control || r.Title);

}



function sectionScores(cis) {

  const scores = SECTIONS.map(() => ({ pass: 0, fail: 0 }));

  cis.forEach(r => {

    const prefix = parseInt(String(r.Control).split('.')[0], 10);

    if (Number.isNaN(prefix) || prefix < 0 || prefix > 8) return;

    const bucket = scores[prefix] || scores[0];

    if (String(r.Status).toLowerCase() === 'pass') {

      bucket.pass++;

      scores[0].pass++;

    } else if (String(r.Status).toLowerCase() === 'fail') {

      bucket.fail++;

      scores[0].fail++;

    }

  });

  return scores;

}



function isAutomatedCheckStatus(status) {
  const s = String(status || '').toLowerCase();
  return s === 'pass' || s === 'fail';
}

function automatedChecks(rows, statusKey = 'Status') {
  return (rows || []).filter((r) => isAutomatedCheckStatus(r[statusKey] ?? r.status ?? r.Status));
}

function statusCell(status) {

  const s = String(status).toLowerCase();

  if (s === 'pass') return ICON_TICK;

  if (s === 'fail') return ICON_FAIL;

  // Match klouddbshield_report.html: non-pass/fail (Manual, Unknown, etc.) → manual icon
  return ICON_MANUAL;

}



function renderLegend() {
  return '<div class="legend-row" style="display:flex;gap:16px;font-size:12px;align-items:center;">' +
    '<span class="status-pass">✓ Pass</span>' +
    '<span class="status-fail">✗ Fail</span>' +
    '<span class="status-manual">◎ Manual</span></div>';
}



function renderProgressBars(scores) {

  let html = '<div id="summaryStats"><h3>Summary</h3>';

  SECTIONS.slice(1).forEach(sec => {

    const s = scores[sec.id];

    const total = s.pass + s.fail;

    if (!total) return;

    const pct = (s.pass / total) * 100;

    html += '<div class="progress-bar">' +

      '<div class="progress-label"><span>' + escapeHtml(sec.name) + '</span>' +

      '<span>' + s.pass + '/' + total + ' - (' + pct.toFixed(2) + '%)</span></div>' +

      '<div class="progress"><div class="progress-filled" style="width:' + pct + '%;background:' + sec.color + '"></div></div></div>';

  });

  const overall = scores[0];

  const oTotal = overall.pass + overall.fail;

  if (oTotal) {

    const oPct = (overall.pass / oTotal) * 100;

    html += '<div class="overall-progress-bar">' +

      '<div class="progress-label"><span>' + escapeHtml(SECTIONS[0].name) + '</span>' +

      '<span>' + overall.pass + '/' + oTotal + ' - (' + oPct.toFixed(2) + '%)</span></div>' +

      '<div class="progress"><div class="progress-filled" style="width:' + oPct + '%;background:' + SECTIONS[0].color + '"></div></div></div>';

  }

  html += '</div>';

  return html;

}



function renderCisTable(cis, idPrefix) {
  cis = automatedChecks(cis, 'Status');

  if (!cis.length) {

    return '<p style="color:#666;">No automated CIS controls in this scan.</p>';

  }

  const tableId = idPrefix + '-cis-table';

  let html = '<div class="table-container" data-cis-table="' + tableId + '">' +

    '<div class="table-toolbar">' +

    '<button type="button" class="toggleAll" data-target="' + tableId + '">Expand All</button>' +

    renderLegend() + '</div>' +

    '<table class="table maintable" id="' + tableId + '"><thead><tr>' +

    '<th>Control</th><th class="icon_column">Result</th><th class="icon_column">Details</th></tr></thead><tbody>';



  cis.forEach((r, i) => {

    const anchor = rowAnchorId(r.Control, r.Title);

    const rowId = idPrefix + '-cis-' + i;

    const crit = r.Critical ? ' critical_row' : '';

    html += '<tr id="' + escapeHtml(anchor) + '" class="toggleRow' + crit + '" data-detail="' + rowId + '">' +

      '<td>' + escapeHtml(r.Control + ' ' + r.Title) + '</td>' +

      statusCell(r.Status) + ICON_INFO + '</tr>' +

      '<tr class="childTableRow" id="' + rowId + '"><td colspan="3">' +

      '<div class="scrollable-container"><table class="table" id="innerTable">' +

      '<tr><th>Description</th><td>' + nl2br(r.Description) + '</td></tr>';

    if (r.FailReason) {

      html += '<tr><th>Fail Reason</th><td>' + nl2br(r.FailReason) + '</td></tr>';

    }

    html += '<tr><th>Rationale</th><td>' + nl2br(r.Rationale) + '</td></tr>' +

      '<tr><th>Process to Validate</th><td>' + nl2br(r.Procedure) + '</td></tr>' +

      '<tr><th>References</th><td>' + nl2br(r.References) + '</td></tr></table></div></td></tr>';

  });



  html += '</tbody></table></div>';

  return html;

}



function renderPostgresPanel(cis, version) {
  cis = automatedChecks(cis, 'Status');

  const scores = sectionScores(cis);

  return renderProgressBars(scores) +

    '<h3>Control Details</h3>' +

    (version ? '<p>Postgres Version ' + escapeHtml(version) + '</p>' : '') +

    renderCisTable(cis, 'pg');

}



function renderHbaPanel(checks) {

  if (!checks?.length) {

    return '<p style="color:#666;">No HBA checks in this scan.</p>';

  }

  return '<ul class="hba-lines-list">' + checks.map(c => {

    const pass = c.status === 'pass';

    return '<li class="hba-line ' + (pass ? 'pass' : 'fail') + '">' +

      '<div><span class="line-no">Line ' + escapeHtml(c.n) + '</span> — ' + escapeHtml(c.title) +

      (c.desc ? '<div style="font-size:12px;color:#666;margin-top:4px;">' + escapeHtml(c.desc) + '</div>' : '') +

      '</div><span class="' + (pass ? 'status-pass' : 'status-fail') + '">' + (pass ? 'Pass' : 'Fail') + '</span></li>';

  }).join('') + '</ul>';

}



function renderSslPanel(sslData) {

  if (!sslData?.cells?.length && !Object.keys(sslData?.sslParams || {}).length) {

    return '<p style="color:#666;">No SSL audit in this scan.</p>';

  }

  let html = '';

  if (sslData.cells?.length) {

    html += '<table class="table maintable"><thead><tr><th>Check</th><th>Status</th><th>Details</th></tr></thead><tbody>';

    sslData.cells.forEach(c => {

      const st = String(c.status || 'fail').toLowerCase();

      html += '<tr><td>' + escapeHtml(c.title) + '</td><td>' + escapeHtml(st) + '</td><td>' +

        escapeHtml(c.desc || c.message || '') + '</td></tr>';

    });

    html += '</tbody></table>';

  }

  const params = sslData.sslParams || {};

  const keys = Object.keys(params);

  if (keys.length) {

    html += '<h4 style="margin-top:20px;">SSL Parameters</h4><table class="table"><thead><tr><th>Parameter</th><th>Value</th></tr></thead><tbody>';

    keys.forEach(k => {

      html += '<tr><td><code>' + escapeHtml(k) + '</code></td><td>' + escapeHtml(params[k]) + '</td></tr>';

    });

    html += '</tbody></table>';

  }

  const hbaLines = sslData.hbaLines || [];

  if (hbaLines.length) {

    html += '<h4 style="margin-top:16px;">pg_hba lines (host without hostssl)</h4><ul class="hba-lines-list">' +

      hbaLines.map(line => '<li class="hba-line fail"><code>' + escapeHtml(line) + '</code></li>').join('') +

      '</ul>';

  }

  return html || '<p style="color:#666;">No SSL audit in this scan.</p>';

}



function userReportSections(userData) {
  if (userData == null || userData === '') return [];
  if (typeof userData === 'string') {
    const trimmed = userData.trim();
    if (!trimmed) return [];
    try {
      return userReportSections(JSON.parse(trimmed));
    } catch {
      return [];
    }
  }
  if (Array.isArray(userData)) return userData;
  if (typeof userData === 'object') {
    const tables = userData.Tables || userData.tables;
    if (Array.isArray(tables)) return tables;
    if (userData.Data || userData.data || userData.Title || userData.title) return [userData];
  }
  return [];
}

function renderUserSectionBody(data) {
  if (!data || typeof data !== 'object') {
    return '<p style="color:#666;">No data for this section.</p>';
  }
  const desc = data.Description || data.description || '';
  const list = data.List || data.list || [];
  const table = data.Table || data.table;
  let html = '';
  if (desc) html += '<h6>' + escapeHtml(desc) + '</h6>';
  if (list.length) {
    html += '<ul>' + list.map(item => '<li>' + escapeHtml(item) + '</li>').join('') + '</ul>';
  }
  if (table) {
    const cols = table.Columns || table.columns || [];
    const rows = table.Rows || table.rows || [];
    html += '<div class="scrollable-container"><table class="table" id="manualCheckTable"><thead><tr>';
    cols.forEach(c => { html += '<th>' + escapeHtml(c) + '</th>'; });
    html += '</tr></thead><tbody>';
    rows.forEach(row => {
      html += '<tr>';
      (Array.isArray(row) ? row : []).forEach(cell => {
        html += '<td>' + escapeHtml(cell) + '</td>';
      });
      html += '</tr>';
    });
    html += '</tbody></table></div>';
  }
  return html || '<p style="color:#666;">No data for this section.</p>';
}

function renderUsersPanel(userData) {
  const sections = userReportSections(userData);
  if (!sections.length) {
    return '<p style="color:#666;">No users report in this scan.</p>';
  }
  const tableId = 'users-report-table';
  let html = '<div class="table-container">' +
    '<div class="table-toolbar">' +
    '<button type="button" class="toggleAll" data-target="' + tableId + '">Expand All</button></div>' +
    '<table class="table maintable" id="' + tableId + '"><thead><tr>' +
    '<th>Title</th><th class="icon_column">Details</th></tr></thead><tbody>';
  sections.forEach((sec, i) => {
    const title = sec.Title || sec.title || 'Report';
    const rowId = 'users-sec-' + i;
    html += '<tr class="toggleRow" data-detail="' + rowId + '">' +
      '<td>' + escapeHtml(title) + '</td>' + ICON_INFO + '</tr>' +
      '<tr class="childTableRow" id="' + rowId + '"><td colspan="2">' +
      '<div class="manualCheck">' + renderUserSectionBody(sec.Data || sec.data) + '</div></td></tr>';
  });
  html += '</tbody></table></div>';
  return html;
}



function bindReportInteractions(root) {

  root.querySelectorAll('.infoIcon').forEach(icon => {

    icon.addEventListener('click', e => {

      e.stopPropagation();

      const detail = icon.closest('tr')?.nextElementSibling;

      if (detail?.classList.contains('childTableRow')) {

        detail.classList.toggle('open');

      }

    });

  });

  root.querySelectorAll('.toggleAll').forEach(btn => {

    btn.addEventListener('click', () => {

      const table = root.querySelector('#' + CSS.escape(btn.dataset.target));

      if (!table) return;

      const rows = table.querySelectorAll('.childTableRow');

      const expand = ![...rows].every(r => r.classList.contains('open'));

      rows.forEach(r => r.classList.toggle('open', expand));

      btn.textContent = expand ? 'Collapse All' : 'Expand All';

    });

  });

  root.querySelectorAll('.nav-link').forEach(tab => {

    tab.addEventListener('click', e => {

      e.preventDefault();

      const panelId = tab.dataset.panel;

      root.querySelectorAll('.nav-link').forEach(t => t.classList.remove('active'));

      root.querySelectorAll('.tab-content').forEach(p => p.classList.remove('active-tab'));

      tab.classList.add('active');

      const panel = root.querySelector('#' + panelId);

      if (panel) panel.classList.add('active-tab');

    });

  });

}



function scrollToHashAnchor(root) {

  const hash = decodeURIComponent((location.hash || '').replace(/^#/, ''));

  if (!hash) return;

  const el = root.querySelector('[id="' + CSS.escape(hash) + '"]');

  if (!el) return;

  const detail = el.nextElementSibling;

  if (detail?.classList.contains('childTableRow')) {

    detail.classList.add('open');

  }

  requestAnimationFrame(() => el.scrollIntoView({ behavior: 'smooth', block: 'center' }));

}



function renderHostIdentity(host, hostId) {
  const hostname = host.name || hostId || 'Unknown host';
  const meta = [
    ['Hostname', hostname],
    host.ip ? ['IP address', host.ip] : null,
    host.id && host.id !== hostname ? ['Target ID', host.id] : null,
    host.postgres_version ? ['PostgreSQL', host.postgres_version] : null,
    host.last_audit ? ['Last scan', host.last_audit] : null,
    host.cis_pct ? ['CIS score', host.cis_pct] : null,
    host.failed_controls != null ? ['Failed controls', String(host.failed_controls)] : null,
  ].filter(Boolean);

  return '<div class="kshield-host-identity">' +
    '<div class="kshield-host-identity-label">Single-host report</div>' +
    '<h2 class="kshield-host-identity-name">' + escapeHtml(hostname) + '</h2>' +
    '<p class="kshield-host-identity-note">All critical violations, CIS, user, and HBA findings below belong to <strong>' +
    escapeHtml(hostname) + '</strong> only — not your full fleet.</p>' +
    '<dl class="kshield-host-meta">' +
    meta.map(([k, v]) =>
      '<div class="kshield-host-meta-row"><dt>' + escapeHtml(k) + '</dt><dd>' + escapeHtml(v) + '</dd></div>'
    ).join('') +
    '</dl></div>';
}

function renderCriticalViolationsPanel(checks) {
  checks = checks || [];
  if (!checks.length) {
    return '<p style="color:#666;">No critical violation data in this scan.</p>';
  }

  let pass = 0;
  let fail = 0;
  checks.forEach((c) => {
    const s = String(c.status || c.Status || '').toLowerCase();
    if (s === 'pass') pass++;
    else if (s === 'fail') fail++;
  });
  const total = checks.length;
  const passPct = total ? (pass / total) * 100 : 0;
  const failPct = total ? (fail / total) * 100 : 0;

  let html = '<div id="summaryStats"><h3>Summary</h3>' +
    '<div class="progress-bar">' +
    '<div class="progress-label"><span>Passing checks</span><span>' + pass + '/' + total + ' - (' + passPct.toFixed(1) + '%)</span></div>' +
    '<div class="progress"><div class="progress-filled" style="width:' + passPct + '%;background:#34A853"></div></div></div>' +
    '<div class="progress-bar">' +
    '<div class="progress-label"><span>Failing checks</span><span>' + fail + '/' + total + ' - (' + failPct.toFixed(1) + '%)</span></div>' +
    '<div class="progress"><div class="progress-filled" style="width:' + failPct + '%;background:#EA4335"></div></div></div></div>' +
    '<h3>Violation Details</h3>' + renderLegend() +
    '<div class="table-container"><table class="table maintable"><thead><tr>' +
    '<th style="width:70px;">Check #</th><th>Violation</th><th class="icon_column">Result</th><th>Source</th><th>Details</th>' +
    '</tr></thead><tbody>';

  checks.forEach((c) => {
    const status = c.status || c.Status || 'Manual';
    const num = String(c.id ?? c.ID ?? '').padStart(2, '0');
    html += '<tr><td>' + escapeHtml(num) + '</td><td>' + escapeHtml(c.title || c.Title || '') + '</td>' +
      statusCell(status) +
      '<td>' + escapeHtml(c.source || c.Source || '—') + '</td>' +
      '<td>' + escapeHtml(c.details || c.Details || '—') + '</td></tr>';
  });

  html += '</tbody></table></div>';
  return html;
}



function buildReportHtml(data, hbaChecks, sslData, hostId) {

  const host = data.host || {};

  const cis = cisFromHostData(data);

  const version = host.postgres_version || '';

  const usersData = data.user_list_result;

  const criticalChecks = data.critical_checks || [];

  const hostname = host.name || hostId || 'Unknown host';



  const tabs = [{ id: 'kshield-tab-all', label: 'All' }];

  if (criticalChecks.length) tabs.push({ id: 'kshield-tab-critical', label: 'Critical Violations' });

  if (cis.length) tabs.push({ id: 'kshield-tab-postgres', label: 'Postgres Security Report' });

  if (usersData) tabs.push({ id: 'kshield-tab-users', label: 'Users Report' });

  if (hbaChecks?.length) tabs.push({ id: 'kshield-tab-hba', label: 'HBA Scanner Report' });

  if (sslData?.cells?.length || Object.keys(sslData?.sslParams || {}).length) {

    tabs.push({ id: 'kshield-tab-ssl', label: 'SSL Report' });

  }



  let html = '<div class="kshield-html-report">' +

    '<header><div class="kshield-header-inner">' +

    '<h1 class="kshield-report-title">KloudDBShield — ' + escapeHtml(hostname) + '</h1>' +

    renderHostIdentity(host, hostId) +

    '</div></header>' +

    '<nav class="kshield-report-toolbar" aria-label="Report sections">';



  tabs.forEach((t, i) => {

    html += '<button type="button" class="nav-link' + (i === 0 ? ' active' : '') + '" data-panel="' + t.id + '">' + escapeHtml(t.label) + '</button>';

  });

  html += '</nav>';



  const postgresBody = renderPostgresPanel(cis, version);

  const criticalBody = renderCriticalViolationsPanel(criticalChecks);



  html += '<div class="tab-content active-tab" id="kshield-tab-all">';

  if (criticalChecks.length) {

    html += '<h3 class="all-title">Critical Violations</h3>' + criticalBody;

  }

  if (cis.length) {

    html += '<h3 class="all-title">Postgres Security Report</h3>' + postgresBody;

  }

  if (usersData) {

    html += '<h3 class="all-title">Users Report</h3>' + renderUsersPanel(usersData);

  }

  if (hbaChecks?.length) {

    html += '<h3 class="all-title">HBA Scanner Report</h3>' + renderHbaPanel(hbaChecks);

  }

  if (sslData?.cells?.length || Object.keys(sslData?.sslParams || {}).length) {

    html += '<h3 class="all-title">SSL Report</h3>' + renderSslPanel(sslData);

  }

  html += '</div>';



  if (criticalChecks.length) {

    html += '<div class="tab-content" id="kshield-tab-critical">' + criticalBody + '</div>';

  }

  if (cis.length) {

    html += '<div class="tab-content" id="kshield-tab-postgres">' + postgresBody + '</div>';

  }

  if (usersData) {

    html += '<div class="tab-content" id="kshield-tab-users">' + renderUsersPanel(usersData) + '</div>';

  }

  if (hbaChecks?.length) {

    html += '<div class="tab-content" id="kshield-tab-hba">' + renderHbaPanel(hbaChecks) + '</div>';

  }

  if (sslData?.cells?.length || Object.keys(sslData?.sslParams || {}).length) {

    html += '<div class="tab-content" id="kshield-tab-ssl">' + renderSslPanel(sslData) + '</div>';

  }



  html += '<footer>KloudDB Shield · Report for <strong>' + escapeHtml(hostname) + '</strong>' +
    ' · Generated from latest scan data</footer></div>';

  return html;

}



export async function mountHtmlReportViewer(container, hostId) {

  if (!container || !hostId) return;

  container.innerHTML = '<div class="kshield-html-report"><div class="kshield-loading">Loading full HTML report…</div></div>';

  try {

    const data = await getHostReport(hostId);

    if (!data?.host) {

      container.innerHTML = '<p style="color:var(--muted);">No scan data for this host.</p>';

      return;

    }

    let hbaChecks = null;

    try {

      const hba = await getHbaScanner(hostId);

      if (hba?.checks?.length) hbaChecks = hba.checks;

    } catch { /* optional */ }

    if (!hbaChecks?.length && data.hba_scan_result?.length) {

      hbaChecks = data.hba_scan_result.map((h, i) => ({

        n: h.n ?? h.line ?? i + 1,

        title: h.title ?? h.Title ?? '',

        desc: h.desc ?? h.description ?? '',

        status: (h.status ?? h.Status ?? '').toLowerCase(),

      }));

    }

    let sslData = null;

    try {

      const ssl = await getSslScanner(hostId);

      if (ssl?.available) sslData = ssl;

    } catch { /* optional */ }

    if (!sslData && data.ssl_scan_result) {

      const raw = data.ssl_scan_result;

      sslData = {

        cells: (raw.cells || raw.Cells || []).map(c => ({

          title: c.title ?? c.Title ?? '',

          status: (c.status ?? c.Status ?? '').toLowerCase(),

          message: c.message ?? c.Message ?? '',

        })),

        sslParams: raw.ssl_params || raw.sslParams || raw.SSLParams || {},

        hbaLines: raw.hba_lines || raw.hbaLines || raw.HBALines || [],

      };

    }



    const resolvedHost = data.host.name || data.host.id || hostId;
    document.title = resolvedHost + ' — KloudDBShield Report';

    container.innerHTML = buildReportHtml(data, hbaChecks, sslData, hostId);

    const reportRoot = container.querySelector('.kshield-html-report') || container;

    bindReportInteractions(reportRoot);

    scrollToHashAnchor(reportRoot);

  } catch (err) {

    container.innerHTML = '<p style="color:var(--danger);">Failed to load report: ' + escapeHtml(err.message) + '</p>';

  }

}



export async function initHtmlReportPage(hostId) {

  const root = document.getElementById('html-report-root');

  if (!root) return;

  const sub = document.getElementById('html-report-subtitle');

  if (sub && hostId) sub.textContent = hostId + ' — KloudDBShield multi-tab export';

  await mountHtmlReportViewer(root, hostId);

}


