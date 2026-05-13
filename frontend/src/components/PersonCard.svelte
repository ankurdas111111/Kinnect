<script>
  import { onDestroy } from 'svelte';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';
  import FreshnessChip from './primitives/FreshnessChip.svelte';
  import TiltCard from './primitives/TiltCard.svelte';
  import { focusUser, myLocation } from '../lib/stores/map.js';
  import { sosNarratives } from '../lib/stores/sos.js';
  import { calculateDistance, formatDistance } from '../lib/tracking.js';
  import { computeActivityStatus, formatActivityAge } from '../lib/activityStatus.js';

  export let user = null;
  export let onClose = null;

  $: color      = user ? getUserColor(user.userId) : '#6366f1';
  $: colorLight = user ? getUserColorLight(user.userId) : 'rgba(99,102,241,0.15)';

  $: initials = user
    ? (user.displayName || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) || '?'
    : '?';

  // ── Activity status ──────────────────────────────────────────────────────
  $: activityStatus = computeActivityStatus(user);
  $: activityAge    = formatActivityAge(user?.lastSeen);

  // ── Distance from me ─────────────────────────────────────────────────────
  $: distanceText = (user?.lat != null && user?.lng != null && $myLocation)
    ? formatDistance(calculateDistance($myLocation.latitude, $myLocation.longitude, user.lat, user.lng))
    : null;

  function locateOnMap() {
    if (user?.userId) focusUser.set(user.userId);
    if (onClose) onClose();
  }

  // Emergency card — shown when this user has an active SOS with medical data
  $: emergencyCard = user?.sos?.active ? $sosNarratives.get(user.userId)?.medicalCard || null : null;
  $: emergencyContacts = emergencyCard?.emergencyContacts?.filter(c => c.name || c.phone) ||
    ((emergencyCard?.emergencyName || emergencyCard?.emergencyPhone)
      ? [{ name: emergencyCard.emergencyName, phone: emergencyCard.emergencyPhone, relation: '' }]
      : []);

  function copyCoords() {
    if (!user?.lat || !user?.lng) return;
    const text = `${user.lat.toFixed(6)}, ${user.lng.toFixed(6)}`;
    navigator.clipboard?.writeText(text).catch(() => {});
  }

  // Flash stat cells when coordinates update — timer cleaned up on destroy
  let coordFlash = false;
  let _prevLat = null;
  let _prevLng = null;
  let _coordFlashTimer = null;

  $: if (user?.lat !== _prevLat || user?.lng !== _prevLng) {
    if (_prevLat !== null) {
      coordFlash = true;
      if (_coordFlashTimer) clearTimeout(_coordFlashTimer);
      _coordFlashTimer = setTimeout(() => {
        coordFlash = false;
        _coordFlashTimer = null;
      }, 850);
    }
    _prevLat = user?.lat ?? null;
    _prevLng = user?.lng ?? null;
  }

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
      <div class="sos-banner" role="alert">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        SOS Active — {user.sos.reason || 'Emergency'}
      </div>
    {/if}

    <!-- Emergency medical card — shown inline when this user has active SOS + medical data -->
    {#if emergencyCard}
      <div class="ec-card" role="region" aria-label="Emergency medical info for {user.displayName || 'this user'}">

        <!-- Blood type + key pills row -->
        <div class="ec-top">
          {#if emergencyCard.bloodType}
            <div class="ec-blood">
              <span class="ec-blood-label">Blood</span>
              <span class="ec-blood-val">{emergencyCard.bloodType}</span>
            </div>
          {/if}
          {#if emergencyCard.allergies?.trim()}
            <span class="ec-pill ec-pill-alert" title="Allergies: {emergencyCard.allergies}">
              ⚠ {emergencyCard.allergies.length > 28 ? emergencyCard.allergies.slice(0, 28) + '…' : emergencyCard.allergies}
            </span>
          {/if}
          {#if emergencyCard.conditions?.trim()}
            <span class="ec-pill" title="Conditions: {emergencyCard.conditions}">
              {emergencyCard.conditions.length > 28 ? emergencyCard.conditions.slice(0, 28) + '…' : emergencyCard.conditions}
            </span>
          {/if}
        </div>

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
                  <a class="ec-call" href="tel:{c.phone}" aria-label="Call {c.name || 'emergency contact'}" on:click|stopPropagation>
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
      <div
        class="avatar"
        class:avatar-sos={user.sos?.active}
        class:avatar-live={user.online && !user.sos?.active}
        class:avatar-offline={!user.online}
        style="background:{colorLight};border-color:{color}"
        aria-hidden="true"
      >
        <span class="avatar-initials" style="color:{color}">{initials}</span>
      </div>

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
        <button class="close-btn btn btn-icon btn-ghost" on:click={onClose} aria-label="Close">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      {/if}
    </div>

    <!-- Stat grid -->
    {#if user.lat && user.lng}
      <div class="stat-grid">
        <div class="stat" class:animate-coord-flash={coordFlash}>
          <span class="stat-label">Latitude</span>
          <span class="stat-value">{user.lat.toFixed(5)}°</span>
        </div>
        <div class="stat" class:animate-coord-flash={coordFlash}>
          <span class="stat-label">Longitude</span>
          <span class="stat-value">{user.lng.toFixed(5)}°</span>
        </div>

        {#if user.accuracy != null}
          <div class="stat">
            <span class="stat-label">Accuracy</span>
            <span class="stat-value">±{Math.round(user.accuracy)}m</span>
          </div>
        {/if}

        {#if distanceText}
          <div class="stat">
            <span class="stat-label">Distance</span>
            <span class="stat-value">{distanceText}</span>
          </div>
        {/if}

        {#if user.speed != null && user.speed > 0.5}
          <div class="stat">
            <span class="stat-label">Speed</span>
            <span class="stat-value">{(user.speed * 3.6).toFixed(1)} km/h</span>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Actions -->
    <div class="card-actions">
      <button class="btn btn-primary btn-sm action-btn" on:click={locateOnMap} disabled={!user.lat}>
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
        Locate
      </button>
      <button class="btn btn-secondary btn-sm action-btn" on:click={copyCoords} disabled={!user.lat}>
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        Copy
      </button>
    </div>
  </div>
</TiltCard>
{/if}

<style>
  .person-card {
    background: var(--glass-3d, rgba(255,255,255,0.65));
    backdrop-filter: var(--glass-3d-blur, blur(24px) saturate(2.0));
    -webkit-backdrop-filter: var(--glass-3d-blur, blur(24px) saturate(2.0));
    border: 1px solid var(--user-color-light, var(--glass-3d-border));
    border-top-color: rgba(255, 255, 255, 0.25);
    /* 3D layered depth with user color glow */
    box-shadow:
      0 8px 32px rgba(0,0,0,0.14),
      0 2px 8px rgba(0,0,0,0.08),
      0 12px 40px var(--user-color-light, transparent),
      inset 0 1px 0 rgba(255,255,255,0.20),
      inset 0 -1px 0 rgba(0,0,0,0.05);
    border-radius: var(--radius-xl, 20px);
    overflow: hidden;
    isolation: isolate;
    transform-style: preserve-3d;
    transition:
      box-shadow var(--duration-3d, 250ms) var(--ease-out),
      border-color var(--duration-normal) var(--ease-out),
      transform var(--duration-3d, 250ms) var(--ease-spring);
  }

  /* ── Inline emergency card ─────────────────────────────────────────────── */
  .ec-card {
    background: rgba(239, 68, 68, 0.06);
    border-bottom: 1px solid rgba(239, 68, 68, 0.14);
  }
  .ec-top {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    padding: 8px 12px;
  }
  .ec-blood {
    display: flex;
    flex-direction: column;
    align-items: center;
    background: rgba(239, 68, 68, 0.14);
    border: 1px solid rgba(239, 68, 68, 0.30);
    border-radius: 8px;
    padding: 3px 8px;
    flex-shrink: 0;
  }
  .ec-blood-label {
    font-size: 8px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--danger-500, #ef4444);
    line-height: 1;
  }
  .ec-blood-val {
    font-size: 16px;
    font-weight: 900;
    color: var(--danger-600, #dc2626);
    letter-spacing: -0.03em;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
  }
  .ec-pill {
    display: inline-flex;
    align-items: center;
    padding: 3px 8px;
    border-radius: 6px;
    font-size: 10px;
    font-weight: 600;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.10);
    color: var(--text-secondary);
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ec-pill-alert {
    background: rgba(239, 68, 68, 0.10);
    border-color: rgba(239, 68, 68, 0.25);
    color: var(--danger-600, #dc2626);
  }
  .ec-contacts {
    display: flex;
    flex-direction: column;
    gap: 1px;
    border-top: 1px solid rgba(239, 68, 68, 0.10);
  }
  .ec-contact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 6px 12px;
    background: rgba(255, 255, 255, 0.02);
  }
  .ec-contact-info {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .ec-contact-name {
    font-size: 11px;
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ec-contact-rel {
    font-size: 9px;
    font-weight: 500;
    color: var(--text-tertiary);
  }
  .ec-call {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 10px 12px;
    min-height: 44px;
    border-radius: 7px;
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.28);
    color: var(--success-600, #059669);
    font-size: 10px;
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .ec-call:hover { background: rgba(16, 185, 129, 0.22); }

  /* ── SOS banner ─────────────────────────────────────────────────────────── */
  .sos-banner {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    background: rgba(239, 68, 68, 0.12);
    border-bottom: 1px solid rgba(239, 68, 68, 0.20);
    color: var(--danger-600, #dc2626);
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    animation: sos-breathe 2s ease-in-out infinite;
  }
  @keyframes sos-breathe {
    0%, 100% { background: rgba(239, 68, 68, 0.12); }
    50%       { background: rgba(239, 68, 68, 0.22); }
  }

  /* ── Card header ─────────────────────────────────────────────────────────── */
  .card-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-3-5) var(--space-3-5) var(--space-2-5);
  }

  .avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    border: 2.5px solid var(--user-color);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    position: relative;
    /* 3D sphere avatar */
    box-shadow:
      0 4px 12px rgba(0, 0, 0, 0.15),
      inset 0 2px 4px rgba(255, 255, 255, 0.15),
      inset 0 -2px 4px rgba(0, 0, 0, 0.10);
    transition:
      box-shadow var(--duration-normal) var(--ease-out),
      transform var(--duration-normal) var(--ease-spring);
  }
  .avatar:hover {
    transform: perspective(400px) translateZ(4px) scale(1.05);
  }

  /* SOS: expanding shadow ring — doesn't cover initials */
  .avatar-sos {
    animation: sos-shadow-pulse 1.1s ease-out infinite;
  }
  @keyframes sos-shadow-pulse {
    0%   { box-shadow: 0 4px 12px rgba(0,0,0,0.15), inset 0 2px 4px rgba(255,255,255,0.15), 0 0 0 0 rgba(239,68,68,0.55); }
    70%  { box-shadow: 0 4px 12px rgba(0,0,0,0.15), inset 0 2px 4px rgba(255,255,255,0.15), 0 0 0 14px rgba(239,68,68,0); }
    100% { box-shadow: 0 4px 12px rgba(0,0,0,0.15), inset 0 2px 4px rgba(255,255,255,0.15), 0 0 0 0 rgba(239,68,68,0); }
  }

  /* Online: slow breathing glow — Heartbeat Halo */
  .avatar-live {
    animation: heartbeat-halo var(--pulse-duration, 1.5s) ease-in-out infinite;
  }
  @keyframes heartbeat-halo {
    0%, 100% { box-shadow: 0 4px 12px rgba(0,0,0,0.15), inset 0 2px 4px rgba(255,255,255,0.15), 0 0 0 rgba(16,185,129,0); }
    50%       { box-shadow: 0 4px 12px rgba(0,0,0,0.15), inset 0 2px 4px rgba(255,255,255,0.15), 0 0 14px rgba(16,185,129,0.45); }
  }

  .avatar-offline {
    opacity: 0.5;
    filter: grayscale(0.6);
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
    gap: 3px;
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
    gap: 5px;
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
  }

  /* ── F4: Location label badge ──────────────────────────────────────────── */
  .location-label-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 10px;
    font-weight: 600;
    color: var(--primary-500, #14b8a6);
    background: rgba(20, 184, 166, 0.10);
    border: 1px solid rgba(20, 184, 166, 0.22);
    border-radius: var(--radius-full, 9999px);
    padding: 2px 7px;
    line-height: 1.3;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Close button — actual 44x44px touch target (no pseudo-element workaround) */
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

  /* ── Stat grid ───────────────────────────────────────────────────────────── */
  .stat-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1px;
    background: var(--border-subtle, rgba(0,0,0,0.06));
    border-top: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
    border-bottom: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px 14px;
    background: var(--surface-1);
    /* 3D inset stat cells */
    box-shadow:
      inset 0 1px 3px rgba(0, 0, 0, 0.04),
      0 1px 0 rgba(255, 255, 255, 0.08);
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
    font-weight: 600;
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
</style>
