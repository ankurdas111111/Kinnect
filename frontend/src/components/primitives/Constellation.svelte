<script>
  /**
   * Constellation — the shared shell "world": a prop-driven SVG of family pins
   * and the links between them. One primitive powers Landing hero/story pins,
   * Login (reactive handshake), Register (builder), and ghost empty states.
   * Absorbs the hardcoded Login deco-map (Login.svelte:142–173).
   *
   * Props:
   *   nodes   — [{ x, y, hue?, state:'unlit'|'igniting'|'live'|'converging', label? }]
   *             hue is a CSS token STRING (e.g. 'var(--member-2)'); raw hex is a
   *             call-site lint violation. Defaults to the Login deco-map geometry.
   *   links   — [[i, j], …] index pairs into nodes (dashed connection lines)
   *   mode    — 'dormant' | 'reactive' | 'builder' | 'ghost'
   *   onnodeactivate — ({ index }) => void — ghost mode only; ghosts are 44px CTAs
   *
   * A11y: non-ghost modes are aria-hidden (pure decoration — must never announce
   * on form-state churn). Ghost pins are real <button>s with aria-labels.
   *
   * Motion: transform/opacity only. Pre-rendered elements are class-toggled by
   * state — no per-state SVG node creation. Ambient loops carry `fx-ambient` and
   * are additionally JS-gated on allowMotion && !prefersReducedMotion().
   */
  import { allowMotion } from '../../lib/stores/effects.js';
  import { prefersReducedMotion } from '../../lib/deviceCapability.js';
  import {
    VIEWBOX, DEFAULT_NODES, DEFAULT_LINKS, GRID_DOTS,
  } from './constellationGeometry.js';

  /** @type {{ nodes?: any[], links?: number[][], mode?: string, onnodeactivate?: (e: { index: number }) => void }} */
  let {
    nodes = DEFAULT_NODES,
    links = DEFAULT_LINKS,
    mode = 'dormant',
    onnodeactivate = undefined,
  } = $props();

  // JS-driven loops run only when motion is globally allowed AND the OS
  // reduce-motion switch is off (a stored 'full' pref must not defeat the OS).
  // CSS keyframes are separately media-gated.
  let loops = $derived($allowMotion && !prefersReducedMotion());
  let isGhost = $derived(mode === 'ghost');

  function activate(index) {
    onnodeactivate?.({ index });
  }
</script>

