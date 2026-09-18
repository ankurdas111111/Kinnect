<script>
  import { self } from 'svelte/legacy';

  import { get } from 'svelte/store';
  import { fade, fly } from 'svelte/transition';
  import { authUser } from '../lib/stores/auth.js';
  import { myRooms, myShareCode } from '../lib/stores/rooms.js';
  import { getShareOrigin } from '../lib/env.js';
  import { toasts } from '../lib/stores/toast.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [open]
   */

  /** @type {Props} */
  let { open = $bindable(false) } = $props();

  let copyDone = $state(false);
  let copyCodeDone = $state(false);




  function buildWaText(user, room, url) {
    const name = user?.displayName?.split(' ')[0] ?? 'me';
    if (room) {
      return (
        `Hey! Join ${name} on Kinnect 👋\n` +
        `It's a safety app for family, friends & close ones — see each other's live location 📍\n\n` +
        `Join my group with code: *${room.code}*\n\n` +
        `Join me here: ${url}`
      );
    }
    const code = user?.shareCode ?? '';
    const codeText = code ? `Add me using my code: *${code}*\n\n` : '';
    return (
      `Hey! 👋\n` +
      `I use Kinnect — a safety app for family, friends & close ones — see each other's live location 📍\n\n` +
      `${codeText}` +
      `Join me here: ${url}`
    );
  }

  function shareWhatsApp() {
    window.open('https://wa.me/?text=' + encodeURIComponent(waText), '_blank', 'noopener');
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(appUrl);
      copyDone = true;
      setTimeout(() => { copyDone = false; }, 2500);
    } catch {
      toasts.error('Could not copy link');
    }
  }

  async function copyCode() {
    const code = featuredRoom?.code ?? get(authUser)?.shareCode;
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code);
      copyCodeDone = true;
      setTimeout(() => { copyCodeDone = false; }, 2500);
    } catch {
      toasts.error('Could not copy code');
    }
  }

  function close() { open = false; }

  /** Escape closes the sheet. Without this the backdrop stayed up and swallowed
   *  every click behind it — a keyboard trap, and the app's own modal primitive
   *  (primitives/Modal.svelte) already behaves this way. */
  function onWindowKeydown(e) {
    if (open && e.key === 'Escape') { e.preventDefault(); close(); }
  }
  // App URL
  let appUrl = $derived(getShareOrigin());
  // Best room to feature in the invite (first one, or none)
  let featuredRoom = $derived($myRooms.length > 0 ? $myRooms[0] : null);
  // Invite text — conversational Indian-family tone
  let waText = $derived(buildWaText($authUser, featuredRoom, appUrl));
</script>

