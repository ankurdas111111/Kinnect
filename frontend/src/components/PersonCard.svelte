<script>
  import { run, createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import { onDestroy } from 'svelte';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';
  import WalkieTalkieButton from './WalkieTalkieButton.svelte';
  import { callState, callPeer } from '../lib/stores/webrtc.js';
  import FreshnessChip from './primitives/FreshnessChip.svelte';
  import TiltCard from './primitives/TiltCard.svelte';
  import AvatarRing from './primitives/AvatarRing.svelte';
  import { focusUser, myLocation } from '../lib/stores/map.js';
  import { sosNarratives } from '../lib/stores/sos.js';
  import { calculateDistance, formatDistance } from '../lib/tracking.js';
  import { computeActivityStatus, formatActivityAge } from '../lib/activityStatus.js';
  import { allowMotion } from '../lib/stores/effects.js';

  /**
   * @typedef {Object} Props
   * @property {any} [user]
   * @property {any} [onClose]
   */

  /** @type {Props} */
  let { user = null, onClose = null } = $props();

  let color      = $derived(user ? getUserColor(user.userId) : '#6366f1');
  let colorLight = $derived(user ? getUserColorLight(user.userId) : 'rgba(99,102,241,0.15)');

  let initials = $derived(user
    ? (user.displayName || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) || '?'
    : '?');

  // ── Presence ring state (delegated to AvatarRing primitive) ─────────────
  /** @type {'sos' | 'live' | 'offline' | 'none'} */
  let ringState = $derived(
    user?.sos?.active ? 'sos' :
    user?.online      ? 'live' :
                        'offline'
  );

  // ── Activity status ──────────────────────────────────────────────────────
  let activityStatus = $derived(computeActivityStatus(user));
  let activityAge    = $derived(formatActivityAge(user?.lastSeen));

  // ── Distance from me ─────────────────────────────────────────────────────
  let distanceText = $derived((user?.lat != null && user?.lng != null && $myLocation)
    ? formatDistance(calculateDistance($myLocation.latitude, $myLocation.longitude, user.lat, user.lng))
    : null);

  function locateOnMap() {
    if (user?.userId) focusUser.set(user.userId);
    if (onClose) onClose();
  }

  // Emergency card — shown when this user has an active SOS with medical data
  let emergencyCard = $derived(user?.sos?.active ? $sosNarratives.get(user.userId)?.medicalCard || null : null);
  let emergencyContacts = $derived(emergencyCard?.emergencyContacts?.filter(c => c.name || c.phone) ||
    ((emergencyCard?.emergencyName || emergencyCard?.emergencyPhone)
      ? [{ name: emergencyCard.emergencyName, phone: emergencyCard.emergencyPhone, relation: '' }]
      : []));

  // Split free-text allergies / conditions into individual chips (display-only,
  // no logic change — pure transform of the same source strings).
  let allergyChips   = $derived((emergencyCard?.allergies || '').split(/[,;\n]/).map(s => s.trim()).filter(Boolean));
  let conditionChips = $derived((emergencyCard?.conditions || '').split(/[,;\n]/).map(s => s.trim()).filter(Boolean));

  function copyCoords() {
    if (!user?.lat || !user?.lng) return;
    const text = `${user.lat.toFixed(6)}, ${user.lng.toFixed(6)}`;
    navigator.clipboard?.writeText(text).catch(() => {});
  }

  // Flash stat cells when coordinates update — data-first: text commits before
  // the flash class toggles (never delay live data for a decorative effect).
  // Timer cleaned up on destroy.
  let coordFlash = $state(false);
  let _prevLat = $state(null);
  let _prevLng = $state(null);
  let _coordFlashTimer = $state(null);

  run(() => {
    if (user?.lat !== _prevLat || user?.lng !== _prevLng) {
      // 1. Commit new coordinate references FIRST (data-first ordering).
      _prevLat = user?.lat ?? null;
      _prevLng = user?.lng ?? null;
      // 2. THEN gate the decorative flash behind allowMotion.
      if (_prevLat !== null && $allowMotion) {
        coordFlash = true;
        if (_coordFlashTimer) clearTimeout(_coordFlashTimer);
        _coordFlashTimer = setTimeout(() => {
          coordFlash = false;
          _coordFlashTimer = null;
        }, 850);
      }
    }
  });

  let pttActive =
    $derived(($callState === 'calling' || $callState === 'connected') &&
    $callPeer?.userID === user?.userId);

  onDestroy(() => {
    if (_coordFlashTimer) clearTimeout(_coordFlashTimer);
  });
</script>

{#if user}
<TiltCard intensity={10} shine={true}>
  <div
    class="person-card"
    style:--user-color={color}
    style:--user-color-light={colorLight}
  >
    <!-- SOS banner -->
    {#if user.sos?.active}
      <div class="sos-banner fx-ambient" role="alert">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        SOS Active — {user.sos.reason || 'Emergency'}
      </div>
    {/if}

    <!-- Emergency medical card — shown inline when this user has active SOS + medical data.
         Reads as a danger-glow region: MASSIVE blood type + colored risk chips. -->
    {#if emergencyCard}
      <div class="ec-card" role="region" aria-label="Emergency medical info for {user.displayName || 'this user'}">

        <!-- MASSIVE blood type — instantly scannable in an emergency -->
        {#if emergencyCard.bloodType}
          <div class="ec-blood" aria-label="Blood type {emergencyCard.bloodType}">
            <span class="ec-blood-label">Blood Type</span>
            <span class="ec-blood-val">{emergencyCard.bloodType}</span>
          </div>
        {/if}

        <!-- Allergies (danger) & conditions (warning) as full, wrapping chips -->
        {#if allergyChips.length || conditionChips.length}
          <div class="ec-chips">
            {#each allergyChips as a}
              <span class="ec-chip ec-chip--danger">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                {a}
              </span>
            {/each}
            {#each conditionChips as c}
              <span class="ec-chip ec-chip--warning">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
                {c}
              </span>
            {/each}
          </div>
        {/if}

        <!-- Emergency contacts with call buttons -->
        {#if emergencyContacts.length}
          <div class="ec-contacts">
            {#each emergencyContacts as c, i}
              <div class="ec-contact">
                <div class="ec-contact-info">
                  <span class="ec-contact-name">{c.name || 'Contact ' + (i + 1)}</span>
                  {#if c.relation}<span class="ec-contact-rel">{c.relation}</span>{/if}
                </div>
                {#if c.phone}
                  <a class="ec-call tactile" href="tel:{c.phone}" aria-label="Call {c.name || 'emergency contact'}" onclick={stopPropagation(bubble('click'))}>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 3.07 9.81a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 2 .92h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L6.09 8.91A16 16 0 0 0 15.1 17.9l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/>
                    </svg>
                    {c.phone}
                  </a>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

      </div>
    {/if}

    <!-- Avatar + name row -->
    <div class="card-header">
      <!-- AvatarRing owns the presence ring (sos/live/offline/none).
           Inner avatar div owns the surface: background, initials. -->
      <AvatarRing
        ring={ringState}
        size={44}
        label="{user.displayName || 'User'}: {ringState === 'sos' ? 'SOS active' : ringState === 'live' ? 'online' : 'offline'}"
      >
        {#snippet children()}
          <div
            class="avatar-inner"
            style="background:{colorLight};border-color:{color}"
          >
            <span class="avatar-initials" style="color:{color}">{initials}</span>
          </div>
        {/snippet}
      </AvatarRing>

      <div class="name-block">
        <span class="display-name">{user.displayName || 'Unknown'}</span>

        <!-- Activity status line -->
        {#if activityStatus}
          <span
            class="activity-status"
            style:color={activityStatus.color}
            aria-label="Activity: {activityStatus.label}{activityAge ? ', ' + activityAge : ''}"
          >
            <span class="activity-dot" style:background={activityStatus.dotColor} aria-hidden="true"></span>
            {activityStatus.label}{activityAge ? ' · ' + activityAge : ''}
          </span>
        {/if}

        <!-- F4: Location label badge — shown when backend infers a named place -->
        {#if user.locationLabel}
          <span class="location-label-badge" aria-label="Currently at {user.locationLabel}">
            <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
            {user.locationLabel}
          </span>
        {/if}

        <FreshnessChip
          lastSeenMs={user.lastSeen}
          accuracy={user.accuracy}
          online={!!user.online}
          sos={!!user.sos?.active}
        />
      </div>

      {#if onClose}
        <button class="close-btn btn btn-icon btn-ghost" onclick={onClose} aria-label="Close">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      {/if}
    </div>

    <!-- Stat grid — bento tiles: icon + tinted background per metric type -->
    {#if user.lat && user.lng}
      <div class="stat-grid">
        <div class="stat stat--coord" class:animate-coord-flash={coordFlash}>
          <span class="stat-icon" aria-hidden="true">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </span>
          <span class="stat-body">
            <span class="stat-label">Latitude</span>
            <span class="stat-value tabular-nums">{user.lat.toFixed(5)}°</span>
          </span>
        </div>
        <div class="stat stat--coord" class:animate-coord-flash={coordFlash}>
          <span class="stat-icon" aria-hidden="true">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="2" x2="12" y2="22"/><path d="M2 12a15.3 15.3 0 0 1 10-4 15.3 15.3 0 0 1 10 4 15.3 15.3 0 0 1-10 4 15.3 15.3 0 0 1-10-4z"/></svg>
          </span>
          <span class="stat-body">
            <span class="stat-label">Longitude</span>
            <span class="stat-value tabular-nums">{user.lng.toFixed(5)}°</span>
          </span>
        </div>

        {#if user.accuracy != null}
          <div class="stat stat--accuracy">
            <span class="stat-icon" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1.5" fill="currentColor"/></svg>
            </span>
            <span class="stat-body">
              <span class="stat-label">Accuracy</span>
              <span class="stat-value tabular-nums">±{Math.round(user.accuracy)}m</span>
            </span>
          </div>
        {/if}

        {#if distanceText}
          <div class="stat stat--distance">
            <span class="stat-icon" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="3 11 22 2 13 21 11 13 3 11"/></svg>
            </span>
            <span class="stat-body">
              <span class="stat-label">Distance</span>
              <span class="stat-value">{distanceText}</span>
            </span>
          </div>
        {/if}

        {#if user.speed != null && user.speed > 0.5}
          <div class="stat stat--speed">
            <span class="stat-icon" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
            </span>
            <span class="stat-body">
              <span class="stat-label">Speed</span>
              <span class="stat-value tabular-nums">{(user.speed * 3.6).toFixed(1)} km/h</span>
            </span>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Actions -->
    {#if pttActive}
      <WalkieTalkieButton {user} />
    {:else}
      <div class="card-actions">
        <button class="btn btn-primary btn-sm action-btn tactile" onclick={locateOnMap} disabled={!user.lat}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
          Locate
        </button>
        <button class="btn btn-secondary btn-sm action-btn tactile" onclick={copyCoords} disabled={!user.lat}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
          Copy
        </button>
        <WalkieTalkieButton {user} />
      </div>
    {/if}
  </div>
</TiltCard>
{/if}

<style>
  .person-card {
    animation: depth-card-arrive 380ms var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
    background: var(--glass-3d);
    backdrop-filter: var(--glass-3d-blur);
    -webkit-backdrop-filter: var(--glass-3d-blur);
    border: 1px solid var(--user-color-light, var(--glass-3d-border));
    border-top-color: var(--glass-3d-border);
    box-shadow:
      var(--shadow-lg),
      0 12px 40px var(--user-color-light, transparent),
      var(--glass-3d-inner);
    border-radius: var(--radius-xl, 20px);
    overflow: hidden;
    isolation: isolate;
    transform-style: preserve-3d;
    transition:
      box-shadow var(--duration-3d, 250ms) var(--ease-out),
      border-color var(--duration-normal) var(--ease-out),
      transform var(--duration-3d, 250ms) var(--ease-spring);
  }

  /* ── Inline emergency card — danger-glow region ────────────────────────── */
  .ec-card {
    background: var(--danger-500-12);
    border-bottom: 1px solid var(--danger-500-20);
    box-shadow: inset 0 0 0 1px var(--danger-500-12), var(--glow-sos);
  }

  /* MASSIVE blood type — centered, high contrast, the first thing you see */
  .ec-blood {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    gap: var(--space-1);
    padding: var(--space-3) var(--space-3) var(--space-2);
  }
  .ec-blood-label {
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.14em;
    color: var(--danger-500);
    line-height: 1;
  }
  .ec-blood-val {
    font-size: var(--text-3xl);
    font-weight: 900;
    color: var(--danger-600);
    letter-spacing: -0.04em;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    text-shadow: 0 1px 2px var(--danger-500-20);
  }

  /* Risk chips — full text, wrapping, colored by severity */
  .ec-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1-5);
    padding: 0 var(--space-3) var(--space-2-5);
    justify-content: center;
  }
  .ec-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-2xs);
    font-weight: 700;
    line-height: 1.25;
  }
  .ec-chip--danger {
    background: var(--danger-500-12);
    border: 1px solid var(--danger-500-20);
    color: var(--danger-600);
  }
  .ec-chip--warning {
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    border: 1px solid color-mix(in oklch, var(--warning-500) 26%, transparent);
    color: var(--warning-600);
  }

  .ec-contacts {
    display: flex;
    flex-direction: column;
    gap: 1px;
    border-top: 1px solid var(--danger-500-12);
  }
  .ec-contact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-1-5) var(--space-3);
    background: var(--surface-inset);
  }
  .ec-contact-info {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .ec-contact-name {
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ec-contact-rel {
    font-size: var(--text-2xs);
    font-weight: 500;
    color: var(--text-tertiary);
  }
  .ec-call {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-2-5) var(--space-3);
    min-height: 44px;
    border-radius: var(--radius-md);
    background: var(--success-500-12);
    border: 1px solid var(--success-500-20);
    color: var(--success-600);
    font-size: var(--text-2xs);
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .ec-call:hover { background: var(--success-500-20); }

  /* ── SOS banner — breathe is decorative (fx-ambient gates it) ───────────
     Static background is the fallback when motion is disabled. */
  .sos-banner {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-2) var(--space-3-5);
    background: var(--danger-500-12);
    border-bottom: 1px solid var(--danger-500-20);
    color: var(--danger-600);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    animation: sos-breathe 2s ease-in-out infinite;
  }
  @keyframes sos-breathe {
    0%, 100% { background: var(--danger-500-12); }
    50%       { background: var(--danger-500-20); }
  }

  /* ── Card header ─────────────────────────────────────────────────────────── */
  .card-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-3-5) var(--space-3-5) var(--space-2-5);
  }

  /* Inner avatar surface — AvatarRing owns the ring, this div owns the fill. */
  .avatar-inner {
    width: 100%;
    height: 100%;
    border-radius: 50%;
    border: 2.5px solid var(--user-color);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow:
      inset 0 2px 4px color-mix(in oklch, white 15%, transparent),
      inset 0 -2px 4px color-mix(in oklch, black 10%, transparent);
    transition:
      box-shadow var(--duration-normal) var(--ease-out),
      transform var(--duration-normal) var(--ease-spring);
  }

  .avatar-initials {
    font-weight: 700;
    font-size: 15px;
    line-height: 1;
    user-select: none;
  }

  .name-block {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    min-width: 0;
  }

  .display-name {
    font-weight: 700;
    font-size: var(--text-base, 15px);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
  }

  /* ── Activity status line ───────────────────────────────────────────────── */
  .activity-status {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.01em;
    line-height: 1;
    opacity: 0.9;
  }

  .activity-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
    box-shadow: 0 0 4px currentColor;
  }

  /* ── F4: Location label badge — primary token tint, no raw hex ──────────── */
  .location-label-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    font-size: 10px;
    font-weight: 600;
    color: var(--primary-500);
    background: var(--primary-500-08);
    border: 1px solid var(--primary-500-12);
    border-radius: var(--radius-full, 9999px);
    padding: 2px 7px;
    line-height: 1.3;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Close button — actual 44x44px touch target */
  .close-btn {
    flex-shrink: 0;
    width: 44px;
    height: 44px;
    min-width: 44px;
    min-height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* ── Stat grid — bento tiles ───────────────────────────────────────────── */
  .stat-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3-5) var(--space-1);
  }

  .stat {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-2-5) var(--space-3);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg, 14px);
    box-shadow:
      inset 0 1px 2px color-mix(in oklch, black 3%, transparent),
      0 1px 0 color-mix(in oklch, white 6%, transparent);
    min-height: 44px;
  }

  .stat-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-md, 10px);
    flex-shrink: 0;
    color: var(--text-tertiary);
  }

  .stat-body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  /* Per-type tints — --primary-500-08/-12 pattern */
  .stat--coord {
    background: var(--primary-500-08);
  }
  .stat--coord .stat-icon {
    background: var(--primary-500-12);
    color: var(--primary-500);
  }
  .stat--accuracy .stat-icon {
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    color: var(--warning-600);
  }
  .stat--distance {
    background: var(--primary-500-08);
  }
  .stat--distance .stat-icon {
    background: var(--primary-500-12);
    color: var(--primary-500);
  }
  .stat--speed {
    background: var(--success-500-12);
  }
  .stat--speed .stat-icon {
    background: var(--success-500-20);
    color: var(--success-600);
  }

  .stat-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  .stat-value {
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  /* ── Actions ─────────────────────────────────────────────────────────────── */
  .card-actions {
    display: flex;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3-5) var(--space-3-5);
  }

  .action-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-1);
    min-height: 44px;
  }

  /* Coord-flash: gated by $allowMotion in JS — animation only runs when the
     class is added, which only happens when motion is allowed. */
  @media (prefers-reduced-motion: reduce) {
    .person-card { animation: none; }
    .sos-banner  { animation: none; }
    .animate-coord-flash { animation: none; }
  }
</style>
