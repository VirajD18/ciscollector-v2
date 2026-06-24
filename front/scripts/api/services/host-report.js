import { API_CONFIG } from '../config.js';
import { apiFetch } from '../client.js';

export async function getHostReport(hostId) {
  const url = API_CONFIG.endpoints.server(hostId);
  return apiFetch(url);
}

export async function getHostInstance(instance) {
  return apiFetch(API_CONFIG.endpoints.serverInstance(instance));
}
