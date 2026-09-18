<script>
  import { preventDefault } from 'svelte/legacy';

  import { navigate } from '../lib/viewTransition.js';
  import { authUser, loadSession } from '../lib/stores/auth.js';
  import { markReturningVisitor } from '../lib/entryPoint.js';
  import { apiPost, fetchCsrf, clearCsrf } from '../lib/api.js';
  import { COUNTRY_CODES, COUNTRY_MAP, validateMobileLength } from '../lib/countryCodes.js';
  import { toasts } from '../lib/stores/toast.js';
  import { allowMotion } from '../lib/stores/effects.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';
  import { onMount, onDestroy } from 'svelte';
  import { slide } from 'svelte/transition';

  let mode = $state('email');
  let loginId = $state('');
  let password = $state('');
  let showPassword = $state(false);
  let countryIso = $state('IN');
  let mobileDigits = $state('');
  let error = $state('');
  let loading = $state(false);
  let redirecting = $state(false);
  let mobileHint = $state('');
  let emailTouched = $state(false);
  let passwordTouched = $state(false);
  let mobileTouched = $state(false);

  onMount(() => {
    fetchCsrf();
    // Cold-start race fix: MainApp's mount guard pushes here BEFORE the async
    // loadSession (/api/me) resolves, and a mount-only check would strand a
    // validly-signed-in user on this page. Subscribe instead: the moment the
    // session hydrates, bounce back to the app.
    const unsub = authUser.subscribe((u) => {
      if (u) navigate('/');
    });
    return () => unsub();
  });

  function getCountry() { return COUNTRY_MAP[countryIso]; }

  function mobilePlaceholder() {
    const c = getCountry();
    if (!c) return '';
    return c.min === c.max ? `${c.min} digits` : `${c.min}-${c.max} digits`;
  }

  function validateMobile() {
    if (!mobileDigits) { mobileHint = ''; return false; }
    const r = validateMobileLength(countryIso, mobileDigits);
    mobileHint = r.valid ? '' : r.msg;
    return r.valid;
  }

  let emailValid = $derived(/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(loginId.trim()));
  let emailError = $derived(emailTouched && loginId.trim() && !emailValid);
  let mobileValid = $derived(mobileDigits ? validateMobileLength(countryIso, mobileDigits).valid : false);
  let passwordError = $derived(passwordTouched && password.length > 0 && password.length < 6);

  /* ── The handshake — the pebble trio wired to EXISTING form state.
     Sync format checks only (no async validation on login fields — user
     enumeration risk). Center pebble = self beacon (always warm). Side
     pebbles ignite as the identifier and password become valid; submit is
     the "knock" (gentle self-pebble pulse); success converges the circle
     as a fire-and-forget flourish — navigation is NEVER delayed for it.
     Purely decorative: the cluster stays aria-hidden. */
  let idValid = $derived(mode === 'email' ? emailValid : mobileValid);
  let pwReady = $derived(password.length >= 6);
  let converging = $state(false);
  // Ignition state for pebbles 1–4 (identity, password, knock ×2).
  let memberStates = $state(['unlit', 'unlit', 'unlit', 'unlit']);
  const igniteTimers = [null, null, null, null];

  function setMember(i, lit) {
    if (igniteTimers[i]) { clearTimeout(igniteTimers[i]); igniteTimers[i] = null; }
    if (!lit) { memberStates[i] = 'unlit'; return; }
    if (memberStates[i] === 'live' || memberStates[i] === 'igniting') return;
    // JS-driven sequence: gated on BOTH the effects store and the live OS
    // reduce-motion switch (a stored 'full' pref must not defeat the OS).
    if ($allowMotion && !prefersReducedMotion()) {
      memberStates[i] = 'igniting';
      igniteTimers[i] = setTimeout(() => { memberStates[i] = 'live'; igniteTimers[i] = null; }, 340);
    } else {
      memberStates[i] = 'live';
    }
  }

  $effect(() => { setMember(0, idValid); });
  $effect(() => { setMember(1, pwReady); });
  $effect(() => { setMember(2, loading); setMember(3, loading); });

  onDestroy(() => {
    for (let i = 0; i < igniteTimers.length; i++) {
      if (igniteTimers[i]) clearTimeout(igniteTimers[i]);
    }
  });

  // The knock pulse loops while the request is in flight — JS-gated like
  // every looping decoration (effects store + live OS reduce-motion).
  let pulseOk = $derived($allowMotion && !prefersReducedMotion());

  function onModeToggleKeydown(e, current) {
    var order = ['email', 'mobile'];
    var idx = order.indexOf(current);
    if (idx < 0) return;
    if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
      e.preventDefault();
      mode = order[(idx + 1) % order.length];
    } else if (e.key === 'Home') {
      e.preventDefault();
      mode = order[0];
    } else if (e.key === 'End') {
      e.preventDefault();
      mode = order[order.length - 1];
    }
  }

  async function handleSubmit() {
    error = '';
    emailTouched = true;
    passwordTouched = true;
    mobileTouched = true;

    if (mode === 'email') {
      if (!emailValid) { error = 'Enter a valid email address'; return; }
    } else {
      if (!validateMobile()) { error = mobileHint || 'Enter a valid mobile number'; return; }
    }
    if (!password) { error = 'Password is required'; return; }
    if (password.length < 6) { error = 'Password must be at least 6 characters'; return; }

    loading = true;
    try {
      const body = { password };
      if (mode === 'email') {
        body.login_id = loginId.trim().toLowerCase();
        body.login_method = 'email';
      } else {
        const c = getCountry();
        body.login_id = c.dial + mobileDigits.replace(/\D/g, '');
        body.login_method = 'mobile';
      }
      let res = await apiPost('/api/login', body);
      if (!res.ok && res.error === 'Invalid CSRF token') {
        // Stale session — reset and retry once with a fresh token
        clearCsrf();
        await fetchCsrf();
        res = await apiPost('/api/login', body);
      }
      if (res.ok) {
        redirecting = true;
        // Fire-and-forget convergence flourish, parallel to loadSession().
        // Pure CSS transition, never awaited — navigation wins.
        converging = true;
        toasts.success('Welcome back!');
        // Login creates a new server session with a new CSRF token.
        // Refresh it now so subsequent POSTs from the main app use the correct token.
        clearCsrf();
        await fetchCsrf();
        await loadSession();
        markReturningVisitor();   // never re-pitch the landing to this browser
        // Returning user (signed in, not registered) — mark onboarded so
        // MainApp's first-run "Welcome to Kinnect" overlay doesn't re-fire
        // on a fresh browser. Same key shape as MainApp.svelte.
        if ($authUser?.userId) {
          localStorage.setItem('kinnect_onboarded_' + $authUser.userId, '1');
        }
        // If user arrived via QR add-contact link, redirect back to complete it
        const pendingContact = sessionStorage.getItem('kinnect_pending_contact');
        if (pendingContact) {
          sessionStorage.removeItem('kinnect_pending_contact');
          navigate('/add-contact/' + encodeURIComponent(pendingContact));
        } else {
          navigate('/');
        }
      } else {
        error = res.error || 'Sign in failed — please check your credentials and try again';
      }
    } catch (e) {
      error = 'Network error — check your connection and try again';
    }
    loading = false;
  }
