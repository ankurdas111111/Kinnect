<script>
  import { preventDefault } from 'svelte/legacy';

  /**
   * Landing — "A day with your family, told by scroll."
   *
   * One scrollytelling spine replaces the old Features → How-it-works →
   * tabbed-Demo middle: the hero's 3D-tilt phone mockup IS the story device.
   * On desktop it docks into a sticky right column while four story beats
   * (morning dispersal → geofence arrival → SOS-that-resolves-safe →
   * evening all-home) scroll past on the left. On mobile the hero mockup
   * collapses into a compact sticky card — the only phone chrome on the page.
   *
   * Mechanics:
   *   - IntersectionObserver drives beat/scene classes (PRIMARY path — works
   *     on every WebView; see StoryBeat.svelte).
   *   - CSS scroll-driven animation (animation-timeline) is progressive
   *     enhancement only, behind @supports (reveal-scroll from global.css).
   *   - position: sticky pins the mockup; all scene changes animate
   *     transform/opacity only.
   *   - SOS beat resolves to "safe" within the beat — the page never rests
   *     on an alarming frame.
   */
  import { onMount, onDestroy } from 'svelte';
  import { scale } from 'svelte/transition';
  import { elasticOut } from 'svelte/easing';
  import { spring } from 'svelte/motion';
  import Button from '../components/primitives/Button.svelte';
  import Input  from '../components/primitives/Input.svelte';
  import MagneticButton from '../components/primitives/MagneticButton.svelte';
  import Constellation from '../components/primitives/Constellation.svelte';
  import { DEFAULT_NODES, DEFAULT_LINKS } from '../components/primitives/constellationGeometry.js';
  import StoryBeat from '../components/landing/StoryBeat.svelte';
  import LandingNav from '../components/landing/LandingNav.svelte';
  import LandingFooter from '../components/landing/LandingFooter.svelte';
  import {
    STAGE_W, STAGE_H, PLACES, PINS, ROUTES, BEATS, SCENES, SOS_CHIPS,
  } from '../components/landing/landingStory.js';
  import { allowMotion, allowWebGL } from '../lib/stores/effects.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';
  import { daypartFor } from '../lib/daypart.js';
  import { Capacitor } from '@capacitor/core';

  // ── Routing ──────────────────────────────────────────────────────────────
  import { navigate } from '../lib/viewTransition.js';

  // ── Hero night-sky constellation field (VIGIL) ───────────────────────────
  // Member wheel tokens on the shared primitive — kills the old teal-era
  // hardcoded member hexes. Center beacon stays brand (self ≠ member hue).
  const FIELD_NODES = DEFAULT_NODES.map((n, i) => ({
    ...n,
    hue: i === 0 ? 'var(--primary-400)' : `var(--member-${i})`,
  }));

  // ── Hero tilt — true spring physics (svelte/motion Spring; gated on
  //    allowMotion + OS switch). Low damping gives the card a gentle
  //    settle-overshoot on pointer leave; the glare highlight rides the
  //    same spring so light and geometry always agree. ──────────────────────
  let heroCardEl = $state();
  // spring() (not the Spring class): Button already ships this exact module in
  // the index chunk, so the physics costs zero additional bundle bytes.
  const tilt = spring({ x: 0, y: 0 }, { stiffness: 0.08, damping: 0.34, precision: 0.001 });

  function onHeroMouseMove(e) {
    // JS-driven flourish: must honor the OS reduce-motion switch even when a
    // stored 'full' fx preference exists ($allowMotion alone is not enough).
    if (!heroCardEl || !$allowMotion || prefersReducedMotion()) return;
    const r  = heroCardEl.getBoundingClientRect();
    const dx = (e.clientX - r.left - r.width  / 2) / (r.width  / 2);
    const dy = (e.clientY - r.top  - r.height / 2) / (r.height / 2);
    tilt.set({ x: -dy * 6, y: dx * 6 });
  }

  function onHeroMouseLeave() {
    tilt.set({ x: 0, y: 0 });
  }

  // ── Daypart badge — the watch is live right now, in the visitor's own
  //    morning or midnight (VIGIL: "keeping the watch"). Honest and personal
  //    where a static claim would be neither. ───────────────────────────────
  const DAYPART_BADGE = {
    dawn:  'Dawn watch is on',
    day:   'Day watch is on',
    dusk:  'Dusk watch is on',
    night: 'Night watch is on',
  };
  const badgeText = DAYPART_BADGE[daypartFor(new Date().getHours())] || DAYPART_BADGE.day;

  // ── 3D hero constellation (desktop-web-full only; idle-loaded post-LCP) ────
  // The static SVG hero above is the LCP element and the PERMANENT poster; the
  // WebGL field fades in over it and disposes on any gate flip / route leave.
  // Every gate is re-checked live (a stored 'full' pref must still honor the OS
  // reduce-motion switch and a resize below 1024px).
  let heroSectionEl = $state();
  let constellationCanvasEl = $state();
  let constellationOn = $state(false);   // drives the 400ms opacity crossfade
  let sceneHandle = null;                 // { dispose } from the GL module
  let sceneLoading = false;               // in-flight import guard
  let idleHandle = null;                  // rIC / setTimeout token
  let idleIsTimeout = false;
  let armReady = false;                    // true once the post-LCP idle window
                                           // has fired — only then may a live
                                           // gate flip RE-mount (never pre-LCP)

  function gatesPass() {
    return (
      $allowWebGL &&                                    // fx==='full' && supportsWebGL
      !Capacitor.isNativePlatform() &&                  // never on native shells
      typeof matchMedia === 'function' &&
      matchMedia('(pointer: fine)').matches &&          // desktop pointer
      window.innerWidth >= 1024 &&                      // wide viewport
      !prefersReducedMotion()                           // OS reduce-motion off
    );
  }

  /** Passive scroll progress (0..1) through the hero section — read per-frame
   *  by the GL loop; no scroll listener, no thrash. */
  function heroScroll() {
    if (!heroSectionEl) return 0;
    const vh = window.innerHeight || 1;
    // 0 at hero top, ~1 once the first story beat is fully in view.
    return Math.min(Math.max(-heroSectionEl.getBoundingClientRect().top / vh, 0), 1);
  }

  async function tryMountConstellation() {
    if (sceneHandle || sceneLoading || !constellationCanvasEl) return;
    if (!gatesPass()) return;
    sceneLoading = true;
    try {
      // Never in a static import graph — vite pins this into the async 'three'
      // chunk (and aliases it to capacitor-stub on native builds → mount undefined).
      const mod = await import('../lib/three/heroConstellation.js');
      if (typeof mod?.mount !== 'function') return; // native stub or failed shape
      // Re-check gates AFTER the async gap — user may have flipped a switch.
      if (!gatesPass() || !constellationCanvasEl) return;
      sceneHandle = mod.mount(constellationCanvasEl, {
        getScroll: heroScroll,
        onMounted: () => { constellationOn = true; }, // crossfade in on first frame
      });
    } catch {
      // Failed import / context creation — poster remains, nothing surfaced.
    } finally {
      sceneLoading = false;
    }
  }

  function teardownConstellation() {
    constellationOn = false;
    if (sceneHandle) { sceneHandle.dispose(); sceneHandle = null; }
  }

  /** Schedule the mount attempt after LCP: rIC when available, else a 2500ms
   *  setTimeout shim (Safari — our core audience — has no requestIdleCallback). */
  function scheduleConstellation() {
    if (typeof window === 'undefined' || !gatesPass()) return;
    const fire = () => { armReady = true; tryMountConstellation(); };
    if ('requestIdleCallback' in window) {
      idleIsTimeout = false;
      idleHandle = window.requestIdleCallback(fire, { timeout: 2500 });
    } else {
      idleIsTimeout = true;
      idleHandle = setTimeout(fire, 2500);
    }
  }

  function cancelIdle() {
    if (idleHandle == null) return;
    if (idleIsTimeout) clearTimeout(idleHandle);
    else if ('cancelIdleCallback' in window) window.cancelIdleCallback(idleHandle);
    idleHandle = null;
  }

  onMount(() => {
    scheduleConstellation();
    // Live gate flips: a resize below 1024px or an fx-level change must dispose.
    const onResize = () => { if (sceneHandle && !gatesPass()) teardownConstellation(); };
    window.addEventListener('resize', onResize, { passive: true });
    return () => window.removeEventListener('resize', onResize);
  });

  // Reactively honor fx-level flips (Settings → calm/minimal kills the scene;
  // → full while on the hero re-arms it).
  $effect(() => {
    // reference $allowWebGL so this re-runs when the fx store changes
    const armed = $allowWebGL;
    if (!armed && sceneHandle) {
      teardownConstellation();            // calm/minimal flip → dispose immediately
    } else if (armed && armReady && !sceneHandle && !sceneLoading && constellationCanvasEl) {
      // Re-arm on a live flip back to 'full' — but only AFTER the post-LCP idle
      // window fired, so this never front-runs the LCP-first load sequence.
      tryMountConstellation();
    }
  });

  // ── Story scene state (IO-driven — primary path everywhere) ──────────────
  let activeBeat = $state(0);          // 0 = hero resting frame, 1..4 = beats
  let sosPhase = $state('idle');       // 'idle' | 'alert' | 'safe'
  let sosTimer = null;

  function setBeat(i) {
    if (i === activeBeat) return;
    activeBeat = i;
    clearTimeout(sosTimer);
    if (i === 3) {
      if ($allowMotion && !prefersReducedMotion()) {
        sosPhase = 'alert';
        // SOS must resolve to safe WITHIN the beat — never rest on alarm.
        sosTimer = setTimeout(() => { sosPhase = 'safe'; }, 2400);
      } else {
        sosPhase = 'safe'; // reduced motion: render the resolved frame directly
      }
    } else {
      sosPhase = 'idle';
    }
  }

  let scene = $derived(SCENES[activeBeat]);
  let sosActive = $derived(activeBeat === 3 && sosPhase === 'alert');
  let chip = $derived(
    activeBeat === 3
      ? SOS_CHIPS[sosPhase === 'alert' ? 'alert' : 'safe']
      : scene.chip
  );

  /** Reset to the hero resting frame when the hero copy re-enters view. */
  function heroObserve(node) {
    const io = new IntersectionObserver((entries) => {
      for (const e of entries) if (e.isIntersecting) setBeat(0);
    }, { threshold: 0.4 });
    io.observe(node);
    return { destroy() { io.disconnect(); } };
  }

  // ── Animated counters — product truths only. Invented social proof
  //    ("50,000+ families") is the fastest way to read as template filler;
  //    every number below is a real property of the product. ────────────────
  const stats = [
    { value: 2,  label: 'Second update cadence',  suffix: 's' },
    { value: 30, label: 'Days of route replay',   suffix: '' },
    { value: 6,  label: 'Members free, forever',  suffix: '' },
    { value: 24, label: 'The watch never sleeps', suffix: '/7' },
  ];
  let statDisplays = $state(stats.map(() => 0));
  let statsVisible = false;

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

  // ── Stats IntersectionObserver (counter trigger) ──────────────────────────
  let statsObservers = [];

  function statsObserve(node) {
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
    obs.observe(node);
    statsObservers.push(obs);
    return { destroy() { obs.disconnect(); } };
  }

  // ── Features bento (asymmetric — two featured cells with live visuals,
  //    four compact, one quiet trust cell; equal-card grids read generic) ────
  const FEATURE_GPS = {
    icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>`,
    title: 'Real-time GPS',
    desc: 'Sub-2-second position updates with Kalman-filtered accuracy. Never a stale pin.',
  };

  const FEATURE_SOS = {
    icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
    title: 'One-tap SOS',
    desc: 'Hold to send an emergency signal. Every family pin turns toward you, live — and it always resolves with "marked safe".',
  };

  const FEATURES_COMPACT = [
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8h1a4 4 0 0 1 0 8h-1"/><path d="M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z"/><line x1="6" y1="1" x2="6" y2="4"/><line x1="10" y1="1" x2="10" y2="4"/><line x1="14" y1="1" x2="14" y2="4"/></svg>`,
      title: 'Smart Geofences',
      desc: 'Draw zones around school, home, work. Instant alerts when anyone arrives or leaves.',
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>`,
      title: 'Secret Chat',
      desc: 'End-to-end encrypted family messaging. Messages vanish after read. Zero metadata.',
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 8v4l3 3"/></svg>`,
      title: 'Route History',
      desc: 'Replay anyone\'s route for the last 30 days. Timeline scrubbing, speed heatmaps.',
    },
    {
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      title: 'Guardian Roles',
      desc: 'Granular permissions. Parents see everything. Kids see what you allow. No surprises.',
    },
  ];

  // ── Lifecycle ─────────────────────────────────────────────────────────────
  onDestroy(() => {
    statsObservers.forEach(o => o.disconnect());
    clearTimeout(sosTimer);
    cancelIdle();
    teardownConstellation();   // releases the GL context on route leave
  });

  // ── CTA email ─────────────────────────────────────────────────────────────
  let ctaEmail = $state('');
  let ctaSubmitted = $state(false);

  function onCtaSubmit() {
    if (!ctaEmail) return;
    ctaSubmitted = true;
  }
