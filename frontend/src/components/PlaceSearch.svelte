<script>
  /**
   * PlaceSearch — Google Maps-style search + in-app navigation.
   *
   * States:
   *   'search'    — search bar + autocomplete
   *   'preview'   — place selected, mode tabs + steps + Start button
   *   'navigating'— compact next-turn HUD, map has full space
   */
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { searchPlaces, getDirections, formatDuration, formatDist } from '../lib/geocode.js';
  import { myLocation, navigationState } from '../lib/stores/map.js';

  const dispatch = createEventDispatcher();

  // ── State machine ──────────────────────────────────────────────────────
  let view = 'search'; // 'search' | 'preview' | 'navigating'

  // Search
  let query = '';
  let results = [];
  let loading = false;
  let open = false;
  let inputEl;
  let debounceTimer;
  let highlightIdx = -1;

  // Place + directions
  let selectedPlace = null;
  let mode = 'car';
  let route = null;
  let allRoutes = {};
  let loadingMode = null;

  // Navigation
  let currentStepIdx = 0;
  $: currentStep = route?.steps?.[currentStepIdx] || null;
  $: nextStep = route?.steps?.[currentStepIdx + 1] || null;
  $: isLastStep = currentStepIdx >= (route?.steps?.length || 1) - 1;

  // ── Search ─────────────────────────────────────────────────────────────
  function onInput() {
    clearTimeout(debounceTimer);
    highlightIdx = -1;
    if (selectedPlace) reset();
    if (query.length < 2) { results = []; open = false; return; }
    loading = true;
    debounceTimer = setTimeout(doSearch, 180);
  }

  async function doSearch() {
    if (query.length < 2) { loading = false; return; }
    const opts = {};
    if ($myLocation?.latitude) { opts.lat = $myLocation.latitude; opts.lng = $myLocation.longitude; }
    results = await searchPlaces(query, opts);
    loading = false;
    open = results.length > 0;
  }

  function onFocus() { if (results.length > 0 && view === 'search') open = true; }
  function onBlur() { setTimeout(() => { open = false; }, 220); }
  function onKeydown(e) {
    if (e.key === 'Escape') { if (view !== 'search') reset(); else { open = false; inputEl?.blur(); } return; }
    if (!open || !results.length) return;
    if (e.key === 'ArrowDown') { e.preventDefault(); highlightIdx = Math.min(highlightIdx + 1, results.length - 1); }
    if (e.key === 'ArrowUp') { e.preventDefault(); highlightIdx = Math.max(highlightIdx - 1, 0); }
    if (e.key === 'Enter' && highlightIdx >= 0) { e.preventDefault(); selectPlace(results[highlightIdx]); }
  }

  // ── Place selection ────────────────────────────────────────────────────
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
    if ($myLocation?.latitude) {
      await fetchRoute('car');
      fetchRoute('foot'); // background
    }
  }

  async function fetchRoute(m) {
    if (!selectedPlace || !$myLocation?.latitude) return;
    if (allRoutes[m]) { switchMode(m); return; }
    loadingMode = m;
    const r = await getDirections($myLocation.latitude, $myLocation.longitude, selectedPlace.lat, selectedPlace.lng, m);
    loadingMode = null;
    if (!r) return;
    allRoutes[m] = r;
    if (m === mode || !route) { mode = m; route = r; emitRoute(); }
    allRoutes = allRoutes;
  }

  function switchMode(m) {
    mode = m;
    if (allRoutes[m]) { route = allRoutes[m]; emitRoute(); }
    else fetchRoute(m);
  }

  function emitRoute() {
    if (route?.geometry) dispatch('route', { geometry: route.geometry, dest: selectedPlace, duration: route.duration, distance: route.distance });
  }

  // ── Navigation ─────────────────────────────────────────────────────────
  function startNav() {
    view = 'navigating';
    currentStepIdx = 0;
    navigationState.set({
      active: true,
      destLat: selectedPlace.lat,
      destLng: selectedPlace.lng,
      destName: selectedPlace.name,
      routeCoords: route?.geometry?.coordinates || [],
    });
  }

  function stopNav() {
    view = 'preview';
    currentStepIdx = 0;
    navigationState.set({ active: false });
  }

  function nextTurn() {
    if (!isLastStep) currentStepIdx++;
  }

  function prevTurn() {
    if (currentStepIdx > 0) currentStepIdx--;
  }

  // ── Reset ──────────────────────────────────────────────────────────────
  function reset() {
    query = '';
    results = [];
    open = false;
    selectedPlace = null;
    route = null;
    allRoutes = {};
    view = 'search';
    currentStepIdx = 0;
    highlightIdx = -1;
    navigationState.set({ active: false });
    dispatch('clearRoute');
    inputEl?.focus();
  }

  onDestroy(() => clearTimeout(debounceTimer));

  // ── Turn icons ─────────────────────────────────────────────────────────
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
        <span class="nav-turn-dist">{currentStep.distance > 10 ? formatDist(currentStep.distance) : ''}</span>
        <span class="nav-turn-text">{currentStep.instruction}</span>
      </div>
    </div>
    {#if nextStep}
      <div class="nav-then">Then: {turnIcon(nextStep.instruction)} {nextStep.instruction?.split(' ').slice(0, 4).join(' ')}</div>
    {/if}
    <div class="nav-bottom">
      <div class="nav-eta">
        <span class="nav-eta-time">{formatDuration(route.duration)}</span>
        <span class="nav-eta-dist">{formatDist(route.distance)} · {selectedPlace?.name || ''}</span>
      </div>
      <div class="nav-controls">
        <button class="nav-ctrl" on:click={prevTurn} disabled={currentStepIdx === 0} aria-label="Previous step">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <span class="nav-step-count">{currentStepIdx + 1}/{route.steps.length}</span>
        <button class="nav-ctrl" on:click={nextTurn} disabled={isLastStep} aria-label="Next step">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
        <button class="nav-stop" on:click={stopNav}>Exit</button>
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
        <button class="ps-back" on:click={reset} aria-label="Back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
      {:else}
        <svg class="ps-icon" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      {/if}
      <input
        bind:this={inputEl}
        bind:value={query}
        on:input={onInput}
        on:focus={onFocus}
        on:blur={onBlur}
        on:keydown={onKeydown}
        class="ps-input"
        type="text"
        placeholder={view === 'preview' ? selectedPlace?.name : "Search places, addresses..."}
        autocomplete="off"
        spellcheck="false"
      />
      {#if loading || loadingMode}<div class="ps-spinner"></div>{/if}
      {#if query && view === 'search'}
        <button class="ps-clear" on:click={reset} aria-label="Clear">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      {/if}
    </div>

    <!-- Autocomplete results -->
    {#if open && results.length > 0}
      <ul class="ps-results">
        {#each results as r, i}
          <li>
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <div class="ps-result" class:ps-result-hl={i === highlightIdx} on:click={() => selectPlace(r)}>
              <div class="ps-result-icon">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z"/><circle cx="12" cy="10" r="3"/></svg>
              </div>
              <div class="ps-result-text">
                <span class="ps-result-name">{r.name}</span>
                {#if r.sub}<span class="ps-result-sub">{r.sub}</span>{/if}
              </div>
            </div>
          </li>
        {/each}
      </ul>
    {/if}

    <!-- Preview: mode tabs + summary + steps + actions -->
    {#if view === 'preview' && $myLocation?.latitude}
      <div class="ps-nav">
        <!-- Mode tabs -->
        <div class="ps-modes">
          {#each [['car','Car'],['foot','Walk'],['bike','Cycle']] as [m, label]}
            <button class="ps-mode" class:ps-mode-on={mode === m} class:ps-mode-loading={loadingMode === m} on:click={() => switchMode(m)}>
              <span class="ps-mode-label">{label}</span>
              {#if allRoutes[m]}
                <span class="ps-mode-eta">{formatDuration(allRoutes[m].duration)}</span>
              {:else if loadingMode === m}
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
            {#if route.summary}
              <span class="ps-summary-via">via {route.summary}</span>
            {/if}
          </div>

          <!-- Steps (always visible, scrollable) -->
          {#if route.steps?.length}
            <ol class="ps-steps">
              {#each route.steps as step, i}
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
            <button class="ps-btn ps-btn-start" on:click={startNav}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z"/></svg>
              Start
            </button>
            <button class="ps-btn ps-btn-walk" on:click={() => dispatch('setDestination', selectedPlace)}>
              Walk With Me
            </button>
          </div>
        {:else if loadingMode}
          <div class="ps-loading"><div class="ps-spinner"></div> Getting directions...</div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  /* ══════════════════════════════════════════════════════════════════════ */
  /* NAVIGATION HUD — compact, Google Maps-style next-turn bar            */
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
  .nav-turn-icon {
    width: 44px; height: 44px; border-radius: 12px;
    background: #3b82f6; color: #fff;
    font-size: 22px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
  }
  .nav-turn-body { flex: 1; min-width: 0; }
  .nav-turn-dist {
    display: block; font-size: 18px; font-weight: 800; color: #fff;
    font-family: var(--font-display, system-ui); letter-spacing: -0.02em;
  }
  .nav-turn-text {
    display: block; font-size: 12px; color: rgba(255,255,255,0.65);
    line-height: 1.3; margin-top: 1px;
  }
  .nav-then {
    padding: 6px 16px; font-size: 11px; color: rgba(255,255,255,0.30);
    border-bottom: 1px solid rgba(255,255,255,0.05);
  }
  .nav-bottom {
    display: flex; align-items: center; justify-content: space-between;
    padding: 8px 12px 10px;
  }
  .nav-eta { flex: 1; min-width: 0; }
  .nav-eta-time {
    font-size: 14px; font-weight: 800; color: #10b981;
    font-family: var(--font-display, system-ui);
  }
  .nav-eta-dist {
    display: block; font-size: 10px; color: rgba(255,255,255,0.25);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .nav-controls { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
  .nav-ctrl {
    width: 32px; height: 32px; border-radius: 8px; border: none;
    background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.5);
    display: flex; align-items: center; justify-content: center;
    cursor: pointer;
  }
  .nav-ctrl:hover { background: rgba(255,255,255,0.10); color: #fff; }
  .nav-ctrl:disabled { opacity: 0.2; cursor: default; }
  .nav-step-count {
    font-size: 10px; font-weight: 700; color: rgba(255,255,255,0.25);
    min-width: 28px; text-align: center;
  }
  .nav-stop {
    padding: 6px 12px; border-radius: 8px; font-size: 11px; font-weight: 700;
    background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.22);
    color: #f87171; cursor: pointer; margin-left: 4px;
  }
  .nav-stop:hover { background: rgba(239,68,68,0.22); }

  /* ══════════════════════════════════════════════════════════════════════ */
  /* SEARCH + PREVIEW                                                      */
  /* ══════════════════════════════════════════════════════════════════════ */
  .ps-wrap { position: relative; width: min(380px, calc(100vw - 24px)); z-index: 20; }

  .ps-bar {
    display: flex; align-items: center; gap: 6px;
    background: rgba(5,8,18,0.88);
    backdrop-filter: blur(24px); -webkit-backdrop-filter: blur(24px);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 14px; padding: 9px 12px;
    box-shadow: 0 4px 20px rgba(0,0,0,0.3);
  }
  .ps-bar:focus-within { border-color: rgba(99,102,241,0.35); box-shadow: 0 4px 20px rgba(0,0,0,0.3), 0 0 0 3px rgba(99,102,241,0.08); }
  .ps-icon { color: rgba(255,255,255,0.30); flex-shrink: 0; }
  .ps-back { display: flex; align-items: center; background: none; border: none; color: rgba(255,255,255,0.5); cursor: pointer; flex-shrink: 0; padding: 0; }
  .ps-back:hover { color: #fff; }
  .ps-input { flex: 1; background: none; border: none; outline: none; color: rgba(255,255,255,0.90); font-size: 16px; font-weight: 500; font-family: inherit; min-width: 0; }
  .ps-input::placeholder { color: rgba(255,255,255,0.22); }
  .ps-clear { display: flex; align-items: center; justify-content: center; background: rgba(255,255,255,0.06); border: none; border-radius: 50%; width: 24px; height: 24px; cursor: pointer; color: rgba(255,255,255,0.4); flex-shrink: 0; }
  .ps-clear:hover { background: rgba(255,255,255,0.12); }
  .ps-spinner { width: 14px; height: 14px; border: 2px solid rgba(99,102,241,0.25); border-top-color: rgba(99,102,241,0.8); border-radius: 50%; animation: ps-spin 0.5s linear infinite; flex-shrink: 0; }
  @keyframes ps-spin { to { transform: rotate(360deg); } }

  .ps-results { position: absolute; top: calc(100% + 4px); left: 0; right: 0; background: rgba(8,12,24,0.96); backdrop-filter: blur(24px); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; padding: 4px; list-style: none; margin: 0; max-height: 300px; overflow-y: auto; box-shadow: 0 12px 40px rgba(0,0,0,0.5); }
  .ps-result { display: flex; align-items: flex-start; gap: 10px; padding: 10px 12px; border-radius: 10px; cursor: pointer; transition: background 0.1s; }
  .ps-result:hover, .ps-result-hl { background: rgba(99,102,241,0.10); }
  .ps-result:active { background: rgba(99,102,241,0.18); }
  .ps-result-icon { color: rgba(99,102,241,0.55); flex-shrink: 0; margin-top: 1px; }
  .ps-result-text { display: flex; flex-direction: column; gap: 1px; min-width: 0; flex: 1; }
  .ps-result-name { font-size: 13px; font-weight: 600; color: rgba(255,255,255,0.88); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ps-result-sub { font-size: 11px; color: rgba(255,255,255,0.30); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  /* Nav panel */
  .ps-nav { margin-top: 4px; background: rgba(8,12,24,0.96); backdrop-filter: blur(24px); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; box-shadow: 0 12px 40px rgba(0,0,0,0.5); overflow: hidden; }
  .ps-modes { display: flex; border-bottom: 1px solid rgba(255,255,255,0.06); }
  .ps-mode { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 1px; padding: 10px 4px 8px; border: none; background: transparent; color: rgba(255,255,255,0.30); cursor: pointer; position: relative; transition: all 0.15s; }
  .ps-mode:hover { color: rgba(255,255,255,0.55); background: rgba(255,255,255,0.03); }
  .ps-mode-on { color: var(--primary-300, #a5b4fc) !important; background: rgba(99,102,241,0.08) !important; }
  .ps-mode-on::after { content: ''; position: absolute; bottom: 0; left: 20%; right: 20%; height: 2px; background: var(--primary-400, #818cf8); border-radius: 2px; }
  .ps-mode-loading { opacity: 0.4; }
  .ps-mode-eta { font-size: 11px; font-weight: 800; }
  .ps-mode-label { font-size: 10px; font-weight: 600; opacity: 0.5; }

  .ps-summary { padding: 10px 14px 6px; display: flex; align-items: baseline; gap: 6px; flex-wrap: wrap; }
  .ps-summary-time { font-size: 22px; font-weight: 800; color: #fff; font-family: var(--font-display, system-ui); letter-spacing: -0.03em; }
  .ps-summary-dist { font-size: 13px; color: rgba(255,255,255,0.35); }
  .ps-summary-via { width: 100%; font-size: 11px; color: rgba(255,255,255,0.20); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .ps-steps { list-style: none; margin: 0; padding: 0 10px; max-height: 180px; overflow-y: auto; border-top: 1px solid rgba(255,255,255,0.05); scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.06) transparent; }
  .ps-step { display: flex; align-items: flex-start; gap: 10px; padding: 8px 4px; border-bottom: 1px solid rgba(255,255,255,0.03); }
  .ps-step:last-child { border-bottom: none; }
  .ps-step-icon { width: 22px; height: 22px; border-radius: 50%; flex-shrink: 0; background: rgba(99,102,241,0.10); color: var(--primary-300, #a5b4fc); font-size: 12px; display: flex; align-items: center; justify-content: center; margin-top: 1px; }
  .ps-step-body { flex: 1; min-width: 0; }
  .ps-step-text { font-size: 12px; color: rgba(255,255,255,0.70); display: block; line-height: 1.4; }
  .ps-step-meta { font-size: 10px; color: rgba(255,255,255,0.22); }

  .ps-actions { display: flex; gap: 6px; padding: 8px 10px 10px; border-top: 1px solid rgba(255,255,255,0.05); }
  .ps-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px; padding: 11px 8px; border-radius: 12px; font-size: 13px; font-weight: 700; border: none; cursor: pointer; -webkit-tap-highlight-color: transparent; }
  .ps-btn:active { transform: scale(0.97); }
  .ps-btn-start { background: #3b82f6; color: #fff; box-shadow: 0 2px 12px rgba(59,130,246,0.35); }
  .ps-btn-start:hover { background: #2563eb; }
  .ps-btn-walk { background: rgba(99,102,241,0.12); color: var(--primary-300, #a5b4fc); border: 1px solid rgba(99,102,241,0.22); }
  .ps-btn-walk:hover { background: rgba(99,102,241,0.20); }

  .ps-loading { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 16px; font-size: 12px; color: rgba(255,255,255,0.30); }
</style>
