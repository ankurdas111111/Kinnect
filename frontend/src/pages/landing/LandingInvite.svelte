<script>
  /**
   * Trust, platform & invitation callout — "Begin your circle in under
   * thirty seconds." Join form is wired to the app's real routes: an invite
   * passphrase goes to /add-contact/:code, an empty submit to /register.
   */
  import { push } from 'svelte-spa-router';
  import LandingIcon from './LandingIcon.svelte';

  let joinCode = $state('');

  function onJoin(event) {
    event.preventDefault();
    const code = joinCode.trim();
    if (code) push(`/add-contact/${encodeURIComponent(code)}`);
    else push('/register');
  }
</script>

<section class="invite" id="circle" aria-labelledby="invite-title">
  <div class="invite-card">
    <span class="ember-bar" aria-hidden="true"></span>

    <h2 id="invite-title">Begin your circle in under thirty seconds.</h2>

    <p class="invite-sub">
      Free forever for personal circles. No payment cards, no subscriptions,
      no ads, no trackers. Built on open web push standards and native
      Android companion wrappers.
    </p>

    <form class="join-form" onsubmit={onJoin}>
      <label class="visually-hidden" for="invite-code">Circle invite passphrase</label>
      <input
        id="invite-code"
        type="text"
        placeholder="Enter Circle Invite Passphrase"
        bind:value={joinCode}
        autocomplete="off"
      />
      <button class="join-btn" type="submit">Join Circle</button>
    </form>

    <ul class="trust-row">
      <li>
        <span class="ic-sage"><LandingIcon name="lock" size={16} /></span>
        Client-side Passphrase Key
      </li>
      <li>
        <span class="ic-sage"><LandingIcon name="cloud-off" size={16} /></span>
        No Centralized Telemetry DB
      </li>
      <li>
        <span class="ic-sage"><LandingIcon name="android" size={16} /></span>
        Android APK &amp; PWA Ready
      </li>
    </ul>
  </div>
</section>

<style>
  .invite {
    background: var(--paper-warm);
    border-top: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    padding: clamp(var(--space-16), 7vw, 5rem) clamp(var(--space-4), 4vw, var(--space-6));
    scroll-margin-top: calc(var(--space-16) + var(--space-6)); /* sticky nav clearance */
  }

  .invite-card {
    max-width: 56rem; /* design: callout width */
    margin-inline: auto;
    padding: clamp(var(--space-8), 5vw, calc(var(--space-12) + var(--space-2)));
    background: var(--surface-3);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-2xl);
    box-shadow: var(--shadow-sm);
    text-align: center;
  }

  .ember-bar {
    display: block;
    width: calc(var(--space-16) + var(--space-4)); /* 80px accent rule */
    height: var(--space-1);
    margin: 0 auto var(--space-8);
    background: var(--primary-500);
    border-radius: var(--radius-full);
  }

  h2 {
    margin: 0 0 var(--space-4);
    font-family: var(--font-serif);
    font-weight: 400;
    font-size: clamp(var(--text-3xl), 1.4rem + 1.6vw, var(--text-4xl));
    line-height: 1.2;
    letter-spacing: -0.02em;
    color: var(--text-primary);
  }

  .invite-sub {
    max-width: 36rem;
    margin: 0 auto var(--space-8);
    font-size: var(--text-base);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
  }

  .join-form {
    max-width: 28rem; /* design: action container */
    margin: 0 auto var(--space-8);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .visually-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    padding: 0;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
    border: 0;
  }

  input {
    flex: 1;
    min-width: 0;
    min-height: var(--space-12); /* 48px */
    padding: var(--space-3) var(--space-4);
    background: var(--card);
    border: 1px solid color-mix(in srgb, var(--border-strong) 60%, transparent);
    border-radius: var(--radius-input);
    color: var(--text-primary);
    font-family: inherit;
    font-size: var(--text-base); /* ≥16px: no mobile zoom */
  }

  input::placeholder {
    color: var(--text-tertiary);
  }

  input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary-500) 35%, transparent);
  }

  .join-btn {
    min-height: var(--space-12); /* 48px accent-fill floor */
    padding: 0 var(--space-6);
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-family: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    border: none;
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background-color var(--duration-normal) var(--ease-out),
      transform var(--duration-fast) var(--ease-out);
  }

  .join-btn:hover {
    background: var(--primary-600);
  }

  .join-btn:active {
    transform: scale(0.96);
  }

  .trust-row {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--space-4) var(--space-6);
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .trust-row li {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
  }

  .ic-sage {
    color: var(--success-500);
    display: inline-flex;
  }

  @media (min-width: 40rem) {
    .join-form {
      flex-direction: row;
    }
  }
</style>
