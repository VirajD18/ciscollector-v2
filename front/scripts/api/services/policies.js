import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getPolicies() {
  return fetchApiOrMock(API_CONFIG.endpoints.policies, 'policies.json');
}
