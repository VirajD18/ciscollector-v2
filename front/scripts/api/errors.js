/**
 * Turn API / Postgres errors into short UI messages.
 */
export function formatConnectionError(raw, host, port) {
  const msg = String(raw || '').trim();
  const lower = msg.toLowerCase();
  const target = host && port ? `${host}:${port}` : (host || 'the server');

  if (
    lower.includes('actively refused') ||
    lower.includes('connection refused') ||
    lower.includes('connectex')
  ) {
    return `Cannot connect to PostgreSQL at ${target}. Nothing is listening on that port — start PostgreSQL or check host/port.`;
  }
  if (lower.includes('no such host') || lower.includes('unknown host')) {
    return `Host "${host}" could not be found. Check the hostname or IP address.`;
  }
  if (
    lower.includes('timeout') ||
    lower.includes('i/o timeout') ||
    lower.includes('deadline exceeded') ||
    lower.includes('context deadline')
  ) {
    return `Connection to ${target} timed out. Check host, port, and that PostgreSQL is running.`;
  }
  if (lower.includes('password authentication failed') || lower.includes('invalid password')) {
    return 'Authentication failed. Check username and password.';
  }
  if (lower.includes('does not exist') && lower.includes('database')) {
    return `Database does not exist on ${target}. Check the database name.`;
  }
  if (lower.includes('ssl') && (lower.includes('required') || lower.includes('handshake'))) {
    return 'SSL connection failed. This server may require sslmode= require in advanced config.';
  }
  if (lower.includes('report database is busy') || lower.includes('database is locked') || lower.includes('sqlite_busy')) {
    return 'Report database is busy. Close DB Browser on klouddbshield.db, wait for any running ciscollector scan to finish, then try again.';
  }
  if (msg.startsWith('API ') && msg.includes('failed')) {
    return parseApiErrorText(msg, host, port);
  }
  return msg || 'Connection failed.';
}

function parseApiErrorText(text, host, port) {
  const jsonStart = text.indexOf('{');
  if (jsonStart >= 0) {
    try {
      const body = JSON.parse(text.slice(jsonStart));
      if (body.message) {
        return formatConnectionError(body.message, host, port);
      }
    } catch {
      /* ignore */
    }
  }
  return formatConnectionError(text, host, port);
}

export function errorMessageFromCaught(err, host, port) {
  if (!err) return 'Request failed.';
  const raw = err.message || String(err);
  return formatConnectionError(raw, host, port);
}
