<script>
  import { getAvatarStyle } from './avatarPalette.js';
  import AvatarRing from '../primitives/AvatarRing.svelte';

  let { user } = $props();

  // Map presence state to the AvatarRing token grammar.
  // SOS > offline > live  (three distinguishable ring states).
  let ring = $derived(
    user.sos?.active   ? 'sos'
    : user.online === false ? 'offline'
    : 'live'
  );
</script>

<AvatarRing {ring} size={44}>
  {#snippet children()}
    <div
      class="user-avatar"
      class:av-offline={user.online === false}
      style="{getAvatarStyle(user.displayName)}{user.online === false ? 'filter: saturate(0.4);' : ''}"
    >
      {(user.displayName || 'U')[0].toUpperCase()}
    </div>
  {/snippet}
</AvatarRing>

<style>
  /* Avatar inner — AvatarRing owns the ring; this div owns the face. */
  .user-avatar {
    width: 100%;
    height: 100%;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.0625rem; /* 17px — legible initials */
    text-transform: uppercase;
    line-height: 1;
    transition: transform var(--duration-normal) var(--ease-spring);
  }

  /* Offline — dim only the avatar face; name/sub stay legible */
  .user-avatar.av-offline {
    opacity: 0.50;
  }
</style>
