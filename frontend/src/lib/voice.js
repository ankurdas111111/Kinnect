/**
 * voice.js — spoken-updates engine (opt-in TTS, default OFF).
 *
 * Web SpeechSynthesis primary: zero dependencies, works on web + Android
 * WebView + foregrounded iOS WKWebView. The engine is isolated behind
 * _speakRaw() so @capacitor-community/text-to-speech can slot in later for
 * iOS background speech (mirror the haptics.js lazy-load pattern).
 *
 * Known limits (accepted v1): iOS backgrounding suspends WebView JS — an
 * utterance may drop until foregrounded; desktop Chrome may defer speech
 * started in a hidden tab. Android with the tracking foreground service
 * keeps speaking in the background — the core driving scenario works.
 *
 * Queue semantics:
 *   urgent — SOS only. Cancels everything, bypasses rate limit AND
 *            mute-for-today (safety app: SOS always speaks; the mute copy
 *            says so). Only the master toggle silences it.
 *   normal — verdict transitions, arrivals, SOS-end. Deduped (same string
 *            <30s apart dropped), queue depth 3, token-bucket rate limited.
 *   low    — check-ins. Dropped if anything is speaking/queued; additionally
 *            gated by the voiceCheckins sub-toggle.
 *
 * Debug: localStorage.kinnect_voice_debug = '1' → every accept/drop decision
 * lands in window.__voiceLog (ring buffer) so headless tests can assert
 * behavior without hearing audio.
 */
import { writable, get } from 'svelte/store';

const ENABLED_KEY = 'kinnect_voice_enabled';
const CHECKINS_KEY = 'kinnect_voice_checkins';
const MUTE_KEY = 'kinnect_voice_mute_until';
const DEBUG_KEY = 'kinnect_voice_debug';

const RATE_LIMIT_PER_MIN = 6;
const DEDUPE_MS = 30_000;
const MAX_QUEUE = 3;

function lsGet(k) { try { return localStorage.getItem(k); } catch { return null; } }
function lsSet(k, v) { try { localStorage.setItem(k, v); } catch { /* private mode */ } }

// ── Persisted stores ─────────────────────────────────────────────────────────
export const voiceEnabled = writable(lsGet(ENABLED_KEY) === '1');
export const voiceCheckins = writable(lsGet(CHECKINS_KEY) === '1');
export const voiceMutedUntil = writable(Number(lsGet(MUTE_KEY)) || 0);

voiceCheckins.subscribe((v) => lsSet(CHECKINS_KEY, v ? '1' : '0'));
voiceMutedUntil.subscribe((v) => lsSet(MUTE_KEY, String(v || 0)));

/**
 * Enable/disable from a REAL user gesture (ToggleControl click). The enable
 * path immediately speaks the confirmation — that single utterance is the
 * iOS speech unlock priming.
 */
export function setVoiceEnabledFromGesture(on) {
  voiceEnabled.set(!!on);
  lsSet(ENABLED_KEY, on ? '1' : '0');
  if (on) {
    _speakNow('Voice updates are on.');
  } else {
    cancelAll();
  }
}

/** Mute non-urgent speech until local midnight. SOS still speaks (by design). */
export function muteForToday() {
  const d = new Date();
  d.setHours(23, 59, 59, 999);
  voiceMutedUntil.set(d.getTime());
}
export function unmute() { voiceMutedUntil.set(0); }

export function isSupported() {
  return typeof window !== 'undefined' && !!window.speechSynthesis;
}

// ── Debug observability ──────────────────────────────────────────────────────
function logDecision(decision, text, priority) {
  if (lsGet(DEBUG_KEY) !== '1' || typeof window === 'undefined') return;
  window.__voiceLog = window.__voiceLog || [];
  window.__voiceLog.push({ ts: Date.now(), decision, text, priority });
  if (window.__voiceLog.length > 100) window.__voiceLog.shift();
}

