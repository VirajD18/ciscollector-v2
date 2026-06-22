import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getRuns(limit = 50) {
  const path = `${API_CONFIG.endpoints.runs}?limit=${limit}`;
  return fetchApiOrMock(path, null);
}
