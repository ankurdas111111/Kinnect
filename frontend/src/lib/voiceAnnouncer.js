/**
 * voiceAnnouncer.js — policy layer for spoken updates.
 *
 * Decides WHAT gets spoken and WHEN; lib/voice.js decides HOW. Initialized
 * once from setupSocketHandlers (socket.js) after the activity recorder.
 *
 * ── aria-live double-announcement analysis ─────────────────────────────────
 * 26 aria-live regions exist (banner, toast, verdict line, …). A VoiceOver /
 * TalkBack user who enables spoken updates would hear BOTH the screen reader
 * and the synth voice. There is no reliable screen-reader detection on the
 * web (by design, for privacy), so the mitigations are:
 *   1. Default OFF — SR users are unaffected unless they opt in.
 *   2. The Settings toggle copy explicitly advises SR users to leave it off.
 *   3. TTS speaks the same semantic content the aria-live strings carry
 *      (speech-friendly expansions only), so either channel is complete.
 *   4. TTS never mutates the DOM — it cannot re-trigger aria-live regions.
 * ───────────────────────────────────────────────────────────────────────────
 */
import { get } from 'svelte/store';
import { speak, cancelAll } from './voice.js';
import { alertState } from './stores/sos.js';
import { arrivalProjections } from './stores/arrivals.js';
import { familyVerdict } from './stores/verdict.js';
import { pushActivity } from './activityLog.js';
import { haptics } from './haptics.js';

export { cancelAll as cancelVoice };

// ── Watcher tuning ───────────────────────────────────────────────────────────
const ARRIVAL_ETA_S = 180;        // leaving projections with eta ≤3min = arrived
const ARRIVAL_DIST_M = 250;       // …or within 250m of the place
const ARRIVAL_DEDUPE_MS = 3 * 60_000; // suppress double announce (watcher vs placeAlert)
const VERDICT_HYSTERESIS_MS = 20_000; // tone must persist before speaking
const VERDICT_COOLDOWN_MS = 5 * 60_000; // per-direction re-announce cooldown

/** Speech-friendly age: "12 minutes", never "12m" (TTS reads that as meters). */
export function speechAge(ms) {
  if (ms == null || ms < 0) return 'a while';
  const min = Math.floor(ms / 60_000);
  if (min < 1) return 'moments';
  if (min === 1) return '1 minute';
  if (min < 60) return `${min} minutes`;
  const h = Math.floor(min / 60);
  return h === 1 ? 'an hour' : `${h} hours`;
}

/**
 * SOS start. Urgent priority. If AlertOverlay's alarm is sounding in the
 * foreground (7–10s sawtooth), delay the utterance until just after it —
 * speech over the alarm is unintelligible. Background/driving (no visible
 * overlay) speaks immediately — exactly when voice matters most.
 */
export function announceSosStart(name, reason, isMe, isGeofence) {
  let text;
  if (isMe) text = 'Your SOS is on. Your family has been alerted.';
  else if (isGeofence) text = `Heads up. ${name} left their safe zone.`;
  else {
    text = `SOS. ${name} needs help.`;
    if (reason && !/^sos$/i.test(reason.trim()) && !/emergency/i.test(reason.trim())) {
      text += ` They said: ${reason}.`;
    }
  }
  const a = get(alertState);
  const delay = a?.visible && a?.alarmMs > 0 ? a.alarmMs + 300 : 0;
  const fire = () => {
    speak(text, { priority: 'urgent' });
    // AlertOverlay already buzzes in the foreground — only add haptics when hidden
    if (typeof document !== 'undefined' && document.hidden) haptics.sos?.();
  };
  if (delay) setTimeout(fire, delay);
  else fire();
}

/** SOS cleared. Normal priority — reassurance, not an alarm. */
export function announceSosEnd(name, isMe) {
  speak(isMe ? 'Your SOS is off.' : `${name} cancelled the SOS. Everyone's okay.`, { priority: 'normal' });
}

/** "I'm safe" pulse — low priority chatter, gated by the check-ins sub-toggle. */
export function announceCheckin(name) {
  speak(`${name} checked in. All safe.`, { priority: 'low' });
}

