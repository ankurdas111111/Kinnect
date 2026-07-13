<script>
  /**
   * PlaybackControls — purely presentational transport bar.
   * Holds NO playback state — all display state is driven by props.
   * Zero imports from stores / socket / Map. Props + callbacks only.
   * CONTRACTS.md §11: renaming any event is a breaking-change PR to CONTRACTS.md first.
   *
   * Props:
   *   playing    — boolean               — current play state
   *   progress   — number 0–1            — fill fraction
   *   duration   — number (ms)           — total; for aria-valuetext math
   *   speed      — number                — current speed multiplier
   *   speeds     — number[]              — cycle order for speed button
   *   timestamps — { current, end }      — epoch ms | null; rendered HH:MM mono
   *
   * Callbacks (frozen names — CONTRACTS.md §11):
   *   onplay        — ()               — play pressed while !playing
   *   onpause       — ()               — pause pressed while playing
   *   onscrub       — ({ progress })   — number 0–1; every range input event
   *   onspeedchange — ({ speed })      — next entry in speeds array
   */

  /** @type {{
   *   playing?: boolean,
   *   progress?: number,
   *   duration?: number,
   *   speed?: number,
   *   speeds?: number[],
   *   timestamps?: { current: number | null, end: number | null },
   *   onplay?: () => void,
   *   onpause?: () => void,
   *   onscrub?: (p: { progress: number }) => void,
   *   onspeedchange?: (p: { speed: number }) => void,
   * }} */
  let {
    playing    = false,
    progress   = 0,
    duration   = 0,
    speed      = 1,
    speeds     = [1, 2, 4],
    timestamps = { current: null, end: null },
    onplay,
    onpause,
    onscrub,
    onspeedchange,
  } = $props();

  // ── Range input works in integer steps 0–1000 to avoid float drift ─────────
  // Map progress 0–1 → slider 0–1000 for aria precision and native range step
  const sliderVal  = $derived(Math.round(progress * 1000));

  // aria-valuetext: "3:42 of 12:10"
  function fmtHHMM(epochMs) {
    if (epochMs == null) return '--:--';
    return new Date(epochMs).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function fmtElapsed(progFrac, totalMs) {
    const elapsed = Math.round(progFrac * totalMs / 1000);
    const total   = Math.round(totalMs / 1000);
    const fmt = (s) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
    return `${fmt(elapsed)} of ${fmt(total)}`;
  }

  const valueText = $derived(fmtElapsed(progress, duration));

  // ── Scrubber input → onscrub callback ───────────────────────────────────────
  function handleScrub(e) {
    const p = Number(e.target.value) / 1000;
    onscrub?.({ progress: p });
  }

  // ── Speed cycle ─────────────────────────────────────────────────────────────
  function cycleSpeed() {
    const idx  = speeds.indexOf(speed);
    const next = speeds[(idx + 1) % speeds.length];
    onspeedchange?.({ speed: next });
  }

  // ── Play / pause toggle ─────────────────────────────────────────────────────
  function handlePlayPause() {
    if (playing) onpause?.();
    else         onplay?.();
  }
</script>

<div class="playback-controls" role="region" aria-label="Playback controls">

  <!-- Scrubber row — donated verbatim from RoutePlayback:688–729 -->
  <div class="scrubber-row">
    <!-- Current timestamp -->
    <time class="ts-mono" aria-label="Current time: {fmtHHMM(timestamps.current)}">
      {fmtHHMM(timestamps.current)}
    </time>

    <!-- Layered scrubber: transform-scaled fill + transparent native range input -->
    <div class="scrubber-wrap" style="--frac: {progress};">
      <!-- Static paint layer — pointer-events: none so range input captures all hits -->
      <div class="scrubber-track" aria-hidden="true">
        <div class="scrubber-fill"></div>
      </div>

      <!-- Accessible range input stacked on top -->
      <input
        type="range"
        min="0"
        max="1000"
        step="1"
        value={sliderVal}
        oninput={handleScrub}
        class="scrubber"
        aria-label="Playback position"
        aria-valuemin="0"
        aria-valuemax="1000"
        aria-valuenow={sliderVal}
        aria-valuetext={valueText}
      />
    </div>

    <!-- End timestamp -->
    <time class="ts-mono">{fmtHHMM(timestamps.end)}</time>
  </div>

  <!-- Transport row: play/pause + speed -->
  <div class="transport-row">
    <!-- Play / Pause — single button, aria-label swaps, glyph + text — CONTRACTS.md §11 -->
    <button
      type="button"
      class="play-btn"
      class:playing
      onclick={handlePlayPause}
      aria-label={playing ? 'Pause playback' : 'Play route'}
      aria-pressed={playing}
    >
      {#if playing}
        <!-- Pause icon -->
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <rect x="6" y="4" width="4" height="16"/>
          <rect x="14" y="4" width="4" height="16"/>
        </svg>
        <span>Pause</span>
      {:else}
        <!-- Play icon -->
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        <span>Play</span>
      {/if}
    </button>

    <!-- Speed cycle button — announces "Speed Nx" — CONTRACTS.md §11 -->
    <button
      type="button"
      class="speed-btn"
      onclick={cycleSpeed}
      aria-label={`Speed ${speed}x`}
    >
      {speed}×
    </button>
  </div>
</div>

<style>
  /* ── Wrapper ─────────────────────────────────────────────────────────────── */
  .playback-controls {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-4);
  }

  /* ── Scrubber row ────────────────────────────────────────────────────────── */
  .scrubber-row {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
  }

  .ts-mono {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    flex-shrink: 0;
    font-family: var(--font-mono);
  }

  /* Layered progress — transform-scaled fill + transparent range input.
     RoutePlayback:688–729 pattern, donated verbatim — CONTRACTS.md §11 */
  .scrubber-wrap {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
    height: 44px;      /* 44px min touch-target height */
  }

  .scrubber-track {
    position: absolute;
    left: 0;
    right: 0;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    overflow: hidden;
    pointer-events: none;
  }

  /* GPU-only progress: transform scaleX on the fill child — CONTRACTS.md §11 */
  .scrubber-fill {
    position: absolute;
    inset: 0;
    transform-origin: left center;
    transform: scaleX(var(--frac, 0));
    background: linear-gradient(90deg, var(--primary-600), var(--primary-400));
    border-radius: inherit;
    transition: transform 120ms var(--ease-out);
  }

  /* Transparent native range input stacked above the track */
  .scrubber {
    position: relative;
    z-index: 1;
    flex: 1;
    width: 100%;
    height: 44px;
    margin: 0;
    -webkit-appearance: none;
    appearance: none;
    background: transparent;
    cursor: pointer;
    outline: none;
  }

  .scrubber:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 3px;
    border-radius: var(--radius-full);
  }

  /* Thumb — RoutePlayback:737–746 pattern */
  .scrubber::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--text-inverse, white);
    border: 2.5px solid var(--primary-500);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-spring);
  }

  .scrubber::-webkit-slider-thumb:hover {
    transform: scale(1.2);
  }

  .scrubber::-moz-range-thumb {
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--text-inverse, white);
    border: 2.5px solid var(--primary-500);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
  }

  /* ── Transport row ───────────────────────────────────────────────────────── */
  .transport-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  /* Play / Pause button — glyph + text, ≥44px, aria-label swaps — CONTRACTS.md §11 */
  .play-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    min-height: 44px;
    padding: var(--space-2-5) var(--space-4);
    border-radius: var(--radius-lg);
    background: linear-gradient(135deg, var(--primary-500), var(--primary-700));
    border: none;
    color: var(--text-inverse);
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 700;
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-spring),
                box-shadow var(--duration-normal) var(--ease-out),
                background var(--duration-normal) var(--ease-out);
    box-shadow: var(--shadow-primary);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .play-btn:hover {
    box-shadow: var(--shadow-primary), var(--glow-primary, none);
  }

  .play-btn:active {
    transform: scale(0.97);
  }

  .play-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .play-btn.playing {
    background: var(--surface-2);
    color: var(--text-primary);
    box-shadow: var(--shadow-sm);
  }

  /* Speed button — ≥44×44px tap target */
  .speed-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    min-height: 44px;
    padding: 0 var(--space-3);
    border-radius: var(--radius-md);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    transition: background var(--duration-normal) var(--ease-out),
                color var(--duration-normal) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .speed-btn:hover {
    background: var(--surface-2);
    color: var(--text-primary);
  }

  .speed-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  /* ── Reduced motion — RoutePlayback:805–807 pattern ─────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scrubber-fill { transition: none; }
    .scrubber::-webkit-slider-thumb:hover { transform: none; }
    .play-btn:hover  { box-shadow: var(--shadow-primary); }
    .play-btn:active { transform: none; }
  }
</style>
