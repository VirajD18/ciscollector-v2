import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getViolations() {
  return fetchApiOrMock(API_CONFIG.endpoints.violations, 'violations.json');
}
