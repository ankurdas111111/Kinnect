<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher, onDestroy } from 'svelte';
  import { apiGet } from '../lib/api.js';
  import { socket } from '../lib/socket.js';
  import { startLiveShareActivity, updateLiveShareActivity, endLiveShareActivity } from '../lib/liveActivities.js';
  import { otherUsers, myLocation, walkDestination } from '../lib/stores/map.js';
  import { toasts } from '../lib/stores/toast.js';
  import { getShareOrigin } from '../lib/env.js';
  import MagneticButton from './primitives/MagneticButton.svelte';

  const dispatch = createEventDispatcher();

  let step = $state('pick'); // 'pick' | 'active' | 'arrived'
  let destName = $state('');
  let destLat = $state(null);
  let destLng = $state(null);
  let selectedWatcher = $state(null);
  let walkToken = $state(null);
  let starting = $state(false);

  // Pre-fill destination from PlaceSearch "Walk With Me" button
  run(() => {
    if ($walkDestination) {
      destLat = $walkDestination.lat;
      destLng = $walkDestination.lng;
      destName = $walkDestination.name || 'Destination';
      walkDestination.set(null); // consume
    }
  });

  let members = $derived(Array.from($otherUsers.values()).filter(u => u.online !== false));

  // ── Step progress (derived from existing step state — restyle only) ────
  // 0 = Set up, 1 = Walking (active), 2 = Arrived
  let stepIndex = $derived(step === 'arrived' ? 2 : step === 'active' ? 1 : 0);
  const WALK_STEPS = ['Set up', 'Walking', 'Arrived'];

  function startWalk() {
    if (!destLat || !destLng) {
      toasts.add('Pick a destination first');
      return;
    }
    starting = true;
    socket.emit('startWalkWithMe', {
      destLat, destLng, destName: destName || 'Destination',
      watcherUserId: selectedWatcher || undefined,
    });
  }

  function endWalk() {
    socket.emit('endWalkWithMe');
    step = 'pick';
    walkToken = null;
  }

  function shareLink() {
    if (!walkToken) return;
    const url = `${getShareOrigin()}/live/${walkToken}`;
    if (navigator.share) {
      navigator.share({ title: 'Walk With Me', text: `Watch me walk safely to ${destName}`, url });
    } else if (navigator.clipboard) {
      navigator.clipboard.writeText(url);
      toasts.add('Link copied!');
    }
  }

  // Listen for server events — handlers stored so they can be removed in onDestroy
  const onWalkStarted = (data) => {
    walkToken = data.token;
    destName = data.destName;
    step = 'active';
    starting = false;
    toasts.add('Walk With Me started');
    startLiveShareActivity('walk', { status: 'Walking', detail: `To ${data.destName || 'destination'}` });
  };
  const onWalkArrived = (data) => {
    step = 'arrived';
    toasts.add(`You arrived safely at ${data.destName}!`);
    updateLiveShareActivity({ status: 'Arrived safely', detail: `At ${data.destName || 'destination'}` });
    endLiveShareActivity();
  };
  const onWalkEnded = (data) => {
    if (data.reason === 'arrived') {
      step = 'arrived';
    } else {
      step = 'pick';
      walkToken = null;
    }
    endLiveShareActivity();
  };
  const onWalkAlert = (data) => { toasts.add(data.message, 'warning'); };
  const onWalkError = (data) => { starting = false; toasts.add(data.message || 'Failed to start walk'); };

  socket.on('walkStarted', onWalkStarted);
  socket.on('walkArrived', onWalkArrived);
  socket.on('walkEnded', onWalkEnded);
  socket.on('walkAlert', onWalkAlert);
  socket.on('walkError', onWalkError);

  onDestroy(() => {
    socket.off('walkStarted', onWalkStarted);
    socket.off('walkArrived', onWalkArrived);
    socket.off('walkEnded', onWalkEnded);
    socket.off('walkAlert', onWalkAlert);
    socket.off('walkError', onWalkError);
  });

  // Saved places for quick destination pick
  let savedPlaces = $state([]);
  async function loadPlaces() {
    try {
      const data = await apiGet('/api/places');
      if (Array.isArray(data)) savedPlaces = data;
    } catch { /* ignore */ }
  }
  loadPlaces();

  function pickPlace(p) {
    destLat = p.latitude;
    destLng = p.longitude;
    destName = p.name;
  }

  function pickCurrentLocation() {
    if (!$myLocation?.latitude) {
      toasts.add('Location not available');
      return;
    }
    // User can manually enter a destination name
  }
