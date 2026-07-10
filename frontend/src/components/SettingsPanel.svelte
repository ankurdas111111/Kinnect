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
  import Card from './primitives/Card.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  // Visual-effects segmented control — labels keyed by FX_LEVELS ('full'|'calm'|'minimal').
  const FX_META = {
    full:    { name: 'Full',    desc: 'All animations & effects' },
    calm:    { name: 'Calm',    desc: 'Reduced motion & blur' },
    minimal: { name: 'Minimal', desc: 'Max battery & performance' },
  };

  let firstName = $state('');
  let lastName = $state('');
  let email = $state('');
  let mobile = $state('');
  let currentPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let deletePassword = $state('');
  let saving = $state(false);
  let changingPw = $state(false);
  let showDelete = $state(false);
  let deleting = $state(false);
  let deleteCountdown = $state(0);
  let deleteTimer = $state(null);

  // Plain local state — driven only by clicks and acks, never by stores.
  // This prevents any external store write from overriding what the user just clicked.
  let retentionMode = $state('default');  // 'default' (24h) | '48h'
  let privacyUntil = null;        // ms timestamp or null
  let privacyActive = $state(false);
  let privacyTimeLeft = $state('');
  let _privacyTimer = null;

  // Quiet Hours state
  let quietHoursEnabled = $state(false);

  // Panic Mode state (opt-in: double-tap SOS FAB fires SOS without confirm modal)
  let panicMode = $state(localStorage.getItem('kinnect_panic_mode') === 'true');
  let quietHoursStart = $state('22:00');
  let quietHoursEnd = $state('07:00');

  // Push notifications
  let pushSupported = $state(false);
  let pushEnabled = $state(false);
  let togglingPush = $state(false);
  let _pendingVapidResolve = null;

  // Background tracking / battery optimization (Android native only)
  const isNative = isNativePlatform();
  let batteryOptIgnoring = $state(true);  // true = already unrestricted (no action needed)
  let checkingBattery = $state(false);

  function formatTimeLeft(ms) {
    if (ms <= 0) return '';
    const h = Math.floor(ms / 3600000);
    const m = Math.floor((ms % 3600000) / 60000);
    return h > 0 ? `${h}h ${m}m` : `${m}m`;
  }

  function _updatePrivacy() {
    const now = Date.now();
    privacyActive = !!privacyUntil && privacyUntil > now;
    privacyTimeLeft = privacyActive ? formatTimeLeft(privacyUntil - now) : '';
  }

  // --- Socket event handlers (registered synchronously in onMount before any await) ---

  function onVapidKey(payload) {
    if (_pendingVapidResolve) {
      _pendingVapidResolve(payload);
      _pendingVapidResolve = null;
    }
  }

  function onPushSubscribeAck(payload) {
    togglingPush = false;
    if (payload && !payload.ok) {
      pushEnabled = false; // server rejected — roll back optimistic update
      banner.set({ type: 'sos', text: payload.error || 'Server rejected push subscription', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  function onPushUnsubscribeAck(payload) {
    togglingPush = false;
    if (payload && !payload.ok) {
      pushEnabled = true; // server rejected — roll back
      banner.set({ type: 'sos', text: 'Server could not remove subscription', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  function onPrivacyAck(payload) {
    if (payload && payload.ok) {
      // Server confirmed — update local state and sync shared store.
      privacyUntil = payload.pausedUntil ?? null;
      privacyPause.set(privacyUntil);
      _updatePrivacy();
    }
  }

  onMount(async () => {
    // Register ALL socket listeners FIRST — before any awaits — so no ack is ever missed.
    socket.on('vapidKey', onVapidKey);
    socket.on('pushSubscribeAck', onPushSubscribeAck);
    socket.on('pushUnsubscribeAck', onPushUnsubscribeAck);
    socket.on('privacyPauseAck', onPrivacyAck);

    // Seed privacy from the shared store (populated if another component set it earlier).
    const stored = get(privacyPause);
    if (stored && stored > Date.now()) {
      privacyUntil = stored;
    }
    _updatePrivacy();
    _privacyTimer = setInterval(_updatePrivacy, 10000);

    // Load profile fields — /api/me is the available endpoint.
    const res = await apiGet('/api/me');
    if (res.ok) {
      email = res.email || '';
      mobile = res.mobile || '';
      // firstName/lastName not in /api/me — leave blank (can be typed in)
    }

    // Push support detection — use getRegistration() to avoid hanging on .ready
    // if no SW is registered yet.
    pushSupported = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
    if (pushSupported) {
      try {
        const reg = await navigator.serviceWorker.getRegistration();
        if (reg) {
          const sub = await reg.pushManager.getSubscription();
          pushEnabled = !!sub;
        }
      } catch (_) {
        pushEnabled = false;
      }
    }

    // Check battery optimization status (Android native only)
    if (isNative) {
      batteryOptIgnoring = await isIgnoringBatteryOptimizations();
    }
  });

  async function handleBatteryOptimization() {
    checkingBattery = true;
    await requestIgnoreBatteryOptimizations();
    // Re-check after user returns from settings
    await new Promise(r => setTimeout(r, 800));
    batteryOptIgnoring = await isIgnoringBatteryOptimizations();
    checkingBattery = false;
  }

  onDestroy(() => {
    clearInterval(_privacyTimer);
    if (deleteTimer) clearInterval(deleteTimer);
    socket.off('vapidKey', onVapidKey);
    socket.off('pushSubscribeAck', onPushSubscribeAck);
    socket.off('pushUnsubscribeAck', onPushUnsubscribeAck);
    socket.off('privacyPauseAck', onPrivacyAck);
    _pendingVapidResolve = null;
  });

  // Fetch VAPID key via WS event — resolves via socket.on('vapidKey'), with 5 s timeout.
  function fetchVapidKey() {
    return new Promise((resolve) => {
      _pendingVapidResolve = resolve;
      socket.emit('getVapidKey', {});
      setTimeout(() => {
        if (_pendingVapidResolve === resolve) {
          _pendingVapidResolve = null;
          resolve({ ok: false });
        }
      }, 5000);
    });
  }

  function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
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
        banner.set({ type: 'sos', text: 'Notification permission denied', actions: [] });
        setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
        return;
      }
      const keyPayload = await fetchVapidKey();
      if (!keyPayload.ok || !keyPayload.key) {
        togglingPush = false;
        banner.set({ type: 'sos', text: 'Push notifications not configured on server', actions: [] });
        setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
        return;
      }
      const reg = await Promise.race([
        navigator.serviceWorker.ready,
        new Promise((_, rej) => setTimeout(() => rej(new Error('sw-timeout')), 6000)),
      ]);
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(keyPayload.key),
      });
      const json = sub.toJSON();
      // Optimistic — flip UI immediately, notify server in background.
      pushEnabled = true;
      togglingPush = false;
      socket.emit('pushSubscribe', { endpoint: json.endpoint, keys: json.keys });
      banner.set({ type: 'info', text: "Notifications on. We'll only bother you when it matters.", actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
    } catch (err) {
      togglingPush = false;
      const msg = err.message === 'sw-timeout' ? 'Service worker not ready'
        : (err.message?.includes('denied') || err.message?.includes('permission'))
          ? 'Notification permission denied — enable in browser settings'
          : (err.message || String(err));
      banner.set({ type: 'sos', text: 'Could not enable notifications: ' + msg, actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3500);
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
        pushEnabled = false;
        togglingPush = false;
        socket.emit('pushUnsubscribe', { endpoint });
        banner.set({ type: 'info', text: 'Notifications off. Radio silence.', actions: [] });
        setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
      } else {
        pushEnabled = false;
        togglingPush = false;
      }
    } catch (err) {
      togglingPush = false;
      banner.set({ type: 'sos', text: 'Could not disable notifications: ' + (err.message || err), actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function saveProfile() {
    saving = true;
    const res = await apiPost('/api/profile/update', { firstName, lastName, email, mobile });
    saving = false;
    if (res.ok) {
      banner.set({ type: 'info', text: 'Looking good. Profile saved.', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
    } else {
      banner.set({ type: 'sos', text: res.error || 'Failed to update', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function changePassword() {
    if (newPassword !== confirmPassword) {
      banner.set({ type: 'sos', text: 'Passwords do not match', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
      return;
    }
    changingPw = true;
    const res = await apiPost('/api/profile/password', { currentPassword, newPassword });
    changingPw = false;
    if (res.ok) {
      currentPassword = '';
      newPassword = '';
      confirmPassword = '';
      banner.set({ type: 'info', text: "New password locked in. Don't lose this one.", actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
    } else {
      banner.set({ type: 'sos', text: res.error || 'Failed', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function deleteAccount() {
    deleting = true;
    const res = await apiPost('/api/profile/delete', { password: deletePassword });
    deleting = false;
    if (res.ok) {
      authUser.set(null);
      window.location.hash = '#/login';
    } else {
      banner.set({ type: 'sos', text: res.error || 'Failed', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function logout() {
    await apiPost('/api/logout');
    window.location.hash = '#/login';
    window.location.reload();
  }

  const _retentionLabels = { default: '24 hours', '48h': '2 days', '5d': '5 days', '10d': '10 days', '30d': '30 days' };
  function setRetentionMode(mode) {
    retentionMode = mode; // immediate optimistic update — no callback, no deadlock
    socket.emit('setRetention', { mode });
    const label = _retentionLabels[mode] || mode;
    banner.set({ type: 'info', text: `Location kept for ${label} after going offline`, actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  function startDeleteFlow() {
    showDelete = true;
    deleteCountdown = 3;
    if (deleteTimer) clearInterval(deleteTimer);
    deleteTimer = setInterval(() => {
      deleteCountdown--;
      if (deleteCountdown <= 0) clearInterval(deleteTimer);
    }, 1000);
  }

  function setPrivacyMode(duration) {
    // Optimistic — update local state immediately, backend confirms via privacyPauseAck.
    if (duration === 'resume') {
      privacyUntil = null;
    } else {
      const hours = parseInt(duration, 10) || 1;
      privacyUntil = Date.now() + hours * 3600000;
    }
    privacyPause.set(privacyUntil); // keep shared store in sync for other components
    _updatePrivacy();
    socket.emit('setPrivacyPause', { duration });
  }

  function saveQuietHours() {
    socket.emit('updateQuietHours', {
      enabled: quietHoursEnabled,
      startTime: quietHoursStart,
      endTime: quietHoursEnd,
    });
    banner.set({ type: 'info', text: quietHoursEnabled
      ? `Quiet hours set — contacts see a blurred location ${quietHoursStart}–${quietHoursEnd}`
      : 'Quiet hours disabled', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
  }
</script>

{#if embedded}
  <div class="panel-body settings-panel">
    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Profile</h4>
      <div class="form-section">
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
          {saving ? 'Saving...' : 'Save Profile'}
        </button>
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Visual effects</h4>
      <div class="form-section">
        <p class="hint">Dial down animation and blur for a calmer screen and better battery life.</p>
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
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Privacy Mode</h4>
      <div class="form-section">
        <p class="hint">Temporarily hide your location from everyone. Guardians can still see you.</p>
        {#if privacyActive}
          <div class="privacy-active animate-slide-up">
            <span class="ghost-emoji animate-ghost-float" aria-hidden="true">👻</span>
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
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Quiet Hours</h4>
      <div class="form-section">
        <p class="hint">During Quiet Hours your location is approximate for everyone except your guardians.</p>
        <label class="toggle-row">
          <span>Enable Quiet Hours</span>
          <button
            class="toggle-btn"
            class:on={quietHoursEnabled}
            onclick={() => { quietHoursEnabled = !quietHoursEnabled; }}
            aria-label={quietHoursEnabled ? 'Disable quiet hours' : 'Enable quiet hours'}
          >
            <span class="toggle-knob"></span>
          </button>
        </label>
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
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Panic Mode</h4>
      <div class="form-section">
        <p class="hint">Double-tap the SOS button to send an alert instantly, skipping the confirmation step. Only enable if you might need to trigger SOS one-handed in an emergency.</p>
        <label class="toggle-row">
          <span>Enable Panic Mode</span>
          <button
            class="toggle-btn"
            class:on={panicMode}
            onclick={() => { panicMode = !panicMode; localStorage.setItem('kinnect_panic_mode', String(panicMode)); }}
            aria-label={panicMode ? 'Disable panic mode' : 'Enable panic mode'}
          >
            <span class="toggle-knob"></span>
          </button>
        </label>
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Offline Visibility</h4>
      <div class="form-section">
        <p class="hint">How long your last location stays visible after you close the app.</p>
        <div class="retention-pills">
          {#each [['default','1 Day'],['48h','2 Days'],['5d','5 Days'],['10d','10 Days'],['30d','30 Days']] as [mode, label]}
            <button
              class="retention-pill"
              class:active={retentionMode === mode}
              onclick={() => setRetentionMode(mode)}
              aria-pressed={retentionMode === mode}
            >{label}</button>
          {/each}
        </div>
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Notifications</h4>
      <div class="form-section">
        {#if pushSupported}
          <p class="hint">SOS alerts, check-in reminders, and family updates. We only notify you when it matters.</p>
          <label class="toggle-row">
            <span>{pushEnabled ? 'Notifications are on' : 'Turn on notifications'}</span>
            <button
              class="toggle-btn"
              class:on={pushEnabled}
              disabled={togglingPush}
              onclick={() => pushEnabled ? disablePush() : enablePush()}
              aria-label={pushEnabled ? 'Disable push notifications' : 'Enable push notifications'}
            >
              <span class="toggle-knob"></span>
            </button>
          </label>
          {#if togglingPush}<p class="hint">Updating...</p>{/if}
        {:else}
          <p class="hint">Notifications aren't available in this browser.</p>
        {/if}
      </div>
    </Card>

    {#if isNative}
      <Card variant="glass" hover={false} padding="md">
        <h4 class="section-title-bold">Background Location</h4>
        <div class="form-section">
          {#if batteryOptIgnoring}
            <div class="battery-status battery-status--ok">
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
              Background location is allowed
            </div>
            <p class="hint">Kinnect can share your location even when the app is in the background.</p>
          {:else}
            <div class="battery-status battery-status--warn">
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              Background location may be restricted
            </div>
            <p class="hint">Your phone might stop Kinnect from working in the background. Tap below to fix this.</p>
            <button class="btn btn-primary btn-sm tactile" onclick={handleBatteryOptimization} disabled={checkingBattery}>
              {checkingBattery ? 'Checking...' : 'Fix Background Access'}
            </button>
          {/if}
        </div>
      </Card>
    {/if}

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Change Password</h4>
      <div class="form-section">
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
          {changingPw ? 'Changing...' : 'Change Password'}
        </button>
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Help</h4>
      <div class="form-section">
        <button class="btn btn-secondary btn-sm tactile" onclick={() => dispatch('openGuide')}>How Kinnect Works</button>
      </div>
    </Card>

    <Card variant="glass" hover={false} padding="md">
      <h4 class="section-title-bold">Account</h4>
      <div class="form-section">
        <button class="btn btn-secondary btn-sm tactile logout-btn" onclick={logout}>Sign Out</button>
      </div>
    </Card>

    <Card variant="glass" hover={false} glow="danger" padding="md">
      <h4 class="section-title-bold danger-title">Delete Account</h4>
      <div class="form-section">
        {#if showDelete}
          <p class="hint danger-text">This will permanently delete your account, all your data, and remove you from every group. This cannot be undone.</p>
          <label class="field-label">
            Enter your password to confirm deletion
            <input type="password" bind:value={deletePassword} class="field-input" />
          </label>
          <div class="delete-actions">
            <button
              class="btn btn-sm tactile delete-countdown-btn"
              class:counting={deleteCountdown > 0}
              class:counting3={deleteCountdown === 3}
              class:counting2={deleteCountdown === 2}
              class:counting1={deleteCountdown === 1}
              onclick={deleteAccount}
              disabled={deleting || !deletePassword || deleteCountdown > 0}
            >
              {deleting ? 'Deleting...' : deleteCountdown > 0 ? `Wait ${deleteCountdown}…` : 'Permanently Delete'}
            </button>
            <button class="btn btn-secondary btn-sm tactile" onclick={() => { showDelete = false; deletePassword = ''; deleteCountdown = 0; if (deleteTimer) clearInterval(deleteTimer); }}>Cancel</button>
          </div>
        {:else}
          <button class="btn btn-danger-outline btn-sm tactile" onclick={startDeleteFlow}>Permanently Delete</button>
        {/if}
      </div>
    </Card>
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
  /* Stack of glass Cards, one per settings group */
  .settings-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-3);
  }

  /* Section title lives inside a Card now — no extra horizontal padding */
  h4.section-title-bold {
    margin: 0 0 var(--space-3);
    padding: 0;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2-5);
    padding: 0;
  }

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

  .hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
    line-height: 1.4;
  }

  .danger-text { color: var(--danger-500); }

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
  .btn-danger-outline:hover { background: var(--danger-500-10, rgba(220, 38, 38, 0.08)); }

  .delete-actions {
    display: flex;
    gap: var(--space-2);
  }

  /* ── Visual effects — 3-option segmented control ──────────────────── */
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
      color var(--duration-fast) var(--ease-out),
      border-color var(--duration-fast) var(--ease-out);
  }
  .fx-seg-btn:hover {
    background: var(--surface-2);
    color: var(--text-primary);
  }
  .fx-seg-btn.active {
    background: var(--primary-500-12);
    border-color: var(--primary-500-20);
    color: var(--text-primary);
  }
  .fx-seg-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

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

  /* MERIDIAN: Retention pills with glow on selection */
  .retention-pills {
    display: flex;
    gap: var(--space-1-5);
    flex-wrap: wrap;
  }

  .retention-pill {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    min-height: 44px;
    transition:
      background var(--duration-fast) var(--ease-out),
      color var(--duration-fast) var(--ease-out),
      transform 200ms var(--ease-spring);
  }

  .retention-pill:hover {
    background: var(--surface-2);
    color: var(--text-primary);
  }

  .retention-pill.active {
    background: var(--primary-600);
    color: white;
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary), var(--shadow-xs);
    transform: scale(1.04);
  }

  .danger-title {
    color: var(--danger-600) !important;
  }

  .delete-countdown-btn {
    background: var(--danger-500);
    color: white;
    border: none;
    transition: background 800ms, box-shadow 800ms;
  }

  /* 3-color progression: gray → amber → red as countdown ticks 3 → 2 → 1 */
  .delete-countdown-btn.counting3 {
    background: var(--gray-500);
    box-shadow: none;
  }
  .delete-countdown-btn.counting2 {
    background: var(--warning-500);
    box-shadow: none;
  }
  .delete-countdown-btn.counting1 {
    background: var(--danger-500);
    box-shadow: var(--glow-sos, 0 0 16px rgba(239, 68, 68, 0.40));
  }

  .privacy-btns {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 44px;
    font-size: var(--text-sm);
    color: var(--text-primary);
    cursor: pointer;
  }

  .toggle-btn {
    position: relative;
    width: 44px;
    height: 26px;
    border-radius: var(--radius-full);
    background: var(--border-strong);
    border: none;
    cursor: pointer;
    transition: background var(--duration-normal) var(--ease-out);
    padding: 0;
    flex-shrink: 0;
  }
  .toggle-btn.on { background: var(--primary-500); }

  .toggle-knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 20px;
    height: 20px;
    border-radius: var(--radius-full);
    background: white;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
    /* transform-only, spring easing */
    transition: transform 220ms var(--ease-spring);
    will-change: transform;
  }
  .toggle-btn.on .toggle-knob { transform: translateX(18px); }

  /* ── Ghost mode card ─────────────────────────────────────────────── */
  .privacy-active {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3);
    background: var(--warning-500-08, rgba(245, 158, 11, 0.08));
    border: 1px solid var(--warning-500-20, rgba(245, 158, 11, 0.20));
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

  .quiet-time-row {
    display: flex;
    gap: var(--space-3);
    align-items: center;
    margin: var(--space-2) 0;
  }
  .field-label-inline {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
  }
  .field-time {
    width: 110px;
  }

  .logout-btn {
    width: 100%;
  }

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
    background: var(--success-500-12, rgba(16, 185, 129, 0.12));
    border: 1px solid var(--success-500-28, rgba(16, 185, 129, 0.28));
    color: var(--success-700, #047857);
  }
  .battery-status--warn {
    background: var(--warning-500-12, rgba(245, 158, 11, 0.12));
    border: 1px solid var(--warning-500-30, rgba(245, 158, 11, 0.30));
    color: var(--warning-700, #b45309);
  }
  :global([data-theme="dark"]) .battery-status--ok {
    color: var(--success-400, #34d399);
  }
  :global([data-theme="dark"]) .battery-status--warn {
    color: var(--warning-400, #fbbf24);
  }

  @media (max-width: 767px) {
    .settings-panel {
      width: 100%;
      max-width: 100%;
      max-height: 100dvh;
      overflow-y: auto;
    }
  }

  /* ── Reduced motion ─────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .toggle-knob,
    .retention-pill,
    .fx-seg-btn {
      transition: none;
    }
    .retention-pill.active { transform: none; }
  }
</style>
