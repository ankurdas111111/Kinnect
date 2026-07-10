<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { themeStore, THEMES } from '../../lib/stores/theme.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [open]
   */

  /** @type {Props} */
  let { open = $bindable(false) } = $props();

  const dispatch = createEventDispatcher();

  let current = $state('dark');
  let unsubscribe;

  onMount(() => {
    unsubscribe = themeStore.subscribe(v => { current = v; });
  });
  onDestroy(() => { if (unsubscribe) unsubscribe(); });

  function select(id) {
    themeStore.set(id);
    dispatch('close');
    open = false;
  }

  function close() {
    dispatch('close');
    open = false;
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
  }

  // Category labels
  const CATS = { classic: 'Classic', premium: 'Premium', genz: 'Gen-Z' };
  const CATS_ORDER = ['classic', 'premium', 'genz'];

  let grouped = $derived(CATS_ORDER.map(cat => ({
    cat,
    label: CATS[cat],
    themes: THEMES.filter(t => t.category === cat),
  })));
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <!-- Backdrop -->
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="picker-backdrop" onclick={close} aria-hidden="true"></div>

  <!-- Panel -->
  <div
    class="picker-panel"
    role="dialog"
    aria-modal="true"
    aria-label="Choose theme"
  >
    <!-- Handle bar -->
    <div class="picker-handle" aria-hidden="true"></div>

    <div class="picker-header">
      <h3 class="picker-title">Choose Theme</h3>
      <button class="picker-close" onclick={close} aria-label="Close theme picker">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    <div class="picker-body">
      {#each grouped as { cat, label, themes }}
        <div class="theme-group">
          <p class="group-label">{label}</p>
          <div class="theme-row">
            {#each themes as theme (theme.id)}
              <button
                class="swatch"
                class:active={current === theme.id}
                class:animated={theme.animated}
                data-theme-id={theme.id}
                onclick={() => select(theme.id)}
                title={theme.name}
                aria-pressed={current === theme.id}
                aria-label="{theme.name} — {theme.desc}"
                style="
                  --sw-bg: {theme.colors.bg};
                  --sw-accent: {theme.colors.accent};
                  --sw-secondary: {theme.colors.secondary};
                  --sw-text: {theme.colors.text};
                "
              >
                <!-- 3D mini-scene inside the swatch -->
                <div class="swatch-scene" aria-hidden="true">
                  <!-- Background fill -->
                  <div class="swatch-bg"></div>

                  <!-- Aurora animated orbs -->
                  {#if theme.id === 'aurora'}
                    <div class="swatch-aurora-blob blob-a"></div>
                    <div class="swatch-aurora-blob blob-b"></div>
                    <div class="swatch-aurora-blob blob-c"></div>
                  {/if}

                  <!-- Vapor grid + shimmer -->
                  {#if theme.id === 'vapor'}
                    <div class="swatch-vapor-grid"></div>
                    <div class="swatch-vapor-shimmer"></div>
                  {/if}

                  <!-- Bloom scanlines + neon border -->
                  {#if theme.id === 'bloom'}
                    <div class="swatch-bloom-scan"></div>
                  {/if}

                  <!-- Deep ocean glow particles -->
                  {#if theme.id === 'deep-ocean'}
                    <div class="swatch-ocean-glow"></div>
                  {/if}

                  <!-- Mini map + floating card simulation -->
                  <div class="swatch-map-strip"></div>
                  <div class="swatch-card">
                    <div class="swatch-dot"></div>
                    <div class="swatch-lines">
                      <div class="swatch-line wide"></div>
                      <div class="swatch-line narrow"></div>
                    </div>
                  </div>

                  <!-- Accent dot (presence pin) -->
                  <div class="swatch-pin"></div>
                </div>

                <span class="swatch-name">{theme.name}</span>
                <span class="swatch-desc">{theme.desc}</span>

                <!-- Active check indicator -->
                {#if current === theme.id}
                  <div class="swatch-check" aria-hidden="true">
                    <svg width="10" height="10" viewBox="0 0 12 12" fill="none">
                      <polyline points="2,6 5,9 10,3" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                  </div>
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  /* ── Backdrop ───────────────────────────────────────────────────────── */
  .picker-backdrop {
    position: fixed;
    inset: 0;
    z-index: calc(var(--z-modal) - 1);
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    animation: backdrop-in 200ms var(--ease-out) both;
  }

  @keyframes backdrop-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Panel ──────────────────────────────────────────────────────────── */
  .picker-panel {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: var(--z-modal);
    background: var(--glass-bg-strong);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-top: 1px solid var(--glass-border-strong);
    border-radius: var(--radius-2xl) var(--radius-2xl) 0 0;
    padding-bottom: calc(var(--space-6) + var(--safe-bottom, 0px));
    box-shadow: 0 -24px 64px rgba(0, 0, 0, 0.45), 0 0 0 1px var(--glass-border);
    animation: panel-up 320ms var(--ease-spring) both;
    /* 3D card depth */
    transform-style: preserve-3d;
    max-height: 85vh;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  @keyframes panel-up {
    from { transform: translateY(100%); opacity: 0; }
    to   { transform: translateY(0);   opacity: 1; }
  }

  /* Desktop: floating card instead of bottom sheet */
  @media (min-width: 768px) {
    .picker-panel {
      bottom: auto;
      top: 50%;
      left: 50%;
      right: auto;
      transform: translate(-50%, -50%);
      border-radius: var(--radius-2xl);
      border: 1px solid var(--glass-border-strong);
      max-width: min(92vw, 560px);
      width: 100%;
      max-height: min(85dvh, 44rem);
      animation: panel-scale-in 280ms var(--ease-spring) both;
    }
    @keyframes panel-scale-in {
      from { transform: translate(-50%, -50%) scale(0.92); opacity: 0; }
      to   { transform: translate(-50%, -50%) scale(1);    opacity: 1; }
    }
    .picker-handle { display: none; }
  }

  /* ── Handle ─────────────────────────────────────────────────────────── */
  .picker-handle {
    width: 36px;
    height: 4px;
    background: var(--border-strong);
    border-radius: var(--radius-full);
    margin: var(--space-3) auto var(--space-1);
  }

  /* ── Header ─────────────────────────────────────────────────────────── */
  .picker-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-5) var(--space-5) var(--space-3);
  }

  .picker-title {
    font-family: var(--font-display);
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.02em;
    margin: 0;
  }

  .picker-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    border: 1px solid var(--border-default);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out),
                color     var(--duration-fast) var(--ease-out),
                transform 120ms var(--ease-spring);
  }
  .picker-close:hover  { background: var(--surface-2); color: var(--text-primary); }
  .picker-close:active { transform: scale(0.90); }

  /* ── Body ───────────────────────────────────────────────────────────── */
  .picker-body {
    padding: 0 var(--space-5) var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  /* ── Group ──────────────────────────────────────────────────────────── */
  .theme-group {}

  .group-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-tertiary);
    margin: 0 0 var(--space-2) var(--space-1);
  }

  .theme-row {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
    gap: var(--space-2-5);
  }

  /* ── Swatch card ────────────────────────────────────────────────────── */
  .swatch {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-1-5);
    padding: var(--space-2);
    border-radius: var(--radius-lg);
    border: 1.5px solid var(--border-default);
    background: var(--surface-1);
    cursor: pointer;
    position: relative;
    transition:
      border-color 200ms var(--ease-out),
      transform 200ms var(--ease-spring),
      box-shadow 200ms var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    overflow: hidden;
    text-align: left;
  }

  .swatch:hover {
    border-color: var(--border-strong);
    transform: translateY(-2px) scale(1.02);
    box-shadow: var(--shadow-md);
  }

  .swatch:active {
    transform: scale(0.96);
    transition-duration: 80ms;
  }

  .swatch.active {
    border-color: var(--sw-accent, var(--primary-500));
    box-shadow:
      0 0 0 1px var(--sw-accent, var(--primary-500)),
      0 4px 20px color-mix(in srgb, var(--sw-accent, var(--primary-500)) 30%, transparent),
      var(--shadow-sm);
    transform: none;
  }

  /* ── Mini 3D scene ──────────────────────────────────────────────────── */
  .swatch-scene {
    width: 100%;
    height: 72px;
    border-radius: var(--radius-md);
    overflow: hidden;
    position: relative;
    /* 3D depth */
    transform-style: preserve-3d;
    box-shadow: inset 0 0 0 1px rgba(255,255,255,0.06);
  }

  .swatch-bg {
    position: absolute;
    inset: 0;
    background: var(--sw-bg, #080810);
  }

  /* Map strip — subtle horizontal bands simulating a top-down map */
  .swatch-map-strip {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 40px;
    background:
      linear-gradient(
        0deg,
        color-mix(in srgb, var(--sw-bg, #080810) 60%, var(--sw-accent, #14b8a6)) 0%,
        transparent 100%
      );
    opacity: 0.35;
  }

  /* Floating glass card (like the real InfoPanel) */
  .swatch-card {
    position: absolute;
    bottom: 10px;
    left: 10px;
    width: 62%;
    height: 22px;
    background: color-mix(in srgb, var(--sw-bg, #080810) 78%, white);
    border: 1px solid color-mix(in srgb, var(--sw-accent, #14b8a6) 20%, transparent);
    border-radius: 5px;
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 0 5px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.30);
  }

  .swatch-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--sw-accent, #14b8a6);
    flex-shrink: 0;
    box-shadow: 0 0 6px color-mix(in srgb, var(--sw-accent, #14b8a6) 70%, transparent);
  }

  .swatch-lines {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }
  .swatch-line {
    height: 2px;
    border-radius: 2px;
    background: color-mix(in srgb, var(--sw-text, white) 25%, transparent);
  }
  .swatch-line.wide   { width: 100%; }
  .swatch-line.narrow { width: 60%; }

  /* Presence pin — top right of scene */
  .swatch-pin {
    position: absolute;
    top: 10px;
    right: 12px;
    width: 10px;
    height: 10px;
    border-radius: 50% 50% 50% 0;
    transform: rotate(-45deg);
    background: var(--sw-accent, #14b8a6);
    box-shadow: 0 0 8px color-mix(in srgb, var(--sw-accent, #14b8a6) 65%, transparent);
  }

  /* ── Aurora-specific mini-blobs ─────────────────────────────────────── */
  .swatch-aurora-blob {
    position: absolute;
    border-radius: 50%;
    filter: blur(8px);
    animation: mini-blob 5s ease-in-out infinite alternate;
    pointer-events: none;
  }

  .blob-a {
    width: 50px; height: 50px;
    background: rgba(139, 92, 246, 0.55);
    top: -10px; left: -5px;
    animation-delay: 0s;
  }
  .blob-b {
    width: 40px; height: 40px;
    background: rgba(20, 184, 166, 0.50);
    top: 5px; right: 0px;
    animation-delay: -1.8s;
  }
  .blob-c {
    width: 30px; height: 30px;
    background: rgba(6, 182, 212, 0.45);
    bottom: 5px; left: 50%;
    animation-delay: -3.2s;
  }

  @keyframes mini-blob {
    from { transform: translate(0, 0) scale(1); }
    to   { transform: translate(6px, 8px) scale(1.15); }
  }

  /* ── Vapor-specific: grid + shimmer ────────────────────────────────── */
  .swatch-vapor-grid {
    position: absolute;
    inset: 0;
    background:
      linear-gradient(0deg, rgba(168,85,247,0.08) 1px, transparent 1px),
      linear-gradient(90deg, rgba(168,85,247,0.08) 1px, transparent 1px);
    background-size: 12px 12px;
  }

  .swatch-vapor-shimmer {
    position: absolute;
    top: 0;
    left: -60%;
    width: 40%;
    height: 100%;
    background: linear-gradient(
      105deg,
      transparent 0%,
      rgba(200,120,255,0.18) 45%,
      rgba(236,72,153,0.14) 55%,
      transparent 100%
    );
    animation: mini-shimmer 3s ease-in-out infinite;
    pointer-events: none;
  }

  @keyframes mini-shimmer {
    from { left: -60%; opacity: 0.7; }
    to   { left: 120%; opacity: 0.7; }
  }

  /* ── Bloom-specific: scanlines ──────────────────────────────────────── */
  .swatch-bloom-scan {
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 3px,
      rgba(74,222,128,0.04) 3px,
      rgba(74,222,128,0.04) 4px
    );
    pointer-events: none;
  }

  /* Neon border animation on bloom swatch */
  .swatch[data-theme-id="bloom"] .swatch-scene {
    box-shadow:
      inset 0 0 0 1px rgba(74,222,128,0.25),
      0 0 12px rgba(74,222,128,0.18);
    animation: neon-border-pulse 2s ease-in-out infinite alternate;
  }

  @keyframes neon-border-pulse {
    from { box-shadow: inset 0 0 0 1px rgba(74,222,128,0.18), 0 0 8px rgba(74,222,128,0.12); }
    to   { box-shadow: inset 0 0 0 1px rgba(74,222,128,0.40), 0 0 16px rgba(74,222,128,0.28); }
  }

  /* ── Deep ocean glow ────────────────────────────────────────────────── */
  .swatch-ocean-glow {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(0,212,255,0.18) 0%, transparent 70%);
    animation: ocean-breathe 3s ease-in-out infinite alternate;
    pointer-events: none;
  }

  @keyframes ocean-breathe {
    from { transform: translate(-50%,-50%) scale(0.8); opacity: 0.6; }
    to   { transform: translate(-50%,-50%) scale(1.3); opacity: 1; }
  }

  /* ── Swatch name + desc ─────────────────────────────────────────────── */
  .swatch-name {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0.01em;
    line-height: 1.2;
  }

  .swatch-desc {
    font-family: var(--font-sans);
    font-size: 10px;
    font-weight: 500;
    color: var(--text-tertiary);
    line-height: 1.2;
  }

  /* ── Check indicator ────────────────────────────────────────────────── */
  .swatch-check {
    position: absolute;
    top: var(--space-2);
    right: var(--space-2);
    width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--sw-accent, var(--primary-500));
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 0 8px color-mix(in srgb, var(--sw-accent, var(--primary-500)) 55%, transparent);
    animation: check-pop 200ms var(--ease-spring) both;
  }

  @keyframes check-pop {
    from { transform: scale(0) rotate(-30deg); opacity: 0; }
    to   { transform: scale(1) rotate(0deg);   opacity: 1; }
  }

  /* ── Reduced motion ─────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .swatch-aurora-blob,
    .swatch-vapor-shimmer,
    .swatch-ocean-glow { animation: none; }
    .picker-panel  { animation: none; }
    .picker-backdrop { animation: none; }
    .swatch:hover  { transform: none; }
  }
</style>
