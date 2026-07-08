<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [open]
   */

  /** @type {Props} */
  let { open = $bindable(false) } = $props();

  const dispatch = createEventDispatcher();
  let currentPage = $state(0);

  const pages = [
    {
      icon: 'location',
      tab: 'Track',
      title: 'Live Location Tracking',
      desc: 'Share your real-time GPS location with family members.',
      steps: [
        'Tap the tracking button on the map to start',
        'Your location updates every few seconds',
        'Family members see you move on their map in real-time',
        'Tap again to pause tracking anytime',
      ],
    },
    {
      icon: 'people',
      tab: 'People',
      title: 'See Your Family',
      desc: 'View everyone who is sharing their location with you.',
      steps: [
        'Tap any person to see their details — speed, battery, accuracy',
        'Tap their location to fly to them on the map',
        'Colored dots show who is online, moving, or stationary',
      ],
    },
    {
      icon: 'connect',
      tab: 'Share',
      title: 'Add Family Members',
      desc: 'Connect with family using Signal Codes or QR codes.',
      steps: [
        'Go to the Share tab and enter their 6-character Signal Code',
        'Or have them scan your QR code from Profile tab',
        'You can also create Groups to share with multiple people at once',
        'Create Live Links to let anyone track you temporarily',
      ],
    },
    {
      icon: 'ride',
      tab: 'Share',
      title: 'Share My Ride',
      desc: 'Let family track your cab, auto, or bike ride with one tap.',
      steps: [
        'Open Share tab and tap "Share Ride"',
        'Scan your Uber/Ola/Rapido screenshot to auto-fill ride details',
        'Or paste copied ride info from the ride app',
        'Family gets a live tracking link — tap "Reached Safely" when you arrive',
      ],
    },
    {
      icon: 'omw',
      tab: 'Share',
      title: 'On My Way',
      desc: 'Quick one-tap notification to family via WhatsApp.',
      steps: [
        'Tap "On My Way" in the Share tab',
        'Opens WhatsApp with a pre-written message and your live link',
        'Pick a contact or group and send — done in 3 seconds',
      ],
    },
    {
      icon: 'walkie',
      tab: 'People',
      title: 'Walkie-Talkie',
      desc: 'Push-to-talk voice calls with any family member — no phone call needed.',
      steps: [
        'Tap any family member\'s card on the map to open their details',
        'Tap the "Talk" button — they\'ll get an incoming call notification',
        'Once connected, hold the button to speak and release to listen',
        'Works instantly between contacts — no setup, no phone number needed',
      ],
    },
    {
      icon: 'sos',
      tab: 'Safety',
      title: 'SOS Emergency Alert',
      desc: 'Instantly alert all your contacts when you need help.',
      steps: [
        'Long-press the SOS button in the Safety tab',
        'All contacts get an immediate alert with your live location',
        'A public watch link is created so anyone can track you',
        'Set up your Emergency Profile for medical info',
      ],
    },
    {
      icon: 'geofence',
      tab: 'Safety',
      title: 'Geofences & Auto-SOS',
      desc: 'Get alerts when family leaves safe zones or stops moving.',
      steps: [
        'Set up geofences around home, school, or work',
        'Get notified when someone exits the safe zone',
        'Auto-SOS triggers if someone stops moving unexpectedly',
        'Check-in reminders ensure regular safety updates',
      ],
    },
    {
      icon: 'profile',
      tab: 'Profile',
      title: 'Your Signal Code & QR',
      desc: 'Share your unique code so family can find and add you.',
      steps: [
        'Your 6-character Signal Code is in the Profile tab',
        'Tap "Show QR" to display a scannable QR code',
        'Anyone who scans it or enters the code adds you as a contact',
        'Manage your saved places, settings, and emergency profile here',
      ],
    },
  ];

  function next() {
    if (currentPage < pages.length - 1) currentPage++;
    else close();
  }

  function prev() {
    if (currentPage > 0) currentPage--;
  }

  function close() {
    localStorage.setItem('kinnect_guide_seen', '1');
    open = false;
    dispatch('close');
  }

  function goTo(i) { currentPage = i; }

  // Reset page when opened
  run(() => {
    if (open) currentPage = 0;
  });

  // Icon SVG paths
  const icons = {
    location: 'M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z M12 9a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z',
    people: 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M9 7a4 4 0 1 0 0-8 4 4 0 0 0 0 8z M23 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75',
    connect: 'M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8 M16 6l-4-4-4 4 M12 2v13',
    ride: 'M1 3h15v13H1z M16 8h4l3 3v5h-7 M5.5 21a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z M18.5 21a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z',
    omw: 'M5 12h14 M12 5l7 7-7 7',
    sos: 'M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z M12 9v4 M12 17h.01',
    geofence: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
    profile: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
    walkie: 'M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z M19 10v2a7 7 0 0 1-14 0v-2 M12 19v4 M8 23h8',
  };
</script>

