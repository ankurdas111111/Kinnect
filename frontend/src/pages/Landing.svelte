<script>
  /**
   * Landing — Premium scroll-based storytelling page for Kinnect
   *
   * Sections:
   *   1. Hero    — animated entry, headline, CTA
   *   2. Social proof bar
   *   3. Features — staggered card reveal
   *   4. How it works — numbered steps with connector
   *   5. Demo panel  — interactive mockup
   *   6. CTA footer
   *
   * Techniques used:
   *   - IntersectionObserver for scroll-reveal
   *   - spring / tweened for physics motion
   *   - Svelte fade / fly / scale / crossfade transitions
   *   - rAF tilt on hero graphic
   *   - Animated stat counters
   *   - Particle field (CSS-only, no canvas)
   */
  import { onMount, onDestroy, tick } from 'svelte';
  import { spring, tweened } from 'svelte/motion';
  import { fade, fly, scale, crossfade } from 'svelte/transition';
  import { cubicOut, elasticOut, backOut } from 'svelte/easing';
  import Button from '../components/primitives/Button.svelte';
  import Card   from '../components/primitives/Card.svelte';
  import Input  from '../components/primitives/Input.svelte';

  // ── Routing ──────────────────────────────────────────────────────────────
  import { push } from 'svelte-spa-router';

  // ── Hero tilt ─────────────────────────────────────────────────────────────
  let heroCardEl;
  let heroRaf = null;
  let hcx = 0, hcy = 0, htx = 0, hty = 0;

  function heroLerp(a, b, t) { return a + (b - a) * t; }

  function heroTick() {
    hcx = heroLerp(hcx, htx, 0.08);
    hcy = heroLerp(hcy, hty, 0.08);
    if (heroCardEl) {
      heroCardEl.style.transform =
        `perspective(1000px) rotateX(${hcx}deg) rotateY(${hcy}deg)`;
    }
    if (Math.abs(hcx - htx) > 0.04 || Math.abs(hcy - hty) > 0.04) {
      heroRaf = requestAnimationFrame(heroTick);
    } else {
      hcx = htx; hcy = hty;
      if (heroCardEl) heroCardEl.style.transform =
        `perspective(1000px) rotateX(${hcx}deg) rotateY(${hcy}deg)`;
      heroRaf = null;
    }
  }

  function onHeroMouseMove(e) {
    if (!heroCardEl) return;
    const r  = heroCardEl.getBoundingClientRect();
    const dx = (e.clientX - r.left - r.width  / 2) / (r.width  / 2);
    const dy = (e.clientY - r.top  - r.height / 2) / (r.height / 2);
    htx = -dy * 6;
    hty =  dx * 6;
    if (!heroRaf) heroRaf = requestAnimationFrame(heroTick);
  }

  function onHeroMouseLeave() {
    htx = 0; hty = 0;
    if (!heroRaf) heroRaf = requestAnimationFrame(heroTick);
  }

  // ── Animated counters ─────────────────────────────────────────────────────
  const stats = [
    { value: 50000, label: 'Families protected', suffix: '+' },
    { value: 99.9,  label: 'Uptime SLA',          suffix: '%', decimals: 1 },
    { value: 2,     label: 'Sec avg update speed', suffix: 's' },
    { value: 180,   label: 'Countries supported',  suffix: '+' },
  ];
  let statDisplays = stats.map(() => 0);
  let statsVisible = false;
  let statsEl;

  function animateStat(target, decimals = 0, onUpdate) {
    const duration = 1600;
    const start    = performance.now();
    function frame(now) {
      const t = Math.min((now - start) / duration, 1);
      const eased = 1 - Math.pow(1 - t, 3); // ease-out-cubic
      onUpdate(+(target * eased).toFixed(decimals));
      if (t < 1) requestAnimationFrame(frame);
    }
    requestAnimationFrame(frame);
  }

  // ── Scroll reveal ─────────────────────────────────────────────────────────
  let revealed = new Set();
  let observers = [];

  function observeReveal(el, key) {
    if (!el) return;
    const obs = new IntersectionObserver(entries => {
      for (const e of entries) {
        if (e.isIntersecting) {
          revealed = new Set([...revealed, key]);
          obs.disconnect();
        }
      }
    }, { threshold: 0.15 });
    obs.observe(el);
    observers.push(obs);
  }

  function observeStats(el) {
    if (!el) return;
    const obs = new IntersectionObserver(entries => {
      for (const e of entries) {
        if (e.isIntersecting && !statsVisible) {
          statsVisible = true;
          stats.forEach((s, i) => {
            animateStat(s.value, s.decimals || 0, v => { statDisplays[i] = v; statDisplays = [...statDisplays]; });
          });
          obs.disconnect();
        }
      }
    }, { threshold: 0.3 });
    obs.observe(el);
    observers.push(obs);
  }

  // ── Features list ─────────────────────────────────────────────────────────
  const features = [
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>`,
      title: 'Real-time GPS',
      desc: 'Sub-2-second position updates with Kalman-filtered accuracy. Never a stale pin.',
      glow: 'primary',
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8h1a4 4 0 0 1 0 8h-1"/><path d="M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z"/><line x1="6" y1="1" x2="6" y2="4"/><line x1="10" y1="1" x2="10" y2="4"/><line x1="14" y1="1" x2="14" y2="4"/></svg>`,
      title: 'Smart Geofences',
      desc: 'Draw zones around school, home, work. Instant alerts when anyone arrives or leaves.',
      glow: null,
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
      title: 'SOS Alerts',
      desc: 'One-tap emergency signal. Notifies your entire family and opens live tracking instantly.',
      glow: 'danger',
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>`,
      title: 'Secret Chat',
      desc: 'End-to-end encrypted family messaging. Messages vanish after read. Zero metadata.',
      glow: null,
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 8v4l3 3"/></svg>`,
      title: 'Route History',
      desc: 'Replay anyone\'s route for the last 30 days. Timeline scrubbing, speed heatmaps.',
      glow: null,
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      title: 'Guardian Roles',
      desc: 'Granular permissions. Parents see everything. Kids see what you allow. No surprises.',
      glow: null,
    },
  ];

  // ── How it works steps ────────────────────────────────────────────────────
  const steps = [
    { num: '01', title: 'Create your family', desc: 'Sign up in 30 seconds. Invite family by phone number or QR code.' },
    { num: '02', title: 'Set your zones',     desc: 'Draw geofences around places that matter — home, school, grandma\'s.' },
    { num: '03', title: 'Stay calm',          desc: 'Everyone shows up on the map. Alerts fire automatically. You just live.' },
  ];

  // ── Interactive demo state ────────────────────────────────────────────────
  let demoTab = 'map';
  const demoTabs = ['map', 'alerts', 'chat'];

  // ── Live ping animation ────────────────────────────────────────────────────
  let pingCount = 0;
  let pingInterval;

  onMount(() => {
    pingInterval = setInterval(() => { pingCount = (pingCount + 1) % 3; }, 1800);
  });

  // ── Lifecycle ─────────────────────────────────────────────────────────────
  onDestroy(() => {
    observers.forEach(o => o.disconnect());
    if (heroRaf) cancelAnimationFrame(heroRaf);
    if (pingInterval) clearInterval(pingInterval);
  });

  // ── Reveal action (Svelte use:) ────────────────────────────────────────────
  function reveal(node, key) {
    observeReveal(node, key);
    return { destroy() {} };
  }

  function statsObserve(node) {
    observeStats(node);
    return { destroy() {} };
  }

  // ── CTA email ─────────────────────────────────────────────────────────────
  let ctaEmail = '';
  let ctaSubmitted = false;

  function onCtaSubmit() {
    if (!ctaEmail) return;
    ctaSubmitted = true;
  }