<svelte:window on:keydown={onWindowKeydown} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="inv-backdrop" transition:fade={{ duration: 150 }} onclick={self(close)} role="dialog" aria-modal="true" aria-label="Invite your family">
    <div class="inv-sheet" transition:fly={{ y: 80, duration: 220 }}>
      <div class="inv-drag-handle" aria-hidden="true"></div>

      <!-- Header -->
      <div class="inv-header">
        <div class="inv-icon-ring">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
        </div>
        <!-- Hearth 03c: name the family, and say exactly what the link does. -->
        <div class="inv-title-block">
          <p class="inv-title">{featuredRoom?.name ? `Invite to ${featuredRoom.name}` : 'Invite your family'}</p>
          <p class="inv-sub">
            Anyone with this link joins as a member and sees the whole family.
            You'll get a nudge the moment someone joins, and you can remove anyone later.
          </p>
        </div>
        <button class="inv-close-btn" onclick={close} aria-label="Close">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <!-- Preview card -->
      <div class="inv-preview">
        <div class="inv-preview-wa-icon" aria-hidden="true">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
        </div>
        <p class="inv-preview-text">{waText}</p>
      </div>

      <!-- Code pill (if available) -->
      {#if featuredRoom || $authUser?.shareCode}
        {@const code = featuredRoom?.code ?? $authUser?.shareCode}
        <!-- QR + code side by side (Hearth 03c) -->
        <div class="inv-qr-row">
          <img
            class="inv-qr"
            src="https://api.qrserver.com/v1/create-qr-code/?size=132x132&data={encodeURIComponent(getShareOrigin() + '/#/add-contact/' + code)}&margin=4&bgcolor=ffffff&color=38332e"
            alt="QR code to join {featuredRoom?.name || 'this family'}"
            width="132" height="132" loading="lazy"
          />
          <p class="inv-qr-hint">Scan to join</p>
        </div>
        <div class="inv-code-row">
          <div class="inv-code-block">
            <span class="inv-code-label">{featuredRoom ? 'Family code' : 'My code'}</span>
            <span class="inv-code-value">{code}</span>
          </div>
          <button class="inv-copy-code-btn" class:is-done={copyCodeDone} onclick={copyCode} aria-label="Copy code">
            {#if copyCodeDone}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
              <span>Copied</span>
            {:else}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              <span>Copy</span>
            {/if}
          </button>
        </div>
      {/if}

      <!-- Action buttons -->
      <div class="inv-actions">
        <button class="inv-wa-btn" onclick={shareWhatsApp}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
          Share on WhatsApp
        </button>
        <button class="inv-copy-btn" class:is-done={copyDone} onclick={copyLink}>
          {#if copyDone}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Link copied!</span>
          {:else}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
            <span>Copy link</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* ═══ Hearth: warm-ink veil, then a paper sheet ═══ */
  .inv-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: var(--shadow-color);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }

  .inv-sheet {
    width: 100%;
    max-width: 480px;
    background: var(--surface-1);
    border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
    box-shadow: var(--sh-up);
    padding: 0 0 max(var(--space-4, 16px), env(safe-area-inset-bottom, 0px));
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .inv-drag-handle {
    width: 36px;
    height: 4px;
    border-radius: var(--radius-full);
    background: var(--border-strong);
    margin: 12px auto 4px;
    flex-shrink: 0;
  }

  /* Header — pebble, serif title, plain warm sentence */
  .inv-header {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    padding: 12px 16px 14px;
  }

  .inv-icon-ring {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    background: var(--primary-100);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .inv-title-block { flex: 1; min-width: 0; }

  .inv-title {
    margin: 0;
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-2xl);
    line-height: 1.3;
    letter-spacing: -0.01em;
    color: var(--text-primary);
  }

  .inv-sub {
    margin: var(--space-1) 0 0;
    font-size: var(--text-base);
    line-height: 1.5;
    color: var(--text-secondary);
    font-family: var(--font-sans);
  }

  .inv-close-btn {
    width: 44px; height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    border-radius: var(--radius-md);
    transition: background-color var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .inv-close-btn:hover { background: var(--surface-hover); color: var(--text-primary); }
  .inv-close-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  /* Preview — a quiet paper tier, not a branded tint */
  .inv-preview {
    margin: 0 14px 12px;
    padding: 12px 14px;
    background: var(--surface-2);
    border-radius: var(--radius-lg);
    display: flex;
    gap: 10px;
    align-items: flex-start;
  }

  .inv-preview-wa-icon {
    color: var(--text-tertiary);
    flex-shrink: 0;
    margin-top: 1px;
  }

  .inv-preview-text {
    margin: 0;
    font-size: var(--text-sm);
    line-height: 1.6;
    color: var(--text-secondary);
    font-family: var(--font-sans);
    white-space: pre-wrap;
    word-break: break-word;
  }

  /* QR + code */
  .inv-qr-row {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    margin: 0 14px 12px;
  }
  .inv-qr {
    display: block;
    width: 132px;
    height: 132px;
    border-radius: var(--radius-md);
    /* Always-light quiet zone so the code scans in night mode too */
    background: var(--text-inverse);
    padding: 6px;
    box-sizing: content-box;
    box-shadow: var(--shadow-xs);
  }
  .inv-qr-hint {
    margin: 0;
    font-size: var(--text-2xs);
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-tertiary);
  }

  .inv-code-row {
    margin: 0 14px 12px;
    padding: 10px 14px;
    background: var(--surface-2);
    border-radius: var(--radius-lg);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .inv-code-block {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .inv-code-label {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    font-family: var(--font-sans);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .inv-code-value {
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.1em;
  }

  .inv-copy-code-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 7px 12px;
    min-height: 44px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-default);
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--text-sm);
    font-weight: 600;
    font-family: var(--font-sans);
    cursor: pointer;
    flex-shrink: 0;
    transition: background-color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .inv-copy-code-btn:hover { background: var(--surface-hover); }
  .inv-copy-code-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .inv-copy-code-btn.is-done { color: var(--success-700); border-color: var(--success-500-20); }

  /* Actions — one ember fill, one paper outline */
  .inv-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0 14px 16px;
  }

  .inv-wa-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    width: 100%;
    padding: 14px;
    min-height: 52px;
    border-radius: var(--radius-button);
    border: none;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: var(--text-base);
    font-weight: 700;
    font-family: var(--font-sans);
    cursor: pointer;
    box-shadow: var(--shadow-sm);
    transition: background-color var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .inv-wa-btn:hover { background: var(--primary-600); box-shadow: var(--shadow-primary); }
  .inv-wa-btn:active { transform: scale(0.98); }
  .inv-wa-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .inv-copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    min-height: 48px;
    border-radius: var(--radius-button);
    border: 1px solid var(--border-strong);
    background: transparent;
    color: var(--text-primary);
    font-size: var(--text-base);
    font-weight: 600;
    font-family: var(--font-sans);
    cursor: pointer;
    transition: background-color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .inv-copy-btn:hover { background: var(--surface-hover); }
  .inv-copy-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .inv-copy-btn.is-done { color: var(--success-700); border-color: var(--success-500-20); }

  @media (prefers-reduced-motion: reduce) {
    .inv-close-btn,
    .inv-copy-code-btn,
    .inv-wa-btn,
    .inv-copy-btn { transition: none; }
    .inv-wa-btn:active { transform: none; }
  }
</style>