// ── Voice selection (async via voiceschanged) ────────────────────────────────
let _voice = null;
let _voiceListening = false;
function pickVoice() {
  if (!isSupported()) return null;
  const voices = window.speechSynthesis.getVoices() || [];
  if (!voices.length) {
    if (!_voiceListening) {
      _voiceListening = true;
      window.speechSynthesis.addEventListener('voiceschanged', () => { _voice = null; pickVoice(); }, { once: true });
    }
    return null; // queued utterances go out with the engine default
  }
  const lang = (typeof navigator !== 'undefined' && navigator.language) || 'en';
  _voice =
    voices.find((v) => v.localService && v.lang === lang) ||
    voices.find((v) => v.localService && v.lang?.startsWith('en')) ||
    voices.find((v) => v.lang?.startsWith('en')) ||
    voices[0];
  return _voice;
}

// ── Engine facade ────────────────────────────────────────────────────────────
function _speakRaw(text) {
  if (!isSupported()) return;
  try {
    const u = new SpeechSynthesisUtterance(text);
    u.rate = 0.95;   // calm brand voice — slightly unhurried
    u.pitch = 1.0;
    u.volume = 1.0;
    const v = _voice || pickVoice();
    if (v) u.voice = v;
    // Chrome stuck-paused bug: resume defensively before every speak
    window.speechSynthesis.resume();
    window.speechSynthesis.speak(u);
  } catch { /* engine hiccup — drop silently */ }
}

function _speakNow(text) {
  logDecision('spoken', text, 'confirm');
  _speakRaw(text);
}

// ── Queue + rate limiting ────────────────────────────────────────────────────
let _queue = [];               // [{text, priority}]
let _recent = new Map();       // text → last spoken ts (dedupe)
let _bucket = [];              // timestamps of non-urgent utterances (token bucket)
let _draining = false;

function rateLimited() {
  const cutoff = Date.now() - 60_000;
  _bucket = _bucket.filter((t) => t >= cutoff);
  return _bucket.length >= RATE_LIMIT_PER_MIN;
}

function engineBusy() {
  return isSupported() && (window.speechSynthesis.speaking || window.speechSynthesis.pending);
}

function drain() {
  if (_draining || !_queue.length) return;
  if (engineBusy()) { setTimeout(drain, 400); return; }
  _draining = true;
  const item = _queue.shift();
  _bucket.push(Date.now());
  logDecision('spoken', item.text, item.priority);
  _speakRaw(item.text);
  _draining = false;
  if (_queue.length) setTimeout(drain, 600);
}

/**
 * Speak a line, subject to priority policy. All gating lives here — callers
 * never need to check enabled/mute/support themselves.
 * @param {string} text
 * @param {{ priority?: 'urgent'|'normal'|'low' }} [opts]
 */
export function speak(text, { priority = 'normal' } = {}) {
  if (!text) return;
  if (!get(voiceEnabled) || !isSupported()) { logDecision('off', text, priority); return; }

  const muted = get(voiceMutedUntil) > Date.now();

  if (priority === 'urgent') {
    // SOS: flush everything and speak immediately. Bypasses mute + rate limit.
    _queue = [];
    try { window.speechSynthesis.cancel(); } catch { /* ignore */ }
    logDecision('spoken', text, priority);
    _speakRaw(text);
    return;
  }

  if (muted) { logDecision('muted', text, priority); return; }

  if (priority === 'low') {
    if (!get(voiceCheckins)) { logDecision('checkins-off', text, priority); return; }
    if (engineBusy() || _queue.length) { logDecision('busy-drop', text, priority); return; }
  }

  // Dedupe identical strings inside the window
  const last = _recent.get(text) || 0;
  if (Date.now() - last < DEDUPE_MS) { logDecision('dedupe', text, priority); return; }

  if (rateLimited()) {
    // Collapse to the newest normal utterance instead of stacking chatter
    logDecision('rate-limited', text, priority);
    if (priority === 'normal') _queue = [{ text, priority }];
    return;
  }

  _recent.set(text, Date.now());
  if (_recent.size > 40) _recent.delete(_recent.keys().next().value);

  _queue.push({ text, priority });
  if (_queue.length > MAX_QUEUE) _queue = _queue.slice(-MAX_QUEUE);
  drain();
}

/** Silence everything — toggle-off, logout (resetSocketState). */
export function cancelAll() {
  _queue = [];
  if (isSupported()) {
    try { window.speechSynthesis.cancel(); } catch { /* ignore */ }
  }
}
