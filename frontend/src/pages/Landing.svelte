<script>
  /**
   * Landing — "Kinnect Landing Story" (Hearth, from Claude Design
   * `Kinnect Landing Story.dc.html`).
   *
   * Three acts:
   *   HERO     — serif headline + the promise. "No feeds. No pings. No dashboard."
   *   STORY    — a 700vh scroll-scrubbed Tuesday with the Nair family on a
   *              stylised map: pebbles move along routes, a camera follows,
   *              chapter cards narrate 07:00 → 19:30 (incl. a quiet Nani, an
   *              SOS that resolves), the clock scrubs with you.
   *   CLOSING  — "Everyone's settled." + the same two CTAs.
   *
   * Mechanics: scroll position → story progress p (0..1) → narrative time t
   * (07:00..19:30) → camera/people/routes/chapters, all pure interpolation.
   * Scrubbing is user-driven, so nothing moves on its own — the page is
   * still under prefers-reduced-motion (only the tiny CSS transitions are
   * disabled there). One rAF-throttled scroll listener, no timers.
   */
  import { onMount } from 'svelte';
  import { navigate } from '../lib/viewTransition.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';

  // ── Geometry (verbatim from the design) ─────────────────────────────────
  const H = [1000, 900], SCHOOL = [560, 1000], OFFICE = [1360, 440], TUITION = [740, 1160], MARKET = [1180, 760], SOSP = [860, 1040];
  const off = (p0, o) => [p0[0] + o[0], p0[1] + o[1]];
  const HA = off(H, [-40, 18]), HM = off(H, [34, 22]), HN = off(H, [-10, -44]);

  const pathKeys = (t0, t1, pts) => {
    const d = [0];
    for (let i = 1; i < pts.length; i++) d.push(d[i - 1] + Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]));
    const L = d[d.length - 1];
    return pts.map((q, i) => [t0 + (t1 - t0) * d[i] / L, q[0], q[1]]);
  };
  const hold = (t, p0) => [[t, p0[0], p0[1]]];
  const seq = (...parts) => {
    const out = [];
    for (const part of parts) for (const k of part) {
      if (!out.length || k[0] > out[out.length - 1][0] + 1e-6) out.push(k);
    }
    return out;
  };

  const PEOPLE = [
    { name: 'You', initial: 'P', self: true,
      keys: seq(hold(7, H), pathKeys(11.0, 11.2, [H, [1180, 900], MARKET]), hold(12.0, MARKET), pathKeys(12.0, 12.2, [MARKET, [1180, 900], H]), hold(19.5, H)),
      tags: [[7, 'You', 'live'], [11.0, 'You · market', 'live'], [12.2, 'You', 'live']] },
    { name: 'Arjun', initial: 'A', color: 'var(--member-1)',
      keys: seq(hold(7, HA), pathKeys(9.0, 9.6, [HA, [1100, 918], [1100, 600], [1360, 600], OFFICE]), hold(18.86, OFFICE), pathKeys(18.87, 19.03, [OFFICE, [1360, 600], [1100, 600], [1100, 1040], [830, 1040]]), hold(19.1, [830, 1040]), pathKeys(19.1, 19.3, [[830, 1040], [1000, 1040], HA]), hold(19.5, HA)),
      tags: [[7, 'home', 'live'], [9.0, 'driving', 'live'], [9.6, 'office', 'live'], [18.87, 'on his way', 'live'], [19.03, 'with Meera', 'live'], [19.1, 'driving home', 'live'], [19.3, 'home', 'live']] },
    { name: 'Meera', initial: 'M', color: 'var(--member-2)',
      keys: seq(hold(7, HM), pathKeys(7.6, 7.95, [HM, [900, 922], [900, 1000], SCHOOL]), hold(15.4, SCHOOL), pathKeys(15.4, 15.75, [SCHOOL, [900, 1000], [900, 922], HM]), hold(17.0, HM), pathKeys(17.0, 17.3, [HM, [1034, 1060], [740, 1060], TUITION]), hold(18.6, TUITION), pathKeys(18.6, 18.85, [TUITION, [740, 1040], SOSP]), hold(19.2, SOSP), pathKeys(19.2, 19.35, [SOSP, [1000, 1040], HM]), hold(19.5, HM)),
      tags: [[7, 'home', 'live'], [7.6, 'walking to school', 'live'], [7.95, 'school', 'live'], [15.4, 'walking home', 'live'], [15.75, 'home', 'live'], [17.0, 'to tuition', 'live'], [17.3, 'tuition', 'live'], [18.6, 'walking home', 'live'], [18.85, 'SOS', 'sos'], [19.05, 'safe', 'safe'], [19.2, 'heading home', 'live'], [19.35, 'home', 'live']] },
    { name: 'Nani', initial: 'N', color: 'var(--member-3)',
      keys: seq(hold(7, HN), hold(19.5, HN)),
      tags: [[7, 'home', 'live'], [16.6, 'quiet', 'quiet'], [17.25, 'home', 'live']] },
  ];

  const routeLen = (pts) => { let L = 0; for (let i = 1; i < pts.length; i++) L += Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]); return L; };
  const ROUTES = [
    { t0: 7.6, t1: 7.95, pts: [HM, [900, 922], [900, 1000], SCHOOL], c: 'var(--member-2)' },
    { t0: 9.0, t1: 9.6, pts: [HA, [1100, 918], [1100, 600], [1360, 600], OFFICE], c: 'var(--member-1)' },
    { t0: 11.0, t1: 11.2, pts: [H, [1180, 900], MARKET], c: 'var(--ember)' },
    { t0: 15.4, t1: 15.75, pts: [SCHOOL, [900, 1000], [900, 922], HM], c: 'var(--member-2)' },
    { t0: 17.0, t1: 17.3, pts: [HM, [1034, 1060], [740, 1060], TUITION], c: 'var(--member-2)' },
    { t0: 18.6, t1: 18.85, pts: [TUITION, [740, 1040], SOSP], c: 'var(--member-2)' },
    { t0: 18.87, t1: 19.03, pts: [OFFICE, [1360, 600], [1100, 600], [1100, 1040], [830, 1040]], c: 'var(--member-1)' },
    { t0: 19.2, t1: 19.35, pts: [SOSP, [1000, 1040], HM], c: 'var(--member-2)' },
    { t0: 19.1, t1: 19.3, pts: [[830, 1040], [1000, 1040], HA], c: 'var(--member-1)' },
  ].map((r) => ({ ...r, L: routeLen(r.pts), points: r.pts.map((q) => q.join(',')).join(' ') }));

  const PLACES = [
    { name: 'Home', x: 1000, y: 900, center: true, dy: -78 },
    { name: 'School', x: 560, y: 1000, dx: 32 },
    { name: 'Office', x: 1360, y: 440, dx: 32 },
    { name: 'Tuition', x: 740, y: 1160, dx: 32 },
    { name: 'Market', x: 1180, y: 760, dx: 32 },
    { name: 'Koramangala', x: 1230, y: 960, area: true },
    { name: 'Jayanagar', x: 560, y: 1100, area: true },
    { name: 'Indiranagar', x: 1360, y: 340, area: true },
    { name: 'Ulsoor Lake', x: 1710, y: 270, area: true },
    { name: 'Cubbon Park', x: 740, y: 600, area: true },
  ];

  const CHAPTERS = [
    { a: 7.0, b: 7.6, time: '07:00', title: 'Good morning, Nair family.', body: 'Everyone’s home. The map is quiet — the way it should be.' },
    { a: 7.6, b: 9.0, time: '07:40', title: 'Meera leaves for school.', body: 'Her pebble traces the route. Priya sees a moving dot and a time — 14 minutes — not a speed or a coordinate.' },
    { a: 9.0, b: 12.0, time: '09:00', title: 'Arjun drives to work.', body: 'No pings. No “left home” alerts. Just where he is, when you look.' },
    { a: 12.0, b: 16.6, time: '12:30', title: 'Nothing to report.', body: 'Everyone’s where they should be, so Kinnect says nothing. That’s the point.' },
    { a: 16.6, b: 18.6, time: '16:40', title: 'Nani’s phone goes quiet.', body: 'Not an alarm. Her pebble fades and the page warms: needs a look. Priya calls — she’d switched it off for a nap.' },
    { a: 18.6, b: 18.85, time: '18:40', title: 'Meera walks home from tuition.', body: 'Fifteen minutes, the usual way.' },
    { a: 18.85, b: 19.05, time: '18:52', title: 'Meera holds SOS.', body: 'One sentence lands on every phone: Meera needs help. Arjun is nine minutes away — and already moving.', sos: true },
    { a: 19.05, b: 19.35, time: '19:04', title: 'She’s safe.', body: 'Twelve minutes, start to finish. Everyone saw the same short story.', sage: true },
    { a: 19.35, b: 19.5, time: '19:30', title: 'Everyone’s settled.', body: 'The day ends the way it started. Quiet.' },
  ];

  // scroll progress → narrative time, and time → camera [cx, cy, scale]
  const TM = [[0, 7], [0.11, 7.95], [0.22, 9.6], [0.32, 12.5], [0.42, 16.6], [0.52, 17.4], [0.6, 18.6], [0.7, 18.87], [0.8, 19.05], [0.9, 19.3], [1, 19.5]];
  const CAM = [[7.0, 1000, 900, 1.5], [7.5, 1000, 900, 1.5], [7.95, 790, 960, 1.15], [8.7, 790, 960, 1.15], [9.5, 1160, 690, 0.85], [10.8, 1160, 690, 0.85], [12.5, 1000, 820, 0.68], [16.2, 1000, 820, 0.68], [16.8, 1000, 880, 1.05], [17.8, 1000, 880, 1.05], [18.55, 800, 1090, 1.05], [18.9, 860, 1040, 1.45], [19.15, 940, 990, 1.05], [19.5, 1000, 900, 1.5]];

  const lerp = (a, b, u) => a + (b - a) * u;
  const cl = (u) => Math.max(0, Math.min(1, u));
  const sm = (u) => u * u * (3 - 2 * u);
  function keyed(keys, t, ease) {
    if (t <= keys[0][0]) return keys[0].slice(1);
    for (let i = 1; i < keys.length; i++) {
      if (t <= keys[i][0]) {
        const a = keys[i - 1], b = keys[i];
        let u = (t - a[0]) / (b[0] - a[0]);
        if (ease) u = sm(u);
        return a.slice(1).map((v, j) => lerp(v, b[j + 1], u));
      }
    }
    return keys[keys.length - 1].slice(1);
  }

  // ── Scroll state ────────────────────────────────────────────────────────
  let p = $state(0);
  let vw = $state(1200);
  let vh = $state(800);
  let storyEl = $state(null);
  let raf = 0;

  function measure() {
    vw = window.innerWidth;
    vh = window.innerHeight;
    if (storyEl) {
      const r = storyEl.getBoundingClientRect();
      const total = r.height - vh;
      p = total > 0 ? cl(-r.top / total) : 0;
    }
  }
  function onScroll() {
    if (raf) return;
    raf = requestAnimationFrame(() => { raf = 0; measure(); });
  }

  onMount(() => {
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll);
    measure();
    requestAnimationFrame(measure);
    return () => {
      window.removeEventListener('scroll', onScroll);
      window.removeEventListener('resize', onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  });

  function goStory() {
    if (!storyEl) return;
    const y = storyEl.getBoundingClientRect().top + (window.scrollY || document.documentElement.scrollTop || 0);
    window.scrollTo({ top: y, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  }

  // ── The scene, derived from scroll ──────────────────────────────────────
  let scene = $derived.by(() => {
    const t = keyed(TM, p)[0];
    const [cx, cy, s0] = keyed(CAM, t, true);
    const narrow = vw < 720;
    const s = s0 * Math.max(0.5, Math.min(1, vw / 1100));
    const tx = vw / 2 - cx * s;
    const ty = (narrow ? vh * 0.4 : vh / 2) - cy * s;
    const k = 1 / s;
    // SOS window desaturates the map (design 18.85–19.05)
    const ver = 0.62 * cl((t - 18.83) / 0.05) * (1 - cl((t - 19.03) / 0.06));

    const people = PEOPLE.map((per) => {
      const [x, y] = keyed(per.keys, t);
      let tag = per.tags[0];
      for (const tg of per.tags) if (t >= tg[0]) tag = tg;
      const state = tag[2];
      return {
        initial: per.initial,
        label: per.self ? tag[1] : `${per.name} · ${tag[1]}`,
        x, y,
        self: !!per.self,
        color: per.self ? 'var(--ember)' : per.color,
        sos: state === 'sos',
        quiet: state === 'quiet',
      };
    });

    const routes = ROUTES.map((r) => {
      const f = cl((t - r.t0) / (r.t1 - r.t0));
      const o = Math.min(cl((t - (r.t0 - 0.05)) / 0.05), 1 - cl((t - (r.t1 + 0.5)) / 0.4)) * 0.85;
      return { points: r.points, color: r.c, dash: r.L, offset: r.L * (1 - f), opacity: o };
    });

    const chapters = CHAPTERS.map((c, i) => {
      const fade = 0.12;
      const fi = i === 0 ? 1 : cl((t - c.a) / fade);
      const fo = i === CHAPTERS.length - 1 ? 1 : cl((c.b - t) / fade);
      const o = Math.min(fi, fo);
      return { ...c, o };
    });

    // ambient tint: ochre while Nani is quiet, vermilion during SOS, sage after
    const och = 0.22 * cl((t - 16.55) / 0.15) * (1 - cl((t - 17.2) / 0.2));
    const sag = 0.28 * cl((t - 19.04) / 0.06) * (1 - cl((t - 19.25) / 0.15));
    let tint = { c: 'var(--ochre)', o: och };
    if (ver > tint.o) tint = { c: 'var(--vermilion)', o: ver };
    if (sag > tint.o) tint = { c: 'var(--sage)', o: sag };

    const hh = Math.floor(t), mm = Math.floor((t - hh) * 60);
    const clock = `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`;

    return { t, tx, ty, s, k, ver, people, routes, chapters, tint, clock, hint: p < 0.015 };
  });
</script>

<div class="lp">

  <!-- ═══ HERO ═══════════════════════════════════════════════════════════ -->
  <section class="lp-hero">
    <nav class="lp-nav" aria-label="Landing">
      <span class="lp-wordmark">Kinnect</span>
      <div class="lp-nav-actions">
        <button class="lp-nav-link" onclick={goStory}>How it works</button>
        <button class="lp-nav-link" onclick={() => navigate('/login')}>Sign in</button>
        <button class="lp-nav-pill" onclick={() => navigate('/register')}>Get the app</button>
      </div>
    </nav>

    <div class="lp-hero-body">
      <span class="lp-eyebrow">Family location sharing, without the noise</span>
      <h1 class="lp-headline">A quiet map for the people you love.</h1>
      <p class="lp-sub">
        Kinnect shows your family as a few pebbles on a map and tells you, in one
        sentence, whether everyone's OK. No feeds. No pings. No dashboard.
      </p>
      <div class="lp-cta-row">
        <button class="lp-cta lp-cta-primary" onclick={() => navigate('/register')}>Create your family — free</button>
        <button class="lp-cta lp-cta-ghost" onclick={goStory}>See a day with one family ↓</button>
      </div>
    </div>

    <div class="lp-scroll-cue"><span class="lp-cue-line" aria-hidden="true"></span>Scroll — a Tuesday with the Nair family, Bengaluru</div>
  </section>

  <!-- ═══ STORY (scroll-scrubbed) ════════════════════════════════════════ -->
  <section class="lp-story" bind:this={storyEl} aria-label="A day with the Nair family">
    <div class="lp-stage">
      <div
        class="lp-cam"
        style:transform={`translate(${scene.tx}px, ${scene.ty}px) scale(${scene.s})`}
        style:filter={scene.ver > 0.01 ? `saturate(${1 - scene.ver})` : 'none'}
      >
        <div class="lp-grid-major" aria-hidden="true"></div>
        <div class="lp-grid-minor" aria-hidden="true"></div>
        <div class="lp-water" aria-hidden="true"></div>
        <div class="lp-park" aria-hidden="true"></div>

        <svg viewBox="0 0 2000 1400" width="2000" height="1400" class="lp-routes" aria-hidden="true">
          {#each scene.routes as r, i (i)}
            <polyline
              points={r.points} fill="none" stroke={r.color} stroke-width="5"
              stroke-linecap="round" stroke-linejoin="round"
              style:stroke-dasharray={r.dash}
              style:stroke-dashoffset={r.offset}
              style:opacity={r.opacity}
            />
          {/each}
        </svg>

        {#each PLACES as pl (pl.name)}
          <div class="lp-place" style:left={`${pl.x}px`} style:top={`${pl.y}px`} style:transform={`scale(${scene.k})`}>
            <span class="lp-place-label" class:lp-area={pl.area}
              style:transform={pl.area || pl.center ? `translate(-50%, calc(-50% + ${pl.dy || 0}px))` : `translate(${pl.dx}px, -50%)`}
            >{pl.name}</span>
          </div>
        {/each}

        {#each scene.people as per (per.initial)}
          <div class="lp-person" class:lp-sos={per.sos} class:lp-self={per.self}
            style:left={`${per.x}px`} style:top={`${per.y}px`} style:transform={`scale(${scene.k})`}>
            <div class="lp-person-inner">
              <div class="lp-pebble-wrap">
                <div class="lp-sos-ring" class:on={per.sos} aria-hidden="true"></div>
                <div class="lp-pebble" class:lp-quiet={per.quiet}
                  style:background={per.quiet ? 'var(--lp-quiet)' : per.sos ? 'var(--vermilion)' : per.color}>{per.initial}</div>
              </div>
              <div class="lp-tag" class:lp-tag-sos={per.sos} class:lp-tag-quiet={per.quiet}>{per.label}</div>
            </div>
          </div>
        {/each}
      </div>

      <div class="lp-tint" style:background={scene.tint.c} style:opacity={scene.tint.o} aria-hidden="true"></div>

      <div class="lp-story-chrome" aria-hidden="true">
        <span class="lp-wordmark lp-wordmark-sm">Kinnect</span>
        <span class="lp-clock"><i class="lp-clock-dot"></i>{scene.clock} · Tuesday</span>
      </div>

      <div class="lp-hint" class:visible={scene.hint}>Keep scrolling — the day moves with you</div>

      {#each scene.chapters as c (c.time)}
        <div class="lp-chapter" style:opacity={c.o} style:transform={`translateY(${(1 - c.o) * 16}px)`}
          style:pointer-events={c.o > 0.5 ? 'auto' : 'none'} aria-hidden={c.o < 0.1}>
          <div class="lp-chapter-card" class:lp-card-sos={c.sos} class:lp-card-sage={c.sage}>
            <span class="lp-chapter-time">{c.time}</span>
            <p class="lp-chapter-title">{c.title}</p>
            <p class="lp-chapter-body">{c.body}</p>
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- ═══ CLOSING ════════════════════════════════════════════════════════ -->
  <section class="lp-closing">
    <span class="lp-eyebrow">That's the whole app</span>
    <h2 class="lp-closing-headline">Everyone's settled.</h2>
    <p class="lp-closing-body">
      One sentence, a few pebbles, and a hold-to-send SOS for the one evening in
      a thousand. Only people you invite can see anyone — ever.
    </p>
    <div class="lp-cta-row lp-cta-center">
      <button class="lp-cta lp-cta-primary" onclick={() => navigate('/register')}>Create your family — free</button>
      <button class="lp-cta lp-cta-ghost" onclick={() => navigate('/register')}>Get the app</button>
    </div>
    <div class="lp-foot">
      <span class="lp-foot-mark">Kinnect</span>
      <span>Privacy-first</span>
      <span>No ads, ever</span>
      <span>Bengaluru</span>
    </div>
  </section>
</div>

<style>
  .lp {
    --lp-quiet: oklch(0.78 0.012 60);
    background: var(--paper);
    color: var(--ink);
    font-family: var(--font-sans);
  }

  /* ── Hero ─────────────────────────────────────────────────────────────── */
  .lp-hero {
    position: relative;
    min-height: 100vh;
    min-height: 100svh;
    display: flex;
    flex-direction: column;
    padding: calc(var(--safe-top, 0px) + var(--space-5)) var(--space-6) var(--space-6);
    box-sizing: border-box;
  }
  .lp-nav { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); }
  .lp-wordmark {
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 26px;
    letter-spacing: -0.01em;
  }
  .lp-wordmark-sm { font-size: 22px; }
  .lp-nav-actions { display: flex; align-items: center; gap: var(--space-3); }
  .lp-nav-link {
    min-height: 44px; padding: 0 var(--space-1);
    border: none; background: transparent; cursor: pointer;
    color: var(--ink-2); font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    white-space: nowrap;
  }
  .lp-nav-link:hover { color: var(--ink); }
  .lp-nav-link:focus-visible, .lp-nav-pill:focus-visible, .lp-cta:focus-visible {
    outline: 2px solid var(--primary-400); outline-offset: 2px;
  }
  .lp-nav-pill {
    min-height: 44px; padding: 0 var(--space-4);
    border-radius: 999px; border: 1px solid var(--hairline);
    background: transparent; cursor: pointer;
    color: var(--ink); font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    white-space: nowrap;
  }
  .lp-nav-pill:hover { background: var(--surface-hover); }

  .lp-hero-body {
    flex: 1;
    display: flex; flex-direction: column; justify-content: center;
    gap: var(--space-6);
    max-width: 960px;
    padding: var(--space-12) 0;
  }
  .lp-eyebrow {
    font-size: var(--text-xs); font-weight: 600;
    letter-spacing: 0.1em; text-transform: uppercase;
    color: var(--ink-3);
  }
  .lp-headline {
    margin: 0;
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: clamp(46px, 7.6vw, 108px);
    line-height: 0.98;
    letter-spacing: -0.015em;
    text-wrap: balance;
  }
  .lp-sub {
    margin: 0;
    font-size: clamp(16px, 1.4vw, 20px);
    line-height: 1.5;
    color: var(--ink-2);
    max-width: 600px;
    text-wrap: pretty;
  }
  .lp-cta-row { display: flex; gap: var(--space-3); flex-wrap: wrap; }
  .lp-cta-center { justify-content: center; margin-top: var(--space-2); }
  .lp-cta {
    min-height: 52px; padding: 0 var(--space-5);
    border-radius: 999px; cursor: pointer;
    font-family: inherit; font-size: var(--text-base); font-weight: 600;
    white-space: nowrap;
    -webkit-tap-highlight-color: transparent;
  }
  .lp-cta-primary { border: none; background: var(--ember); color: #fff; }
  .lp-cta-primary:hover { background: var(--primary-600); }
  .lp-cta-ghost { border: 1px solid var(--hairline); background: transparent; color: var(--ink); }
  .lp-cta-ghost:hover { background: var(--surface-hover); }

  .lp-scroll-cue {
    display: flex; align-items: center; gap: var(--space-2);
    font-size: 13px; color: var(--ink-3);
  }
  .lp-cue-line { width: 28px; height: 1px; background: var(--ink-3); }

  /* ── Story ────────────────────────────────────────────────────────────── */
  .lp-story { position: relative; height: 700vh; }
  .lp-stage {
    position: sticky; top: 0; height: 100vh; height: 100svh;
    overflow: hidden;
    background: var(--map-base);
    color: var(--ink);
  }
  .lp-cam {
    position: absolute; left: 0; top: 0;
    width: 2000px; height: 1400px;
    transform-origin: 0 0;
    will-change: transform;
  }
  .lp-grid-major {
    position: absolute; inset: 0;
    background-image:
      linear-gradient(90deg, var(--map-street) 0 7px, transparent 7px),
      linear-gradient(0deg, var(--map-street) 0 7px, transparent 7px);
    background-size: 200px 100%, 100% 180px;
    background-position: 60px 0, 0 40px;
  }
  .lp-grid-minor {
    position: absolute; inset: 0;
    background-image:
      linear-gradient(90deg, var(--map-street) 0 2.5px, transparent 2.5px),
      linear-gradient(0deg, var(--map-street) 0 2.5px, transparent 2.5px);
    background-size: 50px 100%, 100% 60px;
    background-position: 20px 0, 0 20px;
    opacity: 0.75;
  }
  .lp-water {
    position: absolute; left: 1480px; top: 120px; width: 460px; height: 300px;
    background: var(--map-water); border-radius: 60% 40% 50% 50%;
  }
  .lp-park {
    position: absolute; left: 560px; top: 480px; width: 360px; height: 240px;
    background: var(--map-park); border-radius: 48% 52% 60% 40%;
  }
  .lp-routes { position: absolute; left: 0; top: 0; overflow: visible; pointer-events: none; }

  .lp-place { position: absolute; width: 0; height: 0; transform-origin: 0 0; z-index: 2; }
  .lp-place-label {
    position: absolute; left: 0; top: 0;
    white-space: nowrap;
    font-size: 12px; font-weight: 600; letter-spacing: 0.01em;
    color: var(--ink-2);
    padding: 2px 8px; border-radius: 999px;
    background: color-mix(in oklch, var(--card) 72%, transparent);
  }
  .lp-place-label.lp-area {
    font-size: 11px; letter-spacing: 0.1em; text-transform: uppercase;
    color: var(--ink-3); padding: 0; background: transparent;
  }

  .lp-person { position: absolute; width: 0; height: 0; transform-origin: 0 0; z-index: 4; }
  .lp-person.lp-self { z-index: 5; }
  .lp-person.lp-sos { z-index: 6; }
  .lp-person-inner {
    position: absolute; left: 0; top: 0;
    transform: translate(-50%, -22px);
    display: flex; flex-direction: column; align-items: center; gap: 6px;
  }
  .lp-pebble-wrap { position: relative; width: 44px; height: 44px; }
  .lp-sos-ring {
    position: absolute; left: 50%; top: 50%;
    width: 104px; height: 104px;
    transform: translate(-50%, -50%);
    border-radius: 50%;
    background: color-mix(in oklch, var(--vermilion) 16%, transparent);
    box-shadow: 0 0 0 26px color-mix(in oklch, var(--vermilion) 7%, transparent);
    opacity: 0;
    transition: opacity 0.4s;
  }
  .lp-sos-ring.on { opacity: 1; }
  .lp-pebble {
    position: relative; box-sizing: border-box;
    width: 44px; height: 44px; border-radius: 50%;
    border: 3px solid var(--card);
    box-shadow: 0 0 0 2px var(--sage), 0 4px 12px rgba(40, 30, 20, 0.25);
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-weight: 700; font-size: 15px;
    transition: background 0.4s, box-shadow 0.4s;
  }
  .lp-pebble.lp-quiet {
    color: var(--ink-2); opacity: 0.85;
    box-shadow: 0 4px 12px rgba(40, 30, 20, 0.22);
  }
  .lp-person.lp-sos .lp-pebble {
    box-shadow: 0 0 0 3px var(--card), 0 4px 12px rgba(40, 30, 20, 0.3);
  }
  .lp-tag {
    padding: 3px 9px; border-radius: 999px;
    background: var(--card);
    box-shadow: 0 1px 4px rgba(40, 30, 20, 0.18);
    font-size: 12px; font-weight: 600; color: var(--ink);
    white-space: nowrap;
    transition: background 0.4s;
  }
  .lp-tag-quiet { color: var(--ink-2); }
  .lp-tag-sos { background: var(--vermilion); color: #fff; font-weight: 700; }

  .lp-tint { position: absolute; inset: 0; pointer-events: none; }

  .lp-story-chrome {
    position: absolute; top: 0; left: 0; right: 0;
    padding: var(--space-5) var(--space-6);
    display: flex; justify-content: space-between; align-items: center;
    pointer-events: none;
  }
  .lp-clock {
    display: inline-flex; align-items: center; gap: 8px;
    padding: 8px 14px; border-radius: 999px;
    background: var(--card);
    box-shadow: var(--sh);
    font-size: 14px; font-weight: 600;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .lp-clock-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--sage); }

  .lp-hint {
    position: absolute; left: 50%; top: 76px;
    transform: translateX(-50%);
    padding: 8px 14px; border-radius: 999px;
    background: var(--card); box-shadow: var(--sh);
    font-size: 13px; font-weight: 600; color: var(--ink-2);
    opacity: 0; transition: opacity 0.3s;
    pointer-events: none; white-space: nowrap;
  }
  .lp-hint.visible { opacity: 1; }

  .lp-chapter {
    position: absolute; left: var(--space-6); bottom: var(--space-6);
    width: min(460px, calc(100vw - 48px));
    z-index: 10;
  }
  .lp-chapter-card {
    background: var(--card); color: var(--ink);
    border-radius: 24px;
    padding: 22px 24px 24px;
    box-shadow: var(--sh);
    display: flex; flex-direction: column; gap: 10px;
  }
  .lp-card-sos { background: var(--vermilion); color: #fff; }
  .lp-card-sage { background: oklch(0.93 0.04 150); }
  .lp-chapter-time {
    font-size: 11px; font-weight: 700;
    letter-spacing: 0.1em; text-transform: uppercase;
    opacity: 0.65;
    font-variant-numeric: tabular-nums;
  }
  .lp-chapter-title {
    margin: 0;
    font-family: var(--font-serif);
    font-style: italic;
    font-size: clamp(26px, 2.4vw, 32px);
    line-height: 1.05;
    letter-spacing: -0.01em;
    text-wrap: balance;
  }
  .lp-chapter-body {
    margin: 0;
    font-size: 15px; line-height: 1.5;
    opacity: 0.78;
    text-wrap: pretty;
  }

  /* ── Closing ──────────────────────────────────────────────────────────── */
  .lp-closing {
    padding: 120px var(--space-6) var(--space-12);
    display: flex; flex-direction: column; align-items: center;
    text-align: center;
    gap: var(--space-5);
  }
  .lp-closing-headline {
    margin: 0;
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: clamp(40px, 5.5vw, 72px);
    line-height: 1;
    letter-spacing: -0.015em;
  }
  .lp-closing-body {
    margin: 0;
    font-size: 17px; line-height: 1.55;
    color: var(--ink-2);
    max-width: 560px;
    text-wrap: pretty;
  }
  .lp-foot {
    display: flex; gap: var(--space-4); flex-wrap: wrap; justify-content: center;
    margin-top: 72px;
    font-size: 13px; color: var(--ink-3);
  }
  .lp-foot-mark { font-family: var(--font-serif); font-style: italic; font-size: 15px; }

  @media (prefers-reduced-motion: reduce) {
    .lp-sos-ring, .lp-pebble, .lp-tag, .lp-hint { transition: none; }
  }
</style>
