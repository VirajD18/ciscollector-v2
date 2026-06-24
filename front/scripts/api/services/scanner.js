import { API_CONFIG } from '../config.js';
import { apiFetch } from '../client.js';

export async function getHbaScanner(host) {
  const q = host ? `?host=${encodeURIComponent(host)}` : '';
  return apiFetch(`${API_CONFIG.endpoints.hbaScanner}${q}`);
}

export async function getSslScanner(host) {
  const q = host ? `?host=${encodeURIComponent(host)}` : '';
  return apiFetch(`${API_CONFIG.endpoints.sslScanner}${q}`);
}

export async function getPiiScanner(host, options = {}) {
  const q = host ? `?host=${encodeURIComponent(host)}` : '';
  return apiFetch(`${API_CONFIG.endpoints.piiScanner}${q}`, options);
}

export async function getLogParserScanner(host) {
  const q = host ? `?host=${encodeURIComponent(host)}` : '';
  return apiFetch(`${API_CONFIG.endpoints.logParserScanner}${q}`);
}

export async function getLogReadinessFleet() {
  return apiFetch(API_CONFIG.endpoints.logReadiness);
}
