<script>
  import { get } from 'svelte/store';
  import { fade, fly } from 'svelte/transition';
  import { authUser } from '../lib/stores/auth.js';
  import { myRooms, myShareCode } from '../lib/stores/rooms.js';
  import { getShareOrigin } from '../lib/env.js';
  import { toasts } from '../lib/stores/toast.js';

  export let open = false;

  let copyDone = false;
  let copyCodeDone = false;

  // App URL
  $: appUrl = getShareOrigin();

  // Best room to feature in the invite (first one, or none)
  $: featuredRoom = $myRooms.length > 0 ? $myRooms[0] : null;

  // Invite text — conversational Indian-family tone
  $: waText = buildWaText($authUser, featuredRoom, appUrl);

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
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="inv-backdrop" transition:fade={{ duration: 150 }} on:click|self={close}>
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
        <div class="inv-title-block">
          <p class="inv-title">Invite Family</p>
          <p class="inv-sub">Invite family, friends &amp; close ones via WhatsApp</p>
        </div>
        <button class="inv-close-btn" on:click={close} aria-label="Close">
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
        <div class="inv-code-row">
          <div class="inv-code-block">
            <span class="inv-code-label">{featuredRoom ? 'Group code' : 'My code'}</span>
            <span class="inv-code-value">{code}</span>
          </div>
          <button class="inv-copy-code-btn" on:click={copyCode} aria-label="Copy code">
            {#if copyCodeDone}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
              <span style="color:#4ade80">Copied</span>
            {:else}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              <span>Copy</span>
            {/if}
          </button>
        </div>
      {/if}

      <!-- Action buttons -->
      <div class="inv-actions">
        <button class="inv-wa-btn" on:click={shareWhatsApp}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
          Share on WhatsApp
        </button>
        <button class="inv-copy-btn" on:click={copyLink}>
          {#if copyDone}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            <span style="color:#4ade80">Link copied!</span>
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
  .inv-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0,0,0,0.5);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }

  .inv-sheet {
    width: 100%;
    max-width: 480px;
    background: var(--surface-1);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 20px 20px 0 0;
    padding: 0 0 max(var(--space-4, 16px), env(safe-area-inset-bottom, 0px));
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .inv-drag-handle {
    width: 36px;
    height: 4px;
    border-radius: 2px;
    background: rgba(255,255,255,0.15);
    margin: 12px auto 4px;
    flex-shrink: 0;
  }

  /* Header */
  .inv-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px 14px;
  }

  .inv-icon-ring {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: rgba(74,222,128,0.09);
    border: 1px solid rgba(74,222,128,0.18);
    color: var(--success-400, #4ade80);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .inv-title-block { flex: 1; min-width: 0; }

  .inv-title {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: var(--text-primary);
    font-family: var(--font-body, system-ui), sans-serif;
  }

  .inv-sub {
    margin: 2px 0 0;
    font-size: 12px;
    color: rgba(255,255,255,0.35);
    font-family: system-ui, sans-serif;
  }

  .inv-close-btn {
    width: 32px; height: 32px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.3);
    cursor: pointer;
    border-radius: 8px;
    transition: background 0.15s;
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .inv-close-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.6); }

  /* Preview */
  .inv-preview {
    margin: 0 14px 12px;
    padding: 12px 14px;
    background: rgba(37,211,102,0.05);
    border: 1px solid rgba(37,211,102,0.14);
    border-radius: 12px;
    display: flex;
    gap: 10px;
    align-items: flex-start;
  }

  .inv-preview-wa-icon {
    color: var(--whatsapp-green, #25d366);
    flex-shrink: 0;
    margin-top: 1px;
  }

  .inv-preview-text {
    margin: 0;
    font-size: 12px;
    line-height: 1.65;
    color: rgba(255,255,255,0.5);
    font-family: system-ui, sans-serif;
    white-space: pre-wrap;
    word-break: break-word;
  }

  /* Code row */
  .inv-code-row {
    margin: 0 14px 12px;
    padding: 10px 14px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 10px;
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
    font-size: 10px;
    color: rgba(255,255,255,0.3);
    font-family: system-ui, sans-serif;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .inv-code-value {
    font-size: 15px;
    font-weight: 700;
    color: var(--text-primary);
    font-family: system-ui, monospace, sans-serif;
    letter-spacing: 0.1em;
  }

  .inv-copy-code-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 7px 12px;
    min-height: 44px;
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: rgba(255,255,255,0.5);
    font-size: 12px;
    font-weight: 500;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s;
    touch-action: manipulation;
  }
  .inv-copy-code-btn:hover { background: rgba(255,255,255,0.08); }

  /* Actions */
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
    border-radius: 14px;
    border: none;
    background: #25d366;
    color: #fff;
    font-size: 15px;
    font-weight: 700;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
    touch-action: manipulation;
  }
  .inv-wa-btn:hover { background: #1fb855; }
  .inv-wa-btn:active { transform: scale(0.98); }

  .inv-copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    min-height: 48px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: rgba(255,255,255,0.55);
    font-size: 14px;
    font-weight: 500;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    transition: background 0.15s;
    touch-action: manipulation;
  }
  .inv-copy-btn:hover { background: rgba(255,255,255,0.07); }
</style>
