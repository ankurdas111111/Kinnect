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
          src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data={encodeURIComponent(getShareOrigin() + '/#/add-contact/' + $myShareCode)}&margin=6&bgcolor=ffffff&color=0f0f23"
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
  /* ── Identity Card ──────────────────────────────────────────────── */
  .identity-card {
    background: linear-gradient(135deg, rgba(20, 184, 166, 0.06) 0%, rgba(99, 102, 241, 0.04) 60%, transparent 100%);
    border: 1px solid rgba(20, 184, 166, 0.18);
    border-top-color: rgba(45, 212, 191, 0.32);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    position: relative;
    box-shadow:
      0 0 16px rgba(20, 184, 166, 0.08),
      inset 0 1px 0 rgba(45, 212, 191, 0.10);
    overflow: hidden;
  }

  /* Shimmer sweep on identity card */
  .identity-card::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(
      105deg,
      transparent 30%,
      rgba(20,184,166,0.06) 50%,
      transparent 70%
    );
    transform: translateX(-100%);
    pointer-events: none;
    animation: holo-travel 6s ease-in-out 2s infinite;
    border-radius: inherit;
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
    color: var(--primary-400);
    text-shadow:
      0 0 10px rgba(20, 184, 166, 0.55),
      0 0 24px rgba(20, 184, 166, 0.25);
  }

  /* Skeleton placeholder while the share code is loading */
  .signal-code-skeleton {
    display: inline-block;
    width: 110px;
    height: var(--space-6, 24px);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, rgba(255,255,255,0.05));
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
    width: 28px;
    height: 28px;
    border-radius: var(--radius-md);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-tertiary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms var(--ease-out), color 150ms, transform 120ms var(--ease-spring);
  }
  .qr-icon-btn:hover {
    background: var(--surface-2);
    color: var(--primary-400);
    transform: scale(1.08);
  }
  .qr-icon-btn:active { transform: scale(0.93); }

  /* ── QR modal ────────────────────────────────────────────────────── */
  .qr-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(5, 5, 18, 0.72);
    backdrop-filter: blur(8px) saturate(1.4);
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
    border-radius: var(--radius-2xl, 20px);
    padding: var(--space-6, 24px) var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    max-width: 260px;
    width: 100%;
    box-shadow: 0 24px 60px rgba(0,0,0,0.5);
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
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }

  .qr-image-wrap {
    background: white;
    border-radius: var(--radius-lg);
    padding: 8px;
    line-height: 0;
    box-shadow: 0 4px 16px rgba(0,0,0,0.2);
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
    color: var(--primary-400);
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
    .identity-card::after { animation: none; }
  }
</style>
