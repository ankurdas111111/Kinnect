<script>
  /**
   * LiveBroadcastRow — one active live-share link (CONTRACTS.md §9 CountdownRing).
   * Presentational: copy/WhatsApp/stop are callback props; the parent owns the
   * socket revoke + URL building.
   *
   * Temporal-decay grammar: when the link has an expiry, a CountdownRing shows
   * time-remaining as a green→amber→red depleting ring, matching the viewer
   * expiry strip. `total` is captured ONCE at mount from the remaining window so
   * the ring starts full and depletes (we don't persist original creation time).
   * No expiry → a static "no expiry" chip, no ring.
   */
  import CopyButton from '../primitives/CopyButton.svelte';
  import CountdownRing from '../primitives/CountdownRing.svelte';

  /**
   * @type {{
   *   url: string,
   *   waHref: string,
   *   deadline?: number | null,
   *   onstop?: () => void,
   * }}
   */
  let { url, waHref, deadline = null, onstop } = $props();

  // Capture the full remaining window once so the ring depletes from full.
  const total = deadline ? Math.max(1, deadline - Date.now()) : 0;
</script>

<div class="broadcast-row animate-slide-up">
  <div class="broadcast-head">
    {#if deadline}
      <CountdownRing {deadline} {total} size={40} label="Live share expires" />
    {:else}
      <span class="rec-dot fx-ambient" aria-hidden="true"></span>
    {/if}
    <div class="broadcast-meta">
      <span class="broadcast-label">Live</span>
      <span class="broadcast-expiry">
        {deadline ? 'Expires ' + new Date(deadline).toLocaleTimeString() : 'No expiry'}
      </span>
    </div>
  </div>
  <div class="broadcast-actions">
    <CopyButton text={url} label="Copy Link" />
    <a
      class="btn-wa"
      href={waHref}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Share via WhatsApp"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 0 1-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 0 1-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 0 1 2.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0 0 12.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 0 0 5.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 0 0-3.48-8.413Z"/>
      </svg>
      WhatsApp
    </a>
    <button class="btn btn-danger btn-sm tactile" onclick={() => onstop?.()}>Stop Sharing</button>
  </div>
</div>

<style>
  .broadcast-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-lg);
    background: var(--glass-card-bg, var(--surface-inset));
    border: 1px solid var(--glass-card-border, var(--border-subtle));
    box-shadow: var(--glass-card-shadow, var(--shadow-xs));
  }
  .broadcast-head {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
  }
  .broadcast-meta {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .rec-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-rec);
    flex-shrink: 0;
    animation: rec-blink 1.2s var(--ease-in-out) infinite;
  }
  @keyframes rec-blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.35; }
  }
  .broadcast-label {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: var(--text-sm);
    color: var(--text-primary);
  }
  .broadcast-expiry {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .broadcast-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    align-items: center;
  }
  .btn-wa {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-1-5) var(--space-3);
    min-height: 44px;
    background: var(--whatsapp-green);
    color: var(--text-on-primary);
    border-radius: var(--radius-full);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    text-decoration: none;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    flex-shrink: 0;
    white-space: nowrap;
  }
  .btn-wa:hover { background: var(--whatsapp-dark); }
  .btn-wa:focus-visible { outline: 2px solid var(--whatsapp-green); outline-offset: 2px; }

  @media (prefers-reduced-motion: reduce) {
    .rec-dot { animation: none; }
  }
</style>
