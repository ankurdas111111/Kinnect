<script>
  import { getAvatarStyle, getPresenceRingStyle } from './avatarPalette.js';

  let { user } = $props();
</script>

<div class="user-avatar" class:av-offline={user.online === false} style="{getAvatarStyle(user.displayName)}{user.online === false ? 'filter: saturate(0.4);' : ''}">
  {(user.displayName || 'U')[0].toUpperCase()}
  <!-- Presence ring overlay — colored ring signals status -->
  <span
    class="presence-ring"
    class:ring-sos={user.sos?.active}
    class:ring-offline={user.online === false}
    class:ring-online={user.online !== false && !user.sos?.active}
    style={user.online !== false && !user.sos?.active ? getPresenceRingStyle(user) : ''}
    aria-hidden="true"
  ></span>
  <!-- TECHNIQUE 9: second pulse ring for SOS — staggered multi-ring radar effect -->
  {#if user.sos?.active}
    <span class="presence-ring-sos-outer" aria-hidden="true"></span>
  {/if}
</div>

<style>
  /* ── Avatar — 44px social-grade ────────────────────────────────────────── */
  .user-avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.0625rem; /* 17px — legible initials */
    flex-shrink: 0;
    text-transform: uppercase;
    line-height: 1;
    position: relative;
    transition: transform var(--duration-normal) var(--ease-spring);
  }

  /* Offline — dim only the avatar; name/sub stay legible */
  .user-avatar.av-offline {
    opacity: 0.50;
  }

  /* Presence ring — wraps avatar, shows relationship status */
  .presence-ring {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    pointer-events: none;
    transition: box-shadow var(--duration-normal) var(--ease-out);
  }

  /* SOS presence ring — animated danger */
  .presence-ring.ring-sos {
    box-shadow:
      0 0 0 2.5px var(--danger-500),
      0 0 0 5px rgba(0, 0, 0, 0.5),
      0 0 16px rgba(239, 68, 68, 0.45);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
  }

  /* Offline presence ring — muted */
  .presence-ring.ring-offline {
    box-shadow: 0 0 0 2px rgba(107, 114, 128, 0.30);
  }

  /* Online presence ring — gentle scale breathe (no box-shadow conflict with inline style) */
  .presence-ring.ring-online {
    animation: ring-scale-breathe 2.8s ease-in-out infinite;
  }

  /* TECHNIQUE 9: Second (outer) SOS pulse ring — staggered 0.4s for radar sweep */
  .presence-ring-sos-outer {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    pointer-events: none;
    box-shadow:
      0 0 0 2.5px var(--danger-500),
      0 0 16px rgba(239, 68, 68, 0.35);
    animation: sos-urgent-pulse 1s ease-in-out 0.4s infinite;
    opacity: 0.6;
  }

  @keyframes ring-scale-breathe {
    0%, 100% { transform: scale(1);    opacity: 0.85; }
    50%       { transform: scale(1.07); opacity: 1;    }
  }

  @media (prefers-reduced-motion: reduce) {
    .presence-ring.ring-online { animation: none; }
    .presence-ring-sos-outer { animation: none; }
  }
</style>
