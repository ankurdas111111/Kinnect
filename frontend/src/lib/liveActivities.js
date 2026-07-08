/**
 * iOS Live Activities (Lock Screen + Dynamic Island) for live-share and SOS.
 *
 * Wraps @capgo/capacitor-live-activities behind a platform guard: every call
 * is a silent no-op on web and Android, so callers never need to branch.
 * Layouts are JSON-defined (no per-change SwiftUI edits) and rendered by the
 * Widget Extension target — see docs/live-activities-setup.md for the
 * one-time Xcode setup.
 *
 * Requires iOS 16.2+ (Live Activities), iOS 17.2+ for push-to-start.
 */
let plugin = null;
let currentActivityId = null;

async function getPlugin() {
  // window.Capacitor is injected by the native shell — same detection pattern
  // as haptics.js; avoids importing @capacitor/core (stubbed on web builds).
  if (typeof window === 'undefined' || window.Capacitor?.getPlatform?.() !== 'ios') return null;
  if (plugin) return plugin;
  try {
    const mod = await import('@capgo/capacitor-live-activities');
    plugin = mod.LiveActivities;
  } catch (_) {
    plugin = null;
  }
  return plugin;
}

// Shared layout: title row + status line + big countdown/ETA, matching the
// calm dark aesthetic. Colors are literal because Widget Extensions cannot
// read the web app's CSS tokens; values mirror --bg-* / --brand-* anchors.
function buildLayout({ title, subtitle, accent }) {
  return {
    id: 'kinnect-live',
    layout: {
      type: 'container',
      direction: 'vertical',
      spacing: 6,
      padding: 14,
      backgroundColor: '#0b1220',
      children: [
        {
          type: 'container',
          direction: 'horizontal',
          spacing: 8,
          children: [
            { type: 'text', text: title, fontSize: 15, fontWeight: 'bold', color: '#f1f5f9' },
            { type: 'text', text: '{{status}}', fontSize: 13, color: accent }
          ]
        },
        { type: 'text', text: subtitle, fontSize: 13, color: '#94a3b8' },
        { type: 'text', text: '{{detail}}', fontSize: 13, color: '#cbd5e1' }
      ]
    }
  };
}

/**
 * Start (or replace) the live-share activity. Safe to call repeatedly.
 * kind: 'walk' | 'ride' | 'sos'
 */
export async function startLiveShareActivity(kind, { personName = '', status = '', detail = '' } = {}) {
  const la = await getPlugin();
  if (!la) return;
  const configs = {
    walk: { title: 'Walk With Me', subtitle: personName ? `Walking with ${personName}` : 'Sharing live location', accent: '#2dd4bf' },
    ride: { title: 'Ride Share', subtitle: personName ? `${personName}'s ride` : 'Ride in progress', accent: '#818cf8' },
    sos: { title: 'SOS Active', subtitle: personName ? `${personName} needs help` : 'Emergency alert active', accent: '#f87171' }
  };
  const cfg = configs[kind] || configs.walk;
  try {
    await endLiveShareActivity(); // one Kinnect activity at a time
    const result = await la.startActivity({
      ...buildLayout(cfg),
      data: { status, detail },
      // SOS must stay visible; normal shares can be dismissed by the system sooner
      staleDate: kind === 'sos' ? undefined : Date.now() + 60 * 60 * 1000
    });
    currentActivityId = result?.activityId ?? null;
  } catch (_) {
    currentActivityId = null;
  }
}

/** Update the running activity's dynamic fields ({{status}}, {{detail}}). */
export async function updateLiveShareActivity({ status = '', detail = '' } = {}) {
  const la = await getPlugin();
  if (!la || !currentActivityId) return;
  try {
    await la.updateActivity({ activityId: currentActivityId, data: { status, detail } });
  } catch (_) {}
}

/** End the running activity (arrival, cancel, SOS resolved). */
export async function endLiveShareActivity() {
  const la = await getPlugin();
  if (!la || !currentActivityId) return;
  const id = currentActivityId;
  currentActivityId = null;
  try {
    await la.endActivity({ activityId: id });
  } catch (_) {}
}
