import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getCriticalChecks() {
  return fetchApiOrMock(API_CONFIG.endpoints.criticalChecks, 'critical-checks.json');
}
