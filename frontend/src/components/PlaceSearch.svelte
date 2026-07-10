<script>
  /**
   * PlaceSearch — search, mode-aware preview, and live turn-by-turn navigation.
   *
   * States:
   *   'search'     — blended search bar + dropdown (Saved / Recent / Places)
   *   'preview'    — place selected, mode tabs + steps + Start button
   *   'navigating' — compact next-turn HUD with live follower ETA
   */
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { searchPlaces, getDirections, formatDuration, formatDist } from '../lib/geocode.js';
  import { myLocation, navigationState, navFollow } from '../lib/stores/map.js';
  import { haptics } from '../lib/haptics.js';
  import { savedPlaces as savedPlacesStore } from '../lib/stores/places.js';
  import { searchHistory } from '../lib/stores/searchHistory.js';
  import { createFollower } from '../lib/navigation.js';
  import { toasts } from '../lib/stores/toast.js';
  import SearchResults from './placeSearch/SearchResults.svelte';

  const dispatch = createEventDispatcher();

  // ── Constants ─────────────────────────────────────────────────────────────
  const SCOOTER_D = 'M5 17a2 2 0 1 0 4 0 2 2 0 0 0-4 0m10 0a2 2 0 1 0 4 0 2 2 0 0 0-4 0M7 17h2m5 0h3m-3 0V8.5l-4-3H5.5A1.5 1.5 0 0 0 4 7v3h3M14 5h3l3 3';
  const MODES = [
    { id: 'car',     label: 'Car' },
    { id: 'foot',    label: 'Walk' },
    { id: 'bike',    label: 'Cycle' },
    { id: 'scooter', label: 'Scooter', svgD: SCOOTER_D },
  ];

  // ── State machine ──────────────────────────────────────────────────────────
  let view = $state('search');

  // Search
  let query = $state('');
  let results = $state([]);
  let loading = $state(false);
  let open = $state(false);
  let inputEl = $state();
  let debounceTimer;
  let highlightIdx = $state(-1);

  // Place + directions
  let selectedPlace = $state(null);
  let mode = $state('car');
  let route = $state(null);
  let allRoutes = $state({});
  let loadingMode = $state(null);

  // Navigation
  let currentStepIdx = $state(0);
  let liveEta = $state(null);
  let liveRemainingM = $state(null);
  let liveDistanceToStepM = $state(null); // live distance to the upcoming step maneuver
  let _prevNavStepIdx = 0;               // tracks step changes for haptic feedback
  let locUnsub = null;
  let follower = null;

  // Derived steps
  let currentStep = $derived(route?.steps?.[currentStepIdx] || null);
  let nextStep = $derived(route?.steps?.[currentStepIdx + 1] || null);
  let isLastStep = $derived(currentStepIdx >= (route?.steps?.length || 1) - 1);

  // ── Blended search sections ───────────────────────────────────────────────
  let savedMatches = $derived.by(() => {
    const sp = $savedPlacesStore;
    if (!Array.isArray(sp) || !sp.length) return [];
    const q = query.toLowerCase().trim();
    const filtered = q ? sp.filter(p => p.name?.toLowerCase().includes(q)) : sp.slice(0, 3);
    return filtered.slice(0, 3).map(p => ({ _source: 'saved', name: p.name, sub: '', lat: p.lat, lng: p.lng, icon: p.icon || 'pin' }));
  });

  let recentMatches = $derived.by(() => {
    const q = query.toLowerCase().trim();
    const hist = $searchHistory;
    const filtered = q ? hist.filter(h => h.name?.toLowerCase().includes(q)) : hist.slice(0, 5);
    return filtered.slice(0, 5).map(h => ({ _source: 'recent', name: h.name, sub: h.sub || '', lat: h.lat, lng: h.lng, mode: h.mode, ts: h.ts }));
  });

  let flatItems = $derived([
    ...savedMatches,
    ...recentMatches,
    ...results.map(r => ({ ...r, _source: 'place' })),
  ]);

  // ── Search ────────────────────────────────────────────────────────────────
  function onInput() {
    clearTimeout(debounceTimer);
    highlightIdx = -1;
    if (selectedPlace) reset();
    if (query.length < 2) { results = []; return; }
    loading = true;
    debounceTimer = setTimeout(doSearch, 180);
  }

  // Persist the freshest fix so searches before GPS lock (or with location
  // off) can still be proximity-ranked from the last known position.
  $effect(() => {
    if ($myLocation?.latitude) {
      try {
        localStorage.setItem('kinnect_last_fix',
          JSON.stringify({ lat: $myLocation.latitude, lng: $myLocation.longitude }));
      } catch (_) {}
    }
  });

  function searchBias() {
    if ($myLocation?.latitude) return { lat: $myLocation.latitude, lng: $myLocation.longitude };
    try {
      const last = JSON.parse(localStorage.getItem('kinnect_last_fix') || 'null');
      if (last?.lat) return last;
    } catch (_) {}
    return null;
  }

  async function doSearch() {
    if (query.length < 2) { loading = false; return; }
    const opts = {};
    const bias = searchBias();
    if (bias) { opts.lat = bias.lat; opts.lng = bias.lng; }
    results = await searchPlaces(query, opts);
    loading = false;
  }

  function onFocus() { if (view === 'search') open = true; }
  function onBlur() { setTimeout(() => { open = false; }, 220); }

  function onKeydown(e) {
    if (e.key === 'Escape') { if (view !== 'search') reset(); else { open = false; inputEl?.blur(); } return; }
    if (!open || !flatItems.length) return;
    if (e.key === 'ArrowDown') { e.preventDefault(); highlightIdx = Math.min(highlightIdx + 1, flatItems.length - 1); }
    if (e.key === 'ArrowUp')   { e.preventDefault(); highlightIdx = Math.max(highlightIdx - 1, 0); }
    if (e.key === 'Enter' && highlightIdx >= 0) { e.preventDefault(); selectItem(flatItems[highlightIdx]); }
  }

  // ── Place selection ───────────────────────────────────────────────────────
  function selectItem(item) {
    const storedMode = item._source === 'recent' ? item.mode : null;
    selectPlace({ name: item.name, sub: item.sub || '', lat: item.lat, lng: item.lng, type: item.type || '' })
      .then(() => { if (storedMode && storedMode !== mode) switchMode(storedMode); });
  }

  async function selectPlace(r) {
    selectedPlace = r;
    query = r.name;
    open = false;
    results = [];
    highlightIdx = -1;
    allRoutes = {};
    route = null;
    view = 'preview';
    dispatch('select', { lat: r.lat, lng: r.lng, name: r.name });
    inputEl?.blur();
    searchHistory.add({ name: r.name, sub: r.sub || '', lat: r.lat, lng: r.lng, mode });
    if ($myLocation?.latitude) {
      await fetchRoute('car');
      fetchRoute('foot');
    }
  }

  async function fetchRoute(m) {
    if (!selectedPlace || !$myLocation?.latitude) return null;
    if (allRoutes[m]) { switchMode(m); return allRoutes[m]; }
    loadingMode = m;
    const r = await getDirections($myLocation.latitude, $myLocation.longitude, selectedPlace.lat, selectedPlace.lng, m);
    loadingMode = null;
    if (!r) return null;
    allRoutes[m] = r;
    if (m === mode || !route) { mode = m; route = r; emitRoute(); }
    allRoutes = allRoutes;
    return r;
  }

  function switchMode(m) {
    mode = m;
    if (allRoutes[m]) { route = allRoutes[m]; emitRoute(); }
    else fetchRoute(m);
  }

  function emitRoute() {
    if (route?.geometry) dispatch('route', { geometry: route.geometry, dest: selectedPlace, duration: route.duration, distance: route.distance });
  }

  // ── Navigation + follower ─────────────────────────────────────────────────
  function attachFollower(r) {
    follower?.stop();
    _prevNavStepIdx = 0;
    follower = createFollower({
      geometry: r.geometry,
      durationS: r.duration,
      distanceM: r.distance,
      mode,
      steps: r.steps,
      onUpdate: u => {
        liveEta = u.etaS;
        liveRemainingM = u.remainingDistanceM;
        liveDistanceToStepM = u.distanceToStepM;
        // Auto-advance the current step and fire haptic on change
        if (u.currentStepIndex !== _prevNavStepIdx) {
          _prevNavStepIdx = u.currentStepIndex;
          currentStepIdx = u.currentStepIndex;
          try { haptics.tap(); } catch (_) {}
        }
        navFollow.set({ active: true, lat: u.snappedLat, lng: u.snappedLng, bearing: u.bearing });
      },
      onOffRoute: async () => { const nr = await fetchRoute(mode); if (nr) attachFollower(nr); },
      onArrive: () => { toasts.success(`Arrived at ${selectedPlace?.name || 'destination'}`); stopNav(); },
    });
  }

  function startNav() {
    view = 'navigating';
    currentStepIdx = 0;
    _prevNavStepIdx = 0;
    liveEta = null;
    liveRemainingM = null;
    liveDistanceToStepM = null;
    navigationState.set({ active: true, destLat: selectedPlace.lat, destLng: selectedPlace.lng, destName: selectedPlace.name, routeCoords: route?.geometry?.coordinates || [] });
    // Immediately zoom-in to the user's current position so the map snaps to nav
    // view before the first follower update arrives.
    const initLoc = $myLocation;
    if (initLoc?.latitude) {
      navFollow.set({ active: true, lat: initLoc.latitude, lng: initLoc.longitude, bearing: 0 });
    }
    if (route?.geometry) attachFollower(route);
    locUnsub = myLocation.subscribe(loc => {
      if (loc?.latitude && follower) follower.feed({ lat: loc.latitude, lng: loc.longitude, timestamp: Date.now() });
    });
  }

  function cleanupNav() {
    locUnsub?.(); locUnsub = null;
    follower?.stop(); follower = null;
    liveEta = null; liveRemainingM = null; liveDistanceToStepM = null;
    navFollow.set({ active: false, lat: 0, lng: 0, bearing: 0 });
  }

  function stopNav() {
    view = 'preview';
    currentStepIdx = 0;
    navigationState.set({ active: false });
    cleanupNav();
  }

  function nextTurn() { if (!isLastStep) currentStepIdx++; }
  function prevTurn() { if (currentStepIdx > 0) currentStepIdx--; }

  // ── Reset ─────────────────────────────────────────────────────────────────
  function reset() {
    query = ''; results = []; open = false; selectedPlace = null;
    route = null; allRoutes = {}; view = 'search'; currentStepIdx = 0; highlightIdx = -1;
    navigationState.set({ active: false });
    cleanupNav();
    dispatch('clearRoute');
    inputEl?.focus();
  }

  onDestroy(() => { clearTimeout(debounceTimer); cleanupNav(); });

  // ── Turn icons ────────────────────────────────────────────────────────────
  function turnIcon(text) {
    if (!text) return '→';
    const t = text.toLowerCase();
    if (t.includes('left')) return '↰';
    if (t.includes('right')) return '↱';
    if (t.includes('arrive')) return '◉';
    if (t.includes('head') || t.includes('depart')) return '↑';
    if (t.includes('straight') || t.includes('continue')) return '↑';
    if (t.includes('roundabout')) return '↻';
    if (t.includes('u-turn') || t.includes('uturn')) return '↩';
    return '→';
  }

  function turnIconLarge(text) {
    if (!text) return '→';
    const t = text.toLowerCase();
    if (t.includes('slight left')) return '↖';
    if (t.includes('sharp left')) return '↰';
    if (t.includes('left')) return '←';
    if (t.includes('slight right')) return '↗';
    if (t.includes('sharp right')) return '↱';
    if (t.includes('right')) return '→';
    if (t.includes('arrive')) return '⬤';
    if (t.includes('roundabout')) return '↻';
    if (t.includes('u-turn') || t.includes('uturn')) return '↩';
    return '↑';
  }