<div class="cst cst-{mode}" class:cst-loops={loops} class:cst-ghost={isGhost}>
  <svg
    class="cst-svg"
    viewBox={VIEWBOX}
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden={isGhost ? undefined : 'true'} role={isGhost ? 'group' : undefined}
  >
    <!-- Connection links: dashed line + a spark dot that travels node→node on ignite -->
    {#each links as [a, b] (a + '-' + b)}
      {#if nodes[a] && nodes[b]}
        <line
          class="cst-link"
          class:is-live={nodes[a].state === 'live' && nodes[b].state === 'live'}
          x1={nodes[a].x} y1={nodes[a].y} x2={nodes[b].x} y2={nodes[b].y}
        />
      {/if}
    {/each}

    <!-- Static grid dots (dormant/reactive only) -->
    {#if !isGhost}
      {#each GRID_DOTS as d}
        <circle class="cst-grid" cx={d.x} cy={d.y} r="2" />
      {/each}
    {/if}

    <!-- Center-beacon breathe halo (pre-layered radial, toggled by opacity) -->
    {#if nodes[0]}
      <circle class="cst-beacon" cx={nodes[0].x} cy={nodes[0].y} r="34" />
      <circle class="cst-beacon-inner" cx={nodes[0].x} cy={nodes[0].y} r="22" />
    {/if}

    <!-- Nodes: pins in dormant/reactive/builder; interactive ghosts otherwise -->
    {#each nodes as node, i (i)}
      {#if isGhost && i !== 0}
        <g
          class="cst-node cst-ghostnode"
          data-state={node.state}
          style="--x:{node.x}; --y:{node.y}; --hue:{node.hue || 'var(--primary-400)'}"
        >
          <circle class="cst-ghost-ring" cx={node.x} cy={node.y} r="12" />
          {#if node.label}
            <text class="cst-ghost-label" x={node.x} y={node.y + 26}>{node.label}</text>
          {/if}
          <!-- 44px transparent hit target over the ghost pin -->
          <rect
            class="cst-hit"
            x={node.x - 22} y={node.y - 22} width="44" height="44"
            role="button" tabindex="0"
            aria-label={node.label || 'Add a family member'}
            onclick={() => activate(i)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); activate(i); } }}
          />
        </g>
      {:else}
        <g
          class="cst-node"
          class:is-center={i === 0}
          data-state={node.state}
          style="--x:{node.x}; --y:{node.y}; --hue:{node.hue || 'var(--primary-400)'}; --idx:{i}"
        >
          <circle class="cst-halo" cx={node.x} cy={node.y} r={i === 0 ? 16 : 13} />
          <circle class="cst-pin" cx={node.x} cy={node.y} r={i === 0 ? 10 : 7} />
          <circle class="cst-core" cx={node.x} cy={node.y} r={i === 0 ? 4 : 3} />
        </g>
      {/if}
    {/each}
  </svg>
</div>

<style>
  .cst { position: relative; width: 100%; height: 100%; }
  .cst-svg { display: block; width: 100%; height: 100%; overflow: visible; }
  .cst-node, .cst-pin, .cst-beacon, .cst-beacon-inner, .cst-ghost-ring { transform-box: fill-box; transform-origin: center; }
  /* Links: dashed lines, faint until both endpoints live. */
  .cst-link {
    stroke: color-mix(in oklch, var(--primary-400) 22%, transparent);
    stroke-width: 1.5; stroke-dasharray: 5 4; opacity: 0.25;
    transition: opacity var(--duration-slow) var(--ease-out);
  }
  .cst-link.is-live { opacity: 1; }
  .cst-grid { fill: color-mix(in oklch, var(--text-primary) 6%, transparent); }

  /* Center beacon halos — pre-layered radials, opacity-toggled (never filter). */
  .cst-beacon, .cst-beacon-inner {
    fill: none; stroke-width: 1;
    stroke: color-mix(in oklch, var(--primary-400) 14%, transparent);
  }
  .cst-beacon-inner {
    fill: color-mix(in oklch, var(--primary-400) 6%, transparent);
    stroke: color-mix(in oklch, var(--primary-400) 18%, transparent);
  }

  /* Nodes: pin fill = hue token; halo lights on live/igniting. */
  .cst-halo {
    fill: none; stroke-width: 1; opacity: 0;
    stroke: color-mix(in oklch, var(--hue) 22%, transparent);
    transition: opacity var(--duration-slow) var(--ease-out);
  }
  .cst-pin {
    fill: var(--hue);
    transition: transform var(--duration-normal) var(--ease-spring),
                opacity var(--duration-normal) var(--ease-out);
  }
  .cst-core { fill: var(--surface-0, #fff); }
  .cst-node[data-state='unlit'] .cst-pin { opacity: 0.4; }
  .cst-node[data-state='live'] .cst-halo,
  .cst-node[data-state='igniting'] .cst-halo { opacity: 1; }

  /* Ignite / converge — JS-state-driven, only when loops allowed. Max 6 nodes
     animate at once (ignite scheduler cap lives at the consumer). */
  .cst-loops .cst-node[data-state='igniting'] .cst-pin { animation: cst-ignite 320ms var(--ease-spring) both; }
  .cst-loops .cst-node[data-state='converging'] .cst-pin {
    animation: cst-converge 420ms var(--ease-in) both;
    animation-delay: calc(var(--idx, 0) * var(--stagger-step));
  }

  /* Dormant/reactive: dim pins + one slow center-beacon breathe (ambient). */
  .cst-dormant .cst-pin { opacity: 0.4; }
  .cst-dormant .cst-node.is-center .cst-pin { opacity: 0.7; }
  .cst-loops.cst-dormant .cst-beacon,
  .cst-loops.cst-dormant .cst-beacon-inner { animation: cst-breathe 4.5s var(--ease-in-out) infinite; }
  .cst-loops.cst-dormant .cst-beacon-inner { animation-duration: 3.8s; animation-delay: 0.7s; }

  /* Ghost mode: dashed 25% outlines + label; each is a 44px tappable CTA. */
  .cst-ghost-ring {
    fill: none; stroke-width: 1.5; stroke-dasharray: 4 4; opacity: 0.25;
    stroke: color-mix(in oklch, var(--hue) 55%, transparent);
  }
  .cst-ghost-label {
    fill: var(--text-tertiary); font-size: 11px;
    font-family: var(--font-sans, inherit); text-anchor: middle; opacity: 0.55;
  }
  .cst-hit { fill: transparent; cursor: pointer; outline: none; }
  .cst-hit:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  /* Quietest thing on screen: ghost float capped at opacity ≤ 0.15. */
  .cst-loops.cst-ghost .cst-ghost-ring { animation: cst-ghost-float 6s var(--ease-in-out) infinite; }

  @keyframes cst-ignite { 0% { transform: scale(0.6); } 60% { transform: scale(1.08); } 100% { transform: scale(1); } }
  @keyframes cst-converge { to { transform: scale(0.5); opacity: 0; } }
  @keyframes cst-breathe { 0%, 100% { opacity: 0.6; transform: scale(1); } 50% { opacity: 1; transform: scale(1.1); } }
  @keyframes cst-ghost-float { 0%, 100% { opacity: 0.1; } 50% { opacity: 0.15; } }

  /* Reduced motion / minimal FX: land in the static fully-lit final state. */
  @media (prefers-reduced-motion: reduce) {
    .cst-beacon, .cst-beacon-inner, .cst-pin, .cst-ghost-ring { animation: none !important; }
    .cst-node[data-state='igniting'] .cst-pin,
    .cst-node[data-state='live'] .cst-pin { opacity: 1; transform: none; }
    .cst-node[data-state='converging'] .cst-pin { opacity: 0; }
    .cst-pin, .cst-link, .cst-halo { transition: opacity 150ms var(--ease-out); }
  }
</style>
