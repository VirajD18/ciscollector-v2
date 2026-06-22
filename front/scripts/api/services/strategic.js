import { API_CONFIG } from '../config.js';
import { fetchApiOrMock } from '../client.js';

export async function getStrategicMatrix(range = '30d') {
  const data = await fetchApiOrMock(
    `${API_CONFIG.endpoints.strategic}?range=${encodeURIComponent(range)}`,
    'strategic-30d.json',
  );
  return data.ranges?.[range] || data.ranges?.['30d'] || data;
}
