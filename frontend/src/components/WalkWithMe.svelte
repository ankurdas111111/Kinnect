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

  // Companion name for the live-status line — resolved from the same store
  // the watcher picker reads; presentation only.
  let companionName = $derived(
    selectedWatcher
      ? (members.find(m => m.userId === selectedWatcher)?.displayName?.split(' ')[0] || null)
      : null
  );

  // ── Journey timeline — quiet sentence entries built from the session's
  //    real events (started / alerts / shares / arrival). Presentation-layer
  //    log only: every socket emit, toast and Live Activity call is unchanged.
  let journey = $state([]);
  function logMoment(text, tone) {
    journey = [...journey, { text, at: Date.now(), tone }];
  }
  function softTime(at) {
    return new Date(at)
      .toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
      .toLowerCase();
  }

  function initials(name) {
    if (!name) return '?';
    return name.trim().split(/\s+/).slice(0, 2).map(w => w[0]).join('').toUpperCase();
  }

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
    journey = [];
  }

  function shareLink() {
    if (!walkToken) return;
    const url = `${getShareOrigin()}/live/${walkToken}`;
    if (navigator.share) {
      navigator.share({ title: 'Walk With Me', text: `Watch me walk safely to ${destName}`, url })
        .then(() => logMoment('You invited your circle to follow along', 'quiet'))
        .catch(() => { /* share sheet dismissed — nothing to record */ });
    } else if (navigator.clipboard) {
      navigator.clipboard.writeText(url);
      toasts.add('Link copied!');
      logMoment('You invited your circle to follow along', 'quiet');
    }
  }

  // Listen for server events — handlers stored so they can be removed in onDestroy
  const onWalkStarted = (data) => {
    walkToken = data.token;
    destName = data.destName;
    step = 'active';
    starting = false;
    toasts.add('Walk With Me started');
    journey = [{ text: `You set out for ${data.destName || 'your destination'}`, at: Date.now(), tone: 'ember' }];
    startLiveShareActivity('walk', { status: 'Walking', detail: `To ${data.destName || 'destination'}` });
  };
  const onWalkArrived = (data) => {
    step = 'arrived';
    toasts.add(`You arrived safely at ${data.destName}!`);
    logMoment(`You arrived at ${data.destName || 'your destination'}`, 'sage');
    updateLiveShareActivity({ status: 'Arrived safely', detail: `At ${data.destName || 'destination'}` });
    endLiveShareActivity();
  };
  const onWalkEnded = (data) => {
    if (data.reason === 'arrived') {
      step = 'arrived';
    } else {
      step = 'pick';
      walkToken = null;
      journey = [];
    }
    endLiveShareActivity();
  };
  const onWalkAlert = (data) => {
    toasts.add(data.message, 'warning');
    if (data.message) logMoment(data.message, 'note');
  };
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
  <!-- Step progress — same derived state, rendered as a quiet bead path -->
  <ol class="wwm-steps" aria-label="Walk progress">
    {#each WALK_STEPS as label, i}
      <li
        class="wwm-step"
        class:is-active={stepIndex === i}
        class:is-done={stepIndex > i}
        aria-current={stepIndex === i ? 'step' : undefined}
      >
        <span class="wwm-step-bead" aria-hidden="true">
          {#if stepIndex > i}
            <svg width="9" height="9" viewBox="0 0 10 10" fill="none" aria-hidden="true">
              <polyline points="2,5.5 4.5,8 8,2.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          {/if}
        </span>
        <span class="wwm-step-label">{label}</span>
      </li>
    {/each}
  </ol>

  {#if step === 'pick'}
    <!-- Verdict crest — the one honest sentence, serif italic -->
    <header class="wwm-crest">
      <h3 class="wwm-verdict">
        {destName ? `You're heading to ${destName}.` : 'Where are you heading?'}
      </h3>
      <p class="wwm-sub">Someone who loves you keeps you company until you're safely there.</p>
    </header>

    <section class="wwm-row" role="group" aria-labelledby="wwm-dest-label">
      <h4 class="wwm-label" id="wwm-dest-label">Where you're going</h4>
      {#if savedPlaces.length > 0}
        <div class="wwm-places">
          {#each savedPlaces as place}
            <button
              type="button"
              class="wwm-place"
              class:is-selected={destLat === place.latitude && destLng === place.longitude}
              aria-pressed={destLat === place.latitude && destLng === place.longitude}
              onclick={() => pickPlace(place)}
            >
              {place.name}
            </button>
          {/each}
        </div>
      {:else}
        <p class="wwm-hint">Save places like Home or Work in Settings for quick access.</p>
      {/if}
      <input
        class="wwm-input"
        type="text"
        bind:value={destName}
        placeholder="Or type a place name…"
        aria-label="Destination name"
      />
    </section>

    <section class="wwm-row" role="group" aria-labelledby="wwm-comp-label">
      <h4 class="wwm-label" id="wwm-comp-label">Who walks with you</h4>
      <div class="wwm-watchers">
        {#each members as user (user.userId)}
          <button
            type="button"
            class="wwm-watcher"
            class:is-selected={selectedWatcher === user.userId}
            aria-pressed={selectedWatcher === user.userId}
            onclick={() => selectedWatcher = selectedWatcher === user.userId ? null : user.userId}
          >
            <span class="wwm-pebble" aria-hidden="true">
              {initials(user.displayName)}
              <!-- presence pip: shape + position carry meaning, not color alone;
                   vermilion appears ONLY when this person is in an active SOS -->
              <span class="wwm-pip" class:wwm-pip-sos={user.sos?.active}></span>
            </span>
            <span class="wwm-watcher-name">{user.displayName?.split(' ')[0] || 'User'}</span>
            {#if user.sos?.active}
              <span class="visually-hidden">— in an active SOS</span>
            {/if}
          </button>
        {/each}
        {#if members.length === 0}
          <p class="wwm-hint">No one is online right now — you can still share a live link once you set out.</p>
        {/if}
      </div>
    </section>

    <MagneticButton strength={5} className="mag-full">
      <button type="button" class="wwm-start" onclick={startWalk} disabled={!destLat || starting}>
        {starting ? 'Starting…' : 'Start walking'}
      </button>
    </MagneticButton>

  {:else if step === 'active'}
    <!-- 1 · Verdict banner + live status (reference: destination header + remaining row) -->
    <header class="wwm-crest">
      <h3 class="wwm-verdict">You're walking to {destName}.</h3>
      <p class="wwm-live" aria-live="polite">
        <span class="wwm-live-dot" aria-hidden="true"><span class="wwm-live-halo fx-ambient"></span></span>
        <span class="wwm-live-word">Live</span>
        <span class="wwm-live-rest">
          — {companionName ? `${companionName} is walking with you` : 'your circle can follow your journey'}
        </span>
      </p>
    </header>

    <!-- 2 · Journey timeline — quiet sentences with soft timestamps -->
    <section class="wwm-row" aria-labelledby="wwm-journey-label">
      <h4 class="wwm-label" id="wwm-journey-label">Along the way</h4>
      {#if journey.length > 0}
        <ol class="wwm-journey">
          {#each journey as entry}
            <li class="wwm-entry wwm-entry-{entry.tone}">
              <span class="wwm-entry-dot" aria-hidden="true"></span>
              <span class="wwm-entry-text">{entry.text}</span>
              <time class="wwm-entry-time" datetime={new Date(entry.at).toISOString()}>{softTime(entry.at)}</time>
            </li>
          {/each}
        </ol>
      {:else}
        <p class="wwm-hint">Your walk has just begun.</p>
      {/if}
    </section>

    <!-- 3 · Gentle reassurance row (reference: warm-ping row → our real share feature) -->
    <section class="wwm-ping">
      <span class="wwm-ping-pebble" aria-hidden="true">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
      </span>
      <span class="wwm-ping-copy">
        <span class="wwm-ping-title">Bring someone along</span>
        <span class="wwm-ping-cap">A live link lets them walk beside you, quietly.</span>
      </span>
      <button type="button" class="wwm-share" onclick={shareLink}>Share link</button>
    </section>

    <!-- 4 · How this ends (reference: arrival safety protocol) -->
    <section class="wwm-ends" aria-labelledby="wwm-ends-label">
      <div class="wwm-ends-head">
        <span class="wwm-ends-icon" aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
        </span>
        <h4 class="wwm-ends-title" id="wwm-ends-label">How this walk ends</h4>
        <span class="wwm-tag">Automatic</span>
      </div>
      <p class="wwm-ends-body">
        When you reach {destName}, your companions are told you made it and the walk closes on its own.
        You can also end it yourself below.
      </p>
    </section>

    <!-- 5 · Session controls (reference footer register — no SOS affordance exists here) -->
    <div class="wwm-controls">
      <button type="button" class="wwm-arrive" onclick={endWalk}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        <span>I've arrived safely</span>
      </button>
    </div>

  {:else if step === 'arrived'}
    <!-- Arrival — the settled sage moment: one gentle entry, then still -->
    <section class="wwm-arrived">
      <span class="wwm-arrived-pebble" aria-hidden="true">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
      </span>
      <h3 class="wwm-verdict">You arrived safely.</h3>
      <p class="wwm-sub">Your circle has been told you're here. Nothing more to do.</p>
      <button type="button" class="wwm-done" onclick={() => { step = 'pick'; walkToken = null; journey = []; dispatch('close'); }}>
        Close
      </button>
    </section>
  {/if}
</div>

<style>
  /* ═══ Ground — a warm paper sheet; rows separated by whitespace, not rules ═══ */
  .wwm {
    background: var(--paper);
    border-radius: var(--radius-xl); /* sheet register, 20px */
    box-shadow: var(--shadow-xs);
    padding: var(--space-5) var(--space-4) var(--space-4);
    color: var(--text-primary);
    font-family: var(--font-sans);
  }

  /* Full-width magnetic CTA wrapper */
  :global(.mag-full) {
    display: flex;
    width: 100%;
  }

  .visually-hidden {
    position: absolute;
    width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
    border: 0;
  }

  /* ═══ Step progress — quiet beads on a hairline path ═══ */
  .wwm-steps {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    list-style: none;
    margin: 0 0 var(--space-5);
    padding: 0;
  }

  .wwm-step {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
  }

  /* leading track segment on every step after the first */
  .wwm-step + .wwm-step::before {
    content: '';
    width: var(--space-6);
    height: 1px;
    background: var(--hairline);
    border-radius: var(--radius-full);
    margin-right: var(--space-1-5);
    transition: background-color var(--duration-normal) var(--ease-out);
  }
  .wwm-step.is-active::before,
  .wwm-step.is-done::before {
    background: var(--primary-500-30);
  }

  .wwm-step-bead {
    width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    border: 1px solid var(--hairline);
    display: flex;
    align-items: center;
    justify-content: center;
    color: transparent;
    transition:
      background-color var(--duration-normal) var(--ease-out),
      border-color var(--duration-normal) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out),
      color var(--duration-normal) var(--ease-out);
  }

  .wwm-step-label {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--text-tertiary);
    transition: color var(--duration-normal) var(--ease-out);
  }

  /* Active bead: the ember presence, with a static soft ring */
  .wwm-step.is-active .wwm-step-bead {
    background: var(--primary-500);
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-20);
  }
  .wwm-step.is-active .wwm-step-label {
    color: var(--primary-700);
  }

  /* Done bead: sage settled */
  .wwm-step.is-done .wwm-step-bead {
    background: var(--success-500);
    border-color: var(--success-500);
    color: var(--text-inverse);
  }
  .wwm-step.is-done .wwm-step-label {
    color: var(--text-secondary);
  }

  /* ═══ Verdict crest — serif italic, the only serif on the sheet ═══ */
  .wwm-crest {
    text-align: center;
    margin-bottom: var(--space-6);
    padding: 0 var(--space-2);
  }

  .wwm-verdict {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-2xl);
    line-height: 1.35;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    margin: 0 0 var(--space-2);
    overflow-wrap: break-word;
  }

  .wwm-sub {
    font-size: var(--text-base);
    line-height: 1.55;
    color: var(--text-secondary);
    margin: 0;
  }

  /* Live status row — sage breath for "live", ember word for presence */
  .wwm-live {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
    gap: var(--space-1-5);
    margin: 0;
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .wwm-live-word {
    color: var(--primary-700);
    font-weight: 600;
  }

  .wwm-live-dot {
    position: relative;
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full);
    background: var(--status-live);
    flex-shrink: 0;
  }

  /* breathing halo — transform/opacity only, fades out under calm/minimal fx */
  .wwm-live-halo {
    position: absolute;
    inset: 0;
    border-radius: var(--radius-full);
    background: var(--success-500-30);
    animation: wwm-breathe 4s var(--ease-in-out) infinite;
  }

  @keyframes wwm-breathe {
    0%, 100% { transform: scale(1);   opacity: 0.8; }
    50%      { transform: scale(2.1); opacity: 0; }
  }

  /* ═══ Rows — whitespace-separated groups ═══ */
  .wwm-row { margin-bottom: var(--space-5); }

  .wwm-label {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
    margin: 0 0 var(--space-2);
  }

  .wwm-hint {
    font-size: var(--text-sm);
    line-height: 1.5;
    color: var(--text-tertiary);
    margin: 0 0 var(--space-2);
  }

  /* ═══ Saved-place quick picks — soft paper pills ═══ */
  .wwm-places {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }

  .wwm-place {
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-full);
    font-family: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    min-height: 44px;
    display: flex;
    align-items: center;
    background: var(--surface-2);
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    outline: none;
    transition:
      background-color var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-fast) var(--ease-out),
      color var(--duration-fast) var(--ease-out);
  }
  .wwm-place:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .wwm-place:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-place.is-selected {
    background: var(--primary-100);
    color: var(--primary-700);
    box-shadow: inset 0 0 0 1.5px var(--primary-500);
  }

  /* ═══ Destination input — warm inset, ember focus glow ═══ */
  .wwm-input {
    width: 100%;
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-input);
    font-family: inherit;
    font-size: var(--text-base);
    min-height: 48px;
    box-sizing: border-box;
    background: var(--surface-inset);
    border: 1px solid transparent;
    color: var(--text-primary);
    outline: none;
    transition:
      border-color var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-fast) var(--ease-out);
  }
  .wwm-input:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-20);
  }
  .wwm-input::placeholder { color: var(--text-tertiary); }

  /* ═══ Companion picker — people as pebbles, never pins ═══ */
  .wwm-watchers {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .wwm-watcher {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1-5) var(--space-3) var(--space-1-5) var(--space-1-5);
    border-radius: var(--radius-full);
    font-family: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    min-height: 44px;
    background: var(--surface-2);
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    outline: none;
    transition:
      background-color var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-fast) var(--ease-out),
      color var(--duration-fast) var(--ease-out);
  }
  .wwm-watcher:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .wwm-watcher:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-watcher.is-selected {
    background: var(--primary-100);
    color: var(--primary-700);
    box-shadow: inset 0 0 0 1.5px var(--primary-500);
  }

  .wwm-pebble {
    position: relative;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-secondary);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.02em;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition:
      background-color var(--duration-fast) var(--ease-out),
      color var(--duration-fast) var(--ease-out);
  }
  .wwm-watcher.is-selected .wwm-pebble {
    background: var(--primary-500);
    color: var(--text-on-primary);
  }

  /* presence pip — sage when live; vermilion strictly for an active SOS */
  .wwm-pip {
    position: absolute;
    right: -1px;
    bottom: -1px;
    width: 9px;
    height: 9px;
    border-radius: var(--radius-full);
    background: var(--status-live);
    border: 2px solid var(--paper);
  }
  .wwm-pip.wwm-pip-sos {
    background: var(--status-sos);
  }

  /* ═══ Start CTA — the one ember fill ═══ */
  .wwm-start {
    width: 100%;
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-button);
    font-family: inherit;
    font-size: var(--text-base);
    font-weight: 700;
    min-height: 48px;
    background: var(--primary-500);
    color: var(--text-on-primary);
    border: none;
    cursor: pointer;
    outline: none;
    box-shadow: var(--shadow-sm);
    transition:
      background-color var(--duration-normal) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out),
      opacity var(--duration-fast) var(--ease-out);
  }
  .wwm-start:hover:not(:disabled) {
    background: var(--primary-600);
    box-shadow: var(--shadow-primary);
  }
  .wwm-start:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }
  .wwm-start:disabled { opacity: 0.45; cursor: not-allowed; }

  /* ═══ Journey timeline — sentences on quiet paper ═══ */
  .wwm-journey {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .wwm-entry {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
  }

  .wwm-entry-dot {
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--text-tertiary);
    flex-shrink: 0;
    align-self: center;
  }
  .wwm-entry-ember .wwm-entry-dot { background: var(--primary-500); }
  .wwm-entry-note  .wwm-entry-dot { background: var(--warning-500); }
  .wwm-entry-sage  .wwm-entry-dot { background: var(--success-500); }

  .wwm-entry-text {
    flex: 1;
    font-size: var(--text-base);
    line-height: 1.5;
    color: var(--text-primary);
    overflow-wrap: break-word;
    min-width: 0;
  }
  .wwm-entry-note .wwm-entry-text { color: var(--text-secondary); }

  .wwm-entry-time {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  /* ═══ Gentle reassurance row — share the walk ═══ */
  .wwm-ping {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-5);
  }

  .wwm-ping-pebble {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-full);
    background: var(--primary-100);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .wwm-ping-copy {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .wwm-ping-title {
    font-size: var(--text-base);
    font-weight: 500;
    line-height: 1.3;
    color: var(--text-primary);
  }

  .wwm-ping-cap {
    font-size: var(--text-xs);
    line-height: 1.4;
    color: var(--text-tertiary);
  }

  .wwm-share {
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-md);
    font-family: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    min-height: 44px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    background: var(--surface-2);
    border: 1px solid var(--hairline);
    color: var(--primary-700);
    cursor: pointer;
    outline: none;
    transition: background-color var(--duration-fast) var(--ease-out);
  }
  .wwm-share:hover { background: var(--primary-100); }
  .wwm-share:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  /* ═══ How this ends — the sage promise ═══ */
  .wwm-ends { margin-bottom: var(--space-5); }

  .wwm-ends-head {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }

  .wwm-ends-icon {
    color: var(--success-500);
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .wwm-ends-title {
    flex: 1;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
  }

  .wwm-tag {
    font-size: var(--text-2xs);
    font-weight: 600;
    letter-spacing: 0.04em;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-full);
    background: var(--success-500-20);
    color: var(--success-700);
  }

  .wwm-ends-body {
    font-size: var(--text-base);
    line-height: 1.55;
    color: var(--text-secondary);
    margin: 0;
  }

  /* ═══ Session controls — quiet paper, sage check ═══ */
  .wwm-controls { display: flex; }

  .wwm-arrive {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-button);
    font-family: inherit;
    font-size: var(--text-base);
    font-weight: 600;
    min-height: 48px;
    background: transparent;
    border: 1px solid var(--border-strong);
    color: var(--text-primary);
    cursor: pointer;
    outline: none;
    transition: background-color var(--duration-fast) var(--ease-out);
  }
  .wwm-arrive svg { color: var(--success-500); flex-shrink: 0; }
  .wwm-arrive:hover { background: var(--surface-hover); }
  .wwm-arrive:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  /* ═══ Arrived — the settled moment ═══ */
  .wwm-arrived {
    text-align: center;
    padding: var(--space-4) var(--space-2) var(--space-2);
    /* single gentle entry, then still — transform/opacity only */
    animation: wwm-arrived-enter var(--duration-slow) var(--ease-out) both;
  }

  @keyframes wwm-arrived-enter {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .wwm-arrived-pebble {
    width: 64px;
    height: 64px;
    border-radius: var(--radius-full);
    margin: 0 auto var(--space-4);
    background: var(--success-500-20);
    color: var(--success-600);
    display: flex;
    align-items: center;
    justify-content: center;
    /* one pop on entry — never loops */
    animation: wwm-arrived-pop var(--duration-slow) var(--ease-spring) both;
  }

  @keyframes wwm-arrived-pop {
    0%   { transform: scale(0.7); }
    60%  { transform: scale(1.08); }
    100% { transform: scale(1); }
  }

  .wwm-done {
    margin-top: var(--space-5);
    padding: var(--space-3) var(--space-8);
    border-radius: var(--radius-button);
    min-height: 48px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-family: inherit;
    font-size: var(--text-base);
    font-weight: 600;
    background: var(--success-500-08);
    border: 1px solid var(--success-500-28);
    color: var(--text-primary);
    cursor: pointer;
    outline: none;
    transition: background-color var(--duration-fast) var(--ease-out);
  }
  .wwm-done:hover { background: var(--success-500-20); }
  .wwm-done:focus-visible {
    outline: 2px solid var(--success-400);
    outline-offset: 2px;
  }

  /* ═══ Reduced motion — everything lands at its final state ═══ */
  @media (prefers-reduced-motion: reduce) {
    .wwm-step-bead,
    .wwm-step-label,
    .wwm-step + .wwm-step::before,
    .wwm-place,
    .wwm-watcher,
    .wwm-pebble,
    .wwm-input,
    .wwm-start,
    .wwm-share,
    .wwm-arrive,
    .wwm-done { transition: none; }

    .wwm-live-halo { animation: none; opacity: 0; }
    .wwm-arrived { animation: none; }
    .wwm-arrived-pebble { animation: none; }
  }
</style>
