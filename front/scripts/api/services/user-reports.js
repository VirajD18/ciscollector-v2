import { API_CONFIG } from '../config.js';
import { apiFetch } from '../client.js';

export async function getInactiveUsersReport() {
  return apiFetch(API_CONFIG.endpoints.inactiveUsersReport);
}

export async function getCommonUsersReport() {
  return apiFetch(API_CONFIG.endpoints.commonUsersReport);
}
