import { API_CONFIG } from '../config.js';
import { apiFetch, fetchApiOrMock } from '../client.js';

export async function getFleetCategories() {
  const data = await fetchApiOrMock(API_CONFIG.endpoints.fleetCategories, 'fleet-categories.json');
  return data.categories || data;
}
