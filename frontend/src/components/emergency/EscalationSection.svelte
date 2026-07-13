<script>
  /**
   * EscalationSection — merges the former Panic Relay (SMS escalation) and
   * Heartbeat Check sections into one server-backed "if something goes wrong"
   * surface (critique fix: they were two ~50-line siblings doing the same job).
   *
   * These values live server-side, so this section shows a Skeleton while the
   * getEmergencySettings hydration is in flight — otherwise configured values
   * render empty on revisit and the user assumes their SMS escalation is off.
   *
   * Presentational: panic phones + heartbeat state bindable; save/emit stays in
   * the parent via callbacks. The heartbeat toggle uses the shared
   * ToggleControl (safety-switch a11y contract). Field ids preserved:
   * ep-panic-1, ep-panic-2, ep-heartbeat-deadline.
   */
  import ToggleControl from '../primitives/ToggleControl.svelte';
  import Skeleton from '../primitives/Skeleton.svelte';

  /**
   * @type {{
   *   loading?: boolean,
   *   panicPhone1?: string,
   *   panicPhone2?: string,
   *   panicSaving?: boolean,
   *   heartbeatEnabled?: boolean,
   *   heartbeatDeadline?: string,
   *   onsavePanic: () => void,
   *   onsaveHeartbeat: () => void,
   * }}
   */
  let {
    loading = false,
    panicPhone1 = $bindable(''),
    panicPhone2 = $bindable(''),
    panicSaving = false,
    heartbeatEnabled = $bindable(false),
    heartbeatDeadline = $bindable('10:00'),
    onsavePanic,
    onsaveHeartbeat,
  } = $props();
</script>

{#if loading}
  <!-- Hydration in flight — never flash empty escalation fields -->
  <div class="ep-esc-skeleton" role="status" aria-label="Loading escalation settings" aria-busy="true">
    <Skeleton variant="line" width="80%" />
    <Skeleton variant="card" height="44px" />
    <Skeleton variant="card" height="44px" />
    <Skeleton variant="line" width="60%" />
  </div>
{:else}
  <!-- ── Panic Relay: external SMS escalation ── -->
  <div class="ep-esc-group">
    <p class="ep-hint">
      If no one acknowledges your SOS within 3 minutes, Kinnect sends an SMS with your
      live location to these phone numbers.
    </p>
    <div class="ep-field">
      <label class="ep-field-label" for="ep-panic-1">Emergency Phone 1</label>
      <input
        id="ep-panic-1"
        class="ep-input"
        type="tel"
        inputmode="tel"
        bind:value={panicPhone1}
        placeholder="+91 98765 43210"
        maxlength="20"
        autocomplete="tel"
      />
    </div>
    <div class="ep-field">
      <label class="ep-field-label ep-field-label--spaced" for="ep-panic-2">Emergency Phone 2 (optional)</label>
      <input
        id="ep-panic-2"
        class="ep-input"
        type="tel"
        inputmode="tel"
        bind:value={panicPhone2}
        placeholder="+91 98765 43210"
        maxlength="20"
        autocomplete="tel"
      />
    </div>
    <button class="ep-save-inline-btn" onclick={onsavePanic} disabled={panicSaving}>
      {panicSaving ? 'Saving…' : 'Save SMS Contacts'}
    </button>
  </div>

  <!-- ── Heartbeat Check: daily wellness pulse ── -->
  <div class="ep-esc-group ep-esc-group--divided">
    <p class="ep-hint">
      If you don't open the app or share location by the deadline, your family gets a
      gentle "Haven't heard from you today" notification.
    </p>
    <ToggleControl
      label="Daily heartbeat check"
      description="A gentle nudge to your family if you go quiet"
      bind:checked={heartbeatEnabled}
      onchange={onsaveHeartbeat}
    />
    {#if heartbeatEnabled}
      <div class="ep-field">
        <label class="ep-field-label ep-field-label--spaced" for="ep-heartbeat-deadline">Deadline (UTC)</label>
        <input
          id="ep-heartbeat-deadline"
          class="ep-input"
          type="time"
          bind:value={heartbeatDeadline}
          onchange={onsaveHeartbeat}
        />
      </div>
    {/if}
  </div>
{/if}

<style>
  .ep-esc-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2-5);
  }
  .ep-esc-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-2-5);
  }
  .ep-esc-group--divided {
    padding-top: var(--space-3-5);
    border-top: 1px solid var(--border-subtle);
  }
</style>
