<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { socket } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { banner } from '../lib/stores/sos.js';
  import { privacyPause } from '../lib/stores/places.js';
  import { apiPost, apiGet } from '../lib/api.js';
  import { isIgnoringBatteryOptimizations, requestIgnoreBatteryOptimizations } from '../lib/batteryOptimization.js';
  import { isNativePlatform } from '../lib/geoProvider.js';
  import { effects, FX_LEVELS } from '../lib/stores/effects.js';
  import { daypartEnabled, setDaypartEnabled } from '../lib/daypart.js';
  import { voiceEnabled, voiceCheckins, voiceMutedUntil, setVoiceEnabledFromGesture, muteForToday, unmute, isSupported as voiceSupported } from '../lib/voice.js';
  import { haptics } from '../lib/haptics.js';
  import SettingsSection from './settings/SettingsSection.svelte';
  import RetentionPillGroup from './settings/RetentionPillGroup.svelte';
  import ToggleControl from './primitives/ToggleControl.svelte';

  /** @type {{ embedded?: boolean }} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  // Visual-effects segmented control — labels keyed by FX_LEVELS ('full'|'calm'|'minimal').
  const FX_META = {
    full:    { name: 'Full',    desc: 'All animations & effects' },
    calm:    { name: 'Calm',    desc: 'Reduced motion & blur' },
    minimal: { name: 'Minimal', desc: 'No ambient motion, solid panels' },
  };

  // ── Profile ────────────────────────────────────────────────────────────────
  let firstName = $state('');
  let lastName  = $state('');
  let email     = $state('');
  let mobile    = $state('');

  // ── Password ───────────────────────────────────────────────────────────────
  let currentPassword  = $state('');
  let newPassword      = $state('');
  let confirmPassword  = $state('');
  let saving           = $state(false);
  let changingPw       = $state(false);

  // ── Delete ─────────────────────────────────────────────────────────────────
  let deletePassword = $state('');
  let showDelete     = $state(false);
  let deleting       = $state(false);
  let deleteCountdown = $state(0);
  let deleteTimer    = $state(null);

  // ── Retention (offline visibility) ─────────────────────────────────────────
  let retentionMode = $state('default');

  // ── Privacy pause ──────────────────────────────────────────────────────────
  let privacyUntil  = null;
  let privacyActive = $state(false);
  let privacyTimeLeft = $state('');
  let _privacyTimer = null;

  // ── Quiet Hours ────────────────────────────────────────────────────────────
  let quietHoursEnabled = $state(false);
  let quietHoursStart   = $state('22:00');
  let quietHoursEnd     = $state('07:00');

  // ── Panic Mode ─────────────────────────────────────────────────────────────
  let panicMode = $state(localStorage.getItem('kinnect_panic_mode') === 'true');

  // ── Notifications (push) ───────────────────────────────────────────────────
  let pushSupported       = $state(false);
  let pushEnabled         = $state(false);
  let togglingPush        = $state(false);
  let _pendingVapidResolve = null;

  // ── Background location (Android native) ───────────────────────────────────
  const isNative       = isNativePlatform();
  let batteryOptIgnoring = $state(true);
  let checkingBattery    = $state(false);

  // ── Helpers ────────────────────────────────────────────────────────────────
  function formatTimeLeft(ms) {
    if (ms <= 0) return '';
    const h = Math.floor(ms / 3600000);
    const m = Math.floor((ms % 3600000) / 60000);
    return h > 0 ? `${h}h ${m}m` : `${m}m`;
  }

  function _updatePrivacy() {
    const now = Date.now();
    privacyActive   = !!privacyUntil && privacyUntil > now;
    privacyTimeLeft = privacyActive ? formatTimeLeft(privacyUntil - now) : '';
  }

  function flashBanner(type, text, ms = 2500) {
    banner.set({ type, text, actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), ms);
  }

  // ── Socket listeners ───────────────────────────────────────────────────────
  function onVapidKey(payload) {
    if (_pendingVapidResolve) {
      _pendingVapidResolve(payload);
      _pendingVapidResolve = null;
    }
  }

  function onPushSubscribeAck(payload) {
    togglingPush = false;
    if (payload && !payload.ok) {
      pushEnabled = false;
      flashBanner('sos', payload.error || 'Server rejected push subscription', 3000);
    }
  }

  function onPushUnsubscribeAck(payload) {
    togglingPush = false;
    if (payload && !payload.ok) {
      pushEnabled = true;
      flashBanner('sos', 'Server could not remove subscription', 3000);
    }
  }

  function onPrivacyAck(payload) {
    if (payload && payload.ok) {
      privacyUntil = payload.pausedUntil ?? null;
      privacyPause.set(privacyUntil);
      _updatePrivacy();
    }
  }

  onMount(async () => {
    // Register ALL socket listeners FIRST — before any awaits — so no ack is ever missed.
    socket.on('vapidKey',            onVapidKey);
    socket.on('pushSubscribeAck',    onPushSubscribeAck);
    socket.on('pushUnsubscribeAck',  onPushUnsubscribeAck);
    socket.on('privacyPauseAck',     onPrivacyAck);

    // Seed privacy from the shared store.
    const stored = get(privacyPause);
    if (stored && stored > Date.now()) privacyUntil = stored;
    _updatePrivacy();
    _privacyTimer = setInterval(_updatePrivacy, 10000);

    const res = await apiGet('/api/me');
    if (res.ok) {
      email  = res.email  || '';
      mobile = res.mobile || '';
    }

    pushSupported = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
    if (pushSupported) {
      try {
        const reg = await navigator.serviceWorker.getRegistration();
        if (reg) {
          const sub = await reg.pushManager.getSubscription();
          pushEnabled = !!sub;
        }
      } catch (_) { pushEnabled = false; }
    }

    if (isNative) batteryOptIgnoring = await isIgnoringBatteryOptimizations();
  });

  onDestroy(() => {
    clearInterval(_privacyTimer);
    if (deleteTimer) clearInterval(deleteTimer);
    socket.off('vapidKey',           onVapidKey);
    socket.off('pushSubscribeAck',   onPushSubscribeAck);
    socket.off('pushUnsubscribeAck', onPushUnsubscribeAck);
    socket.off('privacyPauseAck',    onPrivacyAck);
    _pendingVapidResolve = null;
  });

  // ── VAPID ──────────────────────────────────────────────────────────────────
  function fetchVapidKey() {
    return new Promise((resolve) => {
      _pendingVapidResolve = resolve;
      socket.emit('getVapidKey', {});
      setTimeout(() => {
        if (_pendingVapidResolve === resolve) { _pendingVapidResolve = null; resolve({ ok: false }); }
      }, 5000);
    });
  }

  function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64  = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const raw = window.atob(base64);
    const arr = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
    return arr;
  }

  async function enablePush() {
    if (togglingPush) return;
    togglingPush = true;
    try {
      const perm = await Notification.requestPermission();
      if (perm !== 'granted') {
        togglingPush = false;
        flashBanner('sos', 'Notification permission denied', 3000);
        return;
      }
      const keyPayload = await fetchVapidKey();
      if (!keyPayload.ok || !keyPayload.key) {
        togglingPush = false;
        flashBanner('sos', 'Push notifications not configured on server', 3000);
        return;
      }
      const reg = await Promise.race([
        navigator.serviceWorker.ready,
        new Promise((_, rej) => setTimeout(() => rej(new Error('sw-timeout')), 6000)),
      ]);
      const sub  = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: urlBase64ToUint8Array(keyPayload.key) });
      const json = sub.toJSON();
      pushEnabled  = true;
      togglingPush = false;
      socket.emit('pushSubscribe', { endpoint: json.endpoint, keys: json.keys });
      flashBanner('info', "Notifications on. We'll only bother you when it matters.", 2500);
    } catch (err) {
      togglingPush = false;
      const msg = err.message === 'sw-timeout' ? 'Service worker not ready'
        : (err.message?.includes('denied') || err.message?.includes('permission'))
          ? 'Notification permission denied — enable in browser settings'
          : (err.message || String(err));
      flashBanner('sos', 'Could not enable notifications: ' + msg, 3500);
    }
  }

  async function disablePush() {
    if (togglingPush) return;
    togglingPush = true;
    try {
      const reg = await Promise.race([
        navigator.serviceWorker.ready,
        new Promise((_, rej) => setTimeout(() => rej(new Error('sw-timeout')), 6000)),
      ]);
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        const endpoint = sub.endpoint;
        await sub.unsubscribe();
        pushEnabled  = false;
        togglingPush = false;
        socket.emit('pushUnsubscribe', { endpoint });
        flashBanner('info', 'Notifications off. Radio silence.', 2500);
      } else {
        pushEnabled  = false;
        togglingPush = false;
      }
    } catch (err) {
      togglingPush = false;
      flashBanner('sos', 'Could not disable notifications: ' + (err.message || err), 3000);
    }
  }

  async function saveProfile() {
    saving = true;
    const res = await apiPost('/api/profile/update', { firstName, lastName, email, mobile });
    saving = false;
    if (res.ok) flashBanner('info', 'Looking good. Profile saved.', 2000);
    else         flashBanner('sos', res.error || 'Failed to update', 3000);
  }

  async function changePassword() {
    if (newPassword !== confirmPassword) {
      flashBanner('sos', 'Passwords do not match', 2000);
      return;
    }
    changingPw = true;
    const res = await apiPost('/api/profile/password', { currentPassword, newPassword });
    changingPw = false;
    if (res.ok) {
      currentPassword = ''; newPassword = ''; confirmPassword = '';
      flashBanner('info', "New password locked in. Don't lose this one.", 2000);
    } else {
      flashBanner('sos', res.error || 'Failed', 3000);
    }
  }

  async function deleteAccount() {
    deleting = true;
    const res = await apiPost('/api/profile/delete', { password: deletePassword });
    deleting = false;
    if (res.ok) { authUser.set(null); window.location.hash = '#/login'; }
    else         flashBanner('sos', res.error || 'Failed', 3000);
  }

  async function logout() {
    await apiPost('/api/logout');
    window.location.hash = '#/login';
    window.location.reload();
  }

  const _retentionLabels = { default: '24 hours', '48h': '2 days', '5d': '5 days', '10d': '10 days', '30d': '30 days' };
  function setRetentionMode(mode) {
    retentionMode = mode;
    socket.emit('setRetention', { mode });
    flashBanner('info', `Location kept for ${_retentionLabels[mode] || mode} after going offline`, 2500);
  }

  function startDeleteFlow() {
    showDelete = true;
    deleteCountdown = 3;
    if (deleteTimer) clearInterval(deleteTimer);
    deleteTimer = setInterval(() => { deleteCountdown--; if (deleteCountdown <= 0) clearInterval(deleteTimer); }, 1000);
  }

  function setPrivacyMode(duration) {
    if (duration === 'resume') {
      privacyUntil = null;
    } else {
      const hours = parseInt(duration, 10) || 1;
      privacyUntil = Date.now() + hours * 3600000;
    }
    privacyPause.set(privacyUntil);
    _updatePrivacy();
    socket.emit('setPrivacyPause', { duration });
  }

  function saveQuietHours() {
    socket.emit('updateQuietHours', { enabled: quietHoursEnabled, startTime: quietHoursStart, endTime: quietHoursEnd });
    flashBanner('info', quietHoursEnabled
      ? `Quiet hours set — contacts see a blurred location ${quietHoursStart}–${quietHoursEnd}`
      : 'Quiet hours disabled', 3000);
  }

  async function handleBatteryOptimization() {
    checkingBattery = true;
    await requestIgnoreBatteryOptimizations();
    await new Promise(r => setTimeout(r, 800));
    batteryOptIgnoring = await isIgnoringBatteryOptimizations();
    checkingBattery = false;
  }

  function cancelDelete() {
    showDelete = false;
    deletePassword = '';
    deleteCountdown = 0;
    if (deleteTimer) clearInterval(deleteTimer);
  }
</script>

{#if embedded}
  <div class="settings-panel">

    <!-- Profile -->
    <SettingsSection title="Profile">
      {#snippet children()}
        <label class="field-label">
          First Name
          <input type="text" bind:value={firstName} class="field-input" maxlength="50" />
        </label>
        <label class="field-label">
          Last Name
          <input type="text" bind:value={lastName} class="field-input" maxlength="50" />
        </label>
        <label class="field-label">
          Email
          <input type="email" bind:value={email} class="field-input" />
        </label>
        <label class="field-label">
          Mobile
          <input type="tel" bind:value={mobile} class="field-input" />
        </label>
        <button class="btn btn-primary btn-sm tactile" onclick={saveProfile} disabled={saving}>
          {saving ? 'Saving…' : 'Save Profile'}
        </button>
      {/snippet}
    </SettingsSection>

    <!-- Visual effects — flagship FX control -->
    <SettingsSection title="Visual effects" description="Dial down animation and blur for a calmer screen and better battery life.">
      {#snippet children()}
        <div class="fx-segmented" role="radiogroup" aria-label="Visual effects level">
          {#each FX_LEVELS as level}
            <button
              type="button"
              class="fx-seg-btn tactile"
              class:active={$effects === level}
              role="radio"
              aria-checked={$effects === level}
              onclick={() => effects.set(level)}
            >
              <span class="fx-seg-name">{FX_META[level].name}</span>
              <span class="fx-seg-desc">{FX_META[level].desc}</span>
            </button>
          {/each}
        </div>
        <ToggleControl
          checked={$daypartEnabled}
          label="Time-of-day tint"
          description="Nav and ambient colors gently warm at dawn and dusk."
          onchange={(v) => setDaypartEnabled(v)}
        />
      {/snippet}
    </SettingsSection>

    <!-- Privacy Mode -->
    <SettingsSection title="Privacy Mode" description="Temporarily hide your location from everyone. Guardians can still see you.">
      {#snippet children()}
        {#if privacyActive}
          <div class="privacy-active">
            <span class="ghost-emoji" aria-hidden="true">👻</span>
            <div class="ghost-info">
              <p class="ghost-status">You're hidden</p>
              <p class="ghost-time">{privacyTimeLeft} left</p>
            </div>
            <button class="btn btn-secondary btn-sm tactile" onclick={() => setPrivacyMode('resume')}>Show My Location</button>
          </div>
        {:else}
          <div class="privacy-btns">
            <button class="btn btn-secondary btn-sm tactile" onclick={() => setPrivacyMode('1h')}>Hide for 1 hour</button>
            <button class="btn btn-secondary btn-sm tactile" onclick={() => setPrivacyMode('4h')}>Hide for 4 hours</button>
            <button class="btn btn-secondary btn-sm tactile" onclick={() => setPrivacyMode('8h')}>Hide for 8 hours</button>
          </div>
        {/if}
      {/snippet}
    </SettingsSection>

    <!-- Quiet Hours — safety toggle via ToggleControl -->
    <SettingsSection title="Quiet Hours" description="During Quiet Hours your location is approximate for everyone except your guardians.">
      {#snippet children()}
        <ToggleControl
          bind:checked={quietHoursEnabled}
          label="Enable Quiet Hours"
          description="Blurs your position for contacts during the set window"
        />
        {#if quietHoursEnabled}
          <div class="quiet-time-row">
            <label class="field-label-inline">
              From
              <input type="time" bind:value={quietHoursStart} class="field-input field-time" />
            </label>
            <label class="field-label-inline">
              To
              <input type="time" bind:value={quietHoursEnd} class="field-input field-time" />
            </label>
          </div>
        {/if}
        <button class="btn btn-primary btn-sm tactile" onclick={saveQuietHours}>Save Quiet Hours</button>
      {/snippet}
    </SettingsSection>

    <!-- Panic Mode — safety toggle via ToggleControl -->
    <SettingsSection title="Panic Mode" description="Double-tap the SOS button to send an alert instantly, skipping the confirmation step. Only enable if you might need to trigger SOS one-handed in an emergency.">
      {#snippet children()}
        <ToggleControl
          bind:checked={panicMode}
          label="Enable Panic Mode"
          description="Skips confirmation — SOS fires on double-tap"
          onchange={(v) => localStorage.setItem('kinnect_panic_mode', String(v))}
        />
      {/snippet}
    </SettingsSection>

    <!-- Offline Visibility — RetentionPillGroup -->
    <SettingsSection title="Offline Visibility" description="How long your last location stays visible after you close the app.">
      {#snippet children()}
        <RetentionPillGroup
          bind:value={retentionMode}
          onchange={setRetentionMode}
        />
      {/snippet}
    </SettingsSection>

    <!-- Notifications — safety toggle via ToggleControl -->
    <SettingsSection title="Notifications" description={pushSupported ? "SOS alerts, check-in reminders, and family updates. We only notify you when it matters." : "Notifications aren't available in this browser."}>
      {#snippet children()}
        {#if pushSupported}
          <ToggleControl
            checked={pushEnabled}
            label={pushEnabled ? 'Notifications are on' : 'Turn on notifications'}
            disabled={togglingPush}
            onchange={(v) => v ? enablePush() : disablePush()}
          />
          {#if togglingPush}<p class="hint">Updating…</p>{/if}
        {/if}
      {/snippet}
    </SettingsSection>

    {#if voiceSupported()}
      <SettingsSection title="Spoken updates" description="Kinnect can say important family updates out loud — SOS alerts, arrivals, and status changes. Handy while driving.">
        {#snippet children()}
          <ToggleControl
            checked={$voiceEnabled}
            label={$voiceEnabled ? 'Spoken updates are on' : 'Speak updates aloud'}
            description="If you use a screen reader, leave this off — these updates are already announced."
            onchange={(v) => { setVoiceEnabledFromGesture(v); if (v) haptics.confirm?.(); }}
          />
          {#if $voiceEnabled}
            <ToggleControl
              checked={$voiceCheckins}
              label="Also speak check-ins"
              description={'Small updates like “Priya checked in”'}
              onchange={(v) => voiceCheckins.set(v)}
            />
            {#if $voiceMutedUntil > Date.now()}
              <p class="hint">Muted until midnight — SOS will still be spoken.</p>
              <button class="btn btn-secondary btn-sm tactile" onclick={unmute}>Unmute</button>
            {:else}
              <button class="btn btn-secondary btn-sm tactile" onclick={muteForToday}>Mute for today</button>
            {/if}
          {/if}
        {/snippet}
      </SettingsSection>
    {/if}

    <!-- Background Location (Android native only) -->
    {#if isNative}
      <SettingsSection title="Background Location" description={batteryOptIgnoring ? "Kinnect can share your location even when the app is in the background." : "Your phone might stop Kinnect from working in the background. Tap below to fix this."}>
        {#snippet children()}
          {#if batteryOptIgnoring}
            <div class="battery-status battery-status--ok">
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
              Background location is allowed
            </div>
          {:else}
            <div class="battery-status battery-status--warn">
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              Background location may be restricted
            </div>
            <button class="btn btn-primary btn-sm tactile" onclick={handleBatteryOptimization} disabled={checkingBattery}>
              {checkingBattery ? 'Checking…' : 'Fix Background Access'}
            </button>
          {/if}
        {/snippet}
      </SettingsSection>
    {/if}

    <!-- Change Password -->
    <SettingsSection title="Change Password">
      {#snippet children()}
        <label class="field-label">
          Current Password
          <input type="password" bind:value={currentPassword} class="field-input" />
        </label>
        <label class="field-label">
          New Password
          <input type="password" bind:value={newPassword} class="field-input" />
        </label>
        <label class="field-label">
          Confirm New Password
          <input type="password" bind:value={confirmPassword} class="field-input" />
        </label>
        <button class="btn btn-primary btn-sm tactile" onclick={changePassword} disabled={changingPw || !currentPassword || !newPassword}>
          {changingPw ? 'Changing…' : 'Change Password'}
        </button>
      {/snippet}
    </SettingsSection>

    <!-- Help -->
    <SettingsSection title="Help">
      {#snippet children()}
        <button class="btn btn-secondary btn-sm tactile" onclick={() => dispatch('openGuide')}>How Kinnect Works</button>
      {/snippet}
    </SettingsSection>

    <!-- Account -->
    <SettingsSection title="Account">
      {#snippet children()}
        <button class="btn btn-secondary btn-sm tactile logout-btn" onclick={logout}>Sign Out</button>
      {/snippet}
    </SettingsSection>

    <!-- Delete Account -->
    <SettingsSection title="Delete Account" danger>
      {#snippet children()}
        {#if showDelete}
          <p class="hint danger-text">This will permanently delete your account, all your data, and remove you from every group. This cannot be undone.</p>
          <label class="field-label">
            Enter your password to confirm deletion
            <input type="password" bind:value={deletePassword} class="field-input" />
          </label>
          <div class="delete-actions">
            <button
              class="btn btn-sm tactile delete-countdown-btn"
              class:counting3={deleteCountdown === 3}
              class:counting2={deleteCountdown === 2}
              class:counting1={deleteCountdown === 1}
              onclick={deleteAccount}
              disabled={deleting || !deletePassword || deleteCountdown > 0}
            >{deleting ? 'Deleting…' : deleteCountdown > 0 ? `Wait ${deleteCountdown}…` : 'Permanently Delete'}</button>
            <button class="btn btn-secondary btn-sm tactile" onclick={cancelDelete}>Cancel</button>
          </div>
        {:else}
          <button class="btn btn-danger-outline btn-sm tactile" onclick={startDeleteFlow}>Permanently Delete</button>
        {/if}
      {/snippet}
    </SettingsSection>

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Settings</h3>
      <button class="panel-close" onclick={() => dispatch('close')} aria-label="Close">&times;</button>
    </div>
    <div class="panel-body">
      <p>Use the sidebar Settings tab.</p>
    </div>
  </div>
{/if}

<style>
  .settings-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-3);
  }

  /* ── Form fields ──────────────────────────────────────────────────────────── */
  .field-label {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
  }

  .field-input {
    padding: var(--space-2) var(--space-2-5);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    background: var(--surface-3);
    color: var(--text-primary);
    min-height: 44px;
  }

  .field-input:focus {
    outline: none;
    border-color: var(--primary-400);
    box-shadow: 0 0 0 3px var(--primary-500-12);
  }

  .field-label-inline {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
  }

  .field-time { width: 110px; }

  .quiet-time-row {
    display: flex;
    gap: var(--space-3);
    align-items: center;
    margin: var(--space-2) 0;
  }

  /* ── Misc text ────────────────────────────────────────────────────────────── */
  .hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
    line-height: 1.4;
  }

  .danger-text { color: var(--danger-500); }

  /* ── Buttons ──────────────────────────────────────────────────────────────── */
  .btn-sm {
    padding: var(--space-1-5) var(--space-3-5, 14px);
    font-size: var(--text-xs);
    min-height: 44px;
  }

  .btn-danger-outline {
    background: transparent;
    color: var(--danger-500);
    border: 1px solid var(--danger-500);
    cursor: pointer;
    border-radius: var(--radius-md);
  }
  .btn-danger-outline:hover { background: var(--danger-500-10, color-mix(in srgb, var(--danger-500) 10%, transparent)); }

  .logout-btn { width: 100%; }

  /* ── Delete countdown ─────────────────────────────────────────────────────── */
  .delete-actions {
    display: flex;
    gap: var(--space-2);
  }

  /* --countdown-color swapped gray→amber→red; transition is functional feedback, survives reduced-motion */
  .delete-countdown-btn {
    --countdown-color: var(--danger-500);
    background: var(--countdown-color);
    color: white; /* raw-color-ok */
    border: none;
    transition: background var(--duration-countdown, 800ms), box-shadow var(--duration-countdown, 800ms);
  }
  .delete-countdown-btn.counting3 { --countdown-color: var(--gray-500, #6b7280); box-shadow: none; }
  .delete-countdown-btn.counting2 { --countdown-color: var(--warning-500); box-shadow: none; }
  .delete-countdown-btn.counting1 { --countdown-color: var(--danger-500); box-shadow: var(--glow-sos, none); }

  /* ── Visual effects segmented control ────────────────────────────────────── */
  .fx-segmented {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-1);
    padding: var(--space-1);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
  }

  .fx-seg-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-0-5, 2px);
    min-height: 56px;
    padding: var(--space-2);
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    text-align: center;
    -webkit-tap-highlight-color: transparent;
    transition:
      background var(--duration-fast) var(--ease-out),
      color      var(--duration-fast) var(--ease-out),
      border-color var(--duration-fast) var(--ease-out);
  }
  .fx-seg-btn:hover { background: var(--surface-2); color: var(--text-primary); }
  .fx-seg-btn.active {
    background: var(--primary-500-12);
    border-color: var(--primary-500-20);
    color: var(--text-primary);
  }
  .fx-seg-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .fx-seg-name {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    line-height: 1.1;
  }
  .fx-seg-btn.active .fx-seg-name { color: var(--primary-400); }

  .fx-seg-desc {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: 1.2;
  }

  /* ── Privacy active card ──────────────────────────────────────────────────── */
  .privacy-active {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3);
    background: var(--warning-500-08, color-mix(in srgb, var(--warning-500, #f59e0b) 8%, transparent));
    border: 1px solid var(--warning-500-20, color-mix(in srgb, var(--warning-500, #f59e0b) 20%, transparent));
    border-radius: var(--radius-md);
    flex-wrap: wrap;
  }
  .ghost-emoji {
    font-size: var(--text-2xl, 24px);
    display: inline-block;
    flex-shrink: 0;
  }
  .ghost-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5, 2px);
  }
  .ghost-status {
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--warning-700, #b45309);
  }
  .ghost-time {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }
  :global([data-theme='dark']) .ghost-status { color: var(--warning-400, #fbbf24); }

  .privacy-btns {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  /* ── Battery status indicators ───────────────────────────────────────────── */
  .battery-status {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    font-weight: 600;
    margin-bottom: var(--space-2);
  }
  .battery-status--ok {
    background: var(--success-500-12, color-mix(in srgb, var(--success-500, #10b981) 12%, transparent));
    border: 1px solid var(--success-500-28, color-mix(in srgb, var(--success-500, #10b981) 28%, transparent));
    color: var(--success-700, #047857);
  }
  .battery-status--warn {
    background: var(--warning-500-12, color-mix(in srgb, var(--warning-500, #f59e0b) 12%, transparent));
    border: 1px solid var(--warning-500-30, color-mix(in srgb, var(--warning-500, #f59e0b) 30%, transparent));
    color: var(--warning-700, #b45309);
  }
  :global([data-theme='dark']) .battery-status--ok  { color: var(--success-400, #34d399); }
  :global([data-theme='dark']) .battery-status--warn { color: var(--warning-400, #fbbf24); }

  /* ── Mobile layout ───────────────────────────────────────────────────────── */
  @media (max-width: 767px) {
    .settings-panel {
      width: 100%;
      max-width: 100%;
      max-height: 100dvh;
      overflow-y: auto;
    }
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .fx-seg-btn { transition: none; }
    /* countdown color is functional feedback — kept even under reduced-motion (no movement) */
    .delete-countdown-btn { transition: background var(--duration-countdown, 800ms); }
  }
</style>
