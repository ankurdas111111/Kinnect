<script>
  /**
   * GhostConstellation — first-run empty state that previews the filled state.
   * Your real avatar (authUser) sits at the constellation center under a quiet
   * radar sweep; three ghost pins are ≥44px invite CTAs. A conventional primary
   * invite button always renders below — ghosts are bonus affordances, never
   * the only path.
   *
   * Props:
   *   title, body — copy for the EmptyState text block
   *   ctaLabel    — primary button label
   *   memberCount — live member count; the mounted-while-empty flag guarantees
   *                 a same-tick filled load renders with ZERO animation
   *   oninvite    — fired by ghosts AND the primary button — must be the same
   *                 action the surface's existing invite CTA performs
   */
  import Constellation from './Constellation.svelte';
  import EmptyState from './EmptyState.svelte';
  import { authUser } from '../../lib/stores/auth.js';
  import { allowMotion } from '../../lib/stores/effects.js';
  import { prefersReducedMotion } from '../../lib/deviceCapability.js';
  import { haptics } from '../../lib/haptics.js';

  /** @type {{ title: string, body?: string, ctaLabel?: string, memberCount?: number, oninvite?: () => void }} */
  let { title, body = '', ctaLabel = 'Invite people', memberCount = 0, oninvite = undefined } = $props();

  // Mounted-while-empty flag (captured once, deliberately non-reactive):
  // members present at first render → everything paints static, zero animation.
  const mountedWhileEmpty = memberCount === 0;
  let materialized = $derived(memberCount > 0);
  let animate = $derived(mountedWhileEmpty && $allowMotion && !prefersReducedMotion());

  // Role-neutral labels only — never presume a family shape ("Add a parent").
  const nodes = [
    { x: 170, y: 122, state: 'live' }, // self — real avatar overlays this pin
    { x: 84, y: 74, state: 'unlit', hue: 'var(--member-1)', label: 'Add family' },
    { x: 256, y: 88, state: 'unlit', hue: 'var(--member-2)', label: 'Add friends' },
    { x: 170, y: 208, state: 'unlit', hue: 'var(--member-3)', label: 'Add anyone' },
  ];
  const links = [[1, 0], [0, 2], [0, 3]];

  let initials = $derived(
    ($authUser?.displayName || '?').split(' ').map((s) => s[0]).join('').toUpperCase().slice(0, 2)
  );

  function invite() {
    haptics.tap();
    oninvite?.();
  }
</script>

<div class="gc" class:gc-anim={animate} class:gc-materialized={materialized}>
  <div class="gc-stage">
    <Constellation {nodes} {links} mode="ghost" onnodeactivate={invite} />
    <div class="gc-center" aria-hidden="true">
      {#if animate}<div class="gc-sweep fx-ambient"></div>{/if}
      <span class="gc-avatar">{initials}</span>
    </div>
  </div>
  <EmptyState {title} {body}>
    {#snippet action()}
      <button class="gc-cta" onclick={invite}>
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
        {ctaLabel}
      </button>
    {/snippet}
  </EmptyState>
</div>

<style>
  .gc { display: flex; flex-direction: column; align-items: center; width: 100%; padding-top: var(--space-4); }
  /* Entrance only when mounted-while-empty AND motion allowed. */
  .gc-anim { transition: opacity var(--duration-slow) var(--ease-out), transform var(--duration-slow) var(--ease-out); }
  @starting-style {
    .gc-anim { opacity: 0; transform: translateY(8px); }
  }

  /* Stage keeps the SVG viewBox (340×280) aspect so the avatar overlay at
     node (170,122) stays glued to the center pin at any width. */
  .gc-stage { position: relative; width: min(100%, 300px); aspect-ratio: 340 / 280; }
  .gc-center {
    position: absolute; left: 50%; top: calc(122 / 280 * 100%);
    width: 48px; height: 48px; transform: translate(-50%, -50%);
  }

  /* Radar sweep — static conic paint, only transform animates (GPU-only).
     Quietest thing on screen: opacity capped at 0.12 (spec ≤ 0.15). */
  .gc-sweep {
    position: absolute; left: 50%; top: 50%; width: 140px; height: 140px;
    border-radius: var(--radius-full, 9999px);
    background: conic-gradient(from 0deg, color-mix(in oklch, var(--primary-400) 60%, transparent) 0deg, transparent 75deg);
    opacity: 0.12; transform: translate(-50%, -50%);
    animation: gc-sweep 6s linear infinite;
  }
  .gc-avatar {
    position: absolute; inset: 0; display: grid; place-items: center;
    border-radius: var(--radius-full, 9999px);
    background: var(--primary-500-12); color: var(--text-primary);
    font-family: var(--font-display); font-size: var(--text-sm); font-weight: 700;
    box-shadow: var(--glow-primary-sm), inset 0 0 0 2px color-mix(in oklch, var(--primary-400) 55%, transparent);
  }

  /* Materialize: a member joined while this was on screen — ghosts yield.
     Without .gc-anim (same-tick filled mount) the first paint is already the
     final state, so there is ZERO animation. */
  .gc-anim .gc-stage :global(.cst-ghostnode) { transition: opacity var(--duration-normal) var(--ease-out); }
  .gc-materialized .gc-stage :global(.cst-ghostnode) { opacity: 0; pointer-events: none; }

  /* Primary CTA — the conventional path; ghosts are bonus affordances. */
  .gc-cta {
    display: inline-flex; align-items: center; gap: var(--space-1-5);
    min-height: 44px; padding: var(--space-2-5) var(--space-4);
    background: linear-gradient(135deg, var(--primary-500), var(--primary-700));
    color: var(--text-on-primary); border: none; border-radius: var(--radius-lg);
    font-family: var(--font-display); font-size: var(--text-sm); font-weight: 700;
    cursor: pointer; box-shadow: var(--glow-primary-sm);
    transition: transform var(--duration-fast) var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
  }
  .gc-cta:hover { transform: translateY(-1px); }
  .gc-cta:active { transform: scale(0.97); }
  .gc-cta:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  /* Trim EmptyState's top padding — the stage supplies the breathing room. */
  .gc :global(.empty) { padding-top: var(--space-2); }

  @keyframes gc-sweep { to { transform: translate(-50%, -50%) rotate(360deg); } }

  @media (prefers-reduced-motion: reduce) {
    .gc-sweep { animation: none; }
    .gc-anim, .gc-cta, .gc-anim .gc-stage :global(.cst-ghostnode) { transition: none; }
    .gc-cta:hover, .gc-cta:active { transform: none; }
  }
</style>
