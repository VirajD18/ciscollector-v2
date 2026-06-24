/** Map Go API DTOs to prototype-app.js data shapes. */

/** Empty strategic range — zeros only; no demo KPIs. */
export function emptyStrategicRange(label = 'Last 30 days') {
  return {
    label,
    health: 0,
    grade: '-',
    gradeColor: 'var(--muted)',
    critical: 0,
    cis: 0,
    servers: 0,
    privs: [],
    hygiene: { active: 0, inactive: 0, common: 0 },
    cred: { hosts: 0, exposed: 0, weak: 0, ok: 0 },
    hba: [],
    hbaScanned: false,
    sslEnforced: 0,
    sslScanned: false,
    drift: [],
    driftLabels: [],
    audit: [],
    heatmap: [],
    heatmapColumns: [],
    piiScanned: false,
  };
}

export function mapHostsResponse(data) {
  if (data?.instances?.length) {
    return data.instances;
  }
  return data?.rows || data || [];
}

export function mapFleetCategories(data) {
  return data?.categories || data || [];
}

export function mapStrategicRange(data, range = '30d') {
  if (data?.ranges?.[range]) return data.ranges[range];
  if (data?.health != null) return data;
  return null;
}

/** Merge API range with empty shell only (no demo KPIs). */
export function normalizeStrategicRange(live, fallback) {
  const f = fallback || emptyStrategicRange();
  const l = live || {};
  return {
    label: l.label ?? f.label,
    health: l.health ?? f.health ?? 0,
    grade: l.grade ?? f.grade ?? '-',
    gradeColor: l.gradeColor ?? f.gradeColor ?? 'var(--muted)',
    critical: l.critical ?? f.critical ?? 0,
    cis: l.cis ?? f.cis ?? 0,
    servers: l.servers ?? f.servers ?? 0,
    privs: l.privs?.length ? l.privs : [],
    hygiene: l.hygiene ?? f.hygiene ?? { active: 0, inactive: 0, common: 0 },
    cred: l.cred ?? f.cred ?? { hosts: 0, exposed: 0, weak: 0, ok: 0 },
    hba: l.hba?.length ? l.hba : [],
    hbaScanned: l.hbaScanned === true,
    sslEnforced: l.sslEnforced ?? f.sslEnforced ?? 0,
    sslScanned: l.sslScanned === true,
    drift: l.drift?.length ? l.drift : [],
    driftLabels: l.driftLabels?.length ? l.driftLabels : [],
    audit: l.audit?.length ? l.audit : [],
    heatmap: l.piiScanned === true && l.heatmap?.length ? l.heatmap : [],
    heatmapColumns: l.piiScanned === true && l.heatmapColumns?.length ? l.heatmapColumns : [],
    piiScanned: l.piiScanned === true,
  };
}

export function mapCriticalChecksResponse(data) {
  const rows = (data?.rows || []).map((r) => ({
    id: r.id,
    checkId: r.check_id != null && r.check_id !== '' ? Number(r.check_id) : null,
    check: r.check,
    server: r.server,
    details: r.details,
    status: r.status || 'Open',
    detected: r.detected || '',
    source: r.source || '',
    violationType: r.violation_type || '',
    severity: r.severity || 'CRITICAL',
  }));
  return {
    rows,
    checks: data?.checks || [],
    hostRows: data?.host_rows || [],
    checkFails: data?.check_fails || [],
    checkOptions: data?.check_options || [],
    serverOptions: data?.server_options || [],
    sourceOptions: data?.source_options || [],
    typeOptions: data?.type_options || [],
    severityOptions: data?.severity_options || [],
  };
}

export function mapViolationsResponse(violations) {
  if (violations?.checks?.length || violations?.rows?.some((r) => r.check_id != null)) {
    return mapCriticalChecksResponse(violations);
  }
  return {
    rows: mapViolationsToCriticalRows(violations),
    typeOptions: violations?.type_options || [],
    severityOptions: violations?.severity_options || [],
    checkOptions: [],
    serverOptions: [],
  };
}

export function mapViolationsToCriticalRows(violations) {
  if (violations?.rows?.length) {
    return violations.rows.map((r) => ({
      id: r.id,
      server: r.server,
      type: r.type,
      details: r.details,
      severity: r.severity,
      detected: r.detected || '',
      status: r.status || 'Open',
      configSection: r.configSection || 'block-cis',
    }));
  }
  const rows = [];
  let id = 1;
  const push = (v, defaultType, defaultSection) => {
    const sev = v.severity === 'critical' ? 'CRITICAL' : 'HIGH';
    rows.push({
      id: `V-${String(id).padStart(3, '0')}`,
      server: v.host,
      type: v.violation_type || defaultType,
      details: v.check,
      severity: sev,
      detected: v.detected_at || '',
      status: 'Open',
      configSection: defaultSection,
    });
    id += 1;
  };
  for (const v of violations?.critical || []) push(v, 'Critical Config', 'block-cis');
  for (const v of violations?.high || []) push(v, 'HBA Violation', 'block-critical-violations');
  return rows;
}

export function mergeStrategicIntoRanges(baseRanges, liveRange, rangeKey = '30d') {
  if (!liveRange) return baseRanges;
  const merged = { ...baseRanges };
  merged[rangeKey] = { ...merged[rangeKey], ...liveRange };
  return merged;
}
