import { API_CONFIG } from '../config.js';
import { apiFetch } from '../client.js';

export async function getGucDrift() {
  return apiFetch(API_CONFIG.endpoints.gucDrift);
}

export async function getGucBaseline() {
  return apiFetch(API_CONFIG.endpoints.gucBaseline);
}

export async function putGucBaseline(payload) {
  return apiFetch(API_CONFIG.endpoints.gucBaseline, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function getGucSnapshots() {
  return apiFetch(API_CONFIG.endpoints.gucSnapshots);
}
