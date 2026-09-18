<script>
  /**
   * FamilyRoster — the family as warm PEBBLES on quiet paper rows
   * (Stitch "Family Circle"): each person is a round initials badge with a
   * gentle breathing halo when live, followed by ONE plain-language sentence
   * ("Settled where they are." / "Heading to Home — about 8 minutes away.").
   * No cards-with-borders, no metrics grids — rows separated by whitespace.
   *
   * Buckets keep answering what a parent actually asks:
   *   Needs a look — SOS, gone silent, or offline (only rendered when non-empty)
   *   All good     — on the move (with speed) or settled
   *
   * Each row still carries the honest 4-state freshness read (fresh/aging/
   * stale/silent) as BOTH a colored dot and a text age (never color alone —
   * a11y), plus a real distance-from-you. Tap flies the map to that member.
   *
   * DB load: ZERO — pure derive over the in-memory member list.
   */
  import { push } from 'svelte-spa-router';
  import { focusUser } from '../../lib/stores/map.js';
  import { pulseMap } from '../../lib/stores/pulses.js';
  import { arrivalProjections } from '../../lib/stores/arrivals.js';
  import { allowMotion } from '../../lib/stores/effects.js';
  import { getUserColor } from '../../lib/getUserColor.js';
  import { calculateDistance, formatDistance } from '../../lib/tracking.js';
  import { formatAge } from '../../lib/presence.js';
  import { presenceOf, freshness, memberSentence } from '../../lib/hubStatus.js';
  import { rhythmEnabled, getRhythmHint } from '../../lib/presenceRhythm.js';
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

  function statusLabel(r) {
    switch (r.pres) {
      case 'sos': return 'SOS';
      case 'offline': return 'Offline';
      case 'silent': return 'No recent signal';
      case 'moving': return `Moving · ${Math.round((r.m.speed || 0) * 3.6)} km/h`;
      default: return 'Settled';
    }
  }
  // The row's one honest sentence — same rules ladder as the verdict, scoped
  // to a single person, arrival-aware when the projection store knows more.
  function sentenceFor(r, arrivals) {
    const arrival = arrivals?.get?.(r.m.userId) || null;
    return memberSentence(r.m, now, arrival).sentence;
  }
  // Live = actively updating (fresh signal or in motion) → breathing halo.
  function isLive(r) {
    return r.pres === 'moving' || (r.pres === 'settled' && r.fresh === 'fresh');
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
      title="Bring your people in"
      body="Invite someone to share locations, or open the map to start watching over your people."
      ctaLabel="Open the map"
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
    <!-- Tactile pebble — never a pin. Breathing halo only while live. -->
    <span class="pebble-wrap">
      <span class="pebble" class:pebble-live={isLive(r)} class:pebble-sos={r.pres === 'sos'}>
        {#if isLive(r) && $allowMotion}<span class="pebble-halo" aria-hidden="true"></span>{/if}
        {#if r.pres === 'sos' && $allowMotion}<span class="pebble-halo pebble-halo-sos" aria-hidden="true"></span>{/if}
        <span class="pebble-init">{initials(r.m.displayName)}</span>
      </span>
      <span class="presence-dot dot-{r.pres}" aria-hidden="true"></span>
    </span>

    <span class="m-body">
      <span class="m-top">
        <span class="m-name">{r.m.displayName}</span>
        {#if $pulseMap.get(r.m.userId)?.type === 'ok'}
          <span class="m-pulse" title="Checked in — all good">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
            Safe
          </span>
        {:else if r.pres === 'moving'}
          <span class="m-when m-when-live">In motion</span>
        {:else if ageText(r.m)}
          <span class="m-when">{ageText(r.m)}</span>
        {/if}
      </span>
      <span class="m-sentence">{sentenceFor(r, $arrivalProjections)}</span>
      <span class="m-meta">
        <span class="fresh-dot fresh-{r.fresh}" aria-hidden="true"></span>
        <span>{statusLabel(r)}{ageText(r.m) ? ` · ${ageText(r.m)}` : ''}</span>
        {#if dist}<span aria-hidden="true">·</span><span>{dist} away</span>{/if}
      </span>
      {#if rhythm}<span class="m-rhythm">{rhythm}</span>{/if}
    </span>
  </button>
{/snippet}

<style>
  .roster { display: flex; flex-direction: column; gap: var(--space-6); }
  .bucket { display: flex; flex-direction: column; gap: var(--space-2); }
  /* Section labels in the reference's "Present Members" register — quiet
     sans, never a KPI heading. */
  .bucket-head {
    margin: 0; font-size: var(--text-sm); font-weight: 500;
    color: var(--text-secondary); letter-spacing: 0.01em;
  }
  .bucket-head-warn { color: var(--warning-700); }
  /* Rows separated by whitespace, not borders. */
  .bucket-list { display: flex; flex-direction: column; gap: var(--space-4); }

  /* ── Quiet paper row — frameless; a soft tier appears only on hover ────── */
  .member {
    display: flex; align-items: flex-start; gap: var(--space-4);
    min-height: 44px;
    padding: var(--space-2);
    margin: calc(-1 * var(--space-2));
    background: transparent;
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer; text-align: left; color: inherit; font: inherit;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .member:hover { background: var(--surface-hover); }
  .member:active { transform: scale(0.99); }
  .member:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .m-offline { opacity: 0.65; }

  /* ── The pebble — water-smoothed stone, initials inside ─────────────────── */
  .pebble-wrap { position: relative; flex-shrink: 0; margin-top: var(--space-1); }
  .pebble {
    position: relative;
    width: 48px; height: 48px;
    display: flex; align-items: center; justify-content: center;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
  }
  .pebble-live { background: var(--primary-100); }
  .pebble-sos { background: color-mix(in oklch, var(--danger-500) 14%, transparent); }
  .pebble-init {
    position: relative;
    font-size: var(--text-base); font-weight: 600; line-height: 1;
    color: var(--mc, var(--primary-700)); user-select: none;
  }
  .m-sos .pebble-init { color: var(--danger-600); }

  /* Gentle breathing halo — static box-shadow paint, ONLY transform/opacity
     animate (GPU-safe). Rendered only while live and motion is allowed. */
  .pebble-halo {
    position: absolute; inset: 0;
    border-radius: inherit;
    box-shadow: 0 0 0 var(--space-1) color-mix(in oklch, var(--primary-500) 28%, transparent);
    opacity: 0; transform: scale(0.94);
    animation: pebble-breathe 3.6s ease-in-out infinite;
    pointer-events: none;
  }
  .pebble-halo-sos {
    box-shadow: 0 0 0 var(--space-1) color-mix(in oklch, var(--danger-500) 32%, transparent);
    animation-duration: 2s;
  }
  @keyframes pebble-breathe {
    0%, 100% { opacity: 0; transform: scale(0.94); }
    50%      { opacity: 1; transform: scale(1.06); }
  }

  /* Presence dot pinned to the pebble's shoulder — ember in motion, sage
     settled, ochre quiet, vermilion strictly SOS. Always paired with the
     text status below (never color alone). */
  .presence-dot {
    position: absolute; right: 0; bottom: 0;
    width: 14px; height: 14px;
    border-radius: var(--radius-full);
    border: 2px solid var(--paper);
  }
  .dot-moving  { background: var(--primary-500); }
  .dot-settled { background: var(--success-500); }
  .dot-silent  { background: var(--warning-500); }
  .dot-offline { background: var(--status-offline); }
  .dot-sos     { background: var(--danger-500); }

  /* ── Plain-language body ─────────────────────────────────────────────────── */
  .m-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .m-top { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-2); }
  .m-name {
    font-size: var(--text-base); font-weight: 600; color: var(--text-primary);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .m-when { font-size: var(--text-xs); color: var(--text-tertiary); flex-shrink: 0; font-variant-numeric: tabular-nums; }
  .m-when-live { color: var(--primary-700); font-weight: 500; }

  /* The one honest sentence — full body size, reads like a note, not a stat */
  .m-sentence { font-size: var(--text-base); line-height: 1.5; color: var(--text-primary); }
  .m-sos .m-sentence { color: var(--danger-600); font-weight: 600; }

  .m-meta {
    display: flex; align-items: center; gap: var(--space-1);
    margin-top: 2px;
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
  }
  /* Rhythm hint — deliberately muted + neutral, never an alarm */
  .m-rhythm { font-size: var(--text-xs); color: var(--text-tertiary); opacity: 0.8; }
  .m-pulse {
    display: inline-flex; align-items: center; gap: var(--space-1); flex-shrink: 0;
    font-size: var(--text-xs); font-weight: 600; letter-spacing: 0.02em;
    padding: 2px var(--space-2); border-radius: var(--radius-full);
    color: var(--success-600);
    background: color-mix(in oklch, var(--success-500) 10%, transparent);
  }

  /* 4-state honest freshness dot — pairs with the text age (never color alone). */
  .fresh-dot { width: 6px; height: 6px; border-radius: var(--radius-full); flex-shrink: 0; background: var(--text-tertiary); }
  .fresh-fresh  { background: var(--success-500); }
  .fresh-aging  { background: var(--warning-500); }
  .fresh-stale  { background: var(--status-offline); }
  /* silent = needs-a-look ochre (vermilion stays SOS-only); opacity-only pulse */
  .fresh-silent { background: var(--warning-500); animation: fresh-pulse 1.8s ease-in-out infinite; }
  @keyframes fresh-pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.35; } }

  @media (prefers-reduced-motion: reduce) {
    .pebble-halo { animation: none !important; opacity: 0 !important; }
    .fresh-silent { animation: none !important; }
    .member:active { transform: none; }
  }
</style>
