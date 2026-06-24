import { loadHtml } from '../utils/dom.js';
import { PAGE_IDS } from '../router/routes.js';
import {
  hostsApi,
  fleetApi,
  strategicApi,
  violationsApi,
  criticalChecksApi,
  overviewApi,
  runsApi,
  mapHostsResponse,
  mapFleetCategories,
  mapStrategicRange,
  normalizeStrategicRange,
  emptyStrategicRange,
  mapCriticalChecksResponse,
} from '../api/index.js';

const BASE = new URL('.', import.meta.url);

function asset(path) {
  return new URL(`../../${path}`, BASE).pathname;
}

async function loadPages() {
  const htmlParts = await Promise.all(
    PAGE_IDS.map((id) => loadHtml(asset(`pages/${id}.html`))),
  );
  return htmlParts.join('\n');
}

async function loadShell() {
  const [sidebar, topbar, pagesHtml] = await Promise.all([
    loadHtml(asset('components/sidebar.html')),
    loadHtml(asset('components/topbar.html')),
    loadPages(),
  ]);

  const root = document.getElementById('app-root');
  if (!root) throw new Error('#app-root not found');

  root.innerHTML =
    sidebar +
    `<div class="main">${topbar}<main class="content" id="page-root">${pagesHtml}</main></div>`;
}

export async function initApp() {
  await loadShell();

  let hosts = [];
  let fleetCategories = [];
  let strategic30d = null;
  let criticalViolationRows = [];
  let criticalViolationFilters = {
    checkOptions: [],
    checkDefinitions: [],
    serverOptions: [],
    sourceOptions: [],
    typeOptions: [],
    severityOptions: [],
  };
  let overview = null;
  let runs = [];

  try {
    const [hostsData, fleetData, strategicData, criticalChecksData, overviewData, runsData] =
      await Promise.all([
        hostsApi.getHosts(),
        fleetApi.getFleetCategories(),
        strategicApi.getStrategicMatrix('30d'),
        criticalChecksApi.getCriticalChecks(),
        overviewApi.getOverview(),
        runsApi.getRuns(50),
      ]);
    hosts = mapHostsResponse(hostsData);
    fleetCategories = mapFleetCategories(fleetData);
    strategic30d = normalizeStrategicRange(mapStrategicRange(strategicData, '30d'), emptyStrategicRange('Last 30 days'));
    runs = runsData?.runs || [];
    const criticalMapped = mapCriticalChecksResponse(criticalChecksData);
    criticalViolationRows = criticalMapped.rows;
    criticalViolationFilters = {
      checkOptions: criticalMapped.checkOptions || [],
      checkDefinitions: criticalMapped.checks || [],
      serverOptions: criticalMapped.serverOptions || [],
      sourceOptions: criticalMapped.sourceOptions || [],
      typeOptions: criticalMapped.typeOptions || [],
      severityOptions: criticalMapped.severityOptions || [],
    };
    overview = overviewData;
  } catch (err) {
    console.warn('API load failed — dashboard will show empty state until main-server is running:', err);
  }

  window.__SHIELD_BOOT__ = {
    hosts,
    fleetCategories,
    strategic30d,
    criticalViolationRows,
    criticalViolationFilters,
    overview,
    runs,
  };

  const { initGlobalSearch } = await import('../pages/search.js');
  initGlobalSearch();

  const { fetchStrategicForRange, reloadFleetCategories } = await import('./strategic-loader.js');
  window.__SHIELD_API__ = {
    fetchStrategic: fetchStrategicForRange,
    reloadFleetCategories,
  };

  await import('./prototype-app.js');
  document.getElementById('app-root')?.removeAttribute('aria-busy');
}
