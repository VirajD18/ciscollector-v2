import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getOverview() {
  return fetchApiOrMock(API_CONFIG.endpoints.overview, null);
}
