<script>
  import { run } from 'svelte/legacy';

  import { fly, slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { sosNarratives, activeSosUsers } from '../lib/stores/sos.js';
  import { otherUsers } from '../lib/stores/map.js';

  // Collect all SOS-active users that sent a medicalCard
  let sosWithCard = $derived((() => {
    const result = [];
    for (const [userId] of $activeSosUsers) {
      const narrative = $sosNarratives.get(userId);
      if (narrative?.medicalCard) {
        result.push({
          userId,
          user: $otherUsers.get(userId) || null,
          card: narrative.medicalCard,
        });
      }
    }
    return result;
  })());

  let current = $derived(sosWithCard[0] || null);
  let extras  = $derived(sosWithCard.length - 1);

  let expanded = $state(false);
  run(() => {
    if (!current) expanded = false;
  });

  function toggle() { expanded = !expanded; }

  // Returns true when the card has any visible medical data beyond contacts
  let hasMedical = $derived(current && (
    current.card.bloodType  ||
    current.card.allergies?.trim()  ||
    current.card.medications?.trim() ||
    current.card.conditions?.trim()
  ));

  let contacts = $derived(current?.card.emergencyContacts?.length
    ? current.card.emergencyContacts.filter(c => c.name || c.phone)
    : (current?.card.emergencyName || current?.card.emergencyPhone)
      ? [{ name: current.card.emergencyName, phone: current.card.emergencyPhone, relation: '' }]
      : []);
</script>

{#if current}
  <div
    class="sf"
    class:sf-expanded={expanded}
    role="complementary"
    aria-label="Live emergency card for {current.user?.displayName || 'SOS user'}"
    in:fly={{ y: 30, duration: 380, easing: cubicOut }}
    out:fly={{ y: 20, duration: 200, easing: cubicOut }}
  >
    <!-- ── Collapsed pill ───────────────────────────────────────── -->
    <button class="sf-pill" onclick={toggle} aria-expanded={expanded}>
      <span class="sf-pulse" aria-hidden="true"></span>

      {#if current.card.bloodType}
        <span class="sf-blood" aria-label="Blood type {current.card.bloodType}">
          {current.card.bloodType}
        </span>
      {/if}

      <span class="sf-name">{current.user?.displayName || 'Emergency'}</span>
      <span class="sf-tag">SOS</span>

      {#if extras > 0}
        <span class="sf-more">+{extras}</span>
      {/if}

      <span class="sf-chevron" class:sf-chevron-open={expanded} aria-hidden="true">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </span>
    </button>

    <!-- ── Expanded body ─────────────────────────────────────────── -->
    {#if expanded}
      <div
        class="sf-body"
        id="sf-body"
        role="region"
        aria-label="Emergency details"
        transition:slide={{ duration: 260, easing: cubicOut }}
      >

        <!-- Blood type — largest, most critical -->
        {#if current.card.bloodType}
          <div class="sf-bloodtype-row">
            <span class="sf-bt-label">Blood Type</span>
            <span class="sf-bt-value">{current.card.bloodType}</span>
          </div>
        {/if}

        <!-- Allergies — red alert -->
        {#if current.card.allergies?.trim()}
          <div class="sf-med-row sf-med-row-alert">
            <span class="sf-med-icon" aria-hidden="true">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
            </span>
            <div class="sf-med-content">
              <span class="sf-med-label">Allergies</span>
              <span class="sf-med-value">{current.card.allergies}</span>
            </div>
          </div>
        {/if}

        <!-- Conditions -->
        {#if current.card.conditions?.trim()}
          <div class="sf-med-row">
            <div class="sf-med-content">
              <span class="sf-med-label">Conditions</span>
              <span class="sf-med-value">{current.card.conditions}</span>
            </div>
          </div>
        {/if}

        <!-- Medications -->
        {#if current.card.medications?.trim()}
          <div class="sf-med-row">
            <div class="sf-med-content">
              <span class="sf-med-label">Medications</span>
              <span class="sf-med-value">{current.card.medications}</span>
            </div>
          </div>
        {/if}

        <!-- Emergency contacts -->
        {#if contacts.length}
          <div class="sf-contacts">
            <span class="sf-contacts-heading">Emergency Contacts</span>
            {#each contacts as c, i}
              <div class="sf-contact">
                <div class="sf-contact-meta">
                  <span class="sf-contact-name">{c.name || 'Contact ' + (i + 1)}</span>
                  {#if c.relation}
                    <span class="sf-contact-rel">{c.relation}</span>
                  {/if}
                </div>
                {#if c.phone}
                  <a class="sf-call-btn" href="tel:{c.phone}" aria-label="Call {c.name || 'contact'} at {c.phone}">
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 3.07 9.81a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 2 .92h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L6.09 8.91A16 16 0 0 0 15.1 17.9l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/>
                    </svg>
                    {c.phone}
                  </a>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        <!-- Doctor -->
        {#if current.card.doctorName || current.card.doctorPhone}
          <div class="sf-med-row">
            <div class="sf-med-content">
              <span class="sf-med-label">Doctor</span>
              <span class="sf-med-value">
                {current.card.doctorName || ''}
                {#if current.card.doctorPhone}
                  {current.card.doctorName ? ' · ' : ''}<a class="sf-phone-link" href="tel:{current.card.doctorPhone}">{current.card.doctorPhone}</a>
                {/if}
              </span>
            </div>
          </div>
        {/if}

        {#if extras > 0}
          <p class="sf-more-note">+{extras} more person{extras > 1 ? 's' : ''} with active SOS</p>
        {/if}

      </div>
    {/if}
  </div>
{/if}

<style>
  /* ── Float shell — Stitch "SOS Hold Trigger" register: warm paper card,
     vermilion reserved for the emergency accents, sheet radius 20px ────── */
  .sf {
    position: fixed;
    left: var(--space-4, 16px);
    /* sit above the SOS FAB (52px tall) + its spacing + safe-area-inset-bottom */
    bottom: calc(52px + var(--space-4, 16px) * 2 + 4px + env(safe-area-inset-bottom, 0px));
    z-index: calc(var(--z-panel, 1000) + 1);
    width: min(300px, calc(100vw - 96px));
    border-radius: var(--radius-xl);
    overflow: hidden;
    background: color-mix(in oklch, var(--surface-1) 94%, transparent);
    border: 1px solid color-mix(in oklch, var(--danger-500) 35%, transparent);
    box-shadow: var(--shadow-danger), var(--shadow-lg);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    animation: sf-mount-glow 1.2s var(--ease-out) both, sf-beacon 2.2s ease-out 1.2s infinite;

    /* Accent ink that stays AA on paper by day and on dusk paper at night */
    --sf-danger-ink: var(--danger-600);
    --sf-success-ink: var(--success-600);
  }
  :global(:root[data-theme="dark"]) .sf {
    --sf-danger-ink: var(--danger-300);
    --sf-success-ink: var(--success-400);
  }

  /* Emergency beacon — the Stitch expanding-halo pulse, quiet on paper */
  @keyframes sf-beacon {
    0% {
      box-shadow:
        0 0 0 0 color-mix(in oklch, var(--danger-500) 30%, transparent),
        var(--shadow-danger), var(--shadow-lg);
    }
    70%, 100% {
      box-shadow:
        0 0 0 var(--space-3) color-mix(in oklch, var(--danger-500) 0%, transparent),
        var(--shadow-danger), var(--shadow-lg);
    }
  }

  @keyframes sf-mount-glow {
    0% {
      box-shadow:
        0 0 0 0 color-mix(in oklch, var(--danger-500) 0%, transparent),
        var(--shadow-danger), var(--shadow-lg);
    }
    30% {
      box-shadow:
        0 0 0 var(--space-1-5) color-mix(in oklch, var(--danger-500) 20%, transparent),
        var(--shadow-danger), var(--shadow-lg);
    }
    100% {
      box-shadow:
        0 0 0 0 color-mix(in oklch, var(--danger-500) 0%, transparent),
        var(--shadow-danger), var(--shadow-lg);
    }
  }

  /* Mobile: shift higher (above bottom tab + SOS FAB) */
  @media (max-width: 767px) {
    .sf {
      bottom: calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + 52px + var(--space-4, 16px) * 2 + 4px);
    }
  }

  /* Desktop: shift right when sidebar is open */
  :global(.app-layout.sidebar-open:not(.mobile)) .sf {
    left: calc(var(--sidebar-width, 400px) + var(--space-4, 16px));
  }
  :global(.app-layout.tablet.sidebar-open) .sf {
    left: calc(var(--sidebar-tablet, 320px) + var(--space-4, 16px));
  }
  :global(.app-layout.sidebar-closed:not(.mobile)) .sf {
    left: calc(var(--sidebar-collapsed, 56px) + var(--space-4, 16px));
  }

  /* ── Pill (collapsed header) ───────────────────────────────────────────── */
  .sf-pill {
    width: 100%;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3);
    min-height: 44px; /* touch-target floor */
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    -webkit-tap-highlight-color: transparent;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .sf-pill:hover { background: var(--danger-500-10); }

  /* Pulsing live dot — transform/opacity only (GPU), static glow */
  .sf-pulse {
    flex-shrink: 0;
    width: var(--space-2);
    height: var(--space-2);
    border-radius: var(--radius-full);
    background: var(--danger-500);
    box-shadow:
      0 0 var(--space-1-5) color-mix(in oklch, var(--danger-500) 70%, transparent),
      0 0 var(--space-3) color-mix(in oklch, var(--danger-500) 35%, transparent);
    animation: sf-pulse 1.2s ease-in-out infinite;
  }
  @keyframes sf-pulse {
    0%, 100% { transform: scale(1);    opacity: 1; }
    50%      { transform: scale(1.55); opacity: 0.75; }
  }

  /* Blood type badge */
  .sf-blood {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: var(--space-8);
    height: var(--space-6);
    padding: 0 var(--space-1-5);
    border-radius: var(--radius-sm);
    background: color-mix(in oklch, var(--danger-500) 12%, transparent);
    border: 1px solid color-mix(in oklch, var(--danger-500) 40%, transparent);
    color: var(--sf-danger-ink);
    font-size: var(--text-2xs);
    font-weight: 700;
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }

  .sf-name {
    flex: 1;
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  /* Solid vermilion chip — the one loud element on the pill */
  .sf-tag {
    flex-shrink: 0;
    font-size: var(--text-2xs);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-on-danger);
    background: var(--danger-500);
    border: none;
    border-radius: var(--radius-badge);
    padding: var(--space-0-5) var(--space-1-5);
  }

  .sf-more {
    flex-shrink: 0;
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: color-mix(in oklch, var(--text-primary) 8%, transparent);
    border-radius: var(--radius-sm);
    padding: var(--space-0-5) var(--space-1);
  }

  .sf-chevron {
    flex-shrink: 0;
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    transition: transform var(--duration-normal) var(--ease-spring);
  }
  .sf-chevron-open { transform: rotate(180deg); }

  /* ── Expanded body — quiet paper tiers, tonal separation not hard rules ── */
  .sf-body {
    border-top: 1px solid color-mix(in oklch, var(--danger-500) 20%, transparent);
    display: flex;
    flex-direction: column;
    gap: var(--space-px);
    max-height: calc(70vh - 44px);
    overflow-y: auto;
    overscroll-behavior: contain;
    scrollbar-width: none;
  }
  .sf-body::-webkit-scrollbar { display: none; }

  /* ── Blood type row (hero) ─────────────────────────────────────────────── */
  .sf-bloodtype-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-2-5) var(--space-3-5) var(--space-2);
    background: color-mix(in oklch, var(--danger-500) 8%, transparent);
  }
  .sf-bt-label {
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--sf-danger-ink);
  }
  .sf-bt-value {
    font-size: var(--text-3xl);
    font-weight: 700;
    color: var(--sf-danger-ink);
    letter-spacing: -0.04em;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    font-family: var(--font-display, system-ui, sans-serif);
  }

  /* ── Medical rows ──────────────────────────────────────────────────────── */
  .sf-med-row {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3-5);
    background: var(--surface-2);
  }
  .sf-med-row-alert {
    background: color-mix(in oklch, var(--danger-500) 8%, transparent);
  }
  .sf-med-icon {
    flex-shrink: 0;
    color: var(--sf-danger-ink);
    margin-top: var(--space-0-5);
    display: flex;
  }
  .sf-med-content {
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5);
    min-width: 0;
  }
  .sf-med-label {
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-tertiary);
  }
  .sf-med-row-alert .sf-med-label { color: var(--sf-danger-ink); }
  .sf-med-value {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    line-height: var(--leading-normal);
    word-break: break-word;
  }

  /* ── Emergency contacts ────────────────────────────────────────────────── */
  .sf-contacts {
    display: flex;
    flex-direction: column;
    gap: var(--space-px);
    border-top: 1px solid color-mix(in oklch, var(--danger-500) 14%, transparent);
  }
  .sf-contacts-heading {
    display: block;
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-tertiary);
    padding: var(--space-2) var(--space-3-5) var(--space-1);
  }
  .sf-contact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3-5);
    background: var(--surface-2);
  }
  .sf-contact-meta {
    display: flex;
    flex-direction: column;
    gap: var(--space-px);
    min-width: 0;
  }
  .sf-contact-name {
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .sf-contact-rel {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-weight: 500;
  }

  /* Sage call action — confirmed-safe register, never vermilion */
  .sf-call-btn {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-1-5) var(--space-2-5);
    min-height: 44px; /* touch-target floor */
    min-width: 44px;
    border-radius: var(--radius-md);
    background: var(--success-500-20, color-mix(in oklch, var(--success-500) 18%, transparent));
    border: 1px solid color-mix(in oklch, var(--success-500) 35%, transparent);
    color: var(--sf-success-ink);
    font-size: var(--text-xs);
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    touch-action: manipulation;
    transition:
      background var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
  }
  .sf-call-btn:hover {
    background: var(--success-500-28, color-mix(in oklch, var(--success-500) 28%, transparent));
    transform: scale(1.04);
  }
  @media (hover: none) {
    .sf-call-btn:hover {
      transform: none;
    }
  }
  .sf-call-btn:active { transform: scale(0.95); }

  .sf-phone-link {
    color: var(--sf-success-ink);
    text-decoration: none;
    font-weight: 600;
  }
  .sf-phone-link:hover { text-decoration: underline; }

  @media (prefers-reduced-motion: reduce) {
    .sf {
      animation: none;
    }
    .sf-pulse {
      animation: none;
    }
    .sf-chevron {
      transition: none;
    }
    .sf-call-btn {
      transition: background var(--duration-fast) var(--ease-out);
    }
    .sf-call-btn:hover,
    .sf-call-btn:active {
      transform: none;
    }
  }

  .sf-more-note {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    text-align: center;
    padding: var(--space-2) var(--space-3-5);
    margin: 0;
  }
</style>
