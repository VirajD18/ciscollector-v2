import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getHosts() {
  const data = await fetchApiOrMock(API_CONFIG.endpoints.hosts, 'hosts.json');
  if (data?.instances?.length) {
    return data;
  }
  return data?.rows || data;
}
