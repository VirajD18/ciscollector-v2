import { API_CONFIG } from './config.js';

/**
 * HTTP client for KloudDB Shield APIs.
 * Uses mock JSON when API_CONFIG.useMock is true.
 */
export async function apiFetch(path, options = {}) {
  const { method = 'GET', body, headers = {}, signal } = options;
  const url = path.startsWith('http') ? path : `${API_CONFIG.baseUrl}${path}`;

  const init = {
    method,
    credentials: 'same-origin',
    headers: { ...API_CONFIG.defaultHeaders, ...headers },
    signal,
  };

  if (body !== undefined) {
    init.body = typeof body === 'string' ? body : JSON.stringify(body);
  }

  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    let message = text;
    try {
      const body = JSON.parse(text);
      if (body && typeof body.message === 'string' && body.message.trim()) {
        message = body.message.trim();
      }
    } catch {
      /* plain text error body */
    }
    const err = new Error(message);
    err.status = res.status;
    err.apiPath = path;
    throw err;
  }

  const contentType = res.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return res.json();
  }
  return res.text();
}

export async function fetchMock(filename) {
  const url = `${API_CONFIG.mockBase}/${filename}`;
  const res = await fetch(url, { credentials: 'same-origin' });
  if (!res.ok) throw new Error(`Mock ${filename} not found (${res.status})`);
  return res.json();
}

export async function fetchApiOrMock(apiPath, mockFile) {
  if (API_CONFIG.useMock) {
    return fetchMock(mockFile);
  }
  return apiFetch(apiPath);
}
