<script>
  import { self } from 'svelte/legacy';

  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { push } from 'svelte-spa-router';

  const SEEN_KEY = 'kinnect_hub_seen_v2';

  let visible = $state(false);
  let hole = $state({ x: 0, y: 0, w: 0, h: 0 });
  let tooltipLeft = $state(0);
  let tooltipTop = $state(0);
  const PAD = 10;
  const TOOLTIP_W = 304;

  // Member-wheel + status hues only — vermilion stays reserved for live SOS,
  // never a feature bullet.
  const features = [
    { color: 'var(--primary-500)', label: 'Live Map',          desc: 'Everyone, real time'       },
    { color: 'var(--success-500)', label: 'Activity Feed',     desc: 'What\'s been happening'    },
    { color: 'var(--warning-500)', label: 'Route History',     desc: 'Replay recent journeys'    },
    { color: 'var(--member-2)',    label: 'Emergency Profile', desc: 'Critical info, always ready' },
    { color: 'var(--member-4)',    label: 'Check-ins',         desc: 'Scheduled safety pings'    },
  ];

  onMount(() => {
    if (localStorage.getItem(SEEN_KEY)) return;
    if (window.innerWidth < 768) return; // desktop only — navbar hidden on mobile

    setTimeout(() => {
      const btn = document.querySelector('.nav-dashboard-btn');
      if (!btn) return;
      const r = btn.getBoundingClientRect();
      hole = { x: r.left - PAD, y: r.top - PAD, w: r.width + PAD * 2, h: r.height + PAD * 2 };

      tooltipLeft = r.left - 2;
      tooltipTop  = r.bottom + 20;
      // Keep tooltip within viewport horizontally
      if (tooltipLeft + TOOLTIP_W > window.innerWidth - 16) {
        tooltipLeft = window.innerWidth - TOOLTIP_W - 16;
      }
      visible = true;
    }, 1400);
  });

  function dismiss() {
    localStorage.setItem(SEEN_KEY, '1');
    visible = false;
  }

  function explore() {
    localStorage.setItem(SEEN_KEY, '1');
    visible = false;
    setTimeout(() => push('/dashboard'), 60);
  }
</script>

{#if visible}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="hs-overlay"
    role="dialog"
    aria-modal="true"
    aria-label="Discover the Hub"
    onclick={self(dismiss)}
    transition:fade={{ duration: 320 }}
  >
    <!-- Amber spotlight ring positioned exactly over Hub button -->
    <div
      class="hs-hole"
      style="left:{hole.x}px; top:{hole.y}px; width:{hole.w}px; height:{hole.h}px;"
      aria-hidden="true"
    ></div>

    <!-- Dashed connector line from hole to tooltip -->
    <svg
      class="hs-connector"
      style="left:{hole.x + hole.w / 2 - 1}px; top:{hole.y + hole.h}px;"
      width="2"
      height="20"
      viewBox="0 0 2 20"
      aria-hidden="true"
    >
      <line x1="1" y1="0" x2="1" y2="20"
        stroke="color-mix(in oklch, var(--warning-400) 45%, transparent)"
        stroke-width="1.5"
        stroke-dasharray="3 2.5"
      />
    </svg>

    <!-- Tooltip card -->
    <div
      class="hs-card"
      style="left:{tooltipLeft}px; top:{tooltipTop}px; width:{TOOLTIP_W}px;"
      transition:fly={{ y: -10, duration: 380, delay: 180 }}
    >
      <!-- Top accent bar -->
      <div class="hs-accent-bar" aria-hidden="true"></div>

      <p class="hs-eyebrow">New in Kinnect</p>
      <h2 class="hs-headline">Your family command center.</h2>
      <p class="hs-sub">Everything in one place — track, replay, and protect your people.</p>

      <ul class="hs-features" aria-label="Features available in Hub">
        {#each features as f}
          <li class="hs-feat">
            <span class="hs-dot" style="background:{f.color};" aria-hidden="true"></span>
            <span class="hs-feat-name">{f.label}</span>
            <span class="hs-feat-desc">{f.desc}</span>
          </li>
        {/each}
      </ul>

      <div class="hs-actions">
        <button class="hs-btn-primary" onclick={explore}>Explore Hub</button>
        <button class="hs-btn-ghost"   onclick={dismiss}>Maybe later</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .hs-overlay {
    position: fixed;
    inset: 0;
    z-index: calc(var(--z-topmost, 9000) - 100);
    /* No background — the hole's box-shadow creates the vignette */
  }

  /* The spotlight "hole": its box-shadow IS the vignette — a deep ink veil,
     not blue-black. */
  .hs-hole {
    position: absolute;
    border-radius: 22px;
    pointer-events: none;
    /* Static vignette + ring; the breathe lives on ::after as an
       opacity-only pulse (box-shadow keyframes repainted the full
       9999px vignette every frame — GPU-rule violation). */
    box-shadow:
      0 0 0 9999px color-mix(in oklch, var(--ink) 90%, transparent),
      0 0 0 2px  color-mix(in oklch, var(--warning-400) 78%, transparent),
      0 0 0 5px  color-mix(in oklch, var(--warning-400) 12%, transparent);
  }

  .hs-hole::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    box-shadow:
      0 0 0 3px   color-mix(in oklch, var(--warning-400) 30%, transparent),
      0 0 44px 6px color-mix(in oklch, var(--warning-400) 42%, transparent);
    opacity: 0.35;
    animation: spot-breathe 2.6s ease-in-out infinite;
  }

  @keyframes spot-breathe {
    0%, 100% { opacity: 0.35; }
    50%      { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .hs-hole::after { animation: none; opacity: 0.5; }
  }

  .hs-connector {
    position: absolute;
    pointer-events: none;
  }

  /* Tooltip card — a lit sheet of paper inside the dimmed room */
  .hs-card {
    position: absolute;
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: 18px;
    padding: 0 20px 20px;
    overflow: hidden;
    box-shadow: var(--shadow-xl);
    cursor: default;
  }

  /* Ochre top accent stripe — the "look here" cue */
  .hs-accent-bar {
    height: 3px;
    margin: 0 -20px 18px;
    background: var(--warning-500);
    opacity: 0.85;
  }

  .hs-eyebrow {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--warning-700);
    margin: 0 0 7px;
    font-family: var(--font-sans);
  }

  /* .hs-headline picks up the serif italic from the global verdict-voice list */
  .hs-headline {
    font-size: 20px;
    color: var(--text-primary);
    margin: 0 0 6px;
    line-height: 1.25;
  }

  .hs-sub {
    font-size: 14px;
    color: var(--text-secondary);
    margin: 0 0 16px;
    line-height: 1.55;
  }

  /* Feature list */
  .hs-features {
    list-style: none;
    padding: 0;
    margin: 0 0 18px;
    display: flex;
    flex-direction: column;
    gap: 7px;
    border-top: 1px solid var(--border-subtle);
    padding-top: 14px;
  }

  .hs-feat {
    display: grid;
    grid-template-columns: 8px 1fr auto;
    align-items: center;
    gap: 0 9px;
  }

  .hs-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .hs-feat-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    font-family: var(--font-sans);
  }

  .hs-feat-desc {
    font-size: 12px;
    color: var(--text-tertiary);
    text-align: right;
  }

  /* CTA row */
  .hs-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    border-top: 1px solid var(--border-subtle);
    padding-top: 16px;
  }

  .hs-btn-primary {
    flex: 1;
    min-height: 44px;
    border-radius: 11px;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: 14px;
    font-weight: 600;
    font-family: var(--font-sans);
    border: none;
    cursor: pointer;
    letter-spacing: -0.01em;
    transition: transform 0.12s, background 0.15s;
    box-shadow: var(--shadow-sm);
  }

  .hs-btn-primary:hover {
    transform: translateY(-1px);
    background: var(--primary-600);
  }

  .hs-btn-primary:active { transform: scale(0.96); transition-duration: 60ms; }

  .hs-btn-ghost {
    padding: 0 14px;
    min-height: 44px;
    border-radius: 11px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 13px;
    font-weight: 500;
    border: 1px solid var(--border-default);
    cursor: pointer;
    white-space: nowrap;
    transition: color 0.15s, border-color 0.15s;
  }

  .hs-btn-ghost:hover {
    color: var(--text-primary);
    border-color: var(--border-strong);
  }
</style>
