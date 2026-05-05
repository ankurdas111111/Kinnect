import { writable, get } from 'svelte/store';
import { apiGet } from '../api.js';

export const authUser = writable(null);
export const authLoading = writable(true);

function applySessionData(data) {
  authUser.set({
    userId: data.userId,
    displayName: data.displayName,
    role: data.role,
    shareCode: data.shareCode,
    email: data.email,
    mobile: data.mobile,
  });
}

export async function loadSession() {
  authLoading.set(true);
  try {
    const data = await apiGet('/api/me');
    if (data?.ok) {
      applySessionData(data);
    } else if (data?.error === 'Not authenticated') {
      // Definitive 401 — not logged in, redirect immediately
      authUser.set(null);
    } else {
      // Transient failure: network error, 502/503 from server cold-starting, etc.
      // Wait 3s and retry once before concluding the user isn't logged in.
      await new Promise(r => setTimeout(r, 3000));
      const retry = await apiGet('/api/me');
      if (retry?.ok) {
        applySessionData(retry);
      } else {
        authUser.set(null);
      }
    }
  } catch {
    authUser.set(null);
  }
  authLoading.set(false);
}

export function isAdmin() {
  const u = get(authUser);
  return u && u.role === 'admin';
}

export function isAuthenticated() {
  return get(authUser) !== null;
}