</script>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- NAVIGATING: compact next-turn HUD at top                              -->
<!-- ════════════════════════════════════════════════════════════════════════ -->
{#if view === 'navigating' && currentStep}
  <div class="nav-hud">
    <div class="nav-turn-card">
      <div class="nav-turn-icon">{turnIconLarge(currentStep.instruction)}</div>
      <div class="nav-turn-body">
        <span class="nav-turn-dist">{#if liveDistanceToStepM != null && liveDistanceToStepM > 10}in {formatDist(liveDistanceToStepM)}{:else if currentStep.distance > 10}{formatDist(currentStep.distance)}{/if}</span>
        <span class="nav-turn-text">{currentStep.instruction}</span>
      </div>
    </div>
    {#if nextStep}
      <div class="nav-then">Then: {turnIcon(nextStep.instruction)} {nextStep.instruction?.split(' ').slice(0, 4).join(' ')}</div>
    {/if}
    <div class="nav-bottom">
      <div class="nav-eta">
        <span class="nav-eta-time">{formatDuration(liveEta ?? route?.duration)}</span>
        <span class="nav-eta-dist">{formatDist(liveRemainingM ?? route?.distance)} · {selectedPlace?.name || ''}</span>
      </div>
      <div class="nav-controls">
        <button class="nav-ctrl" onclick={prevTurn} disabled={currentStepIdx === 0} aria-label="Previous step">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <span class="nav-step-count">{currentStepIdx + 1}/{route.steps.length}</span>
        <button class="nav-ctrl" onclick={nextTurn} disabled={isLastStep} aria-label="Next step">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
        <button class="nav-stop" onclick={stopNav}>Exit</button>
      </div>
    </div>
  </div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SEARCH + PREVIEW: standard flow                                        -->
<!-- ════════════════════════════════════════════════════════════════════════ -->
{:else}
  <div class="ps-wrap">
    <!-- Search bar -->
    <div class="ps-bar">
      {#if view === 'preview'}
        <button class="ps-back" onclick={reset} aria-label="Back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
      {:else}
        <svg class="ps-icon" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      {/if}
      <input
        bind:this={inputEl}
        bind:value={query}
        oninput={onInput}
        onfocus={onFocus}
        onblur={onBlur}
        onkeydown={onKeydown}
        class="ps-input"
        type="text"
        placeholder={view === 'preview' ? selectedPlace?.name : "Search places, addresses..."}
        autocomplete="off"
        spellcheck="false"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={open && flatItems.length > 0}
      />
      {#if loading || loadingMode}<div class="ps-spinner"></div>{/if}
      {#if query && view === 'search'}
        <button class="ps-clear" onclick={reset} aria-label="Clear">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      {/if}
    </div>

    <!-- Blended dropdown: Saved / Recent / Places -->
    {#if open && flatItems.length > 0}
      <SearchResults
        {savedMatches}
        {recentMatches}
        placeResults={results.map(r => ({ ...r, _source: 'place' }))}
        {highlightIdx}
        onselect={selectItem}
      />
    {/if}

    <!-- Preview: mode tabs + summary + steps + actions -->
    {#if view === 'preview' && $myLocation?.latitude}
      <div class="ps-nav">
        <!-- Mode tabs (car / walk / cycle / scooter) -->
        <div class="ps-modes">
          {#each MODES as m}
            <button class="ps-mode" class:ps-mode-on={mode === m.id} class:ps-mode-loading={loadingMode === m.id} onclick={() => switchMode(m.id)}>
              {#if m.svgD}
                <svg class="ps-mode-svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d={m.svgD}/></svg>
              {/if}
              <span class="ps-mode-label">{m.label}</span>
              {#if allRoutes[m.id]}
                <span class="ps-mode-eta">{formatDuration(allRoutes[m.id].duration)}</span>
              {:else if loadingMode === m.id}
                <span class="ps-mode-eta">...</span>
              {/if}
            </button>
          {/each}
        </div>

        {#if route}
          <!-- Summary -->
          <div class="ps-summary">
            <span class="ps-summary-time">{formatDuration(route.duration)}</span>
            <span class="ps-summary-dist">({formatDist(route.distance)})</span>
            {#if route.summary}<span class="ps-summary-via">via {route.summary}</span>{/if}
          </div>

          <!-- Steps -->
          {#if route.steps?.length}
            <ol class="ps-steps">
              {#each route.steps as step}
                <li class="ps-step">
                  <span class="ps-step-icon">{turnIcon(step.instruction)}</span>
                  <div class="ps-step-body">
                    <span class="ps-step-text">{step.instruction}</span>
                    {#if step.distance > 10}<span class="ps-step-meta">{formatDist(step.distance)}</span>{/if}
                  </div>
                </li>
              {/each}
            </ol>
          {/if}

          <!-- Actions -->
          <div class="ps-actions">
            <button class="ps-btn ps-btn-start" onclick={startNav}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z"/></svg>
              Start
            </button>
            <button class="ps-btn ps-btn-walk" onclick={() => dispatch('setDestination', selectedPlace)}>
              Walk With Me
            </button>
          </div>
          <p class="ps-attr">Routes &copy; OSM contributors &middot; FOSSGIS/Ola</p>
        {:else if loadingMode}
          <div class="ps-loading"><div class="ps-spinner"></div> Getting directions...</div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  /* ══════════════════════════════════════════════════════════════════════ */
  /* NAVIGATION HUD                                                        */
  /* ══════════════════════════════════════════════════════════════════════ */
  .nav-hud {
    background: rgba(5,8,18,0.94);
    backdrop-filter: blur(24px); -webkit-backdrop-filter: blur(24px);
    border-radius: 16px; overflow: hidden;
    box-shadow: 0 8px 32px rgba(0,0,0,0.5);
    border: 1px solid rgba(255,255,255,0.08);
    width: min(380px, calc(100vw - 24px));
  }
  .nav-turn-card {
    display: flex; align-items: center; gap: 14px;
    padding: 14px 16px 10px;
    background: rgba(59,130,246,0.12);
    border-bottom: 1px solid rgba(59,130,246,0.15);
  }
  .nav-turn-icon, .ps-step-icon {
    font-family: 'Apple Color Emoji', 'Segoe UI Symbol', 'Noto Sans Symbols', system-ui, sans-serif;
  }
  .nav-turn-icon {
    width: 44px; height: 44px; border-radius: 12px;
    background: #3b82f6; color: #fff;
    font-size: clamp(1.25rem, 1.6vw, 1.375rem); font-weight: 700;
    display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  }
  .nav-turn-body { flex: 1; min-width: 0; }
  .nav-turn-dist { display: block; font-size: 18px; font-weight: 800; color: #fff; font-family: var(--font-display, system-ui); letter-spacing: -0.02em; }
  .nav-turn-text { display: block; font-size: 12px; color: rgba(255,255,255,0.65); line-height: 1.3; margin-top: 1px; }
  .nav-then { padding: 6px 16px; font-size: 11px; color: rgba(255,255,255,0.30); border-bottom: 1px solid rgba(255,255,255,0.05); }
  .nav-bottom { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px 10px; }
  .nav-eta { flex: 1; min-width: 0; }
  .nav-eta-time { font-size: 14px; font-weight: 800; color: #10b981; font-family: var(--font-display, system-ui); }
  .nav-eta-dist { display: block; font-size: 10px; color: rgba(255,255,255,0.25); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .nav-controls { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
  .nav-ctrl { width: 44px; height: 44px; border-radius: 8px; border: none; background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.5); display: flex; align-items: center; justify-content: center; cursor: pointer; touch-action: manipulation; -webkit-tap-highlight-color: transparent; }
  .nav-ctrl:hover { background: rgba(255,255,255,0.10); color: #fff; }
  .nav-ctrl:disabled { opacity: 0.2; cursor: default; }
  .nav-step-count { font-size: 10px; font-weight: 700; color: rgba(255,255,255,0.25); min-width: 28px; text-align: center; }
  .nav-stop { padding: 6px 12px; border-radius: 8px; font-size: 11px; font-weight: 700; background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.22); color: #f87171; cursor: pointer; margin-left: 4px; min-height: 44px; touch-action: manipulation; -webkit-tap-highlight-color: transparent; }
  .nav-stop:hover { background: rgba(239,68,68,0.22); }

  /* ══════════════════════════════════════════════════════════════════════ */
  /* SEARCH + PREVIEW                                                      */
  /* ══════════════════════════════════════════════════════════════════════ */
  .ps-wrap { position: relative; width: min(380px, calc(100vw - 24px)); z-index: 20; }

  .ps-bar { display: flex; align-items: center; gap: 6px; background: rgba(5,8,18,0.88); backdrop-filter: blur(24px); -webkit-backdrop-filter: blur(24px); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; padding: 9px 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.3); }
  .ps-bar:focus-within { border-color: rgba(99,102,241,0.35); box-shadow: 0 4px 20px rgba(0,0,0,0.3), 0 0 0 3px rgba(99,102,241,0.08); }
  .ps-icon { color: rgba(255,255,255,0.30); flex-shrink: 0; }
  .ps-back { display: flex; align-items: center; justify-content: center; background: none; border: none; color: rgba(255,255,255,0.5); cursor: pointer; flex-shrink: 0; padding: 0; min-width: 44px; min-height: 44px; touch-action: manipulation; -webkit-tap-highlight-color: transparent; }
  .ps-back:hover { color: #fff; }
  .ps-input { flex: 1; background: none; border: none; outline: none; color: rgba(255,255,255,0.90); font-size: 16px; font-weight: 500; font-family: inherit; min-width: 0; }
  .ps-input::placeholder { color: rgba(255,255,255,0.22); }
  .ps-clear { display: flex; align-items: center; justify-content: center; background: transparent; border: none; border-radius: 50%; min-width: 44px; min-height: 44px; cursor: pointer; color: rgba(255,255,255,0.4); flex-shrink: 0; touch-action: manipulation; -webkit-tap-highlight-color: transparent; }
  .ps-clear svg { background: rgba(255,255,255,0.06); border-radius: 50%; padding: 5px; box-sizing: content-box; }
  .ps-clear:hover svg { background: rgba(255,255,255,0.12); }
  .ps-spinner { width: 14px; height: 14px; border: 2px solid rgba(99,102,241,0.25); border-top-color: rgba(99,102,241,0.8); border-radius: 50%; animation: ps-spin 0.5s linear infinite; flex-shrink: 0; }
  @keyframes ps-spin { to { transform: rotate(360deg); } }

  /* Nav panel */
  .ps-nav { margin-top: 4px; background: rgba(8,12,24,0.96); backdrop-filter: blur(24px); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; box-shadow: 0 12px 40px rgba(0,0,0,0.5); overflow: hidden; }
  .ps-modes { display: flex; border-bottom: 1px solid rgba(255,255,255,0.06); }
  .ps-mode { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 1px; padding: 10px 4px 8px; min-height: 44px; border: none; background: transparent; color: rgba(255,255,255,0.30); cursor: pointer; position: relative; transition: all 0.15s; touch-action: manipulation; -webkit-tap-highlight-color: transparent; }
  .ps-mode:hover { color: rgba(255,255,255,0.55); background: rgba(255,255,255,0.03); }
  .ps-mode-on { color: var(--primary-300, #a5b4fc) !important; background: rgba(99,102,241,0.08) !important; }
  .ps-mode-on::after { content: ''; position: absolute; bottom: 0; left: 20%; right: 20%; height: 2px; background: var(--primary-400, #818cf8); border-radius: 2px; }
  .ps-mode-loading { opacity: 0.4; }
  .ps-mode-eta { font-size: 11px; font-weight: 800; }
  .ps-mode-label { font-size: 10px; font-weight: 600; opacity: 0.5; }
  .ps-mode-svg { opacity: 0.7; }

  .ps-summary { padding: 10px 14px 6px; display: flex; align-items: baseline; gap: 6px; flex-wrap: wrap; }
  .ps-summary-time { font-size: clamp(1.25rem, 1.6vw, 1.375rem); font-weight: 800; color: #fff; font-family: var(--font-display, system-ui); letter-spacing: -0.03em; }
  .ps-summary-dist { font-size: 13px; color: rgba(255,255,255,0.35); }
  .ps-summary-via { width: 100%; font-size: 11px; color: rgba(255,255,255,0.20); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .ps-steps { list-style: none; margin: 0; padding: 0 10px; max-height: 180px; overflow-y: auto; border-top: 1px solid rgba(255,255,255,0.05); scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.06) transparent; }
  .ps-step { display: flex; align-items: center; gap: 10px; padding: 8px 4px; min-height: 44px; border-bottom: 1px solid rgba(255,255,255,0.03); }
  .ps-step:last-child { border-bottom: none; }
  .ps-step-icon { width: 22px; height: 22px; border-radius: 50%; flex-shrink: 0; background: rgba(99,102,241,0.10); color: var(--primary-300, #a5b4fc); font-size: 12px; display: flex; align-items: center; justify-content: center; margin-top: 1px; }
  .ps-step-body { flex: 1; min-width: 0; }
  .ps-step-text { font-size: 12px; color: rgba(255,255,255,0.70); display: block; line-height: 1.4; }
  .ps-step-meta { font-size: 10px; color: rgba(255,255,255,0.22); }

  .ps-actions { display: flex; gap: 6px; padding: 8px 10px 4px; border-top: 1px solid rgba(255,255,255,0.05); }
  .ps-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px; padding: 11px 8px; min-height: 44px; border-radius: 12px; font-size: 13px; font-weight: 700; border: none; cursor: pointer; -webkit-tap-highlight-color: transparent; touch-action: manipulation; }
  .ps-btn:active { transform: scale(0.97); }
  .ps-btn-start { background: #3b82f6; color: #fff; box-shadow: 0 2px 12px rgba(59,130,246,0.35); }
  .ps-btn-start:hover { background: #2563eb; }
  .ps-btn-walk { background: rgba(99,102,241,0.12); color: var(--primary-300, #a5b4fc); border: 1px solid rgba(99,102,241,0.22); }
  .ps-btn-walk:hover { background: rgba(99,102,241,0.20); }

  .ps-attr { margin: 0; padding: 5px 12px 8px; font-size: 9px; color: rgba(255,255,255,0.15); text-align: center; }

  .ps-loading { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 16px; font-size: 12px; color: rgba(255,255,255,0.30); }
</style>
