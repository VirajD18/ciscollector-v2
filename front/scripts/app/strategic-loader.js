/**
 * Fleet Overview / strategic dashboard data — always from Go API → SQLite (reportstore).
 * UI layout comes from dba-console-prototype.html (reference only).
 */
import { strategicApi, fleetApi, mapStrategicRange, mapFleetCategories, normalizeStrategicRange, emptyStrategicRange } from '../api/index.js';

export async function fetchStrategicForRange(range) {
  const raw = await strategicApi.getStrategicMatrix(range);
  const live = mapStrategicRange(
    raw?.ranges ? raw : { ranges: { [range]: raw } },
    range,
  );
  const label = range === '7d' ? 'Last 7 days' : range === '24h' ? 'Last 24 hours' : 'Last 30 days';
  return normalizeStrategicRange(live, emptyStrategicRange(label));
}

export async function reloadFleetCategories() {
  const data = await fleetApi.getFleetCategories();
  return mapFleetCategories(data);
}
