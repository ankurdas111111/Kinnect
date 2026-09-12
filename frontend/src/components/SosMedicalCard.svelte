<script>
  /**
   * The medical card from an SOS alert. Extracted from AlertOverlay and loaded
   * on demand: by the alert's own information order this is for the responder
   * who has already arrived, not for the first three seconds, and carrying its
   * markup and styles in the initial route bundle broke the 120 KB ratchet.
   */
  import Card from './primitives/Card.svelte';

  /** @type {{ card: Record<string, any>, open: boolean, ontoggle: () => void }} */
  let { card, open, ontoggle } = $props();

  function hasMedField(c, ...keys) {
    return keys.some(k => c?.[k]?.trim?.());
  }

  // Split a free-text medical field into individual chips (comma / newline / semicolon).
  function chipList(value) {
    return (value || '').split(/[,\n;]+/).map(s => s.trim()).filter(Boolean);
  }
</script>

        <div class="med-card-wrap">
          <Card variant="glass" glow="danger" padding="none" hover={false}>
            <div class="med-card" role="region" aria-label="Emergency medical information">
              <button
                class="med-card-header"
                onclick={ontoggle}
                aria-expanded={open}
                aria-controls="med-card-body"
              >
                <span class="med-card-icon" aria-hidden="true">
                  <!-- Medical cross icon -->
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="3" width="18" height="18" rx="3"/>
                    <line x1="12" y1="8" x2="12" y2="16"/>
                    <line x1="8"  y1="12" x2="16" y2="12"/>
                  </svg>
                </span>
                <span class="med-card-title">Medical Card</span>
                <span class="med-card-chevron" class:open={open} aria-hidden="true">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9"/>
                  </svg>
                </span>
              </button>

              {#if open}
                <div class="med-card-body" id="med-card-body">

                  <!-- Blood type — the single most critical field: extra-large, centered -->
                  {#if card.bloodType}
                    <div class="med-row med-row-bloodtype">
                      <span class="med-blood-label">Blood Type</span>
                      <span class="med-blood-value">{card.bloodType}</span>
                    </div>
                  {/if}

                  <!-- Allergies — highlighted as critical (danger chips) -->
                  {#if card.allergies?.trim()}
                    <div class="med-row med-row-alert">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                          <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Allergies</span>
                        <div class="med-chips">
                          {#each chipList(card.allergies) as item}
                            <span class="med-chip med-chip-danger">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Medications — warning chips -->
                  {#if card.medications?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Medications</span>
                        <div class="med-chips">
                          {#each chipList(card.medications) as item}
                            <span class="med-chip med-chip-warning">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Medical conditions — neutral chips -->
                  {#if card.conditions?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Conditions</span>
                        <div class="med-chips">
                          {#each chipList(card.conditions) as item}
                            <span class="med-chip med-chip-neutral">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Emergency contacts (array — new format) -->
                  {#if card.emergencyContacts?.length}
                    {#each card.emergencyContacts as contact, ci}
                      <div class="med-row med-row-contact">
                        <span class="med-field-icon" aria-hidden="true">
                          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
                          </svg>
                        </span>
                        <div class="med-field-content">
                          <span class="med-field-label">
                            Emergency Contact {card.emergencyContacts.length > 1 ? ci + 1 : ''}
                            {#if contact.relation}<span class="med-relation"> · {contact.relation}</span>{/if}
                          </span>
                          <span class="med-field-value">
                            {contact.name || ''}
                            {#if contact.phone}
                              {contact.name ? ' · ' : ''}<a class="med-phone-link" href="tel:{contact.phone}">{contact.phone}</a>
                            {/if}
                            {#if contact.address}
                              <span class="med-address"> · {contact.address}</span>
                            {/if}
                          </span>
                        </div>
                      </div>
                    {/each}
                  <!-- Fallback: legacy single contact fields -->
                  {:else if hasMedField(card, 'emergencyName', 'emergencyPhone')}
                    <div class="med-row med-row-contact">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Emergency Contact</span>
                        <span class="med-field-value">
                          {card.emergencyName || ''}
                          {#if card.emergencyPhone}
                            {card.emergencyName ? ' · ' : ''}<a class="med-phone-link" href="tel:{card.emergencyPhone}">{card.emergencyPhone}</a>
                          {/if}
                        </span>
                      </div>
                    </div>
                  {/if}

                  <!-- Doctor -->
                  {#if hasMedField(card, 'doctorName', 'doctorPhone')}
                    <div class="med-row med-row-contact">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Primary Doctor</span>
                        <span class="med-field-value">
                          {card.doctorName || ''}
                          {#if card.doctorPhone}
                            {card.doctorName ? ' · ' : ''}<a class="med-phone-link" href="tel:{card.doctorPhone}">{card.doctorPhone}</a>
                          {/if}
                        </span>
                      </div>
                    </div>
                  {/if}

                  <!-- Language / responder notes -->
                  {#if card.language?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Language</span>
                        <span class="med-field-value">{card.language}</span>
                      </div>
                    </div>
                  {/if}

                  {#if card.responderNotes?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Responder Notes</span>
                        <span class="med-field-value">{card.responderNotes}</span>
                      </div>
                    </div>
                  {/if}

                </div>
              {/if}
            </div>
          </Card>
        </div>

<style>
  .med-card-wrap {
    width: 100%;
    max-width: 380px;
    margin-top: var(--space-1);
  }

  .med-card {
    width: 100%;
    overflow: hidden;
    text-align: left;
  }

  .med-card-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3-5);
    min-height: 44px;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .med-card-header:hover { background: var(--danger-500-12); }
  .med-card-header:focus-visible {
    outline: 2px solid var(--danger-400);
    outline-offset: -2px;
  }

  .med-card-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--danger-500-12);
    color: var(--danger-500);
    flex-shrink: 0;
    box-shadow: 0 0 8px color-mix(in oklch, var(--danger-500) 25%, transparent);
  }

  .med-card-title {
    flex: 1;
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--danger-600);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  /* Chevron — spring affordance (transform-only, reduced-motion safe) */
  .med-card-chevron {
    color: var(--danger-400);
    display: flex;
    align-items: center;
    transition: transform var(--duration-normal) var(--ease-spring);
  }
  .med-card-chevron.open { transform: rotate(180deg) scale(1.08); }

  /* Constrain body height so it scrolls on short phones (667px viewport).
     100dvh accounts for iOS browser chrome; safe-area insets cover notch and home indicator. */
  .med-card-body {
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: var(--danger-500-12);
    border-top: 1px solid color-mix(in oklch, var(--danger-500) 15%, transparent);
    max-height: calc(100dvh - 160px - env(safe-area-inset-top, 0px) - env(safe-area-inset-bottom, 0px));
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  /* med-row: dark-first surface — uses surface token so light theme overrides naturally */
  .med-row {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: 9px 14px;
    /* Token-aware: var(--surface-1) resolves to a dark surface in dark mode,
       and a light surface in [data-theme="light"] — no hardcoded white */
    background: var(--surface-1, rgba(15, 23, 42, 0.70));
    backdrop-filter: blur(8px);
  }

  :global([data-theme="light"]) .med-row {
    background: rgba(255, 255, 255, 0.92);
  }

  /* Blood type — extra-large, high-contrast, centered (most critical field) */
  .med-row-bloodtype {
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-1);
    text-align: center;
    padding: var(--space-3) var(--space-3-5);
    background: var(--danger-500-12);
  }
  .med-blood-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--danger-400);
  }
  .med-blood-value {
    font-family: var(--font-display);
    font-size: var(--text-4xl);
    font-weight: 800;
    color: var(--danger-500);
    line-height: 1;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
    text-shadow: 0 0 18px color-mix(in oklch, var(--danger-500) 35%, transparent);
  }

  /* Allergies — red highlight */
  .med-row-alert {
    background: color-mix(in oklch, var(--danger-500) 7%, transparent);
  }
  .med-row-alert .med-field-label { color: var(--danger-600); }

  /* Contact rows — empty rule kept for selector specificity */
  .med-row-contact {}

  .med-field-icon {
    flex-shrink: 0;
    color: var(--danger-400);
    margin-top: 2px;
    display: flex;
    align-items: center;
  }

  .med-field-content {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    min-width: 0;
    flex: 1;
  }

  .med-field-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-tertiary);
  }

  .med-field-value {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    line-height: 1.4;
    word-break: break-word;
  }

  /* ── Medical field chips — hierarchy: danger > warning > neutral ─────────── */
  .med-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1-5);
  }
  .med-chip {
    display: inline-block;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-full);
    line-height: 1.3;
    /* no truncation — long entries wrap within the chip */
    word-break: break-word;
    white-space: normal;
  }
  .med-chip-danger {
    background: var(--danger-500-12);
    color: var(--danger-300);
    border: 1px solid var(--danger-500-20);
  }
  .med-chip-warning {
    background: color-mix(in oklch, var(--warning-500) 14%, transparent);
    color: var(--warning-500);
    border: 1px solid color-mix(in oklch, var(--warning-500) 28%, transparent);
  }
  .med-chip-neutral {
    background: var(--surface-2, rgba(255, 255, 255, 0.06));
    color: var(--text-secondary);
    border: 1px solid var(--border-default);
  }

  .med-phone-link {
    color: var(--primary-500);
    text-decoration: none;
    font-weight: 600;
  }
  .med-phone-link:hover { text-decoration: underline; }
  .med-relation { opacity: 0.75; font-weight: 400; }
  .med-address  { opacity: 0.7; font-size: 0.8em; }
</style>
