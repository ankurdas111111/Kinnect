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
  const CATS = { core: 'Appearance' };
  const CATS_ORDER = ['core'];

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
    background: var(--sw-bg, var(--surface-0));
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
        color-mix(in srgb, var(--sw-bg, var(--surface-0)) 60%, var(--sw-accent, var(--primary-500))) 0%,
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
    background: color-mix(in srgb, var(--sw-bg, var(--surface-0)) 78%, white);
    border: 1px solid color-mix(in srgb, var(--sw-accent, var(--primary-500)) 20%, transparent);
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
    background: var(--sw-accent, var(--primary-500));
    flex-shrink: 0;
    box-shadow: 0 0 6px color-mix(in srgb, var(--sw-accent, var(--primary-500)) 70%, transparent);
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
    background: var(--sw-accent, var(--primary-500));
    box-shadow: 0 0 8px color-mix(in srgb, var(--sw-accent, var(--primary-500)) 65%, transparent);
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
    .picker-panel  { animation: none; }
    .picker-backdrop { animation: none; }
    .swatch:hover  { transform: none; }
  }
</style>
