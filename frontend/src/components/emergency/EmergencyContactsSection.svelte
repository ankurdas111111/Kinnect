<script>
  /**
   * EmergencyContactsSection — the contacts list body. Renders one ContactCard
   * per contact; empty state uses the shared EmptyState action variant with the
   * existing "Add Emergency Contact" button as its CTA (design rules: no empty
   * state without a next step).
   *
   * Presentational: `profile` bindable; add/remove flow via callbacks so all
   * mutation logic stays in the parent orchestrator.
   */
  import ContactCard from './ContactCard.svelte';
  import EmptyState from '../primitives/EmptyState.svelte';
  import { MAX_CONTACTS } from '../../lib/emergencyProfile.js';

  /** @type {{ profile: any, contactPhoneErrors?: string[], onadd: () => void, onremove: (i: number) => void }} */
  let { profile = $bindable(), contactPhoneErrors = [], onadd, onremove } = $props();
</script>

{#if profile.emergencyContacts.length === 0}
  <EmptyState
    title="No emergency contacts yet"
    body="First responders use these to notify your family immediately in an SOS."
    tone="danger"
  >
    {#snippet icon()}
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
           stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
      </svg>
    {/snippet}
    {#snippet action()}
      <button class="ep-add-contact-btn" type="button" onclick={onadd}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Emergency Contact
      </button>
    {/snippet}
  </EmptyState>
{:else}
  {#each profile.emergencyContacts as contact, i (i)}
    <ContactCard
      bind:contact={profile.emergencyContacts[i]}
      index={i}
      phoneError={contactPhoneErrors[i] || ''}
      {onremove}
    />
  {/each}

  {#if profile.emergencyContacts.length < MAX_CONTACTS}
    <button class="ep-add-contact-btn" type="button" onclick={onadd}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <line x1="12" y1="5" x2="12" y2="19"/>
        <line x1="5" y1="12" x2="19" y2="12"/>
      </svg>
      Add Emergency Contact
    </button>
  {/if}
{/if}
