<script>
  import { preventDefault } from 'svelte/legacy';

  import { navigate } from '../lib/viewTransition.js';
  import { authUser, loadSession } from '../lib/stores/auth.js';
  import { markReturningVisitor } from '../lib/entryPoint.js';
  import { apiPost, fetchCsrf, clearCsrf } from '../lib/api.js';
  import { COUNTRY_CODES, COUNTRY_MAP, validateMobileLength } from '../lib/countryCodes.js';
  import { toasts } from '../lib/stores/toast.js';
  import { onMount, tick } from 'svelte';
  import { allowMotion } from '../lib/stores/effects.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';

  let showPassword = $state(false);
  let showConfirm = $state(false);
  // ALL field state lives at script root (never inside step blocks) so values
  // survive step switches — panels below are conditionally rendered.
  let firstName = $state('');
  let lastName = $state('');
  let password = $state('');
  let confirm = $state('');
  let contactType = $state('email');
  let emailValue = $state('');
  let countryIso = $state('IN');
  let mobileDigits = $state('');
  let error = $state('');
  let loading = $state(false);
  let redirecting = $state(false);
  let mobileHint = $state('');
  let emailHint = $state('');

  let firstNameTouched = $state(false);
  let passwordTouched = $state(false);
  let confirmTouched = $state(false);
  let emailTouched = $state(false);
  let mobileTouched = $state(false);

  // ── 3-step progressive disclosure ───────────────────────────────────────
  // Step order (critique fix): Identity → Contact → Security, so the
  // identifier and autocomplete="new-password" land adjacent for password
  // managers (plus a hidden autocomplete="username" mirror on step 3).
  const STEP_TITLES = ['Who you are', 'How to reach you', 'Secure your account'];
  let step = $state(1);

  onMount(() => {
    if ($authUser) navigate('/');
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

  function validateEmail() {
    if (!emailValue.trim()) { emailHint = ''; return false; }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue.trim())) { emailHint = 'Enter a valid email address'; return false; }
    emailHint = '';
    return true;
  }


  function onContactToggleKeydown(e, current) {
    var order = ['email', 'mobile'];
    var idx = order.indexOf(current);
    if (idx < 0) return;
    if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
      e.preventDefault();
      contactType = order[(idx + 1) % order.length];
    } else if (e.key === 'Home') {
      e.preventDefault();
      contactType = order[0];
    } else if (e.key === 'End') {
      e.preventDefault();
      contactType = order[order.length - 1];
    }
  }

  // Strength speaks in the Hearth ladder — ochre (needs a look) → ember →
  // sage (settled). Vermilion is SOS-only, never a form state. `color`
  // fills the bar; `textColor` is an ink-leaning mix so the small label
  // keeps AA contrast on paper and night canvas alike.
  function getPasswordStrength(pw) {
    if (!pw) return { level: 0, label: '', color: '', textColor: '' };
    let score = 0;
    if (pw.length >= 6) score++;
    if (pw.length >= 8) score++;
    if (/[A-Z]/.test(pw)) score++;
    if (/[0-9]/.test(pw)) score++;
    if (/[^A-Za-z0-9]/.test(pw)) score++;

    if (score <= 1) return { level: 1, label: 'Weak', color: 'var(--warning-600)', textColor: 'var(--auth-error-text)' };
    if (score <= 2) return { level: 2, label: 'Fair', color: 'var(--warning-400)', textColor: 'var(--text-secondary)' };
    if (score <= 3) return { level: 3, label: 'Good', color: 'var(--primary-500)', textColor: 'color-mix(in oklch, var(--primary-500) 72%, var(--text-primary))' };
    return { level: 4, label: 'Strong', color: 'var(--success-500)', textColor: 'color-mix(in oklch, var(--success-500) 72%, var(--text-primary))' };
  }

  // Contextual, specific hint for strengthening a weak/fair password.
  function getPasswordHint(pw) {
    if (!pw) return '';
    if (pw.length < 6) return 'Use at least 6 characters';
    if (!/[0-9]/.test(pw) && !/[^A-Za-z0-9]/.test(pw)) return 'Add a number or symbol';
    if (!/[A-Z]/.test(pw)) return 'Add an uppercase letter';
    if (pw.length < 8) return 'A little longer makes it stronger';
    return '';
  }

  // Focus the first field of the current step. preventScroll keeps the step
  // switch scroll-free (reduced-motion users never get a scroll animation).
  async function focusFirstField() {
    await tick();
    const id = step === 1 ? 'first_name'
      : step === 2 ? (contactType === 'email' ? 'reg_email' : 'reg_mobile')
      : 'reg_password';
    document.getElementById(id)?.focus({ preventScroll: true });
  }

  function validateStep(n) {
    if (n === 1) {
      firstNameTouched = true;
      if (!firstName.trim()) { error = 'First name is required to create your account'; return false; }
    } else if (n === 2) {
      if (contactType === 'email') {
        emailTouched = true;
        if (!validateEmail()) { error = emailHint || 'Enter a valid email address'; return false; }
      } else {
        mobileTouched = true;
        if (!validateMobile()) { error = mobileHint || 'Enter a valid mobile number'; return false; }
      }
    }
    return true;
  }

  function goNext() {
    if (!validateStep(step)) return;
    error = '';
    step += 1;
    focusFirstField();
  }

  function goBack() {
    if (step === 1 || loading) return;
    error = '';
    step -= 1;
    focusFirstField();
  }

  // Enter advances steps 1–2; the single server submit happens only on step 3.
  function onFormSubmit() {
    if (step < 3) { goNext(); return; }
    handleSubmit();
  }

  async function handleSubmit() {
    error = '';
    firstNameTouched = true;
    passwordTouched = true;
    confirmTouched = true;
    emailTouched = true;
    mobileTouched = true;

    if (!firstName.trim()) { error = 'First name is required to create your account'; step = 1; focusFirstField(); return; }
    if (password.length < 6) { error = 'Password must be at least 6 characters'; return; }
    if (password !== confirm) { error = 'Passwords do not match — please re-enter'; return; }

    const body = {
      first_name: firstName.trim(),
      last_name: lastName.trim(),
      password,
      confirm,
      contact_type: contactType
    };

    if (contactType === 'email') {
      if (!validateEmail()) { error = emailHint || 'Enter a valid email address'; step = 2; focusFirstField(); return; }
      body.contact_value = emailValue.trim().toLowerCase();
    } else {
      if (!validateMobile()) { error = mobileHint || 'Enter a valid mobile number'; step = 2; focusFirstField(); return; }
      const c = getCountry();
      body.contact_value = c.dial + mobileDigits.replace(/\D/g, '');
    }

    loading = true;
    try {
      const res = await apiPost('/api/register', body);
      if (res.ok) {
        redirecting = true;
        toasts.success('Account created successfully!');
        // Register creates a new server session with a new CSRF token.
        // Refresh it so subsequent POSTs from the main app use the correct token.
        clearCsrf();
        await fetchCsrf();
        await loadSession();
      markReturningVisitor();
        // If user arrived via QR add-contact link, redirect back to complete it
        const pendingContact = sessionStorage.getItem('kinnect_pending_contact');
        if (pendingContact) {
          sessionStorage.removeItem('kinnect_pending_contact');
          navigate('/add-contact/' + encodeURIComponent(pendingContact));
        } else {
          navigate('/');
        }
      } else {
        error = res.error || 'Registration failed — please try again';
      }
    } catch (e) {
      error = 'Network error — check your connection and try again';
    }
    loading = false;
  }
  let passwordStrength = $derived(getPasswordStrength(password));
  let passwordHint = $derived(getPasswordHint(password));
  let emailValid = $derived(emailValue.trim() && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue.trim()));
  let mobileValid = $derived(mobileDigits ? validateMobileLength(countryIso, mobileDigits).valid : false);
  let confirmMatch = $derived(confirm && password === confirm);
  let confirmError = $derived(confirmTouched && confirm && !confirmMatch);

  // Step-validity deriveds drive the builder pebbles.
  let step1Valid = $derived(firstName.trim().length > 0);
  let step2Valid = $derived(contactType === 'email' ? !!emailValid : mobileValid);
  let step3Valid = $derived(password.length >= 6 && !!confirmMatch);

  // Hidden identifier mirrored onto step 3 so password managers pair
  // username ↔ new-password across the step switch.
  let identifierValue = $derived(contactType === 'email'
    ? emailValue.trim().toLowerCase()
    : ((getCountry()?.dial || '') + mobileDigits.replace(/\D/g, '')));

  // Pebble N: unlit socket → 'igniting' (pop) the moment its step validates →
  // locked 'live' once the step is passed. Pure derived, no timers.
  function starState(idx, valid) {
    if (redirecting || step > idx) return 'live';
    if (step === idx && valid) return 'igniting';
    return 'unlit';
  }
  let stepPebbles = $derived([
    starState(1, step1Valid),
    starState(2, step2Valid),
    starState(3, step3Valid),
  ]);
  // Looping decoration is JS-gated: a stored 'full' FX pref must not defeat
  // the OS reduce-motion switch.
  let loops = $derived($allowMotion && !prefersReducedMotion());
