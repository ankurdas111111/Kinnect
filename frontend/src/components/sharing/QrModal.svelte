<script>
  /**
   * QrModal — the family-code QR in a dismissible overlay.
   *
   * Shared by the Share panel's "QR Code" quick action (and mirrors the
   * modal inside infoPanel/IdentityCard). Scanning opens /#/add-contact/
   * with the code prefilled.
   */
  import { myShareCode } from '../../lib/stores/rooms.js';
  import { getShareOrigin } from '../../lib/env.js';

  /** @type {{ open?: boolean }} */
  let { open = $bindable(false) } = $props();

  function close() { open = false; }
</script>

{#if open && $myShareCode}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="qr-backdrop" onclick={close} role="dialog" aria-modal="true" aria-label="Family code QR">
    <div class="qr-modal" onclick={(e) => e.stopPropagation()}>
      <button class="qr-close-btn" onclick={close} aria-label="Close QR code">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <span class="qr-title">Your family code</span>
      <div class="qr-image-wrap">
        <img
          src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data={encodeURIComponent(getShareOrigin() + '/#/add-contact/' + $myShareCode)}&margin=6&bgcolor=ffffff&color=38332e"
          alt="QR code for family code {$myShareCode}"
          width="180"
          height="180"
          class="qr-image"
          loading="lazy"
        />
      </div>
      <code class="qr-code-display">{$myShareCode}</code>
      <p class="qr-hint">Anyone who scans this can connect with you on Kinnect</p>
    </div>
  </div>
{/if}

<style>
  /* ═══ Hearth: warm-ink veil, paper card, serif title ═══ */
  .qr-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: var(--shadow-color);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    animation: qr-fade-in 180ms var(--ease-out) both;
  }
  @keyframes qr-fade-in { from { opacity: 0; } to { opacity: 1; } }

  .qr-modal {
    position: relative;
    background: var(--surface-1);
    border-radius: var(--radius-xl);
    padding: var(--space-6, 24px) var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    max-width: min(100%, 280px);
    width: 100%;
    box-shadow: var(--shadow-xl);
    animation: qr-slide-up 220ms var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }
  @keyframes qr-slide-up { from { transform: translateY(16px) scale(0.95); opacity: 0; } to { transform: none; opacity: 1; } }

  .qr-close-btn {
    position: absolute;
    top: var(--space-1);
    right: var(--space-1);
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: transparent;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    transition: background-color var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
  }
  .qr-close-btn:hover { background: var(--surface-hover); color: var(--text-primary); }
  .qr-close-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .qr-title {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-2xl);
    letter-spacing: -0.01em;
    color: var(--text-primary);
  }

  .qr-image-wrap {
    /* Always-light quiet zone so the code scans in night mode too */
    background: var(--text-inverse);
    border-radius: var(--radius-lg);
    padding: 8px;
    line-height: 0;
    box-shadow: var(--shadow-sm);
  }

  .qr-image {
    display: block;
    width: 180px;
    height: 180px;
    border-radius: var(--radius-sm);
  }

  .qr-code-display {
    font-family: var(--font-mono);
    font-size: var(--text-base);
    font-weight: 700;
    color: var(--primary-700);
    letter-spacing: 0.08em;
    font-variant-numeric: tabular-nums;
    background: var(--primary-100);
    border-radius: var(--radius-md);
    padding: 4px 12px;
  }

  .qr-hint {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    text-align: center;
    line-height: 1.5;
    margin: 0;
    max-width: 210px;
  }

  @media (prefers-reduced-motion: reduce) {
    .qr-backdrop, .qr-modal { animation: none; }
  }
</style>
