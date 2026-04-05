<script>
  import { createEventDispatcher } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { otherUsers, myLocation, walkDestination } from '../lib/stores/map.js';
  import { toasts } from '../lib/stores/toast.js';
  import { getShareOrigin } from '../lib/env.js';

  const dispatch = createEventDispatcher();

  let step = 'pick'; // 'pick' | 'active' | 'arrived'
  let destName = '';
  let destLat = null;
  let destLng = null;
  let selectedWatcher = null;
  let walkToken = null;
  let starting = false;

  // Pre-fill destination from PlaceSearch "Walk With Me" button
  $: if ($walkDestination) {
    destLat = $walkDestination.lat;
    destLng = $walkDestination.lng;
    destName = $walkDestination.name || 'Destination';
    walkDestination.set(null); // consume
  }

  $: members = Array.from($otherUsers.values()).filter(u => u.online !== false);

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

  // Listen for server events
  socket.on('walkStarted', (data) => {
    walkToken = data.token;
    destName = data.destName;
    step = 'active';
    starting = false;
    toasts.add('Walk With Me started');
  });
  socket.on('walkArrived', (data) => {
    step = 'arrived';
    toasts.add(`You arrived safely at ${data.destName}!`);
  });
  socket.on('walkEnded', (data) => {
    if (data.reason === 'arrived') {
      step = 'arrived';
    } else {
      step = 'pick';
      walkToken = null;
    }
  });
  socket.on('walkAlert', (data) => {
    toasts.add(data.message, 'warning');
  });
  socket.on('walkError', (data) => {
    starting = false;
    toasts.add(data.message || 'Failed to start walk');
  });

  // Saved places for quick destination pick
  let savedPlaces = [];
  import API_BASE from '../lib/env.js';
  async function loadPlaces() {
    try {
      const res = await fetch(`${API_BASE}/api/places`, { credentials: 'include' });
      if (res.ok) savedPlaces = await res.json();
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
  {#if step === 'pick'}
    <div class="wwm-header">
      <h3 class="wwm-title">Walk With Me</h3>
      <p class="wwm-desc">A family member will watch your journey until you arrive safe.</p>
    </div>

    <div class="wwm-section">
      <label class="wwm-label">Where are you going?</label>
      {#if savedPlaces.length > 0}
        <div class="wwm-places">
          {#each savedPlaces as place}
            <button
              class="wwm-place" class:wwm-place-sel={destLat === place.latitude && destLng === place.longitude}
              on:click={() => pickPlace(place)}
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

    <div class="wwm-section">
      <label class="wwm-label">Who should watch over you?</label>
      <div class="wwm-watchers">
        {#each members as user (user.userId)}
          <button
            class="wwm-watcher" class:wwm-watcher-sel={selectedWatcher === user.userId}
            on:click={() => selectedWatcher = selectedWatcher === user.userId ? null : user.userId}
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

    <button class="wwm-start" on:click={startWalk} disabled={!destLat || starting}>
      {starting ? 'Starting...' : 'Start Walking'}
    </button>

  {:else if step === 'active'}
    <div class="wwm-active">
      <div class="wwm-active-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="1.5"/><path d="M9 20l1.5-5 2.5 2 2.5-7"/><path d="M6 9h12"/></svg>
      </div>
      <h3 class="wwm-title">Walking to {destName}</h3>
      <p class="wwm-desc">Someone is watching your journey. Stay safe.</p>

      <div class="wwm-actions">
        <button class="wwm-share" on:click={shareLink}>Share Link</button>
        <button class="wwm-end" on:click={endWalk}>I've Arrived</button>
      </div>
    </div>

  {:else if step === 'arrived'}
    <div class="wwm-arrived">
      <div class="wwm-arrived-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
      </div>
      <h3 class="wwm-title">You made it!</h3>
      <p class="wwm-desc">You arrived safely. Your family has been notified.</p>
      <button class="wwm-done" on:click={() => { step = 'pick'; walkToken = null; dispatch('close'); }}>Close</button>
    </div>
  {/if}
</div>

<style>
  .wwm {
    padding: 16px;
    color: rgba(255,255,255,0.88);
  }
  .wwm-header { margin-bottom: 16px; }
  .wwm-title {
    font-size: 16px; font-weight: 800; margin: 0 0 4px;
    font-family: var(--font-display, system-ui);
  }
  .wwm-desc { font-size: 12px; color: rgba(255,255,255,0.40); margin: 0; }
  .wwm-section { margin-bottom: 14px; }
  .wwm-label {
    display: block; font-size: 9px; font-weight: 700;
    text-transform: uppercase; letter-spacing: 0.08em;
    color: rgba(255,255,255,0.30); margin-bottom: 6px;
  }
  .wwm-hint { font-size: 11px; color: rgba(255,255,255,0.25); margin: 0; }
  .wwm-places { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 8px; }
  .wwm-place {
    padding: 5px 10px; border-radius: 8px; font-size: 11px; font-weight: 600;
    background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.6); cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }
  .wwm-place:hover { background: rgba(255,255,255,0.08); }
  .wwm-place-sel {
    background: rgba(99,102,241,0.15) !important;
    border-color: rgba(99,102,241,0.4) !important;
    color: rgba(165,180,252,0.9);
  }
  .wwm-input {
    width: 100%; padding: 8px 10px; border-radius: 8px; font-size: 12px;
    background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.8); outline: none;
    transition: border-color 0.15s;
  }
  .wwm-input:focus { border-color: rgba(99,102,241,0.4); }
  .wwm-input::placeholder { color: rgba(255,255,255,0.2); }
  .wwm-watchers { display: flex; flex-wrap: wrap; gap: 5px; }
  .wwm-watcher {
    display: flex; align-items: center; gap: 5px;
    padding: 5px 10px; border-radius: 8px; font-size: 11px; font-weight: 600;
    background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.08);
    color: rgba(255,255,255,0.6); cursor: pointer;
    transition: all 0.15s;
  }
  .wwm-watcher:hover { background: rgba(255,255,255,0.08); }
  .wwm-watcher-sel {
    background: rgba(16,185,129,0.12) !important;
    border-color: rgba(16,185,129,0.35) !important;
    color: rgba(52,211,153,0.9);
  }
  .wwm-w-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
  .wwm-start {
    width: 100%; padding: 11px; border-radius: 12px; font-size: 13px; font-weight: 700;
    background: linear-gradient(135deg, var(--primary-500, #6366f1), var(--primary-600, #4f46e5));
    color: #fff; border: none; cursor: pointer;
    transition: transform 0.1s, box-shadow 0.2s;
    box-shadow: 0 2px 12px rgba(99,102,241,0.3);
  }
  .wwm-start:hover { transform: translateY(-1px); box-shadow: 0 4px 18px rgba(99,102,241,0.4); }
  .wwm-start:active { transform: scale(0.98); }
  .wwm-start:disabled { opacity: 0.4; cursor: not-allowed; transform: none; }

  /* Active walk state */
  .wwm-active { text-align: center; padding: 16px 0; }
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
    padding: 8px 16px; border-radius: 10px; font-size: 12px; font-weight: 700;
    background: rgba(99,102,241,0.15); border: 1px solid rgba(99,102,241,0.3);
    color: var(--primary-300, #a5b4fc); cursor: pointer;
  }
  .wwm-end {
    padding: 8px 16px; border-radius: 10px; font-size: 12px; font-weight: 700;
    background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.25);
    color: #f87171; cursor: pointer;
  }

  /* Arrived state */
  .wwm-arrived { text-align: center; padding: 20px 0; }
  .wwm-arrived-icon {
    width: 64px; height: 64px; border-radius: 50%; margin: 0 auto 12px;
    background: rgba(16,185,129,0.12); border: 2px solid rgba(16,185,129,0.3);
    display: flex; align-items: center; justify-content: center;
    animation: wwm-arrive-bounce 0.6s cubic-bezier(0.34,1.56,0.64,1);
  }
  @keyframes wwm-arrive-bounce { 0%{transform:scale(0)} 100%{transform:scale(1)} }
  .wwm-done {
    margin-top: 16px; padding: 10px 24px; border-radius: 12px;
    font-size: 13px; font-weight: 700;
    background: rgba(16,185,129,0.15); border: 1px solid rgba(16,185,129,0.3);
    color: #34d399; cursor: pointer;
  }
</style>