{#if open}
  <div class="guide-overlay" role="dialog" aria-modal="true" aria-label="Feature Guide">
    <div class="guide-card">
      <!-- Header -->
      <div class="guide-header">
        <span class="guide-badge">Guide</span>
        <button class="guide-skip" onclick={close}>
          {currentPage === pages.length - 1 ? 'Done' : 'Skip'}
        </button>
      </div>

      <!-- Content -->
      <div class="guide-content">
        <div class="guide-icon-wrap">
          <div class="guide-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              {#each icons[pages[currentPage].icon].split(' M') as seg, i}
                <path d="{i === 0 ? seg : 'M' + seg}"/>
              {/each}
            </svg>
          </div>
          <span class="guide-tab-badge">{pages[currentPage].tab} tab</span>
        </div>

        <h2 class="guide-title">{pages[currentPage].title}</h2>
        <p class="guide-desc">{pages[currentPage].desc}</p>

        <div class="guide-steps">
          {#each pages[currentPage].steps as step, i}
            <div class="guide-step">
              <span class="guide-step-num">{i + 1}</span>
              <span class="guide-step-text">{step}</span>
            </div>
          {/each}
        </div>
      </div>

      <!-- Dots + Nav -->
      <div class="guide-footer">
        <div class="guide-dots" role="tablist" aria-label="Guide pages">
          {#each pages as _, i}
            <button
              class="guide-dot"
              class:active={i === currentPage}
              onclick={() => goTo(i)}
              aria-label="Page {i + 1}"
              aria-selected={i === currentPage}
            ></button>
          {/each}
        </div>
        <div class="guide-nav">
          {#if currentPage > 0}
            <button class="guide-nav-btn guide-prev" onclick={prev}>Back</button>
          {/if}
          <button class="guide-nav-btn guide-next" onclick={next}>
            {currentPage === pages.length - 1 ? 'Get Started' : 'Next'}
          </button>
        </div>
      </div>

      <!-- Page counter -->
      <span class="guide-counter">{currentPage + 1} / {pages.length}</span>
    </div>
  </div>
{/if}

<style>
  .guide-overlay {
    position: fixed;
    inset: 0;
    z-index: var(--z-topmost, 9000);
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
  }

  /*
   * Viewport-proportional card: 25rem (400px, the previous fixed max) is the
   * floor so mobile/tablet render identically; grows with 34vw on desktop
   * and settles at 34rem (544px) on large screens.
   */
  .guide-card {
    background: var(--surface-raised, #1a1a2e);
    border: 1px solid var(--border-default);
    border-radius: 24px;
    width: min(92vw, clamp(25rem, 34vw, 34rem));
    max-height: 85vh;
    overflow-y: auto;
    padding: clamp(20px, 1.8vw, 32px);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.40);
    position: relative;
  }

  .guide-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
  }

  .guide-badge {
    font-family: var(--font-display);
    font-size: var(--text-2xs, 10px);
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--primary-400);
    background: rgba(20, 184, 166, 0.12);
    border: 1px solid rgba(20, 184, 166, 0.25);
    padding: 3px 10px;
    border-radius: 99px;
  }

  .guide-skip {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: none;
    border: none;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 6px;
    transition: color 150ms;
  }
  .guide-skip:hover { color: var(--text-primary); }

  /* Content */
  .guide-content {
    text-align: center;
    margin-bottom: 20px;
  }

  .guide-icon-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }

  .guide-icon {
    width: clamp(56px, 4vw, 68px);
    height: clamp(56px, 4vw, 68px);
    border-radius: 16px;
    background: linear-gradient(135deg, rgba(20, 184, 166, 0.15), rgba(99, 102, 241, 0.10));
    border: 1px solid rgba(20, 184, 166, 0.25);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400);
  }

  .guide-tab-badge {
    font-family: var(--font-display);
    font-size: var(--text-2xs, 10px);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    padding: 2px 10px;
    border-radius: 99px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .guide-title {
    font-family: var(--font-display);
    font-size: clamp(var(--text-xl, 20px), 1.4vw, 26px);
    font-weight: 800;
    color: var(--text-primary);
    margin: 0 0 6px;
    letter-spacing: -0.02em;
  }

  .guide-desc {
    font-size: clamp(var(--text-sm, 13px), 1vw, 15px);
    color: var(--text-secondary);
    margin: 0 0 16px;
    line-height: 1.5;
  }

  /* Steps */
  .guide-steps {
    display: flex;
    flex-direction: column;
    gap: 8px;
    text-align: left;
  }

  .guide-step {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 8px 12px;
    background: var(--surface-inset);
    border-radius: var(--radius-lg, 12px);
    border: 1px solid var(--border-subtle);
  }

  .guide-step-num {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--primary-500);
    color: white;
    font-family: var(--font-display);
    font-size: var(--text-2xs, 10px);
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .guide-step-text {
    font-size: clamp(var(--text-sm, 13px), 0.95vw, 15px);
    color: var(--text-primary);
    line-height: 1.4;
  }

  /* Footer */
  .guide-footer {
    display: flex;
    flex-direction: column;
    gap: 14px;
    align-items: center;
  }

  .guide-dots {
    display: flex;
    gap: 6px;
  }

  .guide-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--border-default);
    border: none;
    cursor: pointer;
    padding: 0;
    transition: background 200ms, transform 200ms;
  }
  .guide-dot.active {
    background: var(--primary-500);
    transform: scale(1.3);
  }

  .guide-nav {
    display: flex;
    gap: 8px;
    width: 100%;
  }

  .guide-nav-btn {
    flex: 1;
    padding: 12px;
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    border-radius: var(--radius-lg, 12px);
    cursor: pointer;
    transition: background 150ms, transform 120ms;
    border: none;
  }
  .guide-nav-btn:active { transform: scale(0.97); }

  .guide-prev {
    background: var(--surface-inset);
    color: var(--text-secondary);
    border: 1px solid var(--border-default);
  }
  .guide-prev:hover { background: var(--surface-hover); }

  .guide-next {
    background: var(--primary-500);
    color: white;
  }
  .guide-next:hover { background: var(--primary-600); }

  .guide-counter {
    position: absolute;
    bottom: 20px;
    right: 20px;
    font-family: var(--font-mono, monospace);
    font-size: var(--text-2xs, 10px);
    color: var(--text-tertiary);
  }

  /* Mobile: card fills the screen (unchanged behavior) */
  @media (max-width: 480px) {
    .guide-card {
      width: 100%;
      max-width: 100%;
      max-height: 90vh;
      border-radius: 20px;
    }
  }
</style>
