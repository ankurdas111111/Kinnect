<script>
  import { onDestroy } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { rideShare } from '../lib/stores/rideShare.js';
  import { parseRideText, ocrImage, readClipboardText } from '../lib/rideImport.js';
  import { toasts } from '../lib/stores/toast.js';
  import { haptics } from '../lib/haptics.js';
  import { getShareOrigin } from '../lib/env.js';

  export let open = false;

  // ── Vehicle type options ──────────────────────────────────────────
  const VEHICLE_TYPES = [
    { id: 'car',     label: 'Car',     svgD: 'M3 9l1.5-3.5A2 2 0 0 1 6.34 4h11.32a2 2 0 0 1 1.84 1.5L21 9m0 0H3m18 0v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V9m4 8a1.5 1.5 0 1 0 3 0 1.5 1.5 0 0 0-3 0m9 0a1.5 1.5 0 1 0 3 0 1.5 1.5 0 0 0-3 0' },
    { id: 'scooter', label: 'Scooter', svgD: 'M5 17a2 2 0 1 0 4 0 2 2 0 0 0-4 0m10 0a2 2 0 1 0 4 0 2 2 0 0 0-4 0M7 17h2m5 0h3m-3 0V8.5l-4-3H5.5A1.5 1.5 0 0 0 4 7v3h3M14 5h3l3 3' },
    { id: 'auto',    label: 'Auto',    svgD: 'M4 17a2 2 0 1 0 4 0 2 2 0 0 0-4 0M2 9h14v8H6M2 9l2-5h10l2 5M16 12h4l2 3v2h-6' },
    { id: 'cab',     label: 'Cab',     svgD: 'M3 9l1.5-3.5A2 2 0 0 1 6.34 4h11.32a2 2 0 0 1 1.84 1.5L21 9m0 0H3m18 0v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V9m3-3h3v3H6zm7 0h2v2h-2zM7 17a1.5 1.5 0 1 0 3 0 1.5 1.5 0 0 0-3 0m7 0a1.5 1.5 0 1 0 3 0 1.5 1.5 0 0 0-3 0' },
    { id: 'train',   label: 'Train',   svgD: 'M8 3h8a4 4 0 0 1 4 4v8l-4 4H8l-4-4V7a4 4 0 0 1 4-4zm0 9h8M8 9h8M10 19l-2 3m6-3 2 3M9 12a1 1 0 1 0 2 0 1 1 0 0 0-2 0m4 0a1 1 0 1 0 2 0 1 1 0 0 0-2 0' },
    { id: 'walk',    label: 'Walk',    svgD: 'M13 4a1 1 0 1 0 2 0 1 1 0 0 0-2 0m1.5 3l-1 4 2.5 3-2 5m-4-7l1.5-1.5L13 13' },
  ];

  // ── ETA quick-pick options ────────────────────────────────────────
  const ETA_OPTS = [
    { label: '15m', mins: 15  },
    { label: '30m', mins: 30  },
    { label: '45m', mins: 45  },
    { label: '1h',  mins: 60  },
    { label: '2h',  mins: 120 },
  ];

  // ── Local form state ──────────────────────────────────────────────
  let vehicleType = '';
  let plateInput  = '';
  let destInput   = '';
  let etaMins     = 0;    // 0 = not set
  let starting    = false;
  let copied      = false;
  let safelyDone  = false;

  // ── Elapsed timer ─────────────────────────────────────────────────
  let elapsedSec     = 0;
  let _timerInterval = null;

  function startTimer() {
    clearInterval(_timerInterval);
    elapsedSec = 0;
    _timerInterval = setInterval(() => { elapsedSec += 1; }, 1000);
  }

  function stopTimer() {
    clearInterval(_timerInterval);
    _timerInterval = null;
  }

  onDestroy(() => stopTimer());

  let _prevActive = false;
  $: {
    if ($rideShare.active && !_prevActive) { startTimer(); _prevActive = true; }
    else if (!$rideShare.active && _prevActive) { stopTimer(); _prevActive = false; }
  }

  function formatElapsed(sec) {
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    if (h > 0) return `${h}h ${m}m`;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
  }

  // ── Derived form state ────────────────────────────────────────────
  $: showPlate = ['car', 'scooter', 'auto', 'cab'].includes(vehicleType);

  $: etaLabel = etaMins > 0
    ? new Date(Date.now() + etaMins * 60000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : '';

  // ETA countdown on the active screen.
  // Reference elapsedSec so Svelte re-evaluates this every second via the timer.
  $: etaMinsLeft = ($rideShare.eta && $rideShare.active && elapsedSec >= 0)
    ? Math.max(0, Math.round(($rideShare.eta - Date.now()) / 60000))
    : -1;

  $: activeVehicleLabel = VEHICLE_TYPES.find(v => v.id === $rideShare.vehicleType)?.label || '';

  // ── Actions ───────────────────────────────────────────────────────
  function startRide() {
    if (starting) return;
    starting = true;
    const eta = etaMins > 0 ? Date.now() + etaMins * 60000 : 0;
    // Pre-write vehicleType + eta so the socket.js merge preserves them
    rideShare.update(s => ({ ...s, vehicleType, eta }));
    socket.emit('shareRide', {
      vehicle: plateInput.trim().slice(0, 15),
      dest: destInput.trim().slice(0, 30),
    });
  }

  function endRide() {
    if (safelyDone) return;
    safelyDone = true;
    setTimeout(() => {
      socket.emit('endRide', {});
      rideShare.set({ active: false, token: '', vehicle: '', vehicleType: '', dest: '', startedAt: 0, eta: 0 });
      stopTimer();
      safelyDone = false;
      open = false;
    }, 1400);
  }

  function buildShareText(url) {
    const vLabel = VEHICLE_TYPES.find(v => v.id === $rideShare.vehicleType)?.label || '';
    let text = "I'm on my way!";
    if (vLabel)               text += ` Travelling by ${vLabel}`;
    if ($rideShare.vehicle)   text += ` (${$rideShare.vehicle})`;
    if ($rideShare.dest)      text += ` to ${$rideShare.dest}`;
    if ($rideShare.eta) {
      const d = new Date($rideShare.eta);
      text += `. ETA ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
    }
    text += `. Track my live location: ${url}`;
    return text;
  }

  function shareOnWhatsApp() {
    const token = $rideShare.token;
    if (!token) return;
    const url = getShareOrigin() + '/#/live/' + token;
    window.open('https://wa.me/?text=' + encodeURIComponent(buildShareText(url)), '_blank', 'noopener');
  }

  async function shareNative() {
    const token = $rideShare.token;
    if (!token) return;
    const url  = getShareOrigin() + '/#/live/' + token;
    const text = buildShareText(url);
    const isNative = typeof window !== 'undefined' && window.Capacitor?.isNativePlatform?.();
    if (isNative) {
      try {
        const { Share } = await import('@capacitor/share');
        await Share.share({ title: 'Track my ride', text, url, dialogTitle: 'Share ride link' });
        return;
      } catch (_) { /* dismissed */ }
    }
    // Web fallback: copy
    navigator.clipboard.writeText(url).catch(() => {});
    copied = true;
    setTimeout(() => { copied = false; }, 2000);
  }

  function copyLink() {
    const token = $rideShare.token;
    if (!token) return;
    const url = getShareOrigin() + '/#/live/' + token;
    navigator.clipboard.writeText(url).catch(() => {});
    copied = true;
    setTimeout(() => { copied = false; }, 2000);
  }

  // Reset form when sheet closes without an active ride
  $: if (!open && !$rideShare.active) {
    vehicleType = ''; plateInput = ''; destInput = ''; etaMins = 0; starting = false;
  }
  $: if ($rideShare.active) { starting = false; }

  // ── Smart Ride Import ───────────────────────────────────────────
  let importing = false;
  let importSource = '';   // 'ocr' | 'clipboard'
  let autoFilled = false;
  let fileInputEl;

  function applyParsed(result) {
    let filled = false;
    if (result.plate) {
      plateInput = result.plate;
      filled = true;
    }
    if (result.vehicleType) {
      vehicleType = result.vehicleType;
      filled = true;
    }
    if (result.etaMins) {
      const closest = ETA_OPTS.reduce((best, opt) =>
        Math.abs(opt.mins - result.etaMins) < Math.abs(best.mins - result.etaMins) ? opt : best
      );
      etaMins = closest.mins;
      filled = true;
    }
    if (filled) {
      autoFilled = true;
      haptics.success();
      setTimeout(() => { autoFilled = false; }, 1500);
    } else {
      toasts.warning("Couldn't read ride details. Enter manually.");
      haptics.warning();
    }
  }

  function handleScreenshot() { fileInputEl?.click(); }

  async function onFileSelected(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    importing = true;
    importSource = 'ocr';
    try {
      const text = await ocrImage(file);
      const result = parseRideText(text);
      applyParsed(result);
    } catch {
      toasts.error('Screenshot scan failed. Try pasting instead.');
    } finally {
      importing = false;
      importSource = '';
      if (fileInputEl) fileInputEl.value = '';
    }
  }

  async function handlePaste() {
    importing = true;
    importSource = 'clipboard';
    try {
      const text = await readClipboardText();
      if (!text || text.trim().length < 5) {
        toasts.warning('Clipboard is empty. Copy ride details from your app first.');
        return;
      }
      const result = parseRideText(text);
      applyParsed(result);
    } catch {
      toasts.error("Couldn't read clipboard. Check permissions.");
    } finally {
      importing = false;
      importSource = '';
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="ride-backdrop" on:click={() => { if (!$rideShare.active) open = false; }} aria-hidden="true"></div>
  <div class="ride-sheet" role="dialog" aria-modal="true" aria-label="Share My Ride">
    <div class="ride-handle" aria-hidden="true"></div>

    {#if !$rideShare.active}
      <!-- ── Setup screen ─────────────────────────────────────────── -->
      <div class="ride-header">
        <div class="ride-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 9l1.5-3.5A2 2 0 0 1 6.34 4h11.32a2 2 0 0 1 1.84 1.5L21 9m0 0H3m18 0v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V9"/>
            <circle cx="7.5" cy="17.5" r="1.5"/>
            <circle cx="16.5" cy="17.5" r="1.5"/>
          </svg>
        </div>
        <div>
          <h3 class="ride-title">Share My Ride</h3>
          <p class="ride-subtitle">Let family track you live until you arrive safely</p>
        </div>
      </div>

      <!-- Smart Import -->
      <div class="import-row">
        <button class="import-btn" on:click={handleScreenshot} disabled={importing}>
          {#if importing && importSource === 'ocr'}
            <span class="import-spinner" aria-hidden="true"></span> Scanning...
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/></svg>
            Scan Screenshot
          {/if}
        </button>
        <button class="import-btn" on:click={handlePaste} disabled={importing}>
          {#if importing && importSource === 'clipboard'}
            <span class="import-spinner" aria-hidden="true"></span> Reading...
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
            Paste from App
          {/if}
        </button>
      </div>
      <input type="file" accept="image/*" class="sr-only" bind:this={fileInputEl} on:change={onFileSelected} />

      <!-- Vehicle type chips -->
      <p class="field-label">How are you travelling?</p>
      <div class="vehicle-chips" class:autofilled={autoFilled} role="group" aria-label="Vehicle type">
        {#each VEHICLE_TYPES as vt}
          <button
            class="vchip"
            class:vchip-active={vehicleType === vt.id}
            on:click={() => vehicleType = vehicleType === vt.id ? '' : vt.id}
            aria-pressed={vehicleType === vt.id}
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d={vt.svgD}/>
            </svg>
            {vt.label}
          </button>
        {/each}
      </div>

      <div class="ride-form">
        {#if showPlate}
          <label class="form-label" for="ride-plate">Number plate / identifier (optional)</label>
          <input
            id="ride-plate"
            class="ride-input"
            type="text"
            maxlength="20"
            placeholder="MH01AB1234, Ola #4532…"
            bind:value={plateInput}
          />
        {/if}

        <label class="form-label" for="ride-dest">Where are you headed? (optional)</label>
        <input
          id="ride-dest"
          class="ride-input"
          type="text"
          maxlength="30"
          placeholder="Home, Office, Station…"
          bind:value={destInput}
        />

        <!-- ETA picker -->
        <p class="field-label" style="margin-bottom: 6px;">Expected arrival (optional)</p>
        <div class="eta-pills" role="group" aria-label="Expected arrival time">
          {#each ETA_OPTS as opt}
            <button
              class="pill-btn"
              class:pill-active={etaMins === opt.mins}
              on:click={() => etaMins = etaMins === opt.mins ? 0 : opt.mins}
              aria-pressed={etaMins === opt.mins}
            >{opt.label}</button>
          {/each}
          {#if etaMins > 0}
            <span class="eta-arrival">by {etaLabel}</span>
          {/if}
        </div>
      </div>

      <div class="ride-actions">
        <button class="btn btn-primary ride-start-btn" on:click={startRide} disabled={starting}>
          {#if starting}
            Creating ride link…
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M5 12h14m-7-7 7 7-7 7"/></svg>
            Share My Ride
          {/if}
        </button>
        <button class="btn btn-ghost ride-cancel-btn" on:click={() => { open = false; }}>Cancel</button>
      </div>

    {:else if safelyDone}
      <!-- ── Reached safely celebration ──────────────────────────── -->
      <div class="safely-done">
        <div class="safely-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </div>
        <p class="safely-text">You reached safely.</p>
        <p class="safely-sub">Ride link ended. Family notified.</p>
      </div>

    {:else}
      <!-- ── Active ride screen ──────────────────────────────────── -->
      <div class="ride-active-header">
        <div class="ride-active-dot" aria-hidden="true"></div>
        <div class="ride-active-meta">
          <h3 class="ride-title">
            Ride Active
            {#if activeVehicleLabel} · {activeVehicleLabel}{/if}
          </h3>
          <p class="ride-subtitle">
            {#if $rideShare.dest}to {$rideShare.dest}{/if}
            {#if $rideShare.vehicle} · {$rideShare.vehicle}{/if}
            {#if etaMinsLeft >= 0}&nbsp;· ETA {etaMinsLeft > 0 ? etaMinsLeft + 'm' : 'now'}{/if}
          </p>
        </div>
        <span class="ride-timer" aria-label="Elapsed time">{formatElapsed(elapsedSec)}</span>
      </div>

      <!-- ETA progress bar -->
      {#if $rideShare.eta && $rideShare.startedAt}
        {@const total   = $rideShare.eta - $rideShare.startedAt}
        {@const elapsed = Date.now() - $rideShare.startedAt}
        {@const pct     = Math.min(100, Math.max(0, (elapsed / total) * 100))}
        <div class="eta-bar-wrap" aria-hidden="true">
          <div class="eta-bar-track">
            <div class="eta-bar-fill" style="width:{pct}%"></div>
          </div>
          <span class="eta-bar-label">{etaMinsLeft > 0 ? etaMinsLeft + 'm left' : 'ETA reached'}</span>
        </div>
      {/if}

      <!-- Share actions -->
      <div class="ride-active-actions">
        <button class="btn-wa ride-wa-btn" on:click={shareOnWhatsApp}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347zm-5.421 7.403h-.004a9.87 9.87 0 0 1-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 0 1-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 0 1 2.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884zm8.413-18.297A11.815 11.815 0 0 0 12.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 0 0 5.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 0 0-3.48-8.413z"/>
          </svg>
          WhatsApp
        </button>

        <div class="ride-secondary-actions">
          <button class="btn-secondary-action" on:click={copyLink} aria-label="Copy link">
            {#if copied}
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
              Copied
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              Copy Link
            {/if}
          </button>

          <button class="btn-secondary-action" on:click={shareNative} aria-label="Share via other apps">
            <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>
            Share
          </button>
        </div>
      </div>

      <button class="btn-safe" on:click={endRide} disabled={safelyDone}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
        Reached Safely
      </button>
    {/if}
  </div>
{/if}

<style>
  .ride-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    z-index: 5200;
    touch-action: none;
  }

  .ride-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 5201;
    background: var(--surface-2, rgba(18, 18, 36, 0.98));
    backdrop-filter: blur(32px) saturate(1.8);
    -webkit-backdrop-filter: blur(32px) saturate(1.8);
    border-top: 1px solid var(--border-subtle, rgba(255,255,255,0.08));
    border-radius: 20px 20px 0 0;
    box-shadow: 0 -8px 48px rgba(0,0,0,0.40);
    padding: 8px 20px calc(24px + env(safe-area-inset-bottom, 0px));
  }

  .ride-handle {
    width: 40px;
    height: 5px;
    background: var(--gray-400, rgba(255,255,255,0.22));
    border-radius: 999px;
    margin: 4px auto 20px;
  }

  /* ── Header ──────────────────────────────────────────────────── */
  .ride-header {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    margin-bottom: 18px;
  }

  .ride-icon {
    width: 48px;
    height: 48px;
    border-radius: 14px;
    background: linear-gradient(135deg, rgba(99,102,241,0.2), rgba(139,92,246,0.15));
    border: 1px solid rgba(99,102,241,0.25);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400, #818cf8);
    flex-shrink: 0;
  }

  .ride-title {
    font-family: var(--font-display);
    font-size: var(--text-lg, 18px);
    font-weight: 700;
    margin: 0 0 3px;
    letter-spacing: -0.02em;
    color: var(--text-primary);
  }

  .ride-subtitle {
    font-size: var(--text-sm, 13px);
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.4;
  }

  /* ── Vehicle type chips ──────────────────────────────────────── */
  .field-label {
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    margin: 0 0 8px;
    display: block;
  }

  .vehicle-chips {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin-bottom: 16px;
  }

  .vchip {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 10px 8px;
    background: var(--surface-inset, rgba(255,255,255,0.05));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.10));
    border-radius: 12px;
    font-family: var(--font-display);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 150ms, border-color 150ms, color 150ms, transform 120ms;
  }

  .vchip:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }

  .vchip.vchip-active {
    background: rgba(99,102,241,0.18);
    border-color: var(--primary-500, #6366f1);
    color: var(--primary-300, #a5b4fc);
    box-shadow: 0 0 0 1px rgba(99,102,241,0.25);
    transform: scale(1.03);
  }

  /* ── Form inputs ─────────────────────────────────────────────── */
  .ride-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-bottom: 18px;
  }

  .form-label {
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    margin-bottom: 4px;
    display: block;
  }

  .ride-input {
    width: 100%;
    padding: 11px 14px;
    background: var(--surface-inset, rgba(255,255,255,0.05));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.10));
    border-radius: var(--radius-lg, 12px);
    color: var(--text-primary);
    font-size: var(--text-base, 15px);
    font-family: var(--font-sans);
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    box-sizing: border-box;
  }

  .ride-input:focus {
    border-color: var(--primary-500, #6366f1);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.12);
  }

  .ride-input::placeholder { color: var(--text-tertiary); }

  /* ── ETA pills ───────────────────────────────────────────────── */
  .eta-pills {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 7px;
  }

  .pill-btn {
    padding: 6px 14px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms, border-color 120ms, transform 100ms;
  }
  .pill-btn:hover { background: var(--surface-2); color: var(--text-primary); }
  .pill-btn.pill-active {
    background: var(--primary-600);
    color: white;
    border-color: var(--primary-500);
    transform: scale(1.06);
  }

  .eta-arrival {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--primary-400);
    padding: 5px 10px;
    background: rgba(99,102,241,0.10);
    border: 1px solid rgba(99,102,241,0.20);
    border-radius: var(--radius-full);
  }

  /* ── Action buttons (setup) ──────────────────────────────────── */
  .ride-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .ride-start-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 14px;
    font-size: var(--text-base, 15px);
    font-weight: 700;
  }

  .ride-cancel-btn {
    width: 100%;
    padding: 12px;
    font-size: var(--text-base, 15px);
  }

  /* ── Active ride header ──────────────────────────────────────── */
  .ride-active-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .ride-active-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--success-500, #10b981);
    flex-shrink: 0;
    box-shadow: 0 0 0 4px rgba(16,185,129,0.20);
    animation: ride-pulse 2s ease-in-out infinite;
  }

  @keyframes ride-pulse {
    0%, 100% { box-shadow: 0 0 0 4px rgba(16,185,129,0.20); }
    50%       { box-shadow: 0 0 0 8px rgba(16,185,129,0.06); }
  }

  .ride-active-meta {
    flex: 1;
    min-width: 0;
  }

  .ride-timer {
    margin-left: auto;
    font-family: var(--font-mono, monospace);
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    color: var(--text-secondary);
    background: var(--surface-inset);
    padding: 4px 10px;
    border-radius: var(--radius-full);
    flex-shrink: 0;
    letter-spacing: 0.04em;
  }

  /* ── ETA progress bar ────────────────────────────────────────── */
  .eta-bar-wrap {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 16px;
  }

  .eta-bar-track {
    flex: 1;
    height: 4px;
    background: var(--surface-inset);
    border-radius: 99px;
    overflow: hidden;
  }

  .eta-bar-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--primary-500), var(--success-500));
    border-radius: 99px;
    transition: width 1s linear;
  }

  .eta-bar-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    flex-shrink: 0;
  }

  /* ── Active ride share actions ───────────────────────────────── */
  .ride-active-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 10px;
  }

  .btn-wa {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 14px;
    font-family: var(--font-display);
    font-size: var(--text-base, 15px);
    font-weight: 700;
    background: #25d366;
    color: white;
    border: none;
    border-radius: var(--radius-lg, 12px);
    cursor: pointer;
    transition: background 150ms, transform 120ms, box-shadow 150ms;
  }
  .btn-wa:hover  { background: #1ebe5d; box-shadow: 0 4px 16px rgba(37,211,102,0.30); }
  .btn-wa:active { transform: scale(0.98); }

  .ride-secondary-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .btn-secondary-action {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    padding: 11px;
    font-family: var(--font-display);
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    background: var(--surface-inset);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-lg, 12px);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 150ms, color 150ms, border-color 150ms;
  }
  .btn-secondary-action:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
    border-color: var(--border-default);
  }

  /* ── Reached Safely button ───────────────────────────────────── */
  .btn-safe {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 14px;
    font-family: var(--font-display);
    font-size: var(--text-base, 15px);
    font-weight: 700;
    background: var(--success-500, #10b981);
    color: white;
    border: none;
    border-radius: var(--radius-lg, 12px);
    cursor: pointer;
    transition: background 150ms, opacity 150ms, transform 120ms;
  }
  .btn-safe:hover  { background: #0ea774; }
  .btn-safe:active { transform: scale(0.98); }
  .btn-safe:disabled { opacity: 0.7; cursor: default; }

  /* ── "Reached Safely" success screen ────────────────────────── */
  .safely-done {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 32px 20px 16px;
    gap: 10px;
    animation: safely-pop 0.35s var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)) forwards;
  }

  @keyframes safely-pop {
    from { opacity: 0; transform: scale(0.88); }
    to   { opacity: 1; transform: scale(1); }
  }

  .safely-icon {
    width: 72px;
    height: 72px;
    border-radius: 50%;
    background: rgba(16,185,129,0.15);
    border: 2px solid rgba(16,185,129,0.30);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--success-500, #10b981);
    box-shadow: 0 0 32px rgba(16,185,129,0.20);
    animation: safely-glow 1.2s ease-in-out infinite;
  }

  @keyframes safely-glow {
    0%, 100% { box-shadow: 0 0 24px rgba(16,185,129,0.18); }
    50%       { box-shadow: 0 0 44px rgba(16,185,129,0.38); }
  }

  .safely-text {
    font-family: var(--font-display);
    font-size: var(--text-xl, 20px);
    font-weight: 800;
    color: var(--success-500, #10b981);
    margin: 0;
    letter-spacing: -0.02em;
  }

  .safely-sub {
    font-size: var(--text-sm, 13px);
    color: var(--text-tertiary);
    margin: 0;
  }

  /* ── Smart Import ──────────────────────────────────────────── */
  .import-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    margin-bottom: 14px;
  }

  .import-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 10px 8px;
    font-family: var(--font-display);
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    color: var(--text-secondary);
    background: var(--surface-inset, rgba(255,255,255,0.05));
    border: 1px dashed var(--border-subtle, rgba(255,255,255,0.12));
    border-radius: 12px;
    cursor: pointer;
    transition: background 150ms, border-color 150ms, color 150ms;
  }
  .import-btn:hover {
    background: var(--surface-hover);
    border-color: var(--primary-500);
    color: var(--text-primary);
  }
  .import-btn:disabled {
    opacity: 0.6;
    cursor: wait;
  }

  .import-spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid var(--text-tertiary);
    border-top-color: var(--primary-400);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .autofilled {
    animation: autofill-glow 1.2s ease-out;
  }
  @keyframes autofill-glow {
    0%   { box-shadow: 0 0 0 3px rgba(99,102,241,0.4); }
    100% { box-shadow: 0 0 0 0 rgba(99,102,241,0); }
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0,0,0,0);
    white-space: nowrap;
    border: 0;
  }
</style>
