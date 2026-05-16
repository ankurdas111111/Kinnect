<script>
  import { push } from 'svelte-spa-router';
  import { authUser, loadSession } from '../lib/stores/auth.js';
  import { apiPost, fetchCsrf, clearCsrf } from '../lib/api.js';
  import { COUNTRY_CODES, COUNTRY_MAP, validateMobileLength } from '../lib/countryCodes.js';
  import { toasts } from '../lib/stores/toast.js';
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import AnimatedMeshBackground from '../components/primitives/AnimatedMeshBackground.svelte';
  import KineticText from '../components/primitives/KineticText.svelte';

  let mode = 'email';
  let loginId = '';
  let password = '';
  let showPassword = false;
  let countryIso = 'IN';
  let mobileDigits = '';
  let error = '';
  let loading = false;
  let redirecting = false;
  let mobileHint = '';
  let emailTouched = false;
  let passwordTouched = false;
  let mobileTouched = false;

  onMount(() => {
    if ($authUser) push('/');
    fetchCsrf();
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

  $: emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(loginId.trim());
  $: emailError = emailTouched && loginId.trim() && !emailValid;
  $: mobileValid = mobileDigits ? validateMobileLength(countryIso, mobileDigits).valid : false;
  $: passwordError = passwordTouched && password.length > 0 && password.length < 6;

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
      if (!emailValid) { error = 'Enter a valid email'; return; }
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
        toasts.success('Welcome back!');
        // Login creates a new server session with a new CSRF token.
        // Refresh it now so subsequent POSTs from the main app use the correct token.
        clearCsrf();
        await fetchCsrf();
        await loadSession();
        // If user arrived via QR add-contact link, redirect back to complete it
        const pendingContact = sessionStorage.getItem('kinnect_pending_contact');
        if (pendingContact) {
          sessionStorage.removeItem('kinnect_pending_contact');
          push('/add-contact/' + encodeURIComponent(pendingContact));
        } else {
          push('/');
        }
      } else {
        error = res.error || 'Login failed';
      }
    } catch (e) {
      error = 'Network error';
    }
    loading = false;
  }
</script>

