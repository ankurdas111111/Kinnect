<script>
  import { fly, slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { sosNarratives, activeSosUsers } from '../lib/stores/sos.js';
  import { otherUsers } from '../lib/stores/map.js';

  // Collect all SOS-active users that sent a medicalCard
  $: sosWithCard = (() => {
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
  })();

  $: current = sosWithCard[0] || null;
  $: extras  = sosWithCard.length - 1;

  let expanded = false;
  $: if (!current) expanded = false;

  function toggle() { expanded = !expanded; }

  // Returns true when the card has any visible medical data beyond contacts
  $: hasMedical = current && (
    current.card.bloodType  ||
    current.card.allergies?.trim()  ||
    current.card.medications?.trim() ||
    current.card.conditions?.trim()
  );

  $: contacts = current?.card.emergencyContacts?.length
    ? current.card.emergencyContacts.filter(c => c.name || c.phone)
    : (current?.card.emergencyName || current?.card.emergencyPhone)
      ? [{ name: current.card.emergencyName, phone: current.card.emergencyPhone, relation: '' }]
      : [];
</script>

{#if current}
  <div
    class="sf"
    class:sf-expanded={expanded}
    role="complementary"
    aria-label="Live emergency card for {current.user?.displayName || 'SOS user'}"
    in:fly={{ y: 20, duration: 300, easing: cubicOut }}
    out:fly={{ y: 20, duration: 200, easing: cubicOut }}
  >
    <!-- ── Collapsed pill ───────────────────────────────────────── -->
    <button class="sf-pill" on:click={toggle} aria-expanded={expanded}>
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
  /* ── Float shell ──────────────────────────────────────────────────────────── */
  .sf {
    position: fixed;
    left: var(--space-4, 16px);
    /* sit above the SOS FAB (52px tall) + its spacing */
    bottom: calc(52px + var(--space-4, 16px) * 2 + 4px);
    z-index: calc(var(--z-panel, 100) + 1);
    width: min(300px, calc(100vw - 96px));
    border-radius: 18px;
    overflow: hidden;
    /* urgent red glass */
    background: rgba(10, 4, 4, 0.88);
    border: 1px solid rgba(239, 68, 68, 0.35);
    box-shadow:
      0 8px 32px rgba(239, 68, 68, 0.22),
      0 2px 8px rgba(0, 0, 0, 0.45),
      inset 0 1px 0 rgba(255, 255, 255, 0.08);
    backdrop-filter: blur(24px) saturate(1.4);
    -webkit-backdrop-filter: blur(24px) saturate(1.4);
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
    gap: 7px;
    padding: 10px 12px;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    -webkit-tap-highlight-color: transparent;
    transition: background 150ms ease;
  }
  .sf-pill:hover { background: rgba(239, 68, 68, 0.06); }

  /* Pulsing live dot */
  .sf-pulse {
    flex-shrink: 0;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ef4444;
    box-shadow: 0 0 6px rgba(239, 68, 68, 0.7);
    animation: sf-pulse 1.2s ease-in-out infinite;
  }
  @keyframes sf-pulse {
    0%, 100% { transform: scale(1); opacity: 1; }
    50%       { transform: scale(1.5); opacity: 0.6; }
  }

  /* Blood type badge */
  .sf-blood {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 30px;
    height: 22px;
    padding: 0 6px;
    border-radius: 6px;
    background: rgba(239, 68, 68, 0.22);
    border: 1px solid rgba(239, 68, 68, 0.45);
    color: #fca5a5;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }

  .sf-name {
    flex: 1;
    font-size: 13px;
    font-weight: 700;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .sf-tag {
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #f87171;
    background: rgba(239, 68, 68, 0.15);
    border: 1px solid rgba(239, 68, 68, 0.30);
    border-radius: 5px;
    padding: 2px 6px;
  }

  .sf-more {
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 700;
    color: rgba(255, 255, 255, 0.45);
    background: rgba(255, 255, 255, 0.08);
    border-radius: 5px;
    padding: 2px 5px;
  }

  .sf-chevron {
    flex-shrink: 0;
    color: rgba(255, 255, 255, 0.35);
    display: flex;
    align-items: center;
    transition: transform 220ms cubic-bezier(0.16, 1, 0.3, 1);
  }
  .sf-chevron-open { transform: rotate(180deg); }

  /* ── Expanded body ─────────────────────────────────────────────────────── */
  .sf-body {
    border-top: 1px solid rgba(239, 68, 68, 0.18);
    display: flex;
    flex-direction: column;
    gap: 1px;
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
    padding: 10px 14px 8px;
    background: rgba(239, 68, 68, 0.08);
  }
  .sf-bt-label {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: #f87171;
  }
  .sf-bt-value {
    font-size: 26px;
    font-weight: 900;
    color: #fca5a5;
    letter-spacing: -0.04em;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    font-family: var(--font-display, system-ui, sans-serif);
  }

  /* ── Medical rows ──────────────────────────────────────────────────────── */
  .sf-med-row {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 8px 14px;
    background: rgba(255, 255, 255, 0.03);
  }
  .sf-med-row-alert {
    background: rgba(239, 68, 68, 0.07);
  }
  .sf-med-icon {
    flex-shrink: 0;
    color: #f87171;
    margin-top: 2px;
    display: flex;
  }
  .sf-med-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .sf-med-label {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: rgba(255, 255, 255, 0.35);
  }
  .sf-med-row-alert .sf-med-label { color: #f87171; }
  .sf-med-value {
    font-size: 12px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.85);
    line-height: 1.45;
    word-break: break-word;
  }

  /* ── Emergency contacts ────────────────────────────────────────────────── */
  .sf-contacts {
    display: flex;
    flex-direction: column;
    gap: 1px;
    border-top: 1px solid rgba(239, 68, 68, 0.14);
  }
  .sf-contacts-heading {
    display: block;
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: rgba(255, 255, 255, 0.30);
    padding: 8px 14px 4px;
  }
  .sf-contact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 7px 14px;
    background: rgba(255, 255, 255, 0.03);
  }
  .sf-contact-meta {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .sf-contact-name {
    font-size: 12px;
    font-weight: 700;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .sf-contact-rel {
    font-size: 10px;
    color: rgba(255, 255, 255, 0.38);
    font-weight: 500;
  }

  .sf-call-btn {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 6px 10px;
    border-radius: 9px;
    background: rgba(16, 185, 129, 0.18);
    border: 1px solid rgba(16, 185, 129, 0.35);
    color: #34d399;
    font-size: 11px;
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    transition: background 150ms ease, transform 100ms ease;
    -webkit-tap-highlight-color: transparent;
  }
  .sf-call-btn:hover {
    background: rgba(16, 185, 129, 0.28);
    transform: scale(1.04);
  }
  .sf-call-btn:active { transform: scale(0.95); }

  .sf-phone-link {
    color: #34d399;
    text-decoration: none;
    font-weight: 600;
  }
  .sf-phone-link:hover { text-decoration: underline; }

  .sf-more-note {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.30);
    text-align: center;
    padding: 8px 14px;
    margin: 0;
  }
</style>
