import { hostsApi } from '../api/index.js';
import { mapHostsResponse } from '../api/mappers.js';
import { getHostInstance } from '../api/services/host-report.js';

function escapeHtml(s) {
  return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export function scannerHostKey(instance, database) {
  if (!instance) return '';
  if (!database) return instance;
  return instance + '/' + database;
}

export function parseScannerHostKey(hostKey) {
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
    return { name: n, hostKey: scannerHostKey(instance, n) };
  }).filter((db) => db.name);
}

/** Instance rows with databases for scanner pages. */
export function normalizeScannerTargets(hostsData) {
  const rows = mapHostsResponse(hostsData) || [];
  if (!Array.isArray(rows)) return [];
  return rows.map((h) => {
    if (Array.isArray(h)) {
      const parsed = parseScannerHostKey(h[0]);
      const instance = parsed.instance || h[0] || '';
      let databases = parsed.database
        ? [{ name: parsed.database, hostKey: scannerHostKey(instance, parsed.database) }]
        : [];
      if (!databases.length) {
        databases = databasesFromLabel(h[2], instance);
      }
      return { instance, ip: h[1] || '', databases };
    }
    const instance = String(h.instance || h[0] || '').trim();
    let databases = Array.isArray(h.databases)
      ? h.databases.map((db) => ({
        name: db.name || db,
        hostKey: db.host_key || scannerHostKey(instance, db.name || db),
      }))
      : [];
    if (!databases.length) {
      databases = databasesFromLabel(h.databases_label, instance);
    }
    return { instance, ip: h.ip || h[1] || '', databases };
  }).filter((h) => h.instance);
}

let scannerTargetsCache = [];

export async function fetchScannerTargets() {
  try {
    const data = await hostsApi.getHosts();
    scannerTargetsCache = normalizeScannerTargets(data);
    return scannerTargetsCache;
  } catch {
    const boot = window.__SHIELD_BOOT__?.hosts;
    scannerTargetsCache = boot?.length ? normalizeScannerTargets(boot) : [];
    return scannerTargetsCache;
  }
}

export async function enrichScannerDatabases(instance, targets) {
  const list = targets || scannerTargetsCache;
  const inst = list.find((x) => x.instance === instance);
  if (!inst || inst.databases?.length) return inst;
  try {
    const overview = await getHostInstance(instance);
    const dbs = (overview?.databases || []).map((db) => ({
      name: db.name || db,
      hostKey: db.host_key || scannerHostKey(instance, db.name || db),
    }));
    if (dbs.length) inst.databases = dbs;
  } catch {
    /* keep empty */
  }
  return inst;
}

export function populateScannerInstanceSelect(select, instances, selectedInstance) {
  if (!select) return '';
  if (!instances?.length) {
    select.innerHTML = '<option value="">No hosts in DB</option>';
    return '';
  }
  const want = selectedInstance || select.value || instances[0].instance;
  select.innerHTML = instances.map((inst) => {
    const sel = inst.instance === want ? ' selected' : '';
    const ip = inst.ip && inst.ip !== '-' ? ' (' + escapeHtml(inst.ip) + ')' : '';
    return '<option value="' + escapeHtml(inst.instance) + '"' + sel + '>' +
      escapeHtml(inst.instance) + ip + '</option>';
  }).join('');
  if (want && [...select.options].some((o) => o.value === want)) {
    select.value = want;
  }
  return select.value || want;
}

export function populateScannerDatabaseSelect(select, databases, selectedDatabase) {
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

export function formatScannerTargetLabel(instance, database) {
  if (!instance) return '';
  if (!database) return instance;
  return instance + ' / ' + database;
}

export function readScannerTarget(instanceSelectId, databaseSelectId) {
  const instEl = document.getElementById(instanceSelectId);
  const dbEl = document.getElementById(databaseSelectId);
  const instance = instEl?.value || '';
  const database = dbEl?.value || '';
  return { instance, database, hostKey: scannerHostKey(instance, database) };
}

export async function initScannerTargetSelects({
  instanceSelectId,
  databaseSelectId,
  keepInstance = '',
  keepDatabase = '',
  onTargetChange,
}) {
  const instSelect = document.getElementById(instanceSelectId);
  const dbSelect = document.getElementById(databaseSelectId);
  const targets = await fetchScannerTargets();
  const instance = populateScannerInstanceSelect(instSelect, targets, keepInstance);
  const inst = await enrichScannerDatabases(instance, targets);
  const database = populateScannerDatabaseSelect(dbSelect, inst?.databases || [], keepDatabase);

  if (!instSelect?.dataset.bound) {
    instSelect.dataset.bound = '1';
    instSelect.addEventListener('change', async () => {
      const row = await enrichScannerDatabases(instSelect.value, targets);
      populateScannerDatabaseSelect(dbSelect, row?.databases || [], '');
      if (onTargetChange) onTargetChange(readScannerTarget(instanceSelectId, databaseSelectId));
    });
  }
  if (!dbSelect?.dataset.bound) {
    dbSelect.dataset.bound = '1';
    dbSelect.addEventListener('change', () => {
      if (onTargetChange) onTargetChange(readScannerTarget(instanceSelectId, databaseSelectId));
    });
  }

  return { instance, database, hostKey: scannerHostKey(instance, database), targets };
}