/** Arrival at a saved place. Normal priority. */
export function announceArrival(name, placeName) {
  const isHome = /home/i.test(placeName || '');
  speak(isHome ? `${name} just got home.` : `${name} just arrived at ${placeName || 'a saved place'}.`, { priority: 'normal' });
}

// ═══ Watchers — initialized once from setupSocketHandlers ════════════════════

let _initialized = false;

/**
 * Global arrival + verdict watchers.
 *
 * Arrivals: HomecomingRail only detects arrivals while the Hub page is
 * mounted — a real gap (the Activity feed misses them too). This watcher runs
 * app-wide: a user leaving arrivalProjections with a small last ETA/distance
 * = arrived. It ALWAYS records to the activity log (voice off included);
 * speech itself is gated inside voice.js. The socket 'placeAlert' arrive
 * event is deduped against it via recentArrivals.
 *
 * Verdict: speaks ONLY safe→caution and caution→safe. Tone 'alert' is
 * exclusively SOS-driven (hubStatus.js rules ladder), and the SOS announcer
 * owns those moments — no double-speak by construction. 20s hysteresis
 * (silent-member flapping at the staleness boundary is common) + 5-minute
 * per-direction cooldown.
 */
export function initVoiceAnnouncer(socket) {
  if (_initialized) return;
  _initialized = true;

  // ── Arrival watcher ────────────────────────────────────────────────────────
  let prevProjections = new Map(); // userId → {etaSeconds, distanceM, displayName, placeName}
  const recentArrivals = new Map(); // userId → ts

  function markArrival(userId, name, placeName) {
    const last = recentArrivals.get(userId) || 0;
    if (Date.now() - last < ARRIVAL_DEDUPE_MS) return;
    recentArrivals.set(userId, Date.now());
    pushActivity({ type: 'contact', userId, userName: name, message: `${name} arrived${placeName ? ` at ${placeName}` : ''}` });
    announceArrival(name, placeName);
  }

  arrivalProjections.subscribe((m) => {
    for (const [userId, prev] of prevProjections) {
      if (!m.has(userId)) {
        const arrived = (prev.etaSeconds != null && prev.etaSeconds <= ARRIVAL_ETA_S) ||
                        (prev.distanceM != null && prev.distanceM <= ARRIVAL_DIST_M);
        if (arrived) markArrival(userId, prev.displayName || 'Someone', prev.placeName);
        // else: stopped converging — silent by design
      }
    }
    prevProjections = new Map(
      [...m.entries()].map(([id, a]) => [id, {
        etaSeconds: a.etaSeconds, distanceM: a.distanceM,
        displayName: a.displayName, placeName: a.placeName,
      }])
    );
  });

  // Authoritative arrive events from configured place alerts — same dedupe path
  socket?.on?.('placeAlert', (data) => {
    if (data?.type === 'arrive' && data.userId) {
      markArrival(data.userId, data.displayName || 'Someone', data.placeName || data.name);
    }
  });

  // ── Verdict transition watcher (safe↔caution only) ─────────────────────────
  let spokenTone = 'safe';
  let candidateTone = null;
  let candidateSince = 0;
  let lastSpokenAt = { caution: 0, safe: 0 };

  familyVerdict.subscribe((v) => {
    const tone = v?.tone || 'safe';
    if (tone === 'alert') { candidateTone = null; return; } // SOS announcer owns alert
    if (tone === spokenTone) { candidateTone = null; return; }

    if (candidateTone !== tone) {
      candidateTone = tone;
      candidateSince = Date.now();
      return;
    }
    // Same candidate persisting — hysteresis + cooldown gate
    if (Date.now() - candidateSince < VERDICT_HYSTERESIS_MS) return;
    if (Date.now() - (lastSpokenAt[tone] || 0) < VERDICT_COOLDOWN_MS) { spokenTone = tone; candidateTone = null; return; }

    spokenTone = tone;
    candidateTone = null;
    lastSpokenAt[tone] = Date.now();
    if (tone === 'caution') {
      // Build from the live sentence — already plain-language ("Dad hasn't
      // updated in a while"); the detail carries the age in screen shorthand,
      // so speak the sentence only.
      speak(`Heads up. ${v.sentence}.`, { priority: 'normal' });
    } else {
      speak("All good again. Everyone's settled.", { priority: 'normal' });
    }
  });
}
