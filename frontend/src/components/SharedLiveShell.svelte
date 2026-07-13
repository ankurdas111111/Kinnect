<script module>
  /**
   * SharedLiveShell — the deduplicated viewer chassis for LiveViewer + WatchViewer.
   *
   * Extracts the identical parts of both recipient-facing map pages:
   *   • MapLibre init with DEFAULT_CENTER / DEFAULT_ZOOM constants (was two
   *     divergent inline [78,20]/z5 and [78.9629,20.5937]/z4 literals)
   *   • glass bottom-card chrome (glass tier tokens, --glass-shadow) with a
   *     StatusBadge (replaces both bespoke .status-badge / .watch-status-badge
   *     CSS blocks) + SignalBars freshness chip
   *   • temporal-decay expiry strip (scaleX depletion, green→amber→red crossfade)
   *
   * connState is derived by the PARENT via lib/presence.js deriveConnState() and
   * passed in — the shell never re-derives it (single source of truth).
   *
   * CALM-TIER CONTRACT: recipients are unauthenticated users on unknown, possibly
   * low-capability devices. This shell forces ambient decoration OFF regardless of
   * the stored fx level — no page-level fx-ambient loops are started here. The only
   * motion is StatusBadge's own dot (1Hz-safe, self-gated inside the primitive) and
   * the functional scaleX expiry depletion (survives reduced-motion — it conveys
   * real state, so it stays as feedback).
   *
   * The parent owns all socket + marker logic. The shell exposes the created
   * `maplibregl.Map` via the `onMap` callback so the parent attaches markers.
   *
   * Layout slots (snippets):
   *   overlay — full-screen modals (name gate, expired card)
   *   header  — top glass header content (WatchViewer identity)
   *   cardExtras — extra rows rendered inside the bottom glass card (check-in,
   *                trailing sign-up link)
   *   docks   — bottom-anchored floating docks (SOS banner, narrative, brand,
   *             follow controls)
   *
   * `position="bottom"` (default) renders the shared glass status card at the
   * bottom (LiveViewer). `position="top"` renders the header slot instead
   * (WatchViewer keeps its large-name identity header).
   */

  /** Default map camera — shared by both viewers (was two divergent literals). */
  export const DEFAULT_CENTER = [78.9629, 20.5937];
  export const DEFAULT_ZOOM = 4;
</script>

<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { MAP_STYLE } from '../lib/mapStyle.js';
  import StatusBadge from './primitives/StatusBadge.svelte';
  import SignalBars from './primitives/SignalBars.svelte';

  /**
   * @type {{
   *   position?: 'bottom' | 'top',
   *   connState?: 'live' | 'connecting' | 'issue' | 'offline',
   *   statusLabel?: string,
   *   announce?: boolean,
   *   sosActive?: boolean,
   *   showSignal?: boolean,
   *   signalLevel?: 0 | 1 | 2 | 3,
   *   freshnessLabel?: string,
   *   expiryPercent?: number | null,
   *   center?: number[],
   *   zoom?: number,
   *   onMap?: (map: import('maplibre-gl').Map) => void,
   *   overlay?: import('svelte').Snippet,
   *   header?: import('svelte').Snippet,
   *   cardExtras?: import('svelte').Snippet,
   *   docks?: import('svelte').Snippet,
   * }}
   */
  let {
    position = 'bottom',
    connState = 'connecting',
    statusLabel = '',
    announce = true,
    sosActive = false,
    showSignal = false,
    signalLevel = 0,
    freshnessLabel = '',
    expiryPercent = null,
    center = DEFAULT_CENTER,
    zoom = DEFAULT_ZOOM,
    onMap = () => {},
    overlay,
    header,
    cardExtras,
    docks,
  } = $props();

  let mapContainer = $state();
  let map = null;

  // Temporal-decay color: green while healthy, amber under 25%, red under 10%.
  // A matching data-attribute drives text/opacity too, so state is never carried
  // by hue alone.
  let expiryStage = $derived(
    expiryPercent == null ? 'ok'
      : expiryPercent <= 10 ? 'critical'
      : expiryPercent <= 25 ? 'low'
      : 'ok'
  );

  onMount(() => {
    map = new maplibregl.Map({
      container: mapContainer,
      style: MAP_STYLE,
      center,
      zoom,
      attributionControl: true,
    });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right');
    onMap(map);
  });

  onDestroy(() => {
    if (map) map.remove();
    map = null;
  });
</script>