</script>

<div class="auth-page page-enter">
  <header class="auth-topbar">
    <div class="auth-wordmark">
      <span class="auth-wordmark-name">Kinnect</span>
      <span class="auth-wordmark-dot" aria-hidden="true"></span>
    </div>
  </header>

  <main class="auth-main">
    <!-- The handshake pebbles — decorative echo of the form state -->
    <div
      class="auth-pebbles"
      class:knocking={loading && !redirecting && pulseOk}
      class:converging
      aria-hidden="true"
    >
      <span class="auth-pebble" class:lit={memberStates[0] === 'live'} class:igniting={memberStates[0] === 'igniting'}>A</span>
      <span class="auth-pebble auth-pebble--self lit">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
        {#if converging}<span class="auth-pebble-presence"></span>{/if}
      </span>
      <span class="auth-pebble" class:lit={memberStates[1] === 'live'} class:igniting={memberStates[1] === 'igniting'}>M</span>
    </div>

    <h1 class="auth-headline">Welcome home to your circle.</h1>
    <p class="auth-subcopy">Sign in with your email or phone to see how everyone's doing.</p>

    <!-- Page-level mode relationship: Sign In ↔ Create your Circle -->
    <nav class="auth-mode-switch" aria-label="Sign in or create a circle">
      <span class="auth-mode-btn active" aria-current="page">Sign in</span>
      <a class="auth-mode-btn" href="#/register">Create your circle</a>
    </nav>

    {#if error}
      <div class="auth-error" role="alert">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <span>{error}</span>
      </div>
    {/if}

    <form onsubmit={preventDefault(handleSubmit)} novalidate>
      <div class="auth-toggle" role="tablist" aria-label="Login method">
        <button type="button" class="auth-toggle-btn" class:active={mode === 'email'} onclick={() => mode = 'email'} onkeydown={(e) => onModeToggleKeydown(e, 'email')} role="tab" aria-selected={mode === 'email'} tabindex={mode === 'email' ? 0 : -1}>Email</button>
        <button type="button" class="auth-toggle-btn" class:active={mode === 'mobile'} onclick={() => mode = 'mobile'} onkeydown={(e) => onModeToggleKeydown(e, 'mobile')} role="tab" aria-selected={mode === 'mobile'} tabindex={mode === 'mobile' ? 0 : -1}>Mobile</button>
      </div>

      {#if mode === 'email'}
        <div class="auth-field" transition:slide={{ duration: 180, axis: 'y' }}>
          <label for="login_email">Email address</label>
          <div class="field-shell" class:is-invalid={emailError} class:is-valid={emailTouched && emailValid}>
            <svg class="field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
            <input
              id="login_email"
              type="email"
              bind:value={loginId}
              placeholder="you@example.com"
              autocomplete="email"
              enterkeyhint="next"
              onblur={() => emailTouched = true}
              aria-invalid={emailError ? 'true' : undefined}
              aria-describedby={emailError ? 'login_email_err' : undefined}
            />
            {#if emailTouched && emailValid}
              <svg class="field-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
            {/if}
          </div>
          <div class="auth-hint-slot" aria-live="polite">{#if emailError}<span class="auth-hint error" id="login_email_err">Enter a valid email address</span>{/if}</div>
        </div>
      {/if}
      {#if mode === 'mobile'}
        <div class="auth-field" transition:slide={{ duration: 180, axis: 'y' }}>
          <label for="login_mobile">Mobile number</label>
          <div class="field-shell" class:is-invalid={mobileTouched && mobileDigits && !mobileValid} class:is-valid={mobileTouched && mobileValid}>
            <select class="field-cc" bind:value={countryIso} onchange={validateMobile} aria-label="Country code">
              {#each COUNTRY_CODES as c}
                <option value={c[1]}>{c[3]} {c[0]}</option>
              {/each}
            </select>
            <input
              id="login_mobile"
              type="tel"
              bind:value={mobileDigits}
              placeholder={mobilePlaceholder()}
              inputmode="numeric"
              enterkeyhint="next"
              onblur={() => { mobileTouched = true; validateMobile(); }}
              aria-invalid={mobileTouched && mobileDigits && !mobileValid ? 'true' : undefined}
              aria-describedby={mobileTouched && mobileHint ? 'login_mobile_err' : undefined}
            />
            {#if mobileTouched && mobileValid}
              <svg class="field-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
            {/if}
          </div>
          <div class="auth-hint-slot" aria-live="polite">{#if mobileTouched && mobileHint}<span class="auth-hint error" id="login_mobile_err">{mobileHint}</span>{/if}</div>
        </div>
      {/if}

      <div class="auth-field">
        <label for="password">Password</label>
        <div class="field-shell" class:is-invalid={passwordError}>
          <svg class="field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2 18v3c0 .6.4 1 1 1h4v-3h3v-3h2l1.4-1.4a6.5 6.5 0 1 0-4-4Z"/><circle cx="16.5" cy="7.5" r=".5" fill="currentColor"/></svg>
          <input
            id="password"
            type={showPassword ? 'text' : 'password'}
            bind:value={password}
            placeholder="Your password"
            autocomplete="current-password"
            enterkeyhint="go"
            onblur={() => passwordTouched = true}
            aria-invalid={passwordError ? 'true' : undefined}
            aria-describedby={passwordError ? 'login_password_err' : undefined}
          />
          <button type="button" class="field-trailing-btn" onclick={() => showPassword = !showPassword} aria-label={showPassword ? 'Hide password' : 'Show password'}>
            {#if showPassword}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
            {:else}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
            {/if}
          </button>
        </div>
        <div class="auth-hint-slot" aria-live="polite">{#if passwordError}<span class="auth-hint error" id="login_password_err">At least 6 characters required</span>{/if}</div>
        <p class="auth-caption">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          <span>Only people you invite can ever see where you are.</span>
        </p>
      </div>

      <button class="auth-submit tactile" type="submit" disabled={loading} class:redirecting={redirecting}>
        {#if loading}
          <span class="submit-spinner" aria-hidden="true"></span>
          <span>{redirecting ? 'Opening your circle...' : 'Signing you in...'}</span>
        {:else}
          <span>Enter your circle</span>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
        {/if}
      </button>
    </form>

    <p class="auth-link">Don't have an account? <a href="#/register">Create your circle</a></p>
  </main>

  <footer class="auth-footer">
    <span class="auth-footer-shield" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/></svg>
    </span>
    <p>Encrypted on-device. Kinnect never sells location data or shows ads — built for quiet peace of mind.</p>
  </footer>
</div>

<style>
  @import '../styles/auth.css';

  /* ── Staggered field entrance ─────────────────────────────────────────────
     Form controls rise + fade in sequentially. GPU-only (transform/opacity).
     Per-child delay derived from --stagger-step (global.css). */
  form > .auth-toggle,
  form > .auth-field,
  form > .auth-submit {
    animation: field-rise var(--duration-normal, 200ms) var(--ease-out) both;
  }
  form > *:nth-child(1) { animation-delay: calc(var(--stagger-step, 40ms) * 1); }
  form > *:nth-child(2) { animation-delay: calc(var(--stagger-step, 40ms) * 2); }
  form > *:nth-child(3) { animation-delay: calc(var(--stagger-step, 40ms) * 3); }
  form > *:nth-child(4) { animation-delay: calc(var(--stagger-step, 40ms) * 4); }

  @keyframes field-rise {
    from { opacity: 0; transform: translateY(var(--space-2-5)); }
    to   { opacity: 1; transform: translateY(0); }
  }

  /* ── Calm error cue ───────────────────────────────────────────────────────
     A brief ochre-tint flash instead of a shake. The banner stays still; a
     pointer-events-free overlay fades from full to zero (GPU-only opacity),
     reading as a soft border + background flash. Ochre, never vermilion. */
  .auth-error {
    position: relative;
    overflow: hidden;
  }
  .auth-error::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    border: 1px solid color-mix(in oklch, var(--warning-500) 45%, transparent);
    background: color-mix(in oklch, var(--warning-500) 16%, transparent);
    pointer-events: none;
    animation: error-tint-flash 700ms var(--ease-out) both;
  }
  @keyframes error-tint-flash {
    0%   { opacity: 1; }
    100% { opacity: 0; }
  }

  /* ── Reduced motion — decorative entrances/cues off ─────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    form > .auth-toggle,
    form > .auth-field,
    form > .auth-submit {
      animation: none;
      opacity: 1;
      transform: none;
    }
    .auth-error::after {
      animation: none;
      opacity: 0;
    }
  }
</style>
