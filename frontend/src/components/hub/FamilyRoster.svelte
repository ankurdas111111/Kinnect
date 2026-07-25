<script>
  /**
   * FamilyRoster — the family, bucketed by what a parent actually asks:
   * "who do I still need to look at?" vs "who's fine?".
   *
   *   Needs a look — SOS, gone silent, or offline (only rendered when non-empty)
   *   All good     — on the move (with speed) or settled
   *
   * Each row carries an honest 4-state freshness read (fresh/aging/stale/silent)
   * shown as BOTH a colored dot and a text age (never color alone — a11y), plus
   * a real distance-from-you (fixed: reads latitude/longitude, not the broken
   * lat/lng the old Hub used). Tap flies the map to that member.
   *
   * DB load: ZERO — pure derive over the in-memory member list.
   */
  import { push } from 'svelte-spa-router';
  import { focusUser } from '../../lib/stores/map.js';
  import { pulseMap } from '../../lib/stores/pulses.js';
  import { getUserColor } from '../../lib/getUserColor.js';
  import { calculateDistance, formatDistance } from '../../lib/tracking.js';
  import { formatAge } from '../../lib/presence.js';
  import { presenceOf, freshness } from '../../lib/hubStatus.js';
  import { rhythmEnabled, getRhythmHint } from '../../lib/presenceRhythm.js';
  import AvatarRing from '../primitives/AvatarRing.svelte';
  import GhostConstellation from '../primitives/GhostConstellation.svelte';

  /** @type {{ members: Array<any>, myLocation: {latitude:number,longitude:number}|null, now: number }} */
  let { members = [], myLocation = null, now = Date.now() } = $props();

  const PRIORITY = { sos: 0, silent: 1, offline: 2, moving: 3, settled: 4 };

  let rows = $derived(
    members
      .map((m) => {
        const pres = presenceOf(m, now);
        return { m, pres, fresh: freshness(m.lastUpdate, now) };
      })
      .sort((a, b) => (PRIORITY[a.pres] ?? 9) - (PRIORITY[b.pres] ?? 9))
  );

  let needsLook = $derived(rows.filter((r) => r.pres === 'sos' || r.pres === 'silent' || r.pres === 'offline'));
  let allGood = $derived(rows.filter((r) => r.pres === 'moving' || r.pres === 'settled'));

  function ringFor(pres) {
    if (pres === 'sos') return 'sos';
    if (pres === 'offline') return 'offline';
    return 'live';
  }
  function statusLabel(r) {
    switch (r.pres) {
      case 'sos': return 'SOS';
      case 'offline': return 'Offline';
      case 'silent': return 'No recent signal';
      case 'moving': return `Moving · ${Math.round((r.m.speed || 0) * 3.6)} km/h`;
      default: return 'Settled';
    }
  }
  function distText(m) {
    if (m.lat == null || m.lng == null || !myLocation?.latitude) return null;
    return formatDistance(calculateDistance(myLocation.latitude, myLocation.longitude, m.lat, m.lng));
  }
  function ageText(m) {
    return m.lastUpdate ? formatAge(now - m.lastUpdate) : '';
  }
  function initials(n) {
    return (n || '?').split(' ').map((s) => s[0]).join('').toUpperCase().slice(0, 2) || '?';
  }
  function open(userId) { focusUser.set(userId); push('/'); }
</script>