</script>

<!-- Scroll progress bar — pure CSS via animation-timeline: scroll(root) in global.css -->
<div class="scroll-progress-bar" aria-hidden="true"></div>

<LandingNav />

<div class="landing" aria-label="Kinnect landing page">

  <!-- ═══════ HERO + STORY SPINE ═══════════════════════════════════════════
       One phone mockup: it opens as the hero visual, then stays pinned
       (position: sticky) while the four story beats scroll past. -->
  <section
    class="hero-story"
    aria-labelledby="hero-headline"
    bind:this={heroSectionEl}
    onmousemove={onHeroMouseMove}
    onmouseleave={onHeroMouseLeave}
  >
    <!-- Background atmosphere: VIGIL night-sky field (first viewport only) -->
    <div class="hero-bg fx-ambient" aria-hidden="true">
      <div class="hero-aurora aurora-hero-bg" aria-hidden="true"></div>
      <div class="hero-orb hero-orb-1"></div>
      <div class="hero-orb hero-orb-2"></div>
      <div class="hero-orb hero-orb-3"></div>
      <div class="hero-grid depth-grid-perspective" aria-hidden="true"></div>
      <div class="hero-field">
        <Constellation mode="dormant" nodes={FIELD_NODES} links={DEFAULT_LINKS} />
      </div>
      {#each Array(14) as _, i}
        <div class="hero-particle" style="
          left:{8 + i * 6.5}%;
          top:{15 + (i * 37 % 70)}%;
          animation-delay:{-i * 0.7}s;
          animation-duration:{5 + (i % 4)}s;
          opacity:{0.2 + (i % 3) * 0.15};
        " aria-hidden="true"></div>
      {/each}

      <!-- WebGL depth field: fades in OVER the static SVG poster above (desktop
           web, fx='full' only). Poster stays beneath permanently → no layout
           shift on failed import / lost context / gate flip. -->
      <canvas
        class="hero-constellation-canvas"
        class:is-on={constellationOn}
        bind:this={constellationCanvasEl}
        aria-hidden="true"
      ></canvas>
    </div>

    <div class="landing-container hs-grid">

      <!-- Hero copy (beat 0) — entrance is CSS-choreographed (.le + --le
           stagger): starts on first paint and honors prefers-reduced-motion,
           which svelte transitions do not. -->
      <header class="hero-copy" use:heroObserve>
        <div class="hero-badge le" style="--le:0">
          <span class="hero-badge-dot" aria-hidden="true"></span>
          {badgeText}
        </div>

        <h1 id="hero-headline" class="hero-headline">
          <span class="hl-line"><span class="hl-inner" style="--hl:0">Know your family</span></span>
          <span class="hl-line"><span class="hl-inner hero-headline-accent" style="--hl:1">is safe.</span></span>
          <span class="hl-line hero-headline-sub"><span class="hl-inner" style="--hl:2">Always.</span></span>
        </h1>

        <p class="hero-tagline le" style="--le:4">
          Kinnect gives families real-time GPS, smart geofence alerts, and
          one-tap SOS — wrapped in a design that feels calm, not clinical.
        </p>

        <div class="hero-actions le" style="--le:5">
          <MagneticButton strength={5}>
            <Button variant="primary" size="lg" on:click={() => navigate('/register')}>
              Start for free
              {#snippet icon()}
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <line x1="3" y1="8" x2="13" y2="8"/>
                  <polyline points="9 4 13 8 9 12"/>
                </svg>
              {/snippet}
            </Button>
          </MagneticButton>

          <Button variant="ghost" size="lg" on:click={() => navigate('/login')}>
            Sign in
          </Button>
        </div>

        <p class="hero-social-proof le" style="--le:6">
          <span class="hero-avatars" aria-hidden="true">
            {#each ['var(--member-1)', 'var(--member-2)', 'var(--member-3)', 'var(--member-4)'] as c}
              <span class="hero-avatar" style="background:{c};"></span>
            {/each}
          </span>
          <span>Made for the people on your map — parents, kids, grandparents.</span>
        </p>
      </header>

      <!-- The story device: hero mockup, sticky through all four beats.
           Tilt + glare ride the same Spring, so light follows geometry. -->
      <div class="hs-visual le" style="--le:3">
        <div class="hs-sticky">
          <div
            class="hero-card-wrap"
            bind:this={heroCardEl}
            style:transform={`perspective(1000px) rotateX(${$tilt.x}deg) rotateY(${$tilt.y}deg)`}
            style:--glare-x={`${($tilt.y * 7).toFixed(2)}px`}
            style:--glare-y={`${(-$tilt.x * 7).toFixed(2)}px`}
          >
            <div class="mockup-card noise-surface">
              <!-- Status bar -->
              <div class="mockup-topbar">
                <div class="mockup-title">Family</div>
                <div class="mockup-ping" aria-label="Live updates active">
                  <span class="ping-ring-2" aria-hidden="true"></span>
                  <span class="ping-ring" aria-hidden="true"></span>
                  <span class="ping-dot"  aria-hidden="true"></span>
                </div>
              </div>

              <!-- Story stage: the miniature family map, scene-driven.
                   HARD CAP: ≤6 concurrently animated nodes per scene
                   (3 pins + geofence + SOS ring + chip). Do not add more. -->
              <div class="mockup-map" aria-hidden="true">
                <div class="mockup-map-grid"></div>
                <svg
                  class="story-stage"
                  viewBox="0 0 {STAGE_W} {STAGE_H}"
                  preserveAspectRatio="xMidYMid slice"
                >
                  {#each ROUTES as r (r.id)}
                    <polyline
                      class="route"
                      class:route-on={scene.routes}
                      style="--rhue:{r.hue}"
                      points={r.points}
                    />
                  {/each}

                  {#each PLACES as pl (pl.id)}
                    <g class="place">
                      <circle class="place-dot" cx={pl.x} cy={pl.y} r="2.5" />
                      <text class="place-label" x={pl.x} y={pl.y - 8}>{pl.label}</text>
                    </g>
                  {/each}

                  {#each PLACES.filter(p => p.ring) as pl (pl.id)}
                    <circle
                      class="gf"
                      class:gf-on={scene.geofence?.id === pl.id}
                      class:gf-safe={scene.geofence?.id === pl.id && scene.geofence?.tone === 'safe'}
                      cx={pl.x} cy={pl.y} r="20"
                    />
                  {/each}

                  {#each PINS as pin (pin.id)}
                    <g
                      class="story-pin"
                      class:sp-sos={sosActive && pin.id === 'zara'}
                      style="--hue:{pin.hue}; transform: translate({scene.pos[pin.id][0]}px, {scene.pos[pin.id][1]}px)"
                    >
                      <circle class="sp-sos-ring fx-ambient" r="6" />
                      <circle class="sp-halo" r="8" />
                      <circle class="sp-dot" r="4" />
                      <text class="sp-label" y="17">{pin.label}</text>
                    </g>
                  {/each}
                </svg>
              </div>

              <!-- Member rows — sublines follow the active scene -->
              <div class="mockup-list stagger-messages" aria-hidden="true">
                {#each PINS as pin, i (pin.id)}
                  <div class="mockup-row" style="animation-delay:{i * 80}ms">
                    <span class="mockup-avatar" style="background:{pin.hue}" aria-hidden="true"></span>
                    <span class="mockup-info">
                      <span class="mockup-name">{pin.label}</span>
                      <span class="mockup-loc">{scene.rows[i].loc}</span>
                    </span>
                    <span class="mockup-ago font-tabular">{scene.rows[i].ago}</span>
                  </div>
                {/each}
              </div>

              <!-- Specular glare — translated by the tilt spring (ambient tier) -->
              <div class="mockup-glare fx-ambient" aria-hidden="true"></div>
            </div>

            <!-- Floating scene chip (decorative twin of the beat copy) -->
            {#if chip}
              {#key chip.text}
                <div class="mockup-chip" data-tone={chip.tone} aria-hidden="true">
                  <span class="chip-icon">{chip.icon}</span> {chip.text}
                </div>
              {/key}
            {/if}
          </div>
        </div>
      </div>

      <!-- The four story beats (IO inside StoryBeat drives the scenes) -->
      <ol class="story-beats" aria-label="A day with Kinnect">
        {#each BEATS as beat, i (beat.time)}
          <StoryBeat {beat} index={i} active={activeBeat === i + 1} onactive={setBeat} />
        {/each}
      </ol>

    </div>

    <!-- Scroll cue (first viewport only) -->
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
            <span class="stat-value font-tabular" aria-live="polite">
              {statDisplays[i].toFixed(s.decimals || 0)}{s.suffix}
            </span>
            <span class="stat-label">{s.label}</span>
          </div>
        {/each}
      </div>
    </div>
  </section>


  <!-- ═══════ FEATURES (quick-scan bento, demoted below the story) ════════ -->
  <section class="features" aria-labelledby="features-heading">
    <div class="landing-container">

      <div class="section-header reveal-scroll">
        <p class="section-eyebrow">Everything you need</p>
        <h2 id="features-heading" class="section-title">
          Built for families.<br>Designed for calm.
        </h2>
      </div>

      <!-- Asymmetric bento: two featured cells carry live visuals, four stay
           compact, one quiet cell carries the privacy promise. Cells are
           page-local glass surfaces (same token recipe as .mockup-card). -->
      <div class="features-bento reveal-scroll-grid" role="list">

        <!-- Featured: Real-time GPS — live mini-map with member dots -->
        <div class="bento-cell bento-gps card-hover-depth" role="listitem">
          <div class="bento-gps-copy">
            <div class="feature-icon-wrap" aria-hidden="true">{@html FEATURE_GPS.icon}</div>
            <h3 class="feature-title">{FEATURE_GPS.title}</h3>
            <p class="feature-desc">{FEATURE_GPS.desc}</p>
          </div>
          <div class="bento-gps-visual" aria-hidden="true">
            <div class="bgv-grid"></div>
            <span class="bgv-dot" style="--hue:var(--member-1); top:32%; left:24%"></span>
            <span class="bgv-dot" style="--hue:var(--member-2); top:58%; left:66%"></span>
            <span class="bgv-dot bgv-live" style="--hue:var(--member-3); top:40%; left:48%">
              <span class="bgv-ring fx-ambient"></span>
            </span>
          </div>
        </div>

        <!-- Featured: SOS — hold-ring motif, resolves to safe -->
        <div class="bento-cell bento-sos card-hover-depth" role="listitem">
          <div class="bento-sos-visual" aria-hidden="true">
            <svg viewBox="0 0 96 96" class="sos-ring-svg">
              <circle class="sos-ring-track" cx="48" cy="48" r="40" />
              <circle class="sos-ring-pulse fx-ambient" cx="48" cy="48" r="40" />
            </svg>
            <span class="sos-ring-core">SOS</span>
          </div>
          <h3 class="feature-title">{FEATURE_SOS.title}</h3>
          <p class="feature-desc">{FEATURE_SOS.desc}</p>
        </div>

        {#each FEATURES_COMPACT as f}
          <div class="bento-cell card-hover-depth" role="listitem">
            <div class="feature-icon-wrap" aria-hidden="true">{@html f.icon}</div>
            <h3 class="feature-title">{f.title}</h3>
            <p class="feature-desc">{f.desc}</p>
          </div>
        {/each}

        <!-- Quiet cell: the privacy promise, stated plainly -->
        <div class="bento-cell bento-quiet" role="listitem">
          <p class="bento-quiet-line">No ads. No data sold. Ever.</p>
          <p class="feature-desc">Your family's location belongs to your family — that's the whole business model.</p>
        </div>

      </div>

    </div>
  </section>


  <!-- ═══════ CTA ════════════════════════════════════════════════════════ -->
  <section class="cta-section" aria-labelledby="cta-heading">
    <div class="cta-bg fx-ambient" aria-hidden="true">
      <div class="cta-orb cta-orb-1"></div>
      <div class="cta-orb cta-orb-2"></div>
    </div>

    <div class="landing-container cta-inner reveal-scroll">
        <h2 id="cta-heading" class="cta-title">Start protecting your family today.</h2>
        <p class="cta-sub">Free forever for families under 6 members. No credit card required.</p>

        {#if !ctaSubmitted}
          <form
            class="cta-form"
            onsubmit={preventDefault(onCtaSubmit)}
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
            <MagneticButton strength={4}>
              <Button variant="primary" size="lg" type="submit">
                Get started free
              </Button>
            </MagneticButton>
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

  <LandingFooter />

</div>

<style>
  /* ── Layout ─────────────────────────────────────────────────────────────── */
  .landing {
    min-height: 100vh;
    background: var(--surface-0);
    color: var(--text-primary);
    font-family: var(--font-sans);
    /* clip (not hidden): an overflow-x:hidden ancestor would silently break
       the position:sticky story column. */
    overflow-x: clip;
  }

  .landing-container {
    width: min(92vw, 90rem);
    margin-inline: auto;
    padding: 0 var(--space-6);
  }

  @media (max-width: 767px) {
    .landing-container { width: 100%; padding: 0 var(--space-4); }
  }

  /* ── Hero + story spine ─────────────────────────────────────────────────── */
  .hero-story {
    position: relative;
  }

  /* Background atmosphere covers the first viewport only */
  .hero-bg {
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 100svh;
    pointer-events: none;
    overflow: clip;
  }

  .hero-aurora {
    position: absolute;
    inset: 0;
    /* aurora-color-cycle keyframe + gradients from global.css .aurora-hero-bg */
  }

  .hero-orb {
    position: absolute;
    border-radius: 50%;
    filter: blur(70px);
    will-change: transform;
  }
  .hero-orb-1 {
    width: 600px; height: 600px;
    background: radial-gradient(circle, color-mix(in oklch, var(--primary-500) 16%, transparent) 0%, transparent 65%);
    top: -10%; left: -10%;
    animation: orb-drift-a 24s ease-in-out infinite;
  }
  .hero-orb-2 {
    width: 500px; height: 500px;
    background: radial-gradient(circle, color-mix(in oklch, var(--member-3) 12%, transparent) 0%, transparent 65%);
    top: 20%; right: -8%;
    animation: orb-drift-b 30s ease-in-out infinite;
  }
  .hero-orb-3 {
    width: 400px; height: 400px;
    background: radial-gradient(circle, color-mix(in oklch, var(--member-4) 8%, transparent) 0%, transparent 65%);
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
    mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
    -webkit-mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
  }

  /* VIGIL night-sky constellation field behind the mockup */
  .hero-field {
    position: absolute;
    top: 8%;
    right: 2%;
    width: min(44vw, 620px);
    height: 72%;
    opacity: 0.45;
  }

  @media (max-width: 900px) {
    .hero-field { width: 80vw; right: -10%; opacity: 0.3; }
  }

  /* Scroll parallax — the night-sky field and depth grid fall away slower
     than the page (progressive enhancement; neither element animates
     transform elsewhere, so the scroll-driven animation owns it). */
  @supports (animation-timeline: scroll()) {
    @media (prefers-reduced-motion: no-preference) {
      .hero-grid  { --plx: 30px; }
      .hero-field { --plx: 56px; }
      .hero-grid,
      .hero-field {
        animation: hero-parallax linear both;
        animation-timeline: scroll(root);
        animation-range: 0svh 120svh;
      }
    }
  }
  @keyframes hero-parallax {
    to { transform: translateY(var(--plx, 40px)); }
  }
  :global([data-fx='minimal']) .hero-grid,
  :global([data-fx='minimal']) .hero-field { animation: none; }

  /* WebGL constellation — absolute over the SVG poster, opacity crossfade in.
     Opacity-only transition (GPU-safe); starts hidden so a mount that never
     fires leaves the poster untouched. */
  .hero-constellation-canvas {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    display: block;
    pointer-events: none;
    opacity: 0;
    transition: opacity 400ms var(--ease-out);
    will-change: opacity;
  }
  .hero-constellation-canvas.is-on { opacity: 1; }

  @media (prefers-reduced-motion: reduce) {
    /* Belt-and-suspenders: the JS gate already blocks the mount, but never
       animate the crossfade if the canvas somehow exists. */
    .hero-constellation-canvas { transition: none; }
  }

  .hero-particle {
    position: absolute;
    width: 2px; height: 2px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--primary-400) 60%, transparent);
    box-shadow: 0 0 6px color-mix(in oklch, var(--primary-400) 50%, transparent);
    animation: particle-float linear infinite;
    will-change: transform, opacity;
  }

  @keyframes particle-float {
    0%  { transform: translateY(0); opacity: 1; }
    50% { transform: translateY(-20px); opacity:0.4; }
    100%{ transform: translateY(0);  opacity:1; }
  }

  /* Grid: hero copy + beats on the left, one sticky mockup column right */
  .hs-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-template-areas:
      'hero  visual'
      'beats visual';
    column-gap: clamp(var(--space-16), 6vw, 7rem);
    align-items: start;
    position: relative;
    z-index: 1;
  }

  .hero-copy {
    grid-area: hero;
    min-height: 100svh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: var(--space-10) 0 var(--space-16);
  }

  .hs-visual { grid-area: visual; }

  .hs-sticky {
    position: sticky;
    top: clamp(var(--space-8), 12svh, 8rem);
    display: flex;
    justify-content: center;
    padding: var(--space-10) 0;
  }

  .story-beats {
    grid-area: beats;
    list-style: none;
    margin: 0;
    padding: 0 0 var(--space-16);
    display: flex;
    flex-direction: column;
  }

  /* Mobile: the hero mockup collapses into the one sticky story card */
  @media (max-width: 900px) {
    .hs-grid { display: block; }
    .hero-copy {
      min-height: auto;
      padding: clamp(var(--space-16), 18svh, 10rem) 0 var(--space-8);
    }
    .hs-visual {
      position: sticky;
      /* clear the fixed nav (56px + safe area) — pre-nav this was space-3 */
      top: calc(env(safe-area-inset-top, 0px) + 56px + var(--space-2));
      z-index: 2;
      display: flex;
      justify-content: center;
    }
    .hs-sticky { position: static; padding: 0; }
    .story-beats { padding: var(--space-10) 0 var(--space-12); }
  }

  /* ── Hero entrance choreography ─────────────────────────────────────────
     CSS-only (.le + --le index): starts at first paint, inherently honors
     prefers-reduced-motion, and needs zero JS. Headline lines rise out of
     clipped line boxes (.hl-line/.hl-inner) — the masked-reveal treatment. */
  .hl-line {
    display: block;
    overflow: clip;
    /* extend the clip box below the baseline so descenders never shear */
    padding-bottom: 0.16em;
    margin-bottom: -0.16em;
  }

  @media (prefers-reduced-motion: no-preference) {
    .le {
      opacity: 0;
      transform: translateY(14px);
      animation: le-rise 480ms cubic-bezier(0.16, 1, 0.3, 1) both;
      animation-delay: calc(var(--le) * 90ms + 60ms);
    }
    .hs-visual.le { transform: translateY(18px) scale(0.985); }

    .hl-inner {
      display: inline-block;
      transform: translateY(120%);
      animation: hl-reveal 480ms cubic-bezier(0.16, 1, 0.3, 1) both;
      animation-delay: calc(var(--hl) * 90ms + 140ms);
    }
  }

  @keyframes le-rise {
    to { opacity: 1; transform: translateY(0) scale(1); }
  }
  @keyframes hl-reveal {
    to { transform: translateY(0); }
  }

  :global([data-fx='minimal']) .le,
  :global([data-fx='minimal']) .hl-inner {
    animation: none;
    opacity: 1;
    transform: none;
  }

  /* Hero copy pieces */
  .hero-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    align-self: flex-start;
    background: color-mix(in oklch, var(--primary-500) 12%, transparent);
    border: 1px solid color-mix(in oklch, var(--primary-500) 28%, transparent);
    border-radius: var(--radius-full, 9999px);
    padding: 5px 14px;
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--primary-400);
    letter-spacing: 0.04em;
    text-transform: uppercase;
    margin-bottom: var(--space-5);
    box-shadow: 0 0 16px color-mix(in oklch, var(--primary-500) 12%, transparent);
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
    font-size: clamp(2.25rem, 5vw, 5rem);
    font-weight: 800;
    line-height: 1.10;
    letter-spacing: -0.03em;
    text-rendering: optimizeLegibility;
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
    font-size: clamp(var(--text-lg), 1.1vw, 1.375rem);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
    max-width: clamp(460px, 34vw, 620px);
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
    margin-right: var(--space-2);
  }
  .hero-avatar {
    width: 28px; height: 28px;
    border-radius: 50%;
    border: 2px solid var(--surface-0);
    margin-right: -8px;
    flex-shrink: 0;
  }

  /* The story device (mockup card) */
  .hero-card-wrap {
    position: relative;
    will-change: transform;
    transform-style: preserve-3d;
  }

  .mockup-card {
    width: clamp(300px, 26vw, 520px);
    background: color-mix(in oklch, var(--surface-0) 88%, transparent);
    border: 1px solid color-mix(in oklch, var(--primary-400) 22%, transparent);
    border-top-color: color-mix(in oklch, var(--primary-300) 40%, transparent);
    border-radius: 20px;
    overflow: hidden;
    box-shadow:
      0 32px 80px color-mix(in srgb, black 60%, transparent),
      0 8px 24px color-mix(in srgb, black 40%, transparent),
      0 0 0 1px color-mix(in oklch, var(--primary-400) 10%, transparent),
      inset 0 1px 0 color-mix(in srgb, white 8%, transparent),
      inset 0 -1px 0 color-mix(in srgb, black 14%, transparent);
    backdrop-filter: blur(32px) saturate(180%) brightness(1.06);
    -webkit-backdrop-filter: blur(32px) saturate(180%) brightness(1.06);
  }

  @media (max-width: 900px) {
    .mockup-card { width: 250px; }
  }

  /* Specular glare — a soft light source that rides the tilt spring
     (translate vars set alongside the card's rotation). Screen-blended
     white at low alpha reads on both themes; .fx-ambient gates it. */
  .mockup-glare {
    position: absolute;
    inset: -30%;
    pointer-events: none;
    z-index: 2;
    background: radial-gradient(
      ellipse 40% 32% at 50% 36%,
      color-mix(in srgb, white 9%, transparent) 0%,
      transparent 70%
    );
    mix-blend-mode: screen;
    transform: translate(var(--glare-x, 0px), var(--glare-y, 0px));
    will-change: transform;
  }

  .mockup-topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid color-mix(in oklch, var(--primary-400) 12%, transparent);
  }

  .mockup-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }

  .mockup-ping {
    position: relative;
    width: 20px; height: 20px;
    flex-shrink: 0;
  }
  /* ping-dot, ping-ring, ping-ring-2 defined in global.css */

  /* ── Story stage (miniature scene-driven map) ───────────────────────────── */
  .mockup-map {
    position: relative;
    height: clamp(150px, 13vw, 240px);
    overflow: hidden;
    /* The map is a NIGHT map in both themes (mixing surface-0 muddied to
       grey on Dawn); ink is a fixed light ramp step for the same reason. */
    --map-ink: var(--gray-100);
    background:
      linear-gradient(135deg,
        color-mix(in oklch, var(--primary-500) 8%, transparent) 0%,
        color-mix(in oklch, var(--member-3) 8%, transparent) 100%),
      var(--surface-midnight);
  }

  @media (max-width: 900px) {
    .mockup-map { height: 150px; }
  }

  .mockup-map-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(0deg, color-mix(in srgb, white 3%, transparent) 1px, transparent 1px),
      linear-gradient(90deg, color-mix(in srgb, white 3%, transparent) 1px, transparent 1px);
    background-size: 24px 24px;
  }

  .story-stage {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
  }

  .place-dot {
    fill: color-mix(in oklch, var(--map-ink) 30%, transparent);
  }
  .place-label {
    fill: color-mix(in oklch, var(--map-ink) 55%, transparent);
    font-size: 9px;
    font-weight: 600;
    font-family: var(--font-sans);
    text-anchor: middle;
  }

  .route {
    fill: none;
    stroke: color-mix(in oklch, var(--rhue) 50%, transparent);
    stroke-width: 1;
    stroke-dasharray: 4 3;
    opacity: 0;
    transition: opacity var(--duration-slower) var(--ease-out);
  }
  .route-on { opacity: 0.55; }

  /* Geofence ring — ignite = scale 0.9→1 + opacity 0→1 (transform/opacity only) */
  .gf {
    fill: color-mix(in oklch, var(--primary-400) 5%, transparent);
    stroke: color-mix(in oklch, var(--primary-400) 35%, transparent);
    stroke-width: 1.5;
    stroke-dasharray: 4 3;
    opacity: 0;
    transform: scale(0.9);
    transform-box: fill-box;
    transform-origin: center;
    transition:
      opacity var(--duration-slower) var(--ease-out),
      transform var(--duration-slower) var(--ease-out);
  }
  .gf-on { opacity: 1; transform: scale(1); }
  .gf-safe {
    stroke: color-mix(in oklch, var(--success-500) 60%, transparent);
    fill: color-mix(in oklch, var(--success-500) 8%, transparent);
  }

  /* Member pins travel between scene positions on transform only */
  .story-pin {
    transition: transform var(--duration-slower) var(--ease-out);
    will-change: transform;
  }
  .sp-halo { fill: color-mix(in oklch, var(--hue) 18%, transparent); }
  .sp-dot  { fill: var(--hue); }
  .sp-label {
    fill: color-mix(in oklch, var(--map-ink) 75%, transparent);
    font-size: 8px;
    font-weight: 700;
    font-family: var(--font-sans);
    text-anchor: middle;
  }
  .sp-sos-ring {
    fill: none;
    stroke: var(--danger-500);
    stroke-width: 1.5;
    opacity: 0;
    transform-box: fill-box;
    transform-origin: center;
  }

  /* SOS beat: pin flips to danger + expanding pulse ring (resolves to safe) */
  .story-pin.sp-sos .sp-dot  { fill: var(--danger-500); }
  .story-pin.sp-sos .sp-halo { fill: color-mix(in oklch, var(--danger-500) 22%, transparent); }
  .story-pin.sp-sos .sp-sos-ring { animation: sos-ping 1.8s var(--ease-out) infinite; }

  @keyframes sos-ping {
    0%   { transform: scale(0.6); opacity: 0.8; }
    100% { transform: scale(2.8); opacity: 0; }
  }

  /* Member rows */
  .mockup-list { padding: var(--space-2) 0; }

  .mockup-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    animation: row-appear 400ms var(--ease-out) both;
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

  @media (max-width: 900px) {
    /* Compact story card on mobile — the map is the message */
    .mockup-list { display: none; }
  }

  /* Floating scene chip — rides the card's BOTTOM edge: the top edge lives
     in the fixed-nav band at rest, and the two collided. */
  .mockup-chip {
    position: absolute;
    bottom: -16px; right: -10px;
    display: flex;
    align-items: center;
    gap: var(--space-1);
    font-size: 11px;
    font-weight: 700;
    padding: 5px 12px;
    border-radius: var(--radius-full, 9999px);
    white-space: nowrap;
    pointer-events: none;
    box-shadow: 0 4px 16px color-mix(in srgb, black 35%, transparent);
    animation: chip-pop var(--duration-normal) var(--ease-spring) both;
  }

  .mockup-chip[data-tone='safe'] {
    background: color-mix(in oklch, var(--success-500) 18%, transparent);
    border: 1px solid color-mix(in oklch, var(--success-500) 40%, transparent);
    color: var(--success-400);
  }
  .mockup-chip[data-tone='info'] {
    background: color-mix(in oklch, var(--primary-500) 15%, transparent);
    border: 1px solid color-mix(in oklch, var(--primary-500) 38%, transparent);
    color: var(--primary-400);
  }
  .mockup-chip[data-tone='alert'] {
    background: color-mix(in oklch, var(--danger-500) 16%, transparent);
    border: 1px solid color-mix(in oklch, var(--danger-500) 42%, transparent);
    color: var(--danger-400, var(--danger-500));
  }

  @keyframes chip-pop {
    from { opacity: 0; transform: translateY(8px) scale(0.95); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  @media (max-width: 900px) {
    .mockup-chip { right: -4px; bottom: -12px; }
  }

  .chip-icon { font-size: 12px; }

  /* Scroll cue — pinned to the bottom of the first viewport */
  .scroll-cue {
    position: absolute;
    top: calc(100svh - var(--space-12));
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
  }
  .scroll-cue-line {
    width: 1px; height: 40px;
    background: linear-gradient(180deg, color-mix(in oklch, var(--primary-400) 60%, transparent) 0%, transparent 100%);
    animation: scroll-line-drop 2s ease-in-out infinite;
  }
  @keyframes scroll-line-drop {
    0%,100% { transform: scaleY(0); transform-origin: top; opacity:0; }
    40%     { transform: scaleY(1); transform-origin: top; opacity:1; }
    80%     { transform: scaleY(1); transform-origin: bottom; opacity:1; }
    100%    { transform: scaleY(0); transform-origin: bottom; opacity:0; }
  }


  /* ── Stats bar ────────────────────────────────────────────────────────────── */
  .stats-bar {
    padding: clamp(var(--space-10), 4vw, 4rem) 0;
    border-top: 1px solid var(--border-subtle);
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-1);
    position: relative;
    z-index: 1;
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
    .stat-item:nth-child(2n) { border-right: none; }
    .stat-item:nth-child(1),
    .stat-item:nth-child(2) { border-bottom: 1px solid var(--border-default); }
  }

  .stat-value {
    font-family: var(--font-display);
    font-size: clamp(var(--text-3xl), 2.6vw, 3.25rem);
    font-weight: 800;
    letter-spacing: -0.03em;
    text-rendering: optimizeLegibility;
    background: linear-gradient(135deg, var(--primary-300) 0%, var(--primary-500) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    min-width: 5ch;
    text-align: center;
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


  /* ── Section shared ─────────────────────────────────────────────────────── */
  .section-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    margin-bottom: clamp(var(--space-12, 48px), 4vw, 4.5rem);
    width: 100%;
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
    font-size: clamp(1.75rem, 3.5vw, 3.25rem);
    font-weight: 800;
    line-height: 1.15;
    letter-spacing: -0.025em;
    text-rendering: optimizeLegibility;
    color: var(--text-primary);
  }


  /* ── Features ─────────────────────────────────────────────────────────────── */
  .features {
    padding: clamp(var(--space-16), 7vw, 7rem) 0 clamp(var(--space-12), 5vw, 5rem);
    position: relative;
    z-index: 1;
    background: var(--surface-0);
  }

  /* Asymmetric bento — cells are page-local glass surfaces (same token
     recipe as .mockup-card); lift/shadow on hover comes from the global
     .card-hover-depth class, so local hover only tints border + nudges icon. */
  .features-bento {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    grid-auto-rows: minmax(150px, auto);
    gap: clamp(var(--space-4), 1.5vw, var(--space-6));
  }

  .bento-cell {
    position: relative;
    padding: var(--space-6);
    border-radius: var(--radius-lg, 16px);
    background: var(--glass-bg);
    border: 1px solid var(--border-default);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    overflow: clip;
    transition: border-color var(--duration-normal) var(--ease-out);
  }

  @media (hover: hover) {
    .bento-cell:hover {
      border-color: color-mix(in oklch, var(--primary-400) 34%, transparent);
    }
    .bento-cell:hover .feature-icon-wrap {
      transform: translateY(-2px) rotate(-3deg);
    }
  }
  .feature-icon-wrap {
    transition: transform var(--duration-normal) var(--ease-spring, var(--ease-out));
  }

  /* Featured: GPS — copy beside a live mini-map */
  .bento-gps {
    grid-column: span 2;
    display: grid;
    grid-template-columns: 1.1fr 0.9fr;
    gap: var(--space-5);
    align-items: center;
  }

  .bento-gps-visual {
    position: relative;
    height: 100%;
    min-height: 132px;
    border-radius: var(--radius-md, 12px);
    border: 1px solid var(--border-subtle);
    overflow: clip;
    background:
      linear-gradient(135deg,
        color-mix(in oklch, var(--primary-500) 8%, transparent) 0%,
        color-mix(in oklch, var(--member-3) 8%, transparent) 100%),
      var(--surface-midnight);
  }

  .bgv-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(0deg, color-mix(in srgb, white 3%, transparent) 1px, transparent 1px),
      linear-gradient(90deg, color-mix(in srgb, white 3%, transparent) 1px, transparent 1px);
    background-size: 20px 20px;
  }

  .bgv-dot {
    position: absolute;
    width: 10px; height: 10px;
    border-radius: 50%;
    background: var(--hue);
    box-shadow: 0 0 10px color-mix(in oklch, var(--hue) 55%, transparent);
  }

  .bgv-ring {
    position: absolute;
    inset: -6px;
    border-radius: 50%;
    border: 1.5px solid var(--hue);
    opacity: 0;
    animation: bgv-ping 2.2s var(--ease-out) infinite;
  }

  @keyframes bgv-ping {
    0%   { transform: scale(0.5); opacity: 0.8; }
    100% { transform: scale(2.1); opacity: 0; }
  }

  /* Featured: SOS — hold-ring motif (transform/opacity pulse only) */
  .bento-sos {
    grid-row: span 2;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .bento-sos-visual {
    position: relative;
    width: 96px; height: 96px;
    margin: 0 auto var(--space-5);
  }

  .sos-ring-svg { width: 100%; height: 100%; }

  .sos-ring-track {
    fill: none;
    stroke: color-mix(in oklch, var(--danger-500) 22%, transparent);
    stroke-width: 5;
  }

  .sos-ring-pulse {
    fill: none;
    stroke: var(--danger-500);
    stroke-width: 3;
    opacity: 0;
    transform-box: fill-box;
    transform-origin: center;
    animation: sos-hold-pulse 2.6s var(--ease-out) infinite;
  }

  @keyframes sos-hold-pulse {
    0%   { transform: scale(0.86); opacity: 0.7; }
    100% { transform: scale(1.24); opacity: 0; }
  }

  .sos-ring-core {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 800;
    letter-spacing: 0.04em;
    color: var(--danger-400, var(--danger-500));
  }

  .bento-sos .feature-title,
  .bento-sos .feature-desc { text-align: center; }

  /* Quiet cell — the privacy promise */
  .bento-quiet {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: var(--space-2);
    background: color-mix(in oklch, var(--success-500) 7%, transparent);
    border-color: color-mix(in oklch, var(--success-500) 26%, transparent);
  }

  .bento-quiet-line {
    font-family: var(--font-display);
    font-size: var(--text-xl);
    font-weight: 800;
    letter-spacing: -0.02em;
    color: var(--text-primary);
    margin: 0;
  }

  /* 7th cell joins the global reveal-scroll-grid stagger (it only covers 6) */
  @supports (animation-timeline: view()) {
    .features-bento > :nth-child(7) {
      animation: reveal-up linear both;
      animation-timeline: view();
      animation-range: entry 0% entry 60%;
      animation-delay: 480ms;
    }
  }

  @media (max-width: 900px) {
    .features-bento { grid-template-columns: repeat(2, 1fr); }
    .bento-gps { grid-column: span 2; grid-template-columns: 1fr; }
    .bento-gps-visual { min-height: 120px; }
    .bento-sos { grid-row: auto; }
  }
  @media (max-width: 580px) {
    .features-bento { grid-template-columns: 1fr; }
    .bento-gps { grid-column: auto; }
  }

  .feature-icon-wrap {
    width: 44px; height: 44px;
    border-radius: var(--radius-lg, 10px);
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg,
      color-mix(in oklch, var(--primary-500) 18%, transparent) 0%,
      color-mix(in oklch, var(--primary-700) 10%, transparent) 100%);
    border: 1px solid color-mix(in oklch, var(--primary-500) 24%, transparent);
    color: var(--primary-400);
    margin-bottom: var(--space-4);
    box-shadow:
      0 4px 16px color-mix(in oklch, var(--primary-500) 12%, transparent),
      inset 0 1px 0 color-mix(in srgb, white 12%, transparent);
  }

  .feature-title {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.015em;
    text-rendering: optimizeLegibility;
    margin-bottom: var(--space-2);
  }

  .feature-desc {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
  }


  /* ── CTA section ────────────────────────────────────────────────────────── */
  .cta-section {
    position: relative;
    padding: clamp(var(--space-16), 7vw, 7rem) 0 clamp(var(--space-10), 4vw, 4rem);
    overflow: hidden;
    border-top: 1px solid var(--border-subtle);
    z-index: 1;
    background: var(--surface-0);
  }

  .cta-bg { position: absolute; inset: 0; pointer-events: none; }

  .cta-orb {
    position: absolute;
    border-radius: 50%;
    filter: blur(80px);
  }
  .cta-orb-1 {
    width: 400px; height: 400px;
    background: radial-gradient(circle, color-mix(in oklch, var(--primary-500) 14%, transparent) 0%, transparent 65%);
    top: -10%; left: 20%;
  }
  .cta-orb-2 {
    width: 300px; height: 300px;
    background: radial-gradient(circle, color-mix(in oklch, var(--member-3) 10%, transparent) 0%, transparent 65%);
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
    font-size: clamp(1.75rem, 3.5vw, 3.25rem);
    font-weight: 800;
    letter-spacing: -0.025em;
    text-rendering: optimizeLegibility;
    color: var(--text-primary);
    max-width: clamp(600px, 42vw, 760px);
  }

  .cta-sub {
    font-size: var(--text-base);
    color: var(--text-secondary);
    max-width: clamp(420px, 30vw, 520px);
    line-height: var(--leading-relaxed);
    margin-top: calc(-1 * var(--space-2));
  }

  .cta-form {
    display: flex;
    gap: var(--space-3);
    width: 100%;
    max-width: clamp(460px, 32vw, 560px);
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
    background: color-mix(in oklch, var(--success-500) 12%, transparent);
    border: 1px solid color-mix(in oklch, var(--success-500) 32%, transparent);
    border-radius: var(--radius-xl, 14px);
    padding: var(--space-4) var(--space-6);
    color: var(--success-400);
    font-weight: 700;
    font-size: var(--text-base);
  }

  .cta-success-icon {
    width: 28px; height: 28px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--success-500) 20%, transparent);
    border: 1px solid color-mix(in oklch, var(--success-500) 40%, transparent);
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

  /* ── FX gating: minimal tier gets instant scene swaps, no loops ──────────── */
  :global([data-fx='minimal']) .story-pin,
  :global([data-fx='minimal']) .gf,
  :global([data-fx='minimal']) .route { transition: none; }
  :global([data-fx='minimal']) .mockup-chip { animation: none; }
  :global([data-fx='minimal']) .story-pin.sp-sos .sp-sos-ring { animation: none; opacity: 0; }
  /* keyframe opacity would override the .fx-ambient opacity gate — kill the
     loops explicitly at minimal (mockup-glare is static; fx-ambient hides it) */
  :global([data-fx='minimal']) .bgv-ring,
  :global([data-fx='minimal']) .sos-ring-pulse { animation: none; opacity: 0; }

  /* ── Reduced motion: final static frames, zero travel, zero loops ────────── */
  @media (prefers-reduced-motion: reduce) {
    .hero-orb, .cta-orb,
    .hero-particle,
    .hero-badge-dot,
    .scroll-cue-line,
    .mockup-row,
    .mockup-chip,
    .hero-aurora { animation: none; opacity: 1; transform: scale(1); }

    .hero-card-wrap { transform: none !important; }
    .mockup-glare { transform: none !important; }

    .story-pin, .gf, .route { transition: none; }
    .story-pin.sp-sos .sp-sos-ring { animation: none; opacity: 0; }
    .bgv-ring, .sos-ring-pulse { animation: none; opacity: 0; }

    .reveal-scroll,
    .reveal-scroll-grid > * {
      opacity: 1 !important;
      transform: none !important;
      animation: none !important;
    }
  }
</style>
