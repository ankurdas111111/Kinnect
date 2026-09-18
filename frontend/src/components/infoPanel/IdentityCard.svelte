<script>
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import { myShareCode, myContactInfo } from '../../lib/stores/rooms.js';
  import { getShareOrigin } from '../../lib/env.js';
  import CopyButton from '../primitives/CopyButton.svelte';

  let showQr = $state(false);
</script>

<!-- ── IDENTITY CARD ────────────────────────────────────────────── -->
<div class="identity-card">
  <span class="card-eyebrow">Your Code</span>
  <div class="identity-body">
    {#if $myShareCode}
      <code class="signal-code">{$myShareCode}</code>
    {:else}
      <!-- Skeleton while the share code loads from the server -->
      <span class="signal-code-skeleton" role="status" aria-label="Loading your code" aria-busy="true"></span>
    {/if}
    <div class="signal-btns">
      <CopyButton text={$myShareCode || ''} label="Copy" />
      {#if $myShareCode}
        <button class="qr-icon-btn" onclick={() => showQr = true} aria-label="Show QR code for signal code" title="QR code">
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/><rect x="18" y="14" width="3" height="3"/><rect x="14" y="18" width="3" height="3"/><rect x="18" y="18" width="3" height="3"/></svg>
        </button>
      {/if}
    </div>
  </div>
  {#if $myContactInfo?.email || $myContactInfo?.mobile}
    <span class="identity-meta">{[$myContactInfo.email, $myContactInfo.mobile].filter(Boolean).join(' · ')}</span>
  {/if}
</div>

<!-- ── QR code modal (fixed overlay) ─────────────────────────────── -->
{#if showQr}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="qr-backdrop" onclick={() => showQr = false} role="dialog" aria-modal="true" aria-label="Signal code QR">
    <div class="qr-modal" onclick={stopPropagation(bubble('click'))}>
      <button class="qr-close-btn" onclick={() => showQr = false} aria-label="Close QR code">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <span class="qr-title">Your Family Code</span>
      <div class="qr-image-wrap">
        <img
          src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data={encodeURIComponent(getShareOrigin() + '/#/add-contact/' + $myShareCode)}&margin=6&bgcolor=ffffff&color=38332e"
          alt="QR code for signal code {$myShareCode}"
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
  /* ── Identity Card — a quiet ember-tinted paper sheet, no holo shimmer ── */
  .identity-card {
    background: var(--primary-500-12, color-mix(in oklch, var(--primary-500) 10%, transparent));
    border-radius: var(--radius-card, 20px);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    /* The parent (.info-root) is a scrolling flex column: without this the
       card is squashed to its eyebrow (~34px) while its body needs ~110px,
       and overflow:hidden then chops the code + Copy/QR buttons clean off. */
    flex-shrink: 0;
    position: relative;
    overflow: hidden;
  }
  .identity-body {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }
  .signal-code {
    font-family: var(--font-mono);
    font-size: var(--text-xl);
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--primary-700);
    font-variant-numeric: tabular-nums;
  }

  /* Skeleton placeholder while the share code is loading */
  .signal-code-skeleton {
    display: inline-block;
    width: 110px;
    height: var(--space-6, 24px);
    border-radius: var(--radius-sm);
    /* Ink-tint shimmer base — the global --skeleton-base is white-on-white
       in light Hearth (flagged for a shared fix). */
    background: color-mix(in oklch, var(--text-primary) 7%, transparent);
    animation: skel-pulse var(--skeleton-duration, 1.6s) ease-in-out infinite;
  }
  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  .identity-meta {
    font-size: 11px;
    color: var(--text-tertiary);
  }

  /* ── Signal code actions group ──────────────────────────────────── */
  .signal-btns {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    flex-shrink: 0;
  }

  /* ── QR icon button ──────────────────────────────────────────────── */
  .qr-icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-input);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms var(--ease-out), color 150ms, transform 120ms var(--ease-spring);
  }
  .qr-icon-btn:hover {
    background: var(--surface-2);
    color: var(--primary-700);
    transform: scale(1.05);
  }
  .qr-icon-btn:active { transform: scale(0.95); }
  .qr-icon-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  /* ── QR modal ────────────────────────────────────────────────────── */
  .qr-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: color-mix(in oklch, var(--ink) 45%, transparent);
    backdrop-filter: blur(8px);
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
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sheet, 24px);
    padding: var(--space-6, 24px) var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    max-width: min(100%, 260px);
    width: 100%;
    box-shadow: var(--shadow-xl);
    animation: qr-slide-up 220ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)) both;
  }
  @keyframes qr-slide-up { from { transform: translateY(16px) scale(0.95); opacity: 0; } to { transform: none; opacity: 1; } }

  .qr-close-btn {
    position: absolute;
    top: var(--space-3);
    right: var(--space-3);
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms;
  }
  .qr-close-btn:hover { background: var(--surface-3, var(--surface-2)); color: var(--text-primary); }

  .qr-title {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
  }

  .qr-image-wrap {
    /* Scanners need a true-white quiet zone in both themes. */
    background: oklch(1 0 0);
    border-radius: var(--radius-lg);
    padding: 8px;
    line-height: 0;
    box-shadow: var(--shadow-sm);
  }

  .qr-image {
    display: block;
    width: 180px;
    height: 180px;
    border-radius: 4px;
  }

  .qr-code-display {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--primary-700);
    letter-spacing: 0.08em;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 4px 10px;
  }

  .qr-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    text-align: center;
    line-height: 1.45;
    margin: 0;
    max-width: 200px;
  }

  @media (prefers-reduced-motion: reduce) {
    .qr-backdrop { animation: none; }
    .qr-modal { animation: none; }
    .signal-code-skeleton { animation: none; opacity: 0.7; }
  }
</style>
