export function parseErrorMessage(body, fallback) {
  return body?.error?.message || fallback;
}

export async function readJSON(res) {
  return res.json().catch(() => ({}));
}

export function etagRevision(res) {
  const raw = res.headers.get('ETag');
  if (!raw) return '0';
  return String(raw).replaceAll('"', '');
}