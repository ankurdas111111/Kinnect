<script>
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { push } from 'svelte-spa-router';

  const SEEN_KEY = 'kinnect_hub_seen_v2';

  let visible = false;
  let hole = { x: 0, y: 0, w: 0, h: 0 };
  let tooltipLeft = 0;
  let tooltipTop = 0;
  const PAD = 10;
  const TOOLTIP_W = 304;

  const features = [
    { color: '#6366f1', label: 'Live Map',          desc: 'Everyone, real time'       },
    { color: '#10b981', label: 'Activity Feed',     desc: 'What\'s been happening'    },
    { color: '#f59e0b', label: 'Route History',     desc: 'Replay recent journeys'    },
    { color: '#ef4444', label: 'Emergency Profile', desc: 'Critical info, always ready' },
    { color: '#8b5cf6', label: 'Check-ins',         desc: 'Scheduled safety pings'    },
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
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="hs-overlay"
    role="dialog"
    aria-modal="true"
    aria-label="Discover the Hub"
    on:click|self={dismiss}
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
        stroke="rgba(251,191,36,0.45)"
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
            <span class="hs-dot" style="background:{f.color}; box-shadow:0 0 7px {f.color}90;" aria-hidden="true"></span>
            <span class="hs-feat-name">{f.label}</span>
            <span class="hs-feat-desc">{f.desc}</span>
          </li>
        {/each}
      </ul>

      <div class="hs-actions">
        <button class="hs-btn-primary" on:click={explore}>Explore Hub</button>
        <button class="hs-btn-ghost"   on:click={dismiss}>Maybe later</button>
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

  /* The spotlight "hole": its box-shadow IS the vignette, the element itself stays transparent */
  .hs-hole {
    position: absolute;
    border-radius: 22px;
    pointer-events: none;
    animation: spot-breathe 2.6s ease-in-out infinite;
  }

  @keyframes spot-breathe {
    0%, 100% {
      box-shadow:
        0 0 0 9999px rgba(4, 3, 14, 0.93),
        0 0 0 2px  rgba(251, 191, 36, 0.78),
        0 0 0 5px  rgba(251, 191, 36, 0.12),
        0 0 32px   rgba(251, 191, 36, 0.30);
    }
    50% {
      box-shadow:
        0 0 0 9999px rgba(4, 3, 14, 0.93),
        0 0 0 2px  rgba(251, 191, 36, 1),
        0 0 0 9px  rgba(251, 191, 36, 0.20),
        0 0 52px   rgba(251, 191, 36, 0.50);
    }
  }

  .hs-connector {
    position: absolute;
    pointer-events: none;
  }

  /* Tooltip card */
  .hs-card {
    position: absolute;
    background: rgba(9, 7, 22, 0.97);
    border: 1px solid rgba(251, 191, 36, 0.20);
    border-radius: 18px;
    padding: 0 20px 20px;
    overflow: hidden;
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
    box-shadow:
      0 28px 64px rgba(0, 0, 0, 0.65),
      0 0 0 1px rgba(251, 191, 36, 0.06),
      inset 0 1px 0 rgba(255, 255, 255, 0.04);
    cursor: default;
  }

  /* Amber top accent stripe */
  .hs-accent-bar {
    height: 3px;
    margin: 0 -20px 18px;
    background: linear-gradient(90deg, #f59e0b 0%, #d97706 60%, transparent 100%);
    opacity: 0.85;
  }

  .hs-eyebrow {
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: #f59e0b;
    margin: 0 0 7px;
    font-family: var(--font-display, system-ui);
  }

  .hs-headline {
    font-size: 19px;
    font-weight: 800;
    color: #fff;
    margin: 0 0 6px;
    font-family: var(--font-display, system-ui);
    line-height: 1.18;
    letter-spacing: -0.025em;
  }

  .hs-sub {
    font-size: 12.5px;
    color: rgba(255, 255, 255, 0.45);
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
    border-top: 1px solid rgba(255, 255, 255, 0.05);
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
    font-size: 12px;
    font-weight: 700;
    color: rgba(255, 255, 255, 0.88);
    font-family: var(--font-display, system-ui);
  }

  .hs-feat-desc {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.35);
    text-align: right;
  }

  /* CTA row */
  .hs-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    padding-top: 16px;
  }

  .hs-btn-primary {
    flex: 1;
    height: 38px;
    border-radius: 11px;
    background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
    color: #0d0a02;
    font-size: 13px;
    font-weight: 800;
    font-family: var(--font-display, system-ui);
    border: none;
    cursor: pointer;
    letter-spacing: -0.01em;
    transition: transform 0.12s, box-shadow 0.15s;
    box-shadow: 0 4px 18px rgba(245, 158, 11, 0.38), 0 0 0 1px rgba(245,158,11,0.3);
  }

  .hs-btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 7px 22px rgba(245, 158, 11, 0.52), 0 0 0 1px rgba(245,158,11,0.4);
  }

  .hs-btn-primary:active { transform: scale(0.96); transition-duration: 60ms; }

  .hs-btn-ghost {
    padding: 0 14px;
    height: 38px;
    border-radius: 11px;
    background: transparent;
    color: rgba(255, 255, 255, 0.38);
    font-size: 12px;
    font-weight: 600;
    border: 1px solid rgba(255, 255, 255, 0.09);
    cursor: pointer;
    white-space: nowrap;
    transition: color 0.15s, border-color 0.15s;
  }

  .hs-btn-ghost:hover {
    color: rgba(255, 255, 255, 0.65);
    border-color: rgba(255, 255, 255, 0.18);
  }
</style>
