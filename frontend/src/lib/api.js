import API_BASE from './env.js';

let csrfToken = null;

function buildApiUrl(path) {
  const normalizedPath = String(path || '').startsWith('/') ? String(path || '') : '/' + String(path || '');
  if (!API_BASE) return normalizedPath;
  return API_BASE + normalizedPath;
}

export async function fetchCsrf() {
  try {
    // Deadline here too: without one, a stalled /api/csrf blocked every
    // mutation behind it indefinitely, since apiPost awaits this first.
    const res = await fetch(buildApiUrl('/api/csrf'), {
      credentials: 'include',
      signal: AbortSignal.timeout(DEFAULT_TIMEOUT)
    });
    if (res.ok) {
      const data = await res.json();
      csrfToken = data.csrfToken;
    }
  } catch {
    // Network error — csrfToken stays null
  }
  return csrfToken;
}

export function getCsrf() {
  return csrfToken;
}

export function clearCsrf() {
  csrfToken = null;
}

function safeJson(res) {
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) return res.json();
  return res.text().then(text => ({ ok: false, error: text || `Request failed (${res.status})` }));
}

/** Nothing in this app should hang forever waiting on a phone's flaky network. */
const DEFAULT_TIMEOUT = 10_000;

/**
 * Failure shape. `retryable` lets a caller say "you appear to be offline —
 * retry?" instead of spinning; `status` lets it tell rejected apart from
 * expired. Success responses are still the parsed API body, untouched, because
 * callers read the API's own `ok` field from it.
 */
const fail = (status, error, retryable) => ({ ok: false, status, error, retryable });

/**
 * fetch with a deadline and an optional caller abort signal.
 * The caller's signal (a route change, say) and our timeout both abort the
 * same request, which is what stops a stale response from overwriting fresh
 * state after the user has moved on.
 */
async function request(url, init = {}, { timeout = DEFAULT_TIMEOUT, signal } = {}) {
  const ctl = new AbortController();
  const onAbort = () => ctl.abort();
  signal?.addEventListener('abort', onAbort, { once: true });
  const timer = setTimeout(() => ctl.abort(new DOMException('timeout', 'TimeoutError')), timeout);
  try {
    return { res: await fetch(buildApiUrl(url), { credentials: 'include', ...init, signal: ctl.signal }) };
  } catch (e) {
    if (e?.name === 'TimeoutError') return { err: fail(0, 'Request timed out', true) };
    if (e?.name === 'AbortError') return { err: fail(0, 'Cancelled', false) };
    return { err: fail(0, 'Network error', true) };
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener('abort', onAbort);
  }
}

/** POST/DELETE share CSRF handling: a 403 means the token rotated or expired. */
async function withCsrfRetry(send) {
  if (!csrfToken) await fetchCsrf();
  let out = await send();
  // Before this, an expired token was permanent: csrfToken was already non-null
  // so fetchCsrf() never re-ran, and every mutation failed until a full reload.
  if (out.res?.status === 403) {
    clearCsrf();
    await fetchCsrf();
    out = await send();
  }
  return out;
}

export async function apiPost(url, body = {}, opts) {
  const { res, err } = await withCsrfRetry(() => request(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'x-csrf-token': csrfToken || '' },
    // The duplicate _csrf in the body is deliberate: the backend reads it, so
    // dropping it is a backend-coupled change, not a frontend one.
    body: JSON.stringify({ ...body, _csrf: csrfToken })
  }, opts));
  if (err) return err;
  return safeJson(res);
}

export async function apiGet(url, opts) {
  // GET is idempotent, so one retry on a retryable failure is safe. POST is not.
  let { res, err } = await request(url, {}, opts);
  if (err?.retryable) ({ res, err } = await request(url, {}, opts));
  if (err) return err;
  if (res.status === 401) return { ok: false, error: 'Not authenticated', status: 401, retryable: false };
  return safeJson(res);
}

export async function apiDelete(url, opts) {
  const { res, err } = await withCsrfRetry(() => request(url, {
    method: 'DELETE',
    headers: { 'x-csrf-token': csrfToken || '' }
  }, opts));
  if (err) return err;
  if (res.status === 204) return { ok: true };
  return safeJson(res);
}