</script>

<div class="auth-page page-enter">
  <header class="auth-topbar">
    <div class="auth-wordmark">
      <span class="auth-wordmark-name">Kinnect</span>
      <span class="auth-wordmark-dot" aria-hidden="true"></span>
    </div>
  </header>

  <main class="auth-main">
    <!-- Builder pebbles: one per step, warming up as each step validates.
         Decorative only — the visible aria-live step label below carries
         the real progress semantics. -->
    <div
      class="auth-pebbles"
      class:knocking={loading && !redirecting && loops}
      class:converging={redirecting}
      aria-hidden="true"
    >
      {#each stepPebbles as pebbleState, i}
        <span
          class="auth-pebble"
          class:auth-pebble--self={step === i + 1 && !redirecting}
          class:lit={pebbleState === 'live'}
          class:igniting={pebbleState === 'igniting'}
        >
          {i + 1}
          {#if redirecting}<span class="auth-pebble-presence"></span>{/if}
        </span>
      {/each}
    </div>

    <h1 class="auth-headline">Create your family circle.</h1>
    <p class="auth-subcopy">Three small steps — under a minute, and free forever for families.</p>

    <!-- Page-level mode relationship: Sign In ↔ Create your Circle -->
    <nav class="auth-mode-switch" aria-label="Sign in or create a circle">
      <a class="auth-mode-btn" href="#/login">Sign in</a>
      <span class="auth-mode-btn active" aria-current="page">Create your circle</span>
    </nav>

    <p class="reg-step-label" aria-live="polite">Step {step} of 3 · {STEP_TITLES[step - 1]}</p>

    {#if error}
      <div class="auth-error" role="alert">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <span>{error}</span>
      </div>
    {/if}

    <form class="reg-form" onsubmit={preventDefault(onFormSubmit)} novalidate>
      <h2 class="sr-only">Step {step} of 3: {STEP_TITLES[step - 1]}</h2>

      {#key step}
        <div class="step-panel">
          {#if step === 1}
            <div class="auth-name-row">
              <div class="auth-field">
                <label for="first_name">First name</label>
                <div class="field-shell" class:is-invalid={firstNameTouched && !firstName.trim()}>
                  <input
                    id="first_name"
                    bind:value={firstName}
                    placeholder="John"
                    autocomplete="given-name"
                    enterkeyhint="next"
                    onblur={() => firstNameTouched = true}
                    aria-invalid={firstNameTouched && !firstName.trim() ? 'true' : undefined}
                    aria-describedby={firstNameTouched && !firstName.trim() ? 'first_name_err' : undefined}
                  />
                </div>
                <div class="auth-hint-slot" aria-live="polite">{#if firstNameTouched && !firstName.trim()}<span class="auth-hint error" id="first_name_err">Required</span>{/if}</div>
              </div>
              <div class="auth-field">
                <label for="last_name">Last name</label>
                <div class="field-shell">
                  <input id="last_name" bind:value={lastName} placeholder="Doe" autocomplete="family-name" enterkeyhint="next" />
                </div>
              </div>
            </div>
          {:else if step === 2}
            <span class="auth-label" id="contact_method_label">Contact method</span>
            <div class="auth-toggle" role="tablist" aria-labelledby="contact_method_label">
              <button type="button" class="auth-toggle-btn" class:active={contactType === 'email'} onclick={() => contactType = 'email'} onkeydown={(e) => onContactToggleKeydown(e, 'email')} role="tab" aria-selected={contactType === 'email'} tabindex={contactType === 'email' ? 0 : -1}>Email</button>
              <button type="button" class="auth-toggle-btn" class:active={contactType === 'mobile'} onclick={() => contactType = 'mobile'} onkeydown={(e) => onContactToggleKeydown(e, 'mobile')} role="tab" aria-selected={contactType === 'mobile'} tabindex={contactType === 'mobile' ? 0 : -1}>Mobile</button>
            </div>

            {#if contactType === 'email'}
              <div class="auth-field">
                <label for="reg_email">Email address</label>
                <div class="field-shell" class:is-invalid={emailTouched && emailValue.trim() && !emailValid} class:is-valid={emailTouched && emailValid}>
                  <svg class="field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
                  <input
                    id="reg_email"
                    type="email"
                    bind:value={emailValue}
                    placeholder="you@example.com"
                    onblur={() => { emailTouched = true; validateEmail(); }}
                    autocomplete="email"
                    enterkeyhint="next"
                    aria-invalid={emailTouched && emailValue.trim() && !emailValid ? 'true' : undefined}
                    aria-describedby={emailTouched && emailHint ? 'reg_email_err' : undefined}
                  />
                  {#if emailTouched && emailValid}
                    <svg class="field-check check-pop" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
                  {/if}
                </div>
                <div class="auth-hint-slot" aria-live="polite">{#if emailTouched && emailHint}<span class="auth-hint error" id="reg_email_err">{emailHint}</span>{/if}</div>
              </div>
            {:else}
              <div class="auth-field">
                <label for="reg_mobile">Mobile number</label>
                <div class="field-shell" class:is-invalid={mobileTouched && mobileDigits && !mobileValid} class:is-valid={mobileTouched && mobileValid}>
                  <select class="field-cc" bind:value={countryIso} onchange={validateMobile} aria-label="Country code">
                    {#each COUNTRY_CODES as c}
                      <option value={c[1]}>{c[3]} {c[0]}</option>
                    {/each}
                  </select>
                  <input
                    id="reg_mobile"
                    type="tel"
                    bind:value={mobileDigits}
                    placeholder={mobilePlaceholder()}
                    inputmode="numeric"
                    autocomplete="tel"
                    enterkeyhint="next"
                    onblur={() => { mobileTouched = true; validateMobile(); }}
                    aria-invalid={mobileTouched && mobileDigits && !mobileValid ? 'true' : undefined}
                    aria-describedby={mobileTouched && mobileHint ? 'reg_mobile_err' : undefined}
                  />
                  {#if mobileTouched && mobileValid}
                    <svg class="field-check check-pop" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
                  {/if}
                </div>
                <div class="auth-hint-slot" aria-live="polite">{#if mobileTouched && mobileHint}<span class="auth-hint error" id="reg_mobile_err">{mobileHint}</span>{/if}</div>
              </div>
            {/if}
          {:else}
            <!-- Hidden identifier from step 2 keeps password managers pairing
                 username ↔ new-password across the step switch. -->
            <input class="sr-only" type="text" name="username" autocomplete="username" value={identifierValue} readonly tabindex="-1" aria-hidden="true" />

            <div class="auth-field">
              <label for="reg_password">Password</label>
              <div class="field-shell" class:is-invalid={passwordTouched && password.length > 0 && password.length < 6}>
                <svg class="field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2 18v3c0 .6.4 1 1 1h4v-3h3v-3h2l1.4-1.4a6.5 6.5 0 1 0-4-4Z"/><circle cx="16.5" cy="7.5" r=".5" fill="currentColor"/></svg>
                <input
                  id="reg_password"
                  type={showPassword ? 'text' : 'password'}
                  bind:value={password}
                  autocomplete="new-password"
                  enterkeyhint="next"
                  onblur={() => passwordTouched = true}
                  aria-invalid={passwordTouched && password.length > 0 && password.length < 6 ? 'true' : undefined}
                  aria-describedby="reg_password_hint"
                />
                <button type="button" class="field-trailing-btn" onclick={() => showPassword = !showPassword} aria-label={showPassword ? 'Hide password' : 'Show password'}>
                  {#if showPassword}
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                  {:else}
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                  {/if}
                </button>
              </div>
              <div id="reg_password_hint">
                {#if password}
                  <div class="password-strength">
                    <div class="strength-bar">
                      <div class="strength-fill" style="width:{passwordStrength.level * 25}%;--strength-color:{passwordStrength.color}"></div>
                    </div>
                    <span class="strength-label" style="color:{passwordStrength.textColor}">{passwordStrength.label}</span>
                  </div>
                  {#if passwordHint}
                    <span class="auth-hint password-hint">{passwordHint}</span>
                  {/if}
                {:else}
                  <span class="auth-hint">Minimum 6 characters</span>
                {/if}
              </div>
            </div>

            <div class="auth-field">
              <label for="reg_confirm">Confirm password</label>
              <div class="field-shell" class:is-invalid={confirmError} class:is-valid={confirmTouched && confirmMatch}>
                <svg class="field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2 18v3c0 .6.4 1 1 1h4v-3h3v-3h2l1.4-1.4a6.5 6.5 0 1 0-4-4Z"/><circle cx="16.5" cy="7.5" r=".5" fill="currentColor"/></svg>
                <input
                  id="reg_confirm"
                  type={showConfirm ? 'text' : 'password'}
                  bind:value={confirm}
                  autocomplete="new-password"
                  enterkeyhint="go"
                  onblur={() => confirmTouched = true}
                  aria-invalid={confirmError ? 'true' : undefined}
                  aria-describedby={confirmError ? 'reg_confirm_err' : undefined}
                />
                {#if confirmTouched && confirmMatch}
                  <svg class="field-check check-pop" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
                {:else}
                  <button type="button" class="field-trailing-btn" onclick={() => showConfirm = !showConfirm} aria-label={showConfirm ? 'Hide password' : 'Show password'}>
                    {#if showConfirm}
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                    {:else}
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                    {/if}
                  </button>
                {/if}
              </div>
              <div class="auth-hint-slot" aria-live="polite">{#if confirmError}<span class="auth-hint error" id="reg_confirm_err">Passwords do not match</span>{/if}</div>
            </div>
          {/if}
        </div>
      {/key}

      <div class="step-nav">
        {#if step > 1}
          <button type="button" class="auth-step-back" onclick={goBack} disabled={loading}>Back</button>
        {/if}
        <button class="auth-submit tactile" type="submit" disabled={loading}>
          {#if loading}
            <span class="submit-spinner" aria-hidden="true"></span>
            <span>{redirecting ? 'Opening your circle...' : 'Creating your circle...'}</span>
          {:else if step < 3}
            <span>Continue</span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
          {:else}
            <span>Create family circle</span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
          {/if}
        </button>
      </div>
    </form>

    <p class="auth-link">Already have a circle? <a href="#/login">Sign in</a></p>
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

  /* Step crossfade — incoming panel opacity/transform, GPU-only. */
  .step-panel {
    animation: step-in var(--duration-normal) var(--ease-out) both;
  }
  @keyframes step-in {
    from { opacity: 0; transform: translateY(var(--space-3)); }
    to   { opacity: 1; transform: none; }
  }

  /* Contact-method label sits above the tablist like a field label. */
  .auth-label {
    display: block;
    margin-bottom: var(--space-1-5);
  }

  /* Email/confirm checkmark — sage spring pop-in when a field settles. */
  .check-pop {
    animation: check-pop var(--duration-normal) var(--ease-spring) both;
  }
  @keyframes check-pop {
    from { transform: scale(0.4); opacity: 0; }
    60%  { transform: scale(1.12); opacity: 1; }
    to   { transform: scale(1); opacity: 1; }
  }

  /* ── Password strength — Hearth ladder (ochre → ember → sage) ─────────── */
  .password-strength {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-1);
  }

  .strength-bar {
    flex: 1;
    height: var(--space-1);
    background: var(--surface-inset);
    border-radius: var(--radius-full);
    overflow: hidden;
  }

  /* Spring-eased fill with a soft glow tinted to the current strength color. */
  .strength-fill {
    height: 100%;
    border-radius: var(--radius-full);
    background: var(--strength-color);
    box-shadow: 0 0 8px color-mix(in oklch, var(--strength-color, transparent) 55%, transparent);
    transition:
      width var(--duration-normal) var(--ease-spring),
      background-color var(--duration-normal) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out);
  }

  .strength-label {
    font-size: var(--text-xs);
    font-weight: 600;
    white-space: nowrap;
  }

  /* Specific, calm hint shown below a weak/fair password. */
  .password-hint {
    display: block;
    margin-top: var(--space-1);
  }

  /* ── Calm error cue — brief ochre flash, never vermilion, never a shake. */
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

  @media (prefers-reduced-motion: reduce) {
    .strength-fill { transition: none; }
    .check-pop { animation: none; }
    .step-panel { animation: none; }
    .auth-error::after { animation: none; opacity: 0; }
  }
</style>
