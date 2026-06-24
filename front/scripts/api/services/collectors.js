import { API_CONFIG } from '../config.js';
import { apiFetch, fetchApiOrMock, fetchMock } from '../client.js';

/** @returns {Promise<{ nodes: object[], updated_at: string }>} */
export async function fetchCollectorNodes() {
  return fetchApiOrMock(API_CONFIG.endpoints.collectorNodes, 'collectors/nodes.json');
}

/** @param {string} nodeId */
export async function fetchCollectorNode(nodeId) {
  return fetchApiOrMock(API_CONFIG.endpoints.collectorNode(nodeId), 'collectors/nodes.json');
}

/** @param {string} nodeId */
export async function fetchCollectorNodeRuns(nodeId) {
  if (API_CONFIG.useMock) {
    return fetchMock('collectors/runs-demo-collector-node-1.json');
  }
  return apiFetch(API_CONFIG.endpoints.collectorNodeRuns(nodeId));
}

/** @param {string} nodeId */
export async function fetchCollectorNodeActivity(nodeId) {
  if (API_CONFIG.useMock) {
    return fetchMock('collectors/activity-demo-collector-node-1.json');
  }
  return apiFetch(API_CONFIG.endpoints.collectorNodeActivity(nodeId));
}

/** @param {string} nodeId */
export async function fetchCollectorNodeLogs(nodeId) {
  return apiFetch(API_CONFIG.endpoints.collectorNodeLogs(nodeId));
}