<div class="auth-page page-enter">
  <!-- 2026 Animated Mesh Background — conic mesh + aurora orbs + particles -->
  <AnimatedMeshBackground variant="brand" grid={true} particles={true} />

  <div class="auth-brand">
    <div class="auth-brand-inner">
      <!-- Decorative floating location pins (visual background element) -->
      <div class="auth-brand-deco" aria-hidden="true">
        <svg class="deco-map" viewBox="0 0 340 280" fill="none" xmlns="http://www.w3.org/2000/svg">
          <!-- Connection lines between pins -->
          <line x1="90" y1="80" x2="180" y2="130" stroke="rgba(99,102,241,0.22)" stroke-width="1.5" stroke-dasharray="5 4"/>
          <line x1="180" y1="130" x2="260" y2="90" stroke="rgba(16,185,129,0.20)" stroke-width="1.5" stroke-dasharray="5 4"/>
          <line x1="180" y1="130" x2="140" y2="200" stroke="rgba(139,92,246,0.18)" stroke-width="1.5" stroke-dasharray="5 4"/>
          <line x1="260" y1="90" x2="290" y2="170" stroke="rgba(6,182,212,0.16)" stroke-width="1.5" stroke-dasharray="5 4"/>
          <!-- Pulse rings on live pins -->
          <circle cx="180" cy="130" r="22" fill="rgba(99,102,241,0.06)" stroke="rgba(99,102,241,0.18)" stroke-width="1"/>
          <circle cx="180" cy="130" r="34" fill="none" stroke="rgba(99,102,241,0.08)" stroke-width="1"/>
          <!-- Pin: self (primary) -->
          <circle cx="180" cy="130" r="10" fill="rgba(99,102,241,0.90)"/>
          <circle cx="180" cy="130" r="4" fill="white"/>
          <!-- Pin: family member 1 (emerald) -->
          <circle cx="90" cy="80" r="8" fill="rgba(16,185,129,0.85)"/>
          <circle cx="90" cy="80" r="3" fill="white"/>
          <circle cx="90" cy="80" r="14" fill="none" stroke="rgba(16,185,129,0.22)" stroke-width="1"/>
          <!-- Pin: family member 2 (violet) -->
          <circle cx="260" cy="90" r="7" fill="rgba(139,92,246,0.80)"/>
          <circle cx="260" cy="90" r="3" fill="white"/>
          <!-- Pin: family member 3 (cyan) -->
          <circle cx="140" cy="200" r="7" fill="rgba(6,182,212,0.75)"/>
          <circle cx="140" cy="200" r="3" fill="white"/>
          <!-- Pin: family member 4 (amber) -->
          <circle cx="290" cy="170" r="6" fill="rgba(245,158,11,0.75)"/>
          <circle cx="290" cy="170" r="2.5" fill="white"/>
          <!-- Subtle grid dots -->
          <circle cx="50" cy="160" r="2" fill="rgba(255,255,255,0.06)"/>
          <circle cx="120" cy="240" r="2" fill="rgba(255,255,255,0.06)"/>
          <circle cx="220" cy="220" r="2" fill="rgba(255,255,255,0.06)"/>
          <circle cx="310" cy="50" r="2" fill="rgba(255,255,255,0.06)"/>
          <circle cx="40" cy="40" r="2" fill="rgba(255,255,255,0.06)"/>
        </svg>
      </div>

      <div class="auth-brand-logo">
        <svg width="24" height="29" viewBox="0 0 20 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10 1C5.029 1 1 5.029 1 10c0 6.938 8.25 13.1 9 14.1.75-1 9-7.162 9-14.1C19 5.029 14.971 1 10 1z" fill="white" fill-opacity="0.95"/>
          <path d="M7 7v6M7 10l3.5-3M7 10l3.5 3" stroke="rgba(255,255,255,0.90)" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <div class="auth-brand-badge">Kinnect = Kin + Connect</div>
      <!-- KineticText — per-character spring entrance for the brand headline -->
      <KineticText
        text="Keep your family close"
        tag="h1"
        delay={120}
        stagger={38}
        className="auth-brand-h1"
        once={true}
      />
      <p>Always know your family is safe. Real-time location sharing, built for families who care.</p>
      <ul class="auth-brand-features">
        <li><span class="feature-check" aria-hidden="true"></span> See your family on a live map</li>
        <li><span class="feature-check" aria-hidden="true"></span> One-tap SOS when you need help</li>
        <li><span class="feature-check" aria-hidden="true"></span> Arrival alerts and safe zones</li>
        <li><span class="feature-check" aria-hidden="true"></span> Works on every phone and browser</li>
      </ul>
    </div>
  </div>

  <div class="auth-form-area">
    <!-- Mobile-only brand header: shown when desktop brand panel is hidden -->
    <div class="mobile-brand-header" aria-hidden="true">
      <div class="mobile-brand-logo">
        <svg width="18" height="22" viewBox="0 0 20 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10 1C5.029 1 1 5.029 1 10c0 6.938 8.25 13.1 9 14.1.75-1 9-7.162 9-14.1C19 5.029 14.971 1 10 1z" fill="white" fill-opacity="0.95"/>
          <path d="M7 7v6M7 10l3.5-3M7 10l3.5 3" stroke="rgba(255,255,255,0.90)" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <span class="mobile-brand-name">Kinnect</span>
      <span class="mobile-brand-tagline">Keep your family close</span>
    </div>

    <div class="auth-card">
      <h2>Welcome back</h2>
      <p class="subtitle">Your family is just a tap away</p>

      {#if error}
        <div class="auth-error" role="alert">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
          {error}
        </div>
      {/if}

      <form on:submit|preventDefault={handleSubmit} novalidate>
        <div class="auth-toggle" role="tablist" aria-label="Login method">
          <button type="button" class="auth-toggle-btn" class:active={mode === 'email'} on:click={() => mode = 'email'} on:keydown={(e) => onModeToggleKeydown(e, 'email')} role="tab" aria-selected={mode === 'email'} tabindex={mode === 'email' ? 0 : -1}>Email</button>
          <button type="button" class="auth-toggle-btn" class:active={mode === 'mobile'} on:click={() => mode = 'mobile'} on:keydown={(e) => onModeToggleKeydown(e, 'mobile')} role="tab" aria-selected={mode === 'mobile'} tabindex={mode === 'mobile' ? 0 : -1}>Mobile</button>
        </div>

        {#if mode === 'email'}
          <div class="auth-field" transition:slide={{ duration: 180, axis: 'y' }}>
            <label for="login_email">Email address</label>
            <div class="input-wrapper">
              <input
                id="login_email"
                type="email"
                class="input"
                class:is-valid={emailTouched && emailValid}
                class:is-invalid={emailError}
                bind:value={loginId}
                placeholder="you@example.com"
                autocomplete="email"
                enterkeyhint="next"
                on:blur={() => emailTouched = true}
              />
              {#if emailTouched && emailValid}
                <span class="input-icon valid" aria-hidden="true">&#10003;</span>
              {/if}
            </div>
            {#if emailError}
              <span class="auth-hint error">Enter a valid email address</span>
            {/if}
          </div>
        {/if}
        {#if mode === 'mobile'}
          <div class="auth-field" transition:slide={{ duration: 180, axis: 'y' }}>
            <label for="login_mobile">Mobile number</label>
            <div class="auth-phone-row">
              <select class="auth-cc-select" bind:value={countryIso} on:change={validateMobile} aria-label="Country code">
                {#each COUNTRY_CODES as c}
                  <option value={c[1]}>{c[3]} {c[0]}</option>
                {/each}
              </select>
              <input
                id="login_mobile"
                type="tel"
                class="input"
                class:is-valid={mobileTouched && mobileValid}
                class:is-invalid={mobileTouched && mobileDigits && !mobileValid}
                bind:value={mobileDigits}
                placeholder={mobilePlaceholder()}
                inputmode="numeric"
                enterkeyhint="next"
                on:blur={() => { mobileTouched = true; validateMobile(); }}
              />
            </div>
            {#if mobileTouched && mobileHint}<span class="auth-hint error">{mobileHint}</span>{/if}
          </div>
        {/if}

        <div class="auth-field">
          <label for="password">Password</label>
          <div class="input-wrapper">
            <input
              id="password"
              type={showPassword ? 'text' : 'password'}
              class="input"
              class:is-invalid={passwordError}
              bind:value={password}
              placeholder="••••••••"
              autocomplete="current-password"
              enterkeyhint="go"
              on:blur={() => passwordTouched = true}
            />
            <button type="button" class="input-icon input-icon--toggle" on:click={() => showPassword = !showPassword} aria-label={showPassword ? 'Hide password' : 'Show password'}>
              {#if showPassword}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              {/if}
            </button>
          </div>
          {#if passwordError}
            <span class="auth-hint error">At least 6 characters</span>
          {/if}
        </div>

        <button class="auth-submit" type="submit" disabled={loading} class:redirecting={redirecting}>
          {#if loading}
            <span class="submit-spinner" aria-hidden="true"></span>
            {redirecting ? 'Opening dashboard...' : 'Signing in...'}
          {:else}
            Sign in
          {/if}
        </button>
      </form>

      <p class="auth-link">Don't have an account? <a href="#/register">Create one</a></p>
    </div>
  </div>
</div>

<style>
  @import '../styles/auth.css';

  /* KineticText headline — matches auth.css h1 sizing but uses span chars */
  :global(.auth-brand-h1) {
    font-size: clamp(2rem, 3.5vw, 2.75rem);
    font-weight: 800;
    color: var(--auth-page-text, #ffffff);
    line-height: 1.15;
    letter-spacing: -0.03em;
    margin-bottom: var(--space-4);
  }

  /* Decorative location-pin cluster */
  .auth-brand-deco {
    position: absolute;
    top: -40px;
    right: -80px;
    width: 340px;
    height: 280px;
    pointer-events: none;
    opacity: 0.65;
  }

  .deco-map {
    width: 100%;
    height: 100%;
    filter: drop-shadow(0 4px 12px rgba(20, 184, 166, 0.12));
  }

  .input-wrapper {
    position: relative;
  }
  .input-wrapper .input {
    padding-right: var(--space-8);
  }
  .input-icon {
    position: absolute;
    right: var(--space-3);
    top: 50%;
    transform: translateY(-50%);
    font-size: var(--text-sm);
    pointer-events: none;
  }
  .input-icon.valid {
    color: var(--success-500);
  }
  .input-icon--toggle {
    pointer-events: auto;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    padding: var(--space-3, 12px);
    min-width: 44px;
    min-height: 44px;
    border-radius: 4px;
    transition: color 0.15s;
  }
  .input-icon--toggle:hover { color: var(--text-secondary); }
  .is-valid {
    border-color: var(--success-400) !important;
  }
  .submit-spinner {
    display: inline-block;
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    margin-right: var(--space-2);
    vertical-align: middle;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── Mobile brand header — only visible on small screens ───────────── */
  .mobile-brand-header {
    display: none;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    margin-bottom: 20px;
    animation: fadeIn 400ms var(--ease-out, ease) both;
  }

  .mobile-brand-logo {
    width: 44px;
    height: 44px;
    border-radius: 14px;
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-700, #0f766e) 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 6px 20px rgba(20, 184, 166, 0.40);
    margin-bottom: 4px;
  }

  .mobile-brand-name {
    font-family: var(--font-display, 'Sora', sans-serif);
    font-size: 1.25rem;
    font-weight: 800;
    color: var(--text-primary, #ffffff);
    letter-spacing: -0.02em;
    line-height: 1;
  }

  .mobile-brand-tagline {
    font-size: 0.8125rem;
    color: rgba(255, 255, 255, 0.62);
    letter-spacing: 0.01em;
  }

  /* Submit button transitions to success green after redirect */
  :global(.auth-submit.redirecting) {
    background: linear-gradient(135deg, var(--success-500, #10b981) 0%, var(--success-700, #047857) 100%) !important;
    box-shadow:
      0 8px 24px rgba(16, 185, 129, 0.40),
      0 3px 8px rgba(16, 185, 129, 0.25),
      inset 0 1px 0 rgba(255, 255, 255, 0.22) !important;
    transition: background 300ms var(--ease-out), box-shadow 300ms var(--ease-out) !important;
  }

  @media (max-width: 767px) {
    .mobile-brand-header {
      display: flex;
    }
  }
</style>
