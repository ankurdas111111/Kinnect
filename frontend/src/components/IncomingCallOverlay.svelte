<script>
  import { incomingOffer, callState } from '../lib/stores/webrtc.js';
  import { acceptCall } from '../lib/webrtc.js';

  function accept() {
    const offer = $incomingOffer;
    if (!offer) return;
    acceptCall(offer.fromUserID, offer.fromName, offer.sdp);
  }

  function decline() {
    incomingOffer.set(null);
  }
</script>

{#if $incomingOffer && $callState === 'idle'}
  <!-- Backdrop -->
  <div class="backdrop" onclick={decline} aria-hidden="true"></div>

  <!-- Bottom-sheet card -->
  <div
    class="call-sheet"
    role="dialog"
    aria-modal="true"
    aria-label="Incoming walkie-talkie call"
  >
    <div class="caller-avatar" aria-hidden="true">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
        <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
        <line x1="12" y1="19" x2="12" y2="23"/>
        <line x1="8" y1="23" x2="16" y2="23"/>
      </svg>
      <span class="ring-pulse"></span>
    </div>

    <div class="call-info">
      <p class="call-label">Incoming walkie-talkie</p>
      <p class="caller-name">{$incomingOffer.fromName}</p>
    </div>

    <div class="call-actions">
      <button class="btn-decline" onclick={decline} aria-label="Decline call">Decline</button>
      <button class="btn-accept" onclick={accept} aria-label="Accept call from {$incomingOffer.fromName}">Accept</button>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    /* Was z:999 = same as BottomSheet backdrop (--z-panel - 1).
       Incoming call must appear above the bottom sheet, so use overlay tier. */
    z-index: calc(var(--z-overlay, 3000) - 1);
  }

  .call-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    /* Was z:1000 = --z-panel, same layer as BottomSheet — causes stacking conflict.
       Incoming call is safety-critical; must render above bottom sheet. */
    z-index: var(--z-overlay, 3000);
    padding: var(--space-6, 24px) var(--space-5, 20px);
    padding-bottom: max(var(--space-6, 24px), env(safe-area-inset-bottom));
    background: var(--bg-card, #1e2435);
    border-top: 1px solid var(--border-subtle, rgba(255,255,255,0.1));
    border-radius: var(--radius-xl, 16px) var(--radius-xl, 16px) 0 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4, 16px);
    animation: slide-up 240ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }

  @keyframes slide-up {
    from { transform: translateY(100%); }
    to   { transform: translateY(0); }
  }

  /* ── Caller avatar with pulse ring ─────────────────────────── */
  .caller-avatar {
    position: relative;
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(20, 184, 166, 0.15);
    border: 2px solid rgba(20, 184, 166, 0.4);
    border-radius: 50%;
    color: var(--brand-primary, #14b8a6);
  }

  .ring-pulse {
    position: absolute;
    inset: -8px;
    border-radius: 50%;
    border: 2px solid rgba(20, 184, 166, 0.35);
    animation: ring 1.4s ease-out infinite;
  }

  @keyframes ring {
    0%   { transform: scale(1); opacity: 0.7; }
    100% { transform: scale(1.5); opacity: 0; }
  }

  /* ── Call info text ─────────────────────────────────────────── */
  .call-info {
    text-align: center;
  }

  .call-label {
    margin: 0;
    font-size: var(--text-xs, 11px);
    color: var(--text-tertiary, rgba(255,255,255,0.45));
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .caller-name {
    margin: var(--space-1, 4px) 0 0;
    font-size: var(--text-lg, 18px);
    font-weight: 600;
    color: var(--text-primary, #f8fafc);
  }

  /* ── Action buttons ─────────────────────────────────────────── */
  .call-actions {
    display: flex;
    gap: var(--space-3, 12px);
    width: 100%;
  }

  .btn-decline,
  .btn-accept {
    flex: 1;
    min-height: 48px;
    border: none;
    border-radius: var(--radius-md, 8px);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    cursor: pointer;
    transition: background 120ms, box-shadow 120ms;
  }

  .btn-decline {
    background: rgba(239, 68, 68, 0.15);
    color: var(--status-danger, #ef4444);
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .btn-decline:hover { background: rgba(239, 68, 68, 0.25); }

  .btn-accept {
    background: var(--brand-primary, #14b8a6);
    color: #fff;
    box-shadow: 0 4px 16px rgba(20, 184, 166, 0.35);
  }

  .btn-accept:hover { background: var(--brand-primary-dark, #0d9488); }

  @media (prefers-reduced-motion: reduce) {
    .call-sheet { animation: none; }
    .ring-pulse { animation: none; }
    .btn-decline, .btn-accept { transition: none; }
  }
</style>