<div class="shell" class:sos-active={sosActive}>
  <div class="shell-map" bind:this={mapContainer}></div>

  {#if overlay}
    {@render overlay()}
  {/if}

  {#if position === 'top' && header}
    <!-- Top glass header slot (WatchViewer identity header). -->
    <div class="shell-header" class:sos-state={sosActive}>
      {@render header()}
    </div>
  {/if}

  {#if position === 'bottom'}
    <!-- Shared bottom glass status card (LiveViewer's MERIDIAN card, unified). -->
    <div class="shell-card" class:sos-state={sosActive}>
      <div class="shell-card-inner">
        <div class="shell-card-row">
          <StatusBadge state={connState} label={statusLabel} {announce} />
          {#if showSignal && freshnessLabel}
            <span class="shell-signal" class:stale={connState !== 'live'} title="Location freshness">
              <SignalBars level={signalLevel} />
              <span class="shell-signal-text">{freshnessLabel}</span>
            </span>
          {/if}
        </div>
        {#if cardExtras}
          {@render cardExtras()}
        {/if}
        {#if expiryPercent != null}
          <div
            class="shell-expiry"
            data-stage={expiryStage}
            role="progressbar"
            aria-label="Share time remaining"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={Math.round(expiryPercent)}
          >
            <div class="shell-expiry-fill" style="transform: scaleX({expiryPercent / 100})"></div>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  {#if docks}
    {@render docks()}
  {/if}
</div>

<style>
  .shell {
    position: relative;
    width: 100%;
    height: 100vh;
    height: 100dvh;
  }

  /* Emergency border — color + the banner text carry meaning; the pulse is a
     calm reinforcement that stops entirely for reduced-motion users. */
  .shell.sos-active {
    outline: var(--space-1) solid var(--status-sos);
    outline-offset: calc(-1 * var(--space-1));
    animation: shell-sos-pulse 2s var(--ease-in-out, ease-in-out) infinite;
  }

  @keyframes shell-sos-pulse {
    0%, 100% { outline-color: var(--status-sos); }
    50%      { outline-color: transparent; }
  }

  .shell-map {
    position: absolute;
    inset: 0;
    z-index: 1;
  }

  /* Top glass header (WatchViewer). Panel glass tier, calm-tier safe. */
  .shell-header {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 50;
    padding: calc(var(--space-3) + env(safe-area-inset-top, 0px)) var(--space-5) var(--space-3);
    background: var(--glass-panel-bg);
    backdrop-filter: var(--glass-panel-blur);
    -webkit-backdrop-filter: var(--glass-panel-blur);
    border-bottom: 1px solid var(--glass-panel-border);
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 64px;
    transition: background var(--duration-slower, 400ms) var(--ease-out);
  }

  .shell-header.sos-state {
    background: color-mix(in oklch, var(--status-sos) 88%, transparent);
    border-bottom-color: color-mix(in oklch, var(--text-inverse, white) 14%, transparent);
    box-shadow: 0 4px 28px color-mix(in oklch, var(--status-sos) 40%, transparent);
    color: var(--text-inverse, white);
  }

  /* Shared bottom glass status card (chip glass tier, --glass-shadow). */
  .shell-card {
    position: absolute;
    bottom: calc(var(--space-5) + env(safe-area-inset-bottom, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    width: min(92vw, 400px);
    background: var(--glass-panel-bg);
    backdrop-filter: var(--glass-panel-blur);
    -webkit-backdrop-filter: var(--glass-panel-blur);
    border: 1px solid var(--glass-panel-border);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
    box-shadow: var(--glass-shadow);
    color: var(--text-primary);
    animation: slide-up-in 380ms var(--ease-out) both;
    transition: bottom var(--duration-slow) var(--ease-out);
  }

  .shell-card.sos-state {
    bottom: calc(var(--space-3) + 96px + env(safe-area-inset-bottom, 0px));
    border-color: var(--danger-500-20);
    box-shadow: var(--glass-shadow), var(--glow-sos-sm);
  }

  .shell-card-inner {
    min-width: 0;
  }

  .shell-card-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .shell-signal {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-shrink: 0;
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--status-live);
  }
  .shell-signal.stale {
    color: var(--text-tertiary);
    font-weight: 500;
  }
  .shell-signal-text {
    white-space: nowrap;
  }

  /* Temporal-decay expiry strip — scaleX depletion (GPU-only), color crossfade
     at thresholds. Functional feedback, so it survives reduced-motion. */
  .shell-expiry {
    margin-top: var(--space-2);
    height: var(--space-1);
    background: color-mix(in oklch, var(--text-primary) 10%, transparent);
    border-radius: var(--radius-xs);
    overflow: hidden;
  }
  .shell-expiry-fill {
    height: 100%;
    width: 100%;
    transform-origin: left center;
    border-radius: var(--radius-xs);
    background: var(--status-live);
    transition:
      transform 1s linear,
      background var(--duration-slower, 500ms) var(--ease-out);
  }
  .shell-expiry[data-stage='low'] .shell-expiry-fill {
    background: var(--warning-500);
  }
  .shell-expiry[data-stage='critical'] .shell-expiry-fill {
    background: var(--status-sos);
  }

  @media (max-width: 767px) {
    .shell-card {
      width: min(96vw, 400px);
      padding: var(--space-2-5) var(--space-3);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .shell.sos-active {
      animation: none;
      outline-color: var(--status-sos);
    }
    .shell-card {
      animation: none;
    }
  }
</style>