<section class="roster" aria-label="Your family">
  {#if members.length === 0}
    <GhostConstellation
      title="Add your first family member"
      body="Invite someone to share locations, or open the map to start watching over your people."
      ctaLabel="Open Map"
      memberCount={0}
      oninvite={() => push('/')}
    />
  {:else}
    {#if needsLook.length}
      <div class="bucket">
        <h3 class="bucket-head bucket-head-warn">Needs a look</h3>
        <div class="bucket-list">
          {#each needsLook as r (r.m.socketId)}
            {@render row(r)}
          {/each}
        </div>
      </div>
    {/if}

    {#if allGood.length}
      <div class="bucket">
        <h3 class="bucket-head">All good</h3>
        <div class="bucket-list">
          {#each allGood as r (r.m.socketId)}
            {@render row(r)}
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</section>

{#snippet row(r)}
  {@const color = getUserColor(r.m.userId)}
  {@const dist = distText(r.m)}
  {@const rhythm = $rhythmEnabled && (r.pres === 'offline' || r.pres === 'silent') ? getRhythmHint(r.m.userId, now) : null}
  <button class="member m-{r.pres}" style="--mc:{color}" onclick={() => open(r.m.userId)}
    aria-label="{r.m.displayName} — {statusLabel(r)}{dist ? `, ${dist} away` : ''}. View on map">
    <AvatarRing ring={ringFor(r.pres)} size={40}>
      <span class="m-init">{initials(r.m.displayName)}</span>
    </AvatarRing>
    <div class="m-info">
      <span class="m-name">{r.m.displayName}</span>
      <span class="m-status">
        <span class="fresh-dot fresh-{r.fresh}" aria-hidden="true"></span>
        {statusLabel(r)}{ageText(r.m) ? ` · ${ageText(r.m)}` : ''}
      </span>
      {#if rhythm}<span class="m-rhythm">{rhythm}</span>{/if}
    </div>
    {#if $pulseMap.get(r.m.userId)?.type === 'ok'}
      <span class="m-pulse" title="Checked in — all good">
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
        Safe
      </span>
    {:else if dist}<span class="m-dist">{dist}</span>{/if}
  </button>
{/snippet}

<style>
  .roster { display: flex; flex-direction: column; gap: var(--space-4); }
  .bucket { display: flex; flex-direction: column; gap: var(--space-2); }
  .bucket-head {
    margin: 0; font-size: var(--text-2xs, 10px); font-weight: 700;
    text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-tertiary);
  }
  .bucket-head-warn { color: var(--warning-400); }
  .bucket-list { display: flex; flex-direction: column; gap: var(--space-2); }

  .member {
    display: flex; align-items: center; gap: var(--space-3);
    min-height: 44px; padding: var(--space-2) var(--space-3);
    background: var(--glass-bg);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md, 12px);
    cursor: pointer; text-align: left; color: inherit; font: inherit;
    transition: border-color var(--duration-fast, 150ms) var(--ease-out, ease),
                background var(--duration-fast, 150ms) var(--ease-out, ease),
                transform var(--duration-fast, 100ms) var(--ease-out, ease);
    -webkit-tap-highlight-color: transparent;
  }
  .member:hover { border-color: var(--glass-border-strong); background: var(--surface-hover); }
  .member:active { transform: scale(0.98); }
  .member:focus-visible { outline: 2px solid var(--mc, var(--primary-400)); outline-offset: 2px; }
  .m-sos { background: var(--danger-500-12); border-color: var(--danger-500-20); }
  .m-offline { opacity: 0.6; }

  .m-init { font-size: 13px; font-weight: 800; color: var(--mc, var(--primary-400)); line-height: 1; user-select: none; }
  .m-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .m-name { font-size: var(--text-sm, 14px); font-weight: 700; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .m-status { display: flex; align-items: center; gap: 6px; font-size: 11px; color: var(--text-tertiary); }
  /* Rhythm hint — deliberately muted + neutral, never an alarm */
  .m-rhythm { font-size: 10px; color: var(--text-tertiary); opacity: 0.75; font-style: italic; }
  .m-dist { font-size: 11px; color: var(--text-tertiary); font-variant-numeric: tabular-nums; flex-shrink: 0; }
  .m-pulse {
    display: inline-flex; align-items: center; gap: 3px; flex-shrink: 0;
    font-size: 10px; font-weight: 700; letter-spacing: 0.02em;
    padding: 3px 8px; border-radius: var(--radius-full, 999px);
    color: var(--success-300);
    background: var(--success-500-12);
    border: 1px solid var(--success-500-20);
  }
  .m-sos .m-status { color: var(--danger-400); font-weight: 700; }

  /* 4-state honest freshness dot — pairs with the text age (never color alone). */
  .fresh-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; background: var(--text-tertiary); }
  .fresh-fresh  { background: var(--success-500); box-shadow: 0 0 5px var(--success-500-30); }
  .fresh-aging  { background: var(--warning-500); }
  .fresh-stale  { background: var(--text-tertiary); }
  .fresh-silent { background: var(--danger-500); animation: fresh-pulse 1.8s ease-in-out infinite; }
  @keyframes fresh-pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.35; } }

  @media (prefers-reduced-motion: reduce) {
    .fresh-silent { animation: none !important; }
    .member:active { transform: none; }
  }
</style>