</script>

<div class="landing" aria-label="Kinnect landing page">

  <!-- ═══════ HERO ════════════════════════════════════════════════════════ -->
  <section class="hero" on:mousemove={onHeroMouseMove} on:mouseleave={onHeroMouseLeave}>
    <!-- Background atmosphere -->
    <div class="hero-bg" aria-hidden="true">
      <div class="hero-orb hero-orb-1"></div>
      <div class="hero-orb hero-orb-2"></div>
      <div class="hero-orb hero-orb-3"></div>
      <div class="hero-grid"></div>
      <!-- Floating particles -->
      {#each Array(14) as _, i}
        <div class="hero-particle" style="
          left:{8 + i * 6.5}%;
          top:{15 + (i * 37 % 70)}%;
          animation-delay:{-i * 0.7}s;
          animation-duration:{5 + (i % 4)}s;
          opacity:{0.2 + (i % 3) * 0.15};
        " aria-hidden="true"></div>
      {/each}
    </div>

    <div class="landing-container">
      <div class="hero-layout">

        <!-- Left: Copy -->
        <div class="hero-copy">
          <div
            class="hero-badge"
            in:fly={{ y: -16, duration: 500, delay: 100, easing: cubicOut }}
          >
            <span class="hero-badge-dot" aria-hidden="true"></span>
            Live on 3 platforms
          </div>

          <h1
            class="hero-headline"
            in:fly={{ y: 24, duration: 600, delay: 200, easing: cubicOut }}
          >
            Know your family
            <span class="hero-headline-accent"> is safe.</span>
            <span class="hero-headline-sub"> Always.</span>
          </h1>

          <p
            class="hero-tagline"
            in:fade={{ duration: 500, delay: 420 }}
          >
            Kinnect gives families real-time GPS, smart geofence alerts, and
            one-tap SOS — wrapped in a design that feels calm, not clinical.
          </p>

          <div
            class="hero-actions"
            in:fly={{ y: 16, duration: 500, delay: 540, easing: cubicOut }}
          >
            <Button variant="primary" size="lg" on:click={() => push('/register')}>
              Start for free
              <svelte:fragment slot="icon">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <line x1="3" y1="8" x2="13" y2="8"/>
                  <polyline points="9 4 13 8 9 12"/>
                </svg>
              </svelte:fragment>
            </Button>

            <Button variant="ghost" size="lg" on:click={() => push('/login')}>
              Sign in
            </Button>
          </div>

          <p class="hero-social-proof" in:fade={{ duration: 500, delay: 700 }}>
            <span class="hero-avatars" aria-hidden="true">
              {#each ['#14b8a6','#a855f7','#f59e0b','#ec4899'] as c}
                <span class="hero-avatar" style="background:{c};"></span>
              {/each}
            </span>
            <span>Join <strong>50,000+</strong> families worldwide</span>
          </p>
        </div>

        <!-- Right: 3D device mockup -->
        <div
          class="hero-visual"
          in:fly={{ x: 40, duration: 700, delay: 350, easing: cubicOut }}
        >
          <div class="hero-card-wrap" bind:this={heroCardEl}>
            <!-- App mockup card -->
            <div class="mockup-card">
              <!-- Status bar -->
              <div class="mockup-topbar">
                <div class="mockup-title">Family</div>
                <div class="mockup-ping" aria-label="Live updates active">
                  <span class="ping-ring" aria-hidden="true"></span>
                  <span class="ping-dot"  aria-hidden="true"></span>
                </div>
              </div>

              <!-- Mini map surface -->
              <div class="mockup-map" aria-hidden="true">
                <div class="mockup-map-grid"></div>
                <!-- Animated user pins -->
                <div class="map-pin pin-1">
                  <div class="pin-dot" style="background:#14b8a6"></div>
                  <div class="pin-ring" style="border-color:#14b8a6"></div>
                  <div class="pin-label">Mom</div>
                </div>
                <div class="map-pin pin-2">
                  <div class="pin-dot" style="background:#a855f7"></div>
                  <div class="pin-ring" style="border-color:#a855f7"></div>
                  <div class="pin-label">Dad</div>
                </div>
                <div class="map-pin pin-3">
                  <div class="pin-dot" style="background:#f59e0b"></div>
                  <div class="pin-ring" style="border-color:#f59e0b"></div>
                  <div class="pin-label">Zara</div>
                </div>
                <!-- Geofence circle -->
                <div class="map-geofence" aria-hidden="true"></div>
              </div>

              <!-- Member rows -->
              <div class="mockup-list">
                {#each [
                  { name: 'Mom',   loc: 'Home',        col: '#14b8a6', ago: 'just now' },
                  { name: 'Dad',   loc: 'Office',      col: '#a855f7', ago: '3 min ago' },
                  { name: 'Zara',  loc: 'School',      col: '#f59e0b', ago: '2 min ago' },
                ] as m, i}
                  <div class="mockup-row" style="animation-delay:{i * 80}ms">
                    <span class="mockup-avatar" style="background:{m.col}" aria-hidden="true"></span>
                    <span class="mockup-info">
                      <span class="mockup-name">{m.name}</span>
                      <span class="mockup-loc">{m.loc}</span>
                    </span>
                    <span class="mockup-ago">{m.ago}</span>
                  </div>
                {/each}
              </div>
            </div>

            <!-- Floating status chips -->
            <div class="mockup-chip chip-safe" aria-label="Family safe">
              <span class="chip-icon" aria-hidden="true">✓</span> All safe
            </div>
            <div class="mockup-chip chip-alert" aria-label="Zara arrived at school">
              <span class="chip-icon" aria-hidden="true">📍</span> Zara arrived
            </div>
          </div>
        </div>

      </div>
    </div>

    <!-- Scroll cue -->
    <div class="scroll-cue" aria-hidden="true">
      <div class="scroll-cue-line"></div>
    </div>
  </section>


  <!-- ═══════ STATS BAR ══════════════════════════════════════════════════ -->
  <section class="stats-bar" use:statsObserve aria-label="Key metrics">
    <div class="landing-container">
      <div class="stats-grid">
        {#each stats as s, i}
          <div class="stat-item">
            <span class="stat-value" aria-live="polite">
              {statDisplays[i].toFixed(s.decimals || 0)}{s.suffix}
            </span>
            <span class="stat-label">{s.label}</span>
          </div>
        {/each}
      </div>
    </div>
  </section>


  <!-- ═══════ FEATURES ════════════════════════════════════════════════════ -->
  <section class="features" aria-labelledby="features-heading">
    <div class="landing-container">

      <div
        class="section-header reveal-block"
        class:is-revealed={revealed.has('features-header')}
        use:reveal={'features-header'}
      >
        <p class="section-eyebrow">Everything you need</p>
        <h2 id="features-heading" class="section-title">
          Built for families.<br>Designed for calm.
        </h2>
      </div>

      <div class="features-grid" role="list">
        {#each features as f, i}
          <div
            class="feature-cell reveal-block"
            class:is-revealed={revealed.has(`feature-${i}`)}
            style="transition-delay: {(i % 3) * 70}ms"
            role="listitem"
            use:reveal={`feature-${i}`}
          >
            <Card variant="glass" glow={f.glow} hover padding="lg">
              <div class="feature-icon-wrap" aria-hidden="true">
                {@html f.icon}
              </div>
              <h3 class="feature-title">{f.title}</h3>
              <p class="feature-desc">{f.desc}</p>
            </Card>
          </div>
        {/each}
      </div>

    </div>
  </section>


  <!-- ═══════ HOW IT WORKS ════════════════════════════════════════════════ -->
  <section class="how-it-works" aria-labelledby="how-heading">
    <div class="landing-container">

      <div
        class="section-header reveal-block"
        class:is-revealed={revealed.has('how-header')}
        use:reveal={'how-header'}
      >
        <p class="section-eyebrow">Simple by design</p>
        <h2 id="how-heading" class="section-title">Up and running in minutes</h2>
      </div>

      <div class="steps-track" role="list">
        {#each steps as s, i}
          <div
            class="step reveal-block"
            class:is-revealed={revealed.has(`step-${i}`)}
            style="transition-delay: {i * 110}ms"
            role="listitem"
            use:reveal={`step-${i}`}
          >
            <div class="step-inner">
              <div class="step-num" aria-hidden="true">{s.num}</div>
              {#if i < steps.length - 1}
                <div class="step-connector" aria-hidden="true"></div>
              {/if}
              <div class="step-body">
                <h3 class="step-title">{s.title}</h3>
                <p class="step-desc">{s.desc}</p>
              </div>
            </div>
          </div>
        {/each}
      </div>

    </div>
  </section>


  <!-- ═══════ INTERACTIVE DEMO ═════════════════════════════════════════════ -->
  <section class="demo-section" aria-labelledby="demo-heading">
    <div class="landing-container">

      <div
        class="section-header reveal-block"
        class:is-revealed={revealed.has('demo-header')}
        use:reveal={'demo-header'}
      >
        <p class="section-eyebrow">See it live</p>
        <h2 id="demo-heading" class="section-title">The app, right here</h2>
      </div>

      <div
        class="demo-frame reveal-block"
        class:is-revealed={revealed.has('demo-frame')}
        use:reveal={'demo-frame'}
        aria-label="Interactive app preview"
      >
        <div>

            <!-- Tab row -->
            <div class="demo-tabs" role="tablist" aria-label="Demo views">
              {#each demoTabs as tab}
                <button
                  class="demo-tab"
                  class:active={demoTab === tab}
                  role="tab"
                  aria-selected={demoTab === tab}
                  on:click={() => demoTab = tab}
                >
                  {tab.charAt(0).toUpperCase() + tab.slice(1)}
                </button>
              {/each}
            </div>

            <!-- Demo content -->
            <div class="demo-content" role="tabpanel">
              {#if demoTab === 'map'}
                <div class="demo-map" in:fade={{ duration: 280 }} aria-label="Map view showing family locations">
                  <div class="demo-map-bg" aria-hidden="true"></div>
                  <!-- Animated pins -->
                  {#each [
                    { label:'Mom',  x:28, y:42, c:'#14b8a6' },
                    { label:'Dad',  x:62, y:30, c:'#a855f7' },
                    { label:'Zara', x:48, y:68, c:'#f59e0b' },
                  ] as p}
                    <div
                      class="demo-pin"
                      style="left:{p.x}%; top:{p.y}%; --pin-c:{p.c}"
                      aria-label="{p.label}'s location"
                    >
                      <div class="demo-pin-inner">
                        <div class="demo-pin-dot"></div>
                        <div class="demo-pin-ripple"></div>
                      </div>
                      <div class="demo-pin-label">{p.label}</div>
                    </div>
                  {/each}
                  <!-- Route line -->
                  <svg class="demo-route" aria-hidden="true" viewBox="0 0 100 100" preserveAspectRatio="none">
                    <polyline
                      points="28,42 35,48 42,55 48,68"
                      fill="none"
                      stroke="#f59e0b"
                      stroke-width="0.8"
                      stroke-dasharray="4 3"
                      opacity="0.5"
                    />
                  </svg>
                </div>

              {:else if demoTab === 'alerts'}
                <div class="demo-alerts" in:fade={{ duration: 280 }} role="log" aria-label="Recent alerts">
                  {#each [
                    { icon:'📍', text:'Zara arrived at School', time:'9:02 AM', color:'var(--success-500)' },
                    { icon:'🏠', text:'Dad left Home',          time:'8:14 AM', color:'var(--primary-400)' },
                    { icon:'⚡', text:'Low battery — Mom (11%)', time:'7:58 AM', color:'var(--warning-500)' },
                    { icon:'✓',  text:'All family members safe', time:'7:30 AM', color:'var(--success-500)' },
                  ] as alert, i}
                    <div
                      class="demo-alert-row"
                      style="animation-delay:{i*60}ms"
                      in:fly={{ x: -16, duration: 300, delay: i * 60, easing: cubicOut }}
                    >
                      <span class="alert-icon" aria-hidden="true">{alert.icon}</span>
                      <span class="alert-text">{alert.text}</span>
                      <span class="alert-time">{alert.time}</span>
                    </div>
                  {/each}
                </div>

              {:else if demoTab === 'chat'}
                <div class="demo-chat" in:fade={{ duration: 280 }} aria-label="Secret chat preview">
                  {#each [
                    { from:'Mom',  msg:'Almost home, 10 mins',    side:'left',  c:'#14b8a6' },
                    { from:'You',  msg:'Ok! 🙌 Dinner is ready',   side:'right', c:'#a855f7' },
                    { from:'Zara', msg:'Can I sleep at Emma\'s?',  side:'left',  c:'#f59e0b' },
                    { from:'Mom',  msg:'Ask Dad 😄',               side:'left',  c:'#14b8a6' },
                  ] as m, i}
                    <div
                      class="demo-msg demo-msg-{m.side}"
                      in:fly={{
                        x: m.side === 'left' ? -12 : 12,
                        duration: 320,
                        delay: i * 80,
                        easing: cubicOut
                      }}
                    >
                      {#if m.side === 'left'}
                        <span class="demo-avatar" style="background:{m.c}" aria-label="{m.from}">{m.from[0]}</span>
                      {/if}
                      <span class="demo-bubble demo-bubble-{m.side}">{m.msg}</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>

        </div>
      </div>

    </div>
  </section>


  <!-- ═══════ CTA ════════════════════════════════════════════════════════ -->
  <section class="cta-section" aria-labelledby="cta-heading">
    <div class="cta-bg" aria-hidden="true">
      <div class="cta-orb cta-orb-1"></div>
      <div class="cta-orb cta-orb-2"></div>
    </div>

    <div
      class="landing-container cta-inner reveal-block"
      class:is-revealed={revealed.has('cta')}
      use:reveal={'cta'}
    >
        <h2 id="cta-heading" class="cta-title">Start protecting your family today.</h2>
        <p class="cta-sub">Free forever for families under 6 members. No credit card required.</p>

        {#if !ctaSubmitted}
          <form
            class="cta-form"
            on:submit|preventDefault={onCtaSubmit}
            aria-label="Sign up form"
          >
            <div class="cta-input-wrap">
              <Input
                label="Your email"
                type="email"
                bind:value={ctaEmail}
                size="lg"
              />
            </div>
            <Button variant="primary" size="lg" type="submit">
              Get started free
            </Button>
          </form>
        {:else}
          <div
            class="cta-success"
            in:scale={{ duration: 400, start: 0.88, easing: elasticOut }}
            role="status"
            aria-live="polite"
          >
            <span class="cta-success-icon" aria-hidden="true">✓</span>
            <span>You're in! Check your inbox to continue.</span>
          </div>
        {/if}

        <div class="cta-trust">
          {#each ['End-to-end encrypted', 'No ads, ever', 'GDPR compliant'] as t}
            <span class="trust-item">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                <circle cx="6" cy="6" r="5.5" stroke="currentColor" stroke-opacity="0.5"/>
                <polyline points="3.5,6 5,7.5 8.5,4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              {t}
            </span>
          {/each}
        </div>
      </div>
  </section>

</div>

<style>
  /* ── Layout ─────────────────────────────────────────────────────────── */
  .landing {
    min-height: 100vh;
    background: var(--surface-0, #0d0b14);
    color: var(--text-primary);
    font-family: var(--font-sans);
    overflow-x: hidden;
  }

  .landing-container {
    max-width: 1120px;
    margin: 0 auto;
    padding: 0 var(--space-6);
  }

  @media (max-width: 767px) {
    .landing-container { padding: 0 var(--space-4); }
  }

  /* ── Hero ────────────────────────────────────────────────────────────── */
  .hero {
    position: relative;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: var(--space-10) 0 var(--space-16);
    overflow: hidden;
  }

  /* Background atmosphere */
  .hero-bg { position: absolute; inset: 0; pointer-events: none; }

  .hero-orb {
    position: absolute;
    border-radius: 50%;
    filter: blur(70px);
    will-change: transform;
  }
  .hero-orb-1 {
    width: 600px; height: 600px;
    background: radial-gradient(circle, rgba(20,184,166,0.16) 0%, transparent 65%);
    top: -10%; left: -10%;
    animation: orb-drift-a 24s ease-in-out infinite;
  }
  .hero-orb-2 {
    width: 500px; height: 500px;
    background: radial-gradient(circle, rgba(168,85,247,0.12) 0%, transparent 65%);
    top: 20%; right: -8%;
    animation: orb-drift-b 30s ease-in-out infinite;
  }
  .hero-orb-3 {
    width: 400px; height: 400px;
    background: radial-gradient(circle, rgba(236,72,153,0.08) 0%, transparent 65%);
    bottom: 0; left: 40%;
    animation: orb-drift-a 20s ease-in-out infinite reverse;
  }

  @keyframes orb-drift-a {
    0%,100% { transform: translate(0,0) scale(1); }
    33%     { transform: translate(40px,60px) scale(1.08); }
    66%     { transform: translate(-30px,30px) scale(0.95); }
  }
  @keyframes orb-drift-b {
    0%,100% { transform: translate(0,0) scale(1); }
    40%     { transform: translate(-50px,40px) scale(1.06); }
    70%     { transform: translate(20px,-30px) scale(0.97); }
  }

  .hero-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(0deg, rgba(168,85,247,0.03) 1px, transparent 1px),
      linear-gradient(90deg, rgba(168,85,247,0.03) 1px, transparent 1px);
    background-size: 48px 48px;
    mask-image: radial-gradient(ellipse 70% 70% at 50% 50%, black 30%, transparent 100%);
    -webkit-mask-image: radial-gradient(ellipse 70% 70% at 50% 50%, black 30%, transparent 100%);
  }

  .hero-particle {
    position: absolute;
    width: 2px; height: 2px;
    border-radius: 50%;
    background: rgba(20,184,166,0.6);
    box-shadow: 0 0 6px rgba(20,184,166,0.5);
    animation: particle-float linear infinite;
    will-change: transform, opacity;
  }

  @keyframes particle-float {
    0%  { transform: translateY(0); opacity: 1; }
    50% { transform: translateY(-20px); opacity:0.4; }
    100%{ transform: translateY(0);  opacity:1; }
  }

  /* Hero layout */
  .hero-layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-16);
    align-items: center;
  }

  @media (max-width: 900px) {
    .hero-layout {
      grid-template-columns: 1fr;
      gap: var(--space-10);
    }
    .hero-visual { order: -1; }
  }

  /* Hero copy */
  .hero-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    background: rgba(20,184,166,0.12);
    border: 1px solid rgba(20,184,166,0.28);
    border-radius: var(--radius-full, 9999px);
    padding: 5px 14px;
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--primary-400);
    letter-spacing: 0.04em;
    text-transform: uppercase;
    margin-bottom: var(--space-5);
    box-shadow: 0 0 16px rgba(20,184,166,0.12);
  }

  .hero-badge-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--primary-400);
    box-shadow: 0 0 8px var(--primary-400);
    animation: badge-pulse 2s ease-in-out infinite;
  }

  @keyframes badge-pulse {
    0%,100% { opacity:1; transform:scale(1); }
    50%     { opacity:0.5; transform:scale(0.8); }
  }

  .hero-headline {
    font-family: var(--font-display);
    font-size: clamp(2.25rem, 5vw, 3.5rem);
    font-weight: 800;
    line-height: 1.10;
    letter-spacing: -0.03em;
    color: var(--text-primary);
    margin-bottom: var(--space-5);
  }

  .hero-headline-accent {
    background: linear-gradient(135deg, var(--primary-300) 0%, var(--primary-500) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .hero-headline-sub {
    display: block;
    font-size: 0.82em;
    color: var(--text-secondary);
    font-weight: 600;
    -webkit-text-fill-color: initial;
    letter-spacing: -0.01em;
    margin-top: var(--space-1);
  }

  .hero-tagline {
    font-size: var(--text-lg);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
    max-width: 460px;
    margin-bottom: var(--space-8);
  }

  .hero-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
    margin-bottom: var(--space-6);
  }

  .hero-social-proof {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-size: var(--text-sm);
    color: var(--text-tertiary);
  }

  .hero-avatars {
    display: flex;
  }
  .hero-avatar {
    width: 28px; height: 28px;
    border-radius: 50%;
    border: 2px solid var(--surface-0, #0d0b14);
    margin-right: -8px;
    flex-shrink: 0;
  }
  .hero-avatars { margin-right: var(--space-2); }

  /* Hero visual / mockup */
  .hero-visual {
    display: flex;
    justify-content: center;
    align-items: center;
  }

  .hero-card-wrap {
    position: relative;
    will-change: transform;
    transform-style: preserve-3d;
  }

  .mockup-card {
    width: 300px;
    background: rgba(13, 11, 20, 0.88);
    border: 1px solid rgba(168,85,247,0.22);
    border-top-color: rgba(200,120,255,0.40);
    border-radius: 20px;
    overflow: hidden;
    box-shadow:
      0 32px 80px rgba(0,0,0,0.60),
      0 8px 24px rgba(0,0,0,0.40),
      0 0 0 1px rgba(168,85,247,0.10),
      inset 0 1px 0 rgba(255,255,255,0.08);
    backdrop-filter: blur(20px);
  }

  @media (max-width: 480px) {
    .mockup-card { width: 260px; }
  }

  .mockup-topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid rgba(168,85,247,0.12);
  }

  .mockup-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }

  .mockup-ping {
    /* Larger hit area so the ring animation has room to expand without
       being clipped by the parent card's overflow:hidden */
    position: relative;
    width: 20px; height: 20px;
    flex-shrink: 0;
  }
  .ping-dot {
    position: absolute;
    /* Center 6px dot in the 20px container */
    top: 50%; left: 50%;
    width: 6px; height: 6px;
    transform: translate(-50%, -50%);
    border-radius: 50%;
    background: var(--success-500);
  }
  .ping-ring {
    position: absolute;
    /* Start centered and same size as dot */
    top: 50%; left: 50%;
    width: 6px; height: 6px;
    margin: -3px 0 0 -3px;
    border-radius: 50%;
    border: 1.5px solid var(--success-500);
    /* Scale up to fill the 20px container (max ×2.4 = 14.4px — fits within 20px) */
    animation: ping-expand 1.8s ease-out infinite;
  }

  @keyframes ping-expand {
    0%   { transform: scale(1);   opacity: 0.8; }
    100% { transform: scale(2.2); opacity: 0; }
  }

  /* Mini map */
  .mockup-map {
    position: relative;
    height: 140px;
    overflow: hidden;
    background:
      linear-gradient(135deg, rgba(20,184,166,0.06) 0%, rgba(168,85,247,0.06) 100%),
      rgba(8,8,16,0.60);
  }

  .mockup-map-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(0deg, rgba(255,255,255,0.03) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px);
    background-size: 24px 24px;
  }

  .map-geofence {
    position: absolute;
    width: 70px; height: 70px;
    border-radius: 50%;
    border: 1.5px dashed rgba(20,184,166,0.35);
    background: rgba(20,184,166,0.05);
    top: 30%; left: 25%;
    transform: translate(-50%,-50%);
  }

  .map-pin {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }
  .pin-1 { top: 35%; left: 32%; animation: pin-float 3.2s ease-in-out infinite; }
  .pin-2 { top: 20%; left: 62%; animation: pin-float 3.8s ease-in-out 0.5s infinite; }
  .pin-3 { top: 58%; left: 55%; animation: pin-float 2.9s ease-in-out 1.1s infinite; }

  @keyframes pin-float {
    0%,100% { transform: translateY(0); }
    50%     { transform: translateY(-4px); }
  }

  .pin-dot {
    width: 10px; height: 10px;
    border-radius: 50%;
    z-index: 2;
    position: relative; /* stacking above the ring */
    box-shadow: 0 0 8px currentColor;
  }
  .pin-ring {
    position: absolute;
    top: 0; left: 0;
    width: 10px; height: 10px;
    border-radius: 50%;
    border: 1.5px solid;
    /* Expand from center of the 10×10 element, not from top-left corner */
    transform-origin: 5px 5px;
    animation: pin-expand 2s ease-out infinite;
  }
  @keyframes pin-expand {
    0%   { transform: scale(1);   opacity: 0.8; }
    100% { transform: scale(2.4); opacity: 0; }
  }
  .pin-label {
    font-size: 9px;
    font-weight: 700;
    color: rgba(255,255,255,0.75);
    background: rgba(0,0,0,0.55);
    padding: 1px 5px;
    border-radius: 3px;
    white-space: nowrap;
    margin-top: 2px;
  }

  /* Member rows */
  .mockup-list { padding: var(--space-2) 0; }

  .mockup-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    animation: row-appear 400ms var(--ease-out, cubic-bezier(0.16,1,0.3,1)) both;
  }

  @keyframes row-appear {
    from { opacity:0; transform:translateX(-8px); }
    to   { opacity:1; transform:translateX(0); }
  }

  .mockup-avatar {
    width: 28px; height: 28px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .mockup-info { flex:1; min-width:0; }
  .mockup-name { display:block; font-size:12px; font-weight:700; color:var(--text-primary); }
  .mockup-loc  { display:block; font-size:10px; color:var(--text-tertiary); }
  .mockup-ago  { font-size:10px; color:var(--text-tertiary); flex-shrink:0; }

  /* Floating chips */
  .mockup-chip {
    position: absolute;
    display: flex;
    align-items: center;
    gap: var(--space-1);
    font-size: 11px;
    font-weight: 700;
    padding: 5px 12px;
    border-radius: var(--radius-full, 9999px);
    white-space: nowrap;
    pointer-events: none;
    box-shadow: 0 4px 16px rgba(0,0,0,0.35);
  }

  .chip-safe {
    background: rgba(16,185,129,0.18);
    border: 1px solid rgba(16,185,129,0.40);
    color: var(--success-400);
    /* Float above the card's top-right corner — inset enough to stay within hero overflow:hidden */
    top: -18px; right: -10px;
    animation: chip-bob 3.5s ease-in-out infinite;
  }

  .chip-alert {
    background: rgba(20,184,166,0.15);
    border: 1px solid rgba(20,184,166,0.38);
    color: var(--primary-400);
    /* Float below the card's bottom-left corner — don't overlap list rows */
    bottom: -20px; left: -10px;
    animation: chip-bob 4s ease-in-out 1s infinite;
  }

  /* Tighten chip offsets on small screens to prevent viewport overflow */
  @media (max-width: 480px) {
    .chip-safe  { right: -4px; top: -16px; }
    .chip-alert { left: -4px;  bottom: -18px; }
  }

  .chip-icon { font-size: 12px; }

  @keyframes chip-bob {
    0%,100% { transform: translateY(0); }
    50%     { transform: translateY(-5px); }
  }

  /* Scroll cue */
  .scroll-cue {
    position: absolute;
    bottom: var(--space-8);
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
  }
  .scroll-cue-line {
    width: 1px; height: 40px;
    background: linear-gradient(180deg, rgba(20,184,166,0.6) 0%, transparent 100%);
    animation: scroll-line-drop 2s ease-in-out infinite;
  }
  @keyframes scroll-line-drop {
    0%,100% { transform: scaleY(0); transform-origin: top; opacity:0; }
    40%     { transform: scaleY(1); transform-origin: top; opacity:1; }
    80%     { transform: scaleY(1); transform-origin: bottom; opacity:1; }
    100%    { transform: scaleY(0); transform-origin: bottom; opacity:0; }
  }


  /* ── Stats bar ────────────────────────────────────────────────────────── */
  .stats-bar {
    padding: var(--space-10) 0;
    border-top: 1px solid var(--border-subtle);
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-1, rgba(168,85,247,0.04));
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0;
  }

  @media (max-width: 640px) {
    .stats-grid { grid-template-columns: repeat(2, 1fr); }
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-5) var(--space-4);
    border-right: 1px solid var(--border-default);
  }
  .stat-item:last-child { border-right: none; }

  @media (max-width: 640px) {
    .stat-item { padding: var(--space-4) var(--space-3); }
    /* 2-col grid: right column has no border-right, bottom row has no border-bottom */
    .stat-item:nth-child(2n) { border-right: none; }
    .stat-item:nth-child(1),
    .stat-item:nth-child(2) { border-bottom: 1px solid var(--border-default); }
  }

  .stat-value {
    font-family: var(--font-display);
    font-size: var(--text-3xl);
    font-weight: 800;
    letter-spacing: -0.03em;
    background: linear-gradient(135deg, var(--primary-300) 0%, var(--primary-500) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    min-width: 5ch;
    text-align: center;
    tabular-nums: auto;
    font-variant-numeric: tabular-nums;
  }

  .stat-label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    text-align: center;
  }


  /* ── Section shared ───────────────────────────────────────────────────── */
  .section-header {
    text-align: center;
    margin-bottom: var(--space-12, 48px);
  }

  .section-eyebrow {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.10em;
    text-transform: uppercase;
    color: var(--primary-400);
    margin-bottom: var(--space-3);
  }

  .section-title {
    font-family: var(--font-display);
    font-size: clamp(1.75rem, 3.5vw, 2.5rem);
    font-weight: 800;
    line-height: 1.15;
    letter-spacing: -0.025em;
    color: var(--text-primary);
  }


  /* ── Features ─────────────────────────────────────────────────────────── */
  .features {
    padding: var(--space-16) 0 var(--space-12);
  }

  .features-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-4);
  }

  @media (max-width: 900px) {
    .features-grid { grid-template-columns: repeat(2, 1fr); }
  }
  @media (max-width: 580px) {
    .features-grid { grid-template-columns: 1fr; }
  }

  .feature-cell { min-height: 160px; }

  .feature-icon-wrap {
    width: 44px; height: 44px;
    border-radius: var(--radius-lg, 10px);
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, rgba(20,184,166,0.18) 0%, rgba(13,148,136,0.10) 100%);
    border: 1px solid rgba(20,184,166,0.24);
    color: var(--primary-400);
    margin-bottom: var(--space-4);
    box-shadow: 0 4px 16px rgba(20,184,166,0.12), inset 0 1px 0 rgba(255,255,255,0.12);
  }

  .feature-title {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    margin-bottom: var(--space-2);
  }

  .feature-desc {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
  }


  /* ── How it works ─────────────────────────────────────────────────────── */
  .how-it-works { padding: var(--space-12) 0; }

  .steps-track {
    display: flex;
    gap: var(--space-6);
    align-items: flex-start;
  }

  @media (max-width: 700px) {
    .steps-track { flex-direction: column; }
  }

  .step { flex: 1; }

  .step-inner {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .step-num {
    font-family: var(--font-display);
    font-size: var(--text-4xl);
    font-weight: 900;
    letter-spacing: -0.05em;
    background: linear-gradient(135deg, var(--primary-400) 0%, var(--primary-700) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    line-height: 1;
  }

  .step-connector {
    position: absolute;
    top: 24px;
    right: calc(-1 * var(--space-6) / 2);
    width: calc(100% + var(--space-6));
    height: 1px;
    background: linear-gradient(90deg, var(--primary-700) 0%, var(--border-default) 100%);
    pointer-events: none;
  }

  @media (max-width: 700px) {
    .step-connector { display: none; }
  }

  .step-title {
    font-family: var(--font-display);
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.01em;
  }

  .step-desc {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
  }


  /* ── Demo section ─────────────────────────────────────────────────────── */
  .demo-section { padding: var(--space-12) 0 var(--space-16); }

  .demo-frame {
    max-width: 640px;
    margin: 0 auto;
    background: var(--surface-1, rgba(13,11,20,0.70));
    border: 1px solid var(--border-default);
    border-top-color: rgba(255,255,255,0.10);
    border-radius: 20px;
    overflow: hidden;
    box-shadow:
      0 24px 64px rgba(0,0,0,0.40),
      0 4px 16px rgba(0,0,0,0.24),
      inset 0 1px 0 rgba(255,255,255,0.07);
  }

  /* Demo tabs */
  .demo-tabs {
    display: flex;
    gap: var(--space-1);
    padding: var(--space-3) var(--space-3) 0;
    border-bottom: 1px solid var(--border-subtle);
  }

  .demo-tab {
    padding: var(--space-2) var(--space-4);
    border: none;
    background: transparent;
    color: var(--text-tertiary);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 600;
    cursor: pointer;
    border-radius: var(--radius-md) var(--radius-md) 0 0;
    border-bottom: 2px solid transparent;
    transition:
      color 180ms var(--ease-out),
      border-color 180ms var(--ease-out),
      background 180ms;
    min-height: 36px;
  }

  .demo-tab:hover { color: var(--text-primary); background: var(--surface-hover, rgba(255,255,255,0.04)); }
  .demo-tab.active {
    color: var(--primary-400);
    border-bottom-color: var(--primary-400);
    background: rgba(20,184,166,0.05);
  }

  .demo-content { min-height: 260px; }

  /* Demo map */
  .demo-map {
    position: relative;
    height: 260px;
    overflow: hidden;
    background: rgba(8,8,20,0.70);
  }
  .demo-map-bg {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(0deg, rgba(255,255,255,0.025) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255,255,255,0.025) 1px, transparent 1px);
    background-size: 32px 32px;
  }

  .demo-pin {
    position: absolute;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0;
  }
  /* Shared 12×12 stacking context — dot and ripple overlap exactly */
  .demo-pin-inner {
    position: relative;
    width: 12px;
    height: 12px;
    flex-shrink: 0;
  }
  .demo-pin-dot {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: var(--pin-c);
    box-shadow: 0 0 10px var(--pin-c);
    z-index: 2;
  }
  .demo-pin-ripple {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    border: 1.5px solid var(--pin-c);
    /* scale from center — transform-origin defaults to 50% 50% */
    animation: demo-pin-expand 2.2s ease-out infinite;
    z-index: 1;
  }
  @keyframes demo-pin-expand {
    0%   { transform: scale(1);   opacity: 0.75; }
    100% { transform: scale(3.2); opacity: 0; }
  }
  .demo-pin-label {
    font-size: 10px;
    font-weight: 700;
    color: rgba(255,255,255,0.9);
    background: rgba(0,0,0,0.65);
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
    margin-top: 5px;
  }

  .demo-route {
    position: absolute;
    inset: 0;
    width: 100%; height: 100%;
  }

  /* Demo alerts */
  .demo-alerts { padding: var(--space-4); display: flex; flex-direction: column; gap: var(--space-2); }

  .demo-alert-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    background: var(--surface-2, rgba(15,15,30,0.60));
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg, 10px);
    animation: row-appear 380ms var(--ease-out) both;
  }

  .alert-icon { font-size: 16px; flex-shrink:0; }
  .alert-text { flex:1; font-size: var(--text-sm); color: var(--text-primary); min-width:0; }
  .alert-time { font-size: var(--text-xs); color: var(--text-tertiary); flex-shrink:0; }

  /* Demo chat */
  .demo-chat {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .demo-msg {
    display: flex;
    align-items: flex-end;
    gap: var(--space-2);
  }
  .demo-msg-right { flex-direction: row-reverse; }

  .demo-avatar {
    width: 24px; height: 24px;
    border-radius: 50%;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    color: #fff;
  }

  .demo-bubble {
    max-width: 72%;
    padding: var(--space-2) var(--space-3);
    border-radius: 14px;
    font-size: var(--text-sm);
    line-height: var(--leading-normal);
  }
  .demo-bubble-left {
    background: var(--surface-2, rgba(255,255,255,0.08));
    border: 1px solid var(--border-subtle);
    color: var(--text-primary);
    border-bottom-left-radius: 4px;
  }
  .demo-bubble-right {
    background: linear-gradient(135deg, var(--primary-500) 0%, var(--primary-700) 100%);
    color: #fff;
    border-bottom-right-radius: 4px;
    box-shadow: 0 4px 14px rgba(20,184,166,0.30);
  }


  /* ── CTA section ──────────────────────────────────────────────────────── */
  .cta-section {
    position: relative;
    padding: var(--space-16) 0 var(--space-10);
    overflow: hidden;
    border-top: 1px solid var(--border-subtle);
  }

  .cta-bg { position: absolute; inset: 0; pointer-events: none; }

  .cta-orb {
    position: absolute;
    border-radius: 50%;
    filter: blur(80px);
  }
  .cta-orb-1 {
    width: 400px; height: 400px;
    background: radial-gradient(circle, rgba(20,184,166,0.14) 0%, transparent 65%);
    top: -10%; left: 20%;
  }
  .cta-orb-2 {
    width: 300px; height: 300px;
    background: radial-gradient(circle, rgba(168,85,247,0.10) 0%, transparent 65%);
    bottom: -5%; right: 15%;
  }

  .cta-inner {
    position: relative;
    z-index: 1;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-5);
  }

  .cta-title {
    font-family: var(--font-display);
    font-size: clamp(1.75rem, 3.5vw, 2.5rem);
    font-weight: 800;
    letter-spacing: -0.025em;
    color: var(--text-primary);
    max-width: 600px;
  }

  .cta-sub {
    font-size: var(--text-base);
    color: var(--text-secondary);
    max-width: 420px;
    line-height: var(--leading-relaxed);
    margin-top: calc(-1 * var(--space-2));
  }

  .cta-form {
    display: flex;
    gap: var(--space-3);
    width: 100%;
    max-width: 460px;
    align-items: flex-end;
  }

  .cta-input-wrap { flex: 1; }

  @media (max-width: 560px) {
    .cta-form { flex-direction: column; }
    .cta-input-wrap { width: 100%; }
  }

  .cta-success {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    background: rgba(16,185,129,0.12);
    border: 1px solid rgba(16,185,129,0.32);
    border-radius: var(--radius-xl, 14px);
    padding: var(--space-4) var(--space-6);
    color: var(--success-400);
    font-weight: 700;
    font-size: var(--text-base);
  }

  .cta-success-icon {
    width: 28px; height: 28px;
    border-radius: 50%;
    background: rgba(16,185,129,0.20);
    border: 1px solid rgba(16,185,129,0.40);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    flex-shrink: 0;
  }

  .cta-trust {
    display: flex;
    gap: var(--space-5);
    flex-wrap: wrap;
    justify-content: center;
    margin-top: var(--space-2);
  }

  .trust-item {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-weight: 500;
  }

  /* ── Scroll reveal ────────────────────────────────────────────────────── */
  .reveal-block {
    opacity: 0;
    transform: translateY(24px);
    transition:
      opacity 0.55s cubic-bezier(0.4, 0, 0.2, 1),
      transform 0.55s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .reveal-block.is-revealed {
    opacity: 1;
    transform: translateY(0);
  }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .hero-orb, .cta-orb,
    .hero-particle,
    .hero-badge-dot,
    .chip-safe, .chip-alert,
    .ping-ring, .pin-ring, .pin-1, .pin-2, .pin-3,
    .demo-pin-ripple,
    .scroll-cue-line { animation: none; opacity: 1; transform: scale(1); }

    .hero-card-wrap { transform: none !important; }
    .demo-route { display: none; }
    .reveal-block { opacity: 1; transform: none; transition: none; }
  }
</style>
