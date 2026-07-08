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

  // ── Step progress (derived from existing step state — no new flow) ────
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
  <!-- Step progress — reflects existing walk lifecycle state -->
  <div class="step-progress" aria-hidden="true">
    {#each WALK_STEPS as label, i}
      <div class="step-item" class:step-active={stepIndex === i} class:step-complete={stepIndex > i}>
        <span class="step-bead"></span>
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
              class="wwm-place tactile" class:wwm-place-sel={destLat === place.latitude && destLng === place.longitude}
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
            class="wwm-watcher tactile" class:wwm-watcher-sel={selectedWatcher === user.userId}
            onclick={() => selectedWatcher = selectedWatcher === user.userId ? null : user.userId}
          >
            <span class="wwm-w-dot" style="background:{user.sos?.active ? '#ef4444' : '#10b981'}"></span>
            {user.displayName?.split(' ')[0] || 'User'}
          </button>
        {/each}
        {#if members.length === 0}
          <p class="wwm-hint">No one is online right now. You can share the link instead.</p>
        {/if}
      </div>
    </div>

    <MagneticButton strength={5} className="mag-full">
      <button class="wwm-start tactile" onclick={startWalk} disabled={!destLat || starting}>
        {starting ? 'Starting...' : 'Start Walking'}
      </button>
    </MagneticButton>

  {:else if step === 'active'}
    <div class="wwm-active wwm-card">
      <div class="wwm-active-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="1.5"/><path d="M9 20l1.5-5 2.5 2 2.5-7"/><path d="M6 9h12"/></svg>
      </div>
      <h3 class="wwm-title">Walking to {destName}</h3>
      <p class="wwm-desc">Someone is watching your journey. Stay safe.</p>

      <div class="wwm-actions">
        <button class="wwm-share tactile" onclick={shareLink}>Share Link</button>
        <button class="wwm-end tactile" onclick={endWalk}>I've Arrived</button>
      </div>
    </div>

  {:else if step === 'arrived'}
    <div class="wwm-arrived wwm-card">
      <div class="wwm-arrived-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
      </div>
      <h3 class="wwm-title">You made it!</h3>
      <p class="wwm-desc">You arrived safely. Your family has been notified.</p>
      <button class="wwm-done tactile" onclick={() => { step = 'pick'; walkToken = null; dispatch('close'); }}>Close</button>
    </div>
  {/if}
</div>

<style>
  .wwm {
    padding: 16px;
    color: rgba(255,255,255,0.88);
  }

  /* Full-width magnetic CTA wrapper — overrides inline-flex so the button fills */
  :global(.mag-full) {
    display: flex;
    width: 100%;
  }

  /* ── Step progress ───────────────────────────────────────────── */
  .step-progress {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-4, 16px);
    margin-bottom: var(--space-4, 16px);
  }
  .step-item {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
  }
  .step-bead {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--border-default, rgba(255,255,255,0.16));
    transition: transform 220ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)),
                background-color 220ms var(--ease-out, ease);
  }
  .step-label {
    font-family: var(--font-display, system-ui);
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    letter-spacing: 0.03em;
    color: var(--text-tertiary, rgba(255,255,255,0.30));
    transition: color 220ms var(--ease-out, ease);
  }
  .step-item.step-active .step-bead {
    background: var(--primary-500, #6366f1);
    transform: scale(1.5);
    box-shadow: 0 0 0 4px var(--primary-500-20, rgba(99,102,241,0.2));
  }
  .step-item.step-active .step-label { color: var(--primary-300, #a5b4fc); }
  .step-item.step-complete .step-bead { background: var(--success-500, #10b981); }
  .step-item.step-complete .step-label { color: var(--text-secondary, rgba(255,255,255,0.6)); }

  .wwm-header { margin-bottom: 16px; }
  .wwm-title {
    font-size: 16px; font-weight: 800; margin: 0 0 4px;
    font-family: var(--font-display, system-ui);
  }
  .wwm-desc { font-size: 12px; color: rgba(255,255,255,0.40); margin: 0; }

  /* ── Card containers ─────────────────────────────────────────── */
  .wwm-section { margin-bottom: 14px; }
  .wwm-card {
    padding: var(--space-3, 12px);
    border-radius: var(--radius-lg, 12px);
    background: var(--surface-1, rgba(255,255,255,0.03));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.08));
    box-shadow: var(--elevation-1, 0 2px 8px rgba(0,0,0,0.08));
  }

  .wwm-label {
    display: block; font-size: 9px; font-weight: 700;
    text-transform: uppercase; letter-spacing: 0.08em;
    color: rgba(255,255,255,0.30); margin-bottom: 6px;
  }
  .wwm-hint { font-size: 11px; color: rgba(255,255,255,0.25); margin: 0; }
  .wwm-places { display: flex; flex-wrap: wrap; gap: var(--space-1); margin-bottom: var(--space-2); }
  .wwm-place {
    padding: var(--space-2) var(--space-3); border-radius: 8px; font-size: 11px; font-weight: 600;
    min-height: 44px; display: flex; align-items: center;
    background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.6); cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }
  .wwm-place:hover { background: rgba(255,255,255,0.08); }
  .wwm-place-sel {
    background: var(--primary-500-20, rgba(99,102,241,0.15)) !important;
    border-color: var(--primary-500-30, rgba(99,102,241,0.4)) !important;
    color: rgba(165,180,252,0.9);
  }
  .wwm-input {
    width: 100%; padding: 10px 12px; border-radius: 8px; font-size: 12px;
    min-height: 44px; box-sizing: border-box;
    background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.8); outline: none;
    transition: border-color 0.15s;
  }
  .wwm-input:focus { border-color: var(--primary-500-30, rgba(99,102,241,0.4)); }
  .wwm-input::placeholder { color: rgba(255,255,255,0.2); }
  .wwm-watchers { display: flex; flex-wrap: wrap; gap: var(--space-1); }
  .wwm-watcher {
    display: flex; align-items: center; gap: var(--space-1);
    padding: var(--space-2) var(--space-3); border-radius: 8px; font-size: 11px; font-weight: 600;
    min-height: 44px;
    background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.6); cursor: pointer;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
  }
  .wwm-watcher:hover { background: rgba(255,255,255,0.08); }
  .wwm-watcher-sel {
    background: rgba(16,185,129,0.12) !important;
    border-color: rgba(16,185,129,0.35) !important;
    color: rgba(52,211,153,0.9);
  }
  .wwm-w-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
  .wwm-start {
    width: 100%; padding: var(--space-3); border-radius: 12px; font-size: 13px; font-weight: 700;
    min-height: 44px;
    background: linear-gradient(135deg, var(--primary-500, #6366f1), var(--primary-600, #4f46e5));
    color: var(--text-on-primary); border: none; cursor: pointer;
    transition: box-shadow 0.2s;
    box-shadow: 0 2px 12px var(--primary-500-30, rgba(99,102,241,0.3));
  }
  .wwm-start:hover { box-shadow: 0 4px 18px var(--primary-500-30, rgba(99,102,241,0.4)); }
  .wwm-start:disabled { opacity: 0.4; cursor: not-allowed; }

  /* Active walk state */
  .wwm-active { text-align: center; padding: 20px 16px; }
  .wwm-active-icon {
    width: 56px; height: 56px; border-radius: 50%; margin: 0 auto 12px;
    background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.25);
    display: flex; align-items: center; justify-content: center;
    color: var(--primary-400, #818cf8);
    animation: wwm-pulse 2s ease-in-out infinite;
  }
  @keyframes wwm-pulse { 0%,100%{box-shadow:0 0 0 0 rgba(99,102,241,0.2)} 50%{box-shadow:0 0 0 12px rgba(99,102,241,0)} }
  .wwm-actions { display: flex; gap: 8px; margin-top: 16px; justify-content: center; }
  .wwm-share {
    padding: var(--space-2) var(--space-4); border-radius: 10px; font-size: 12px; font-weight: 700;
    min-height: 44px; display: flex; align-items: center;
    background: rgba(99,102,241,0.15); border: 1px solid rgba(99,102,241,0.3);
    color: var(--primary-300, #a5b4fc); cursor: pointer;
  }
  .wwm-end {
    padding: var(--space-2) var(--space-4); border-radius: 10px; font-size: 12px; font-weight: 700;
    min-height: 44px; display: flex; align-items: center;
    background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.25);
    color: var(--danger-400, #f87171); cursor: pointer;
  }

  /* Arrived state */
  .wwm-arrived { text-align: center; padding: 24px 16px; }
  .wwm-arrived-icon {
    width: 64px; height: 64px; border-radius: 50%; margin: 0 auto 12px;
    background: rgba(16,185,129,0.12); border: 2px solid rgba(16,185,129,0.3);
    display: flex; align-items: center; justify-content: center;
    animation: wwm-arrive-bounce 0.6s cubic-bezier(0.34,1.56,0.64,1);
  }
  @keyframes wwm-arrive-bounce { 0%{transform:scale(0)} 100%{transform:scale(1)} }
  .wwm-done {
    margin-top: var(--space-4); padding: var(--space-3) var(--space-6); border-radius: 12px;
    min-height: 44px; display: inline-flex; align-items: center; justify-content: center;
    font-size: 13px; font-weight: 700;
    background: rgba(16,185,129,0.15); border: 1px solid rgba(16,185,129,0.3);
    color: var(--success-400, #34d399); cursor: pointer;
  }

  @media (prefers-reduced-motion: reduce) {
    .step-bead,
    .step-label { transition: none; }
    .step-item.step-active .step-bead { transform: none; }
    .wwm-active-icon,
    .wwm-arrived-icon { animation: none; }
  }
</style>