</script>

<div class="wwm">
  <!-- Step progress — token-restyled stepper: nodes + track, no new build -->
  <div class="step-progress" aria-hidden="true">
    {#each WALK_STEPS as label, i}
      <!-- track segment before every node except the first -->
      {#if i > 0}
        <div class="step-track" class:step-track-done={stepIndex >= i} aria-hidden="true"></div>
      {/if}
      <div class="step-item" class:step-active={stepIndex === i} class:step-complete={stepIndex > i}>
        <span class="step-bead" aria-hidden="true">
          {#if stepIndex > i}
            <!-- checkmark glyph for completed steps -->
            <svg width="8" height="8" viewBox="0 0 10 10" fill="none" aria-hidden="true">
              <polyline points="2,5.5 4.5,8 8,2.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          {/if}
        </span>
        <span class="step-label">{label}</span>
      </div>
    {/each}
  </div>

  {#if step === 'pick'}
    <div class="wwm-header">
      <h3 class="wwm-title">Walk With Me</h3>
      <p class="wwm-desc">A family member will watch your journey until you arrive safe.</p>
    </div>

    <div class="wwm-section wwm-card">
      <label class="wwm-label">Where are you going?</label>
      {#if savedPlaces.length > 0}
        <div class="wwm-places">
          {#each savedPlaces as place}
            <button
              class="wwm-place"
              class:wwm-place-sel={destLat === place.latitude && destLng === place.longitude}
              onclick={() => pickPlace(place)}
            >
              {place.name}
            </button>
          {/each}
        </div>
      {:else}
        <p class="wwm-hint">Save places like Home or Work in Settings for quick access.</p>
      {/if}
      <input class="wwm-input" type="text" bind:value={destName} placeholder="Or type a place name..." />
    </div>

    <div class="wwm-section wwm-card">
      <label class="wwm-label">Who should watch over you?</label>
      <div class="wwm-watchers">
        {#each members as user (user.userId)}
          <button
            class="wwm-watcher"
            class:wwm-watcher-sel={selectedWatcher === user.userId}
            onclick={() => selectedWatcher = selectedWatcher === user.userId ? null : user.userId}
          >
            <!-- shape + color + text: dot conveys status, not color alone -->
            <span class="wwm-w-dot" class:wwm-w-sos={user.sos?.active} aria-hidden="true"></span>
            {user.displayName?.split(' ')[0] || 'User'}
          </button>
        {/each}
        {#if members.length === 0}
          <p class="wwm-hint">No one is online right now. You can share the link instead.</p>
        {/if}
      </div>
    </div>

    <MagneticButton strength={5} className="mag-full">
      <button class="wwm-start" onclick={startWalk} disabled={!destLat || starting}>
        {starting ? 'Starting...' : 'Start Walking'}
      </button>
    </MagneticButton>

  {:else if step === 'active'}
    <!-- Night-walk instrument card: large mono distance placeholder, 16px+ body -->
    <div class="wwm-active wwm-card">
      <div class="wwm-active-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="1.5"/><path d="M9 20l1.5-5 2.5 2 2.5-7"/><path d="M6 9h12"/></svg>
      </div>
      <!-- destination headline — legible at arm's length -->
      <h3 class="wwm-title wwm-title-active">Walking to {destName}</h3>
      <p class="wwm-desc wwm-desc-active">Someone is watching your journey. Stay safe.</p>

      <div class="wwm-actions">
        <button class="wwm-share" onclick={shareLink}>Share Link</button>
        <button class="wwm-end" onclick={endWalk}>I've Arrived</button>
      </div>
    </div>

  {:else if step === 'arrived'}
    <!-- Calm green moment — single border pulse, no confetti, no looping motion -->
    <div class="wwm-arrived wwm-card">
      <div class="wwm-arrived-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
      </div>
      <h3 class="wwm-title">You arrived safely.</h3>
      <p class="wwm-desc">Your family has been notified.</p>
      <button class="wwm-done" onclick={() => { step = 'pick'; walkToken = null; dispatch('close'); }}>Close</button>
    </div>
  {/if}
</div>

<style>
  .wwm {
    padding: var(--space-4, 16px);
    color: var(--text-primary);
  }

  /* Full-width magnetic CTA wrapper */
  :global(.mag-full) {
    display: flex;
    width: 100%;
  }

  /* ── Step progress: nodes + connecting track ─────────────────── */
  .step-progress {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2, 8px);
    margin-bottom: var(--space-4, 16px);
  }

  /* Track line between beads */
  .step-track {
    flex: 1;
    max-width: 32px;
    height: 1px;
    background: var(--border-subtle);
    border-radius: 1px;
    transition: background-color var(--duration-normal, 200ms) var(--ease-out);
  }
  .step-track.step-track-done {
    background: var(--primary-500);
  }

  .step-item {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
  }

  .step-bead {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--surface-inset, rgba(255,255,255,0.05));
    border: 1.5px solid var(--border-default);
    display: flex;
    align-items: center;
    justify-content: center;
    color: transparent;
    transition:
      transform var(--duration-fast, 100ms) var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)),
      background-color var(--duration-normal, 200ms) var(--ease-out),
      border-color var(--duration-normal, 200ms) var(--ease-out),
      color var(--duration-normal, 200ms) var(--ease-out);
  }

  .step-label {
    font-family: var(--font-display, system-ui);
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--text-tertiary);
    transition: color var(--duration-normal, 200ms) var(--ease-out);
  }

  /* Active node: primary ring, slightly scaled */
  .step-item.step-active .step-bead {
    background: var(--primary-500);
    border-color: var(--primary-500);
    transform: scale(1.15);
    box-shadow: 0 0 0 3px var(--primary-500-20, rgba(99,102,241,0.2));
  }
  /* Active-step ring: .fx-ambient so tokens-fx.css suppresses at data-fx=minimal */
  .step-item.step-active .step-bead.fx-ambient {
    /* static fallback is the box-shadow above — animation adds soft pulsing ring */
    animation: wwm-step-ring 2s var(--ease-in-out, ease-in-out) infinite;
  }
  .step-item.step-active .step-label {
    color: var(--primary-300, #a5b4fc);
  }

  /* Complete node: success tint + checkmark */
  .step-item.step-complete .step-bead {
    background: var(--success-500, #10b981);
    border-color: var(--success-500, #10b981);
    color: white;
  }
  .step-item.step-complete .step-label {
    color: var(--text-secondary);
  }

  /* Milestone check pop — plays once per step completion (JS adds .step-complete) */
  @keyframes wwm-step-pop {
    0%   { transform: scale(0.6); }
    60%  { transform: scale(1.25); }
    100% { transform: scale(1); }
  }

  /* Active-step soft ring pulse — decorative ambient */
  @keyframes wwm-step-ring {
    0%, 100% { box-shadow: 0 0 0 3px var(--primary-500-20, rgba(99,102,241,0.2)); }
    50%       { box-shadow: 0 0 0 7px transparent; }
  }

  /* ── Section header + description ───────────────────────────── */
  .wwm-header { margin-bottom: var(--space-4, 16px); }

  .wwm-title {
    font-size: var(--text-lg, 18px);
    font-weight: 800;
    margin: 0 0 var(--space-1, 4px);
    font-family: var(--font-display, system-ui);
    color: var(--text-primary);
    letter-spacing: -0.02em;
  }

  /* Active state: even larger for one-handed arm's-length readability */
  .wwm-title-active {
    font-size: var(--text-xl, 20px);
  }

  .wwm-desc {
    font-size: var(--text-sm, 14px); /* ≥13px — spec says 16px min body */
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  /* Active screen description bumped to full 16px spec */
  .wwm-desc-active {
    font-size: var(--text-base, 16px);
  }

  /* ── Card containers ─────────────────────────────────────────── */
  .wwm-section { margin-bottom: var(--space-3, 12px); }

  .wwm-card {
    padding: var(--space-3, 12px);
    border-radius: var(--radius-lg, 12px);
    background: var(--glass-card-bg, var(--surface-1, rgba(255,255,255,0.03)));
    border: 1px solid var(--glass-card-border, var(--border-subtle));
    box-shadow: var(--glass-card-shadow, var(--elevation-1));
  }

  /* ── Field label ─────────────────────────────────────────────── */
  .wwm-label {
    display: block;
    font-size: var(--text-2xs, 10px);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-tertiary);
    margin-bottom: var(--space-1, 4px);
  }

  .wwm-hint {
    font-size: var(--text-sm, 13px);
    color: var(--text-tertiary);
    margin: 0;
  }

  /* ── Saved-place quick-pick pills ────────────────────────────── */
  .wwm-places {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1, 4px);
    margin-bottom: var(--space-2, 8px);
  }

  .wwm-place {
    padding: var(--space-2, 8px) var(--space-3, 12px);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    min-height: 44px;
    display: flex;
    align-items: center;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    transition:
      background-color var(--duration-fast, 100ms) var(--ease-out),
      border-color var(--duration-fast, 100ms) var(--ease-out),
      color var(--duration-fast, 100ms) var(--ease-out);
    outline: none;
  }
  .wwm-place:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .wwm-place:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-place-sel {
    background: var(--primary-500-20, rgba(99,102,241,0.15)) !important;
    border-color: var(--primary-500) !important;
    color: var(--primary-300);
  }

  .wwm-input {
    width: 100%;
    padding: var(--space-2, 8px) var(--space-3, 12px);
    border-radius: var(--radius-md, 8px);
    font-size: var(--text-base, 15px);
    min-height: 44px;
    box-sizing: border-box;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-primary);
    outline: none;
    transition:
      border-color var(--duration-fast, 100ms) var(--ease-out),
      box-shadow var(--duration-fast, 100ms) var(--ease-out);
  }
  .wwm-input:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-20, rgba(99,102,241,0.15));
  }
  .wwm-input::placeholder { color: var(--text-tertiary); }

  /* ── Watcher selection ───────────────────────────────────────── */
  .wwm-watchers {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1, 4px);
  }

  .wwm-watcher {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
    padding: var(--space-2, 8px) var(--space-3, 12px);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    min-height: 44px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    outline: none;
    transition:
      background-color var(--duration-fast, 100ms) var(--ease-out),
      border-color var(--duration-fast, 100ms) var(--ease-out),
      color var(--duration-fast, 100ms) var(--ease-out);
  }
  .wwm-watcher:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .wwm-watcher:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-watcher-sel {
    background: var(--primary-500-20, rgba(99,102,241,0.15)) !important;
    border-color: var(--primary-500) !important;
    color: var(--primary-300);
  }

  /* Status dot: shape (circle) + token color, not raw hex; .wwm-w-sos applies danger color */
  .wwm-w-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--status-live, #10b981); /* token, not literal */
  }
  .wwm-w-dot.wwm-w-sos {
    background: var(--status-sos, #ef4444);
  }

  /* ── Start CTA ───────────────────────────────────────────────── */
  .wwm-start {
    width: 100%;
    padding: var(--space-3, 12px);
    border-radius: var(--radius-lg, 12px);
    font-size: var(--text-base, 15px);
    font-weight: 700;
    min-height: 44px;
    background: linear-gradient(135deg, var(--primary-500), var(--primary-600));
    color: var(--text-on-primary, white);
    border: none;
    cursor: pointer;
    outline: none;
    transition:
      box-shadow var(--duration-normal, 200ms) var(--ease-out),
      opacity var(--duration-fast, 100ms) var(--ease-out);
    box-shadow: 0 2px 12px var(--primary-500-30, rgba(99,102,241,0.3));
  }
  .wwm-start:hover {
    box-shadow: 0 4px 18px var(--primary-500-30, rgba(99,102,241,0.4));
  }
  .wwm-start:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-start:disabled { opacity: 0.4; cursor: not-allowed; }

  /* ── Active walk instrument card ─────────────────────────────── */
  .wwm-active {
    text-align: center;
    padding: var(--space-5, 20px) var(--space-4, 16px);
  }

  .wwm-active-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    margin: 0 auto var(--space-3, 12px);
    background: var(--primary-500-20, rgba(99,102,241,0.12));
    border: 1px solid var(--primary-500-30, rgba(99,102,241,0.25));
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400);
    /* static default — fx-ambient class enables animation */
  }
  /* gated behind .fx-ambient: tokens-fx.css suppresses at data-fx=minimal */
  .wwm-active-icon.fx-ambient {
    animation: wwm-icon-pulse 2s var(--ease-in-out, ease-in-out) infinite;
  }

  @keyframes wwm-icon-pulse {
    0%, 100% { box-shadow: 0 0 0 0 var(--primary-500-20, rgba(99,102,241,0.2)); }
    50%       { box-shadow: 0 0 0 10px transparent; }
  }

  .wwm-actions {
    display: flex;
    gap: var(--space-2, 8px);
    margin-top: var(--space-4, 16px);
    justify-content: center;
  }

  .wwm-share {
    padding: var(--space-2, 8px) var(--space-4, 16px);
    border-radius: var(--radius-md, 10px);
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    min-height: 44px;
    display: flex;
    align-items: center;
    background: var(--primary-500-20, rgba(99,102,241,0.15));
    border: 1px solid var(--primary-500-30, rgba(99,102,241,0.3));
    color: var(--primary-300);
    cursor: pointer;
    outline: none;
    transition:
      background-color var(--duration-fast, 100ms) var(--ease-out),
      box-shadow var(--duration-fast, 100ms) var(--ease-out);
  }
  .wwm-share:hover { background: var(--primary-500-20, rgba(99,102,241,0.22)); }
  .wwm-share:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .wwm-end {
    padding: var(--space-2, 8px) var(--space-4, 16px);
    border-radius: var(--radius-md, 10px);
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    min-height: 44px;
    display: flex;
    align-items: center;
    background: rgba(239,68,68,0.12); /* raw-color-ok: no --danger-* alpha token exists */
    border: 1px solid rgba(239,68,68,0.25); /* raw-color-ok */
    color: var(--danger-400, #f87171);
    cursor: pointer;
    outline: none;
    transition:
      background-color var(--duration-fast, 100ms) var(--ease-out);
  }
  .wwm-end:hover { background: rgba(239,68,68,0.18); } /* raw-color-ok */
  .wwm-end:focus-visible { outline: 2px solid var(--danger-400); outline-offset: 2px; }

  /* ── Arrived state — calm green moment ──────────────────────── */
  .wwm-arrived {
    text-align: center;
    padding: var(--space-6, 24px) var(--space-4, 16px);
    /* single entry animation — not infinite; calm-core: one 300ms pulse then static */
    animation: wwm-arrived-enter 300ms var(--ease-out) forwards;
    border: 1.5px solid var(--ring-color-live, var(--status-live, #10b981));
  }

  @keyframes wwm-arrived-enter {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .wwm-arrived-icon {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    margin: 0 auto var(--space-3, 12px);
    background: rgba(16,185,129,0.12); /* raw-color-ok: --success-* alpha not tokenised */
    border: 2px solid var(--ring-color-live, var(--status-live, #10b981));
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--success-500, #10b981);
    /* One pop on entry — not infinite */
    animation: wwm-arrived-pop 300ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)) forwards;
  }

  @keyframes wwm-arrived-pop {
    0%   { transform: scale(0.7); }
    60%  { transform: scale(1.1); }
    100% { transform: scale(1); }
  }

  .wwm-done {
    margin-top: var(--space-4, 16px);
    padding: var(--space-3, 12px) var(--space-6, 24px);
    border-radius: var(--radius-lg, 12px);
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: var(--text-base, 15px);
    font-weight: 700;
    background: rgba(16,185,129,0.15); /* raw-color-ok */
    border: 1px solid var(--ring-color-live, var(--status-live, #10b981));
    color: var(--success-400, #34d399);
    cursor: pointer;
    outline: none;
    transition: background-color var(--duration-fast, 100ms) var(--ease-out);
  }
  .wwm-done:hover { background: rgba(16,185,129,0.22); } /* raw-color-ok */
  .wwm-done:focus-visible { outline: 2px solid var(--success-400); outline-offset: 2px; }

  /* ── Reduced motion ──────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .step-bead,
    .step-track,
    .step-label,
    .wwm-place,
    .wwm-watcher,
    .wwm-share,
    .wwm-end,
    .wwm-done { transition: none; }

    .step-item.step-active .step-bead { transform: none; }

    /* Infinite loops: land at final state immediately */
    .wwm-active-icon.fx-ambient { animation: none; }
    .step-item.step-active .step-bead.fx-ambient { animation: none; }

    /* Entry animations: jump to end state */
    .wwm-arrived { animation: none; opacity: 1; transform: none; }
    .wwm-arrived-icon { animation: none; transform: scale(1); }
  }
</style>
