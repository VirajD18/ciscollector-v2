/** Auto-synced from dba-console-prototype.html — re-run extract-prototype.mjs after edits */
export const DEFAULT_PAGE = 'strategic-dashboard';

export const PAGE_IDS = [
  "strategic-dashboard",
  "critical-violations",
  "fleet-category",
  "log-readiness",
  "hosts",
  "hba-scanner",
  "ssl-scanner",
  "pii-scanner",
  "log-parser",
  "host-detail",
  "html-report",
  "guc-drift",
  "inactive-users-report",
  "common-users-report",
  "policies",
  "collector-nodes",
];

export const PAGE_META = {
    'fleet-category': { title: 'Fleet category', crumb: 'Category detail' },
    'log-readiness': { title: 'Log Readiness', crumb: 'GUC Gates · Parser Readiness' },
    hosts: { title: 'Hosts', crumb: 'Monitored PostgreSQL hosts' },
    'host-detail': { title: 'Host Audit', crumb: 'Vertical Report · CIS, Config, Access, Ops' },
    'html-report': { title: 'Full HTML report', crumb: 'KloudDBShield multi-tab export' },
    'guc-drift': { title: 'GUC Drift', crumb: 'Vs Golden Baseline' },
    'inactive-users-report': { title: 'Inactive Users Report', crumb: 'Fleet-Wide · Log Parser Menu 6' },
    'common-users-report': { title: 'Common Users Report', crumb: 'Fleet-Wide · Users Report Menu 9' },
    'strategic-dashboard': { title: 'Fleet Overview', crumb: 'Executive Fleet Dashboard' },
    'critical-violations': { title: 'Critical Violations', crumb: 'Group By Violation' },
    'hba-scanner': { title: 'HBA Scanner', crumb: 'pg_hba.conf Checks · Menu 3' },
    'ssl-scanner': { title: 'SSL Scanner', crumb: 'SSL Audit · Menu 15' },
    'pii-scanner': { title: 'Postgres PII Report', crumb: 'PII Scan · Menu 4' },
    'log-parser': { title: 'Log parser', crumb: 'pg_log parser findings' },
    policies: { title: 'Security policies', crumb: 'Templates · groups · schedule · email' },
    'collector-nodes': { title: 'Collector Nodes', crumb: 'Live Fleet · Heartbeat Status' },
};
