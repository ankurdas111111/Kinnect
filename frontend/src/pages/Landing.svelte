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
    { name: 'Home', x: 1000, y: 900, center: true, dy: -116 },
    { name: 'School', x: 560, y: 1000, dx: 32 },
    { name: 'Office', x: 1360, y: 440, dx: 32 },
    { name: 'Tuition', x: 740, y: 1160, dx: 32 },
    { name: 'Market', x: 1180, y: 760, dx: 32 },
    { name: 'Koramangala', x: 1230, y: 960, area: true },
    { name: 'Jayanagar', x: 560, y: 1100, area: true },
    { name: 'Indiranagar', x: 1360, y: 340, area: true },
    // Water/green labels sit at the feature's edge, the way maps caption
    // terrain — never centred inside it like a sticker.
    { name: 'Ulsoor Lake', x: 1622, y: 438, area: true },
    { name: 'Cubbon Park', x: 748, y: 748, area: true },
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
  // pRaw is where the scrollbar IS; pS is where the CAMERA is. pS chases pRaw
  // with exponential damping, which is what gives the scrub its weight —
  // wheel-step scrolling glides instead of stepping. Under reduced motion the
  // chase is skipped (damping is autonomous movement after input stops).
  let pRaw = $state(0);
  let pS = $state(0);
  let vw = $state(1200);
  let vh = $state(800);
  let storyEl = $state(null);
  let raf = 0;
  let chaseRaf = 0;

  function chase() {
    chaseRaf = 0;
    const d = pRaw - pS;
    if (Math.abs(d) < 0.0004) { pS = pRaw; return; }
    pS += d * 0.11;
    chaseRaf = requestAnimationFrame(chase);
  }

  function measure() {
    vw = window.innerWidth;
    vh = window.innerHeight;
    if (storyEl) {
      const r = storyEl.getBoundingClientRect();
      const total = r.height - vh;
      pRaw = total > 0 ? cl(-r.top / total) : 0;
    }
    if (prefersReducedMotion()) { pS = pRaw; return; }
    if (!chaseRaf) chaseRaf = requestAnimationFrame(chase);
  }
  function onScroll() {
    if (raf) return;
    raf = requestAnimationFrame(() => { raf = 0; measure(); });
  }

  onMount(() => {
    document.title = 'Kinnect — a quiet map for the people you love';
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll);
    measure();
    pS = pRaw;   // land where the page loads, no initial chase
    requestAnimationFrame(measure);
    return () => {
      document.title = 'Kinnect';
      window.removeEventListener('scroll', onScroll);
      window.removeEventListener('resize', onScroll);
      if (raf) cancelAnimationFrame(raf);
      if (chaseRaf) cancelAnimationFrame(chaseRaf);
    };
  });

  function goStory() {
    if (!storyEl) return;
    const y = storyEl.getBoundingClientRect().top + (window.scrollY || document.documentElement.scrollTop || 0);
    window.scrollTo({ top: y, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  }

  // ── The scene, derived from the DAMPED scroll ───────────────────────────
  let scene = $derived.by(() => {
    const p = pS;
    const t = keyed(TM, p)[0];
    const [cx, cy, s0] = keyed(CAM, t, true);
    const narrow = vw < 720;
    const s = s0 * Math.max(0.5, Math.min(1, vw / 1100));
    const tx = vw / 2 - cx * s;
    const ty = (narrow ? vh * 0.4 : vh / 2) - cy * s;
    const k = 1 / s;
    // SOS window desaturates the map (design 18.85–19.05). The tone itself is
    // carried by an EDGE tint (radial, clear centre) so the scene stays
    // legible at the exact moment legibility matters most.
    const ver = 0.34 * cl((t - 18.83) / 0.05) * (1 - cl((t - 19.03) / 0.06));

    const people = PEOPLE.map((per) => {
      const [x, y] = keyed(per.keys, t);
      // A pebble in transit lifts slightly — presence, not decoration.
      const [x2, y2] = keyed(per.keys, t + 0.02);
      const moving = Math.hypot(x2 - x, y2 - y) > 0.8;
      let tag = per.tags[0];
      for (const tg of per.tags) if (t >= tg[0]) tag = tg;
      const state = tag[2];
      return {
        initial: per.initial,
        label: per.self ? tag[1] : `${per.name} · ${tag[1]}`,
        x, y, moving,
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
      // Type choreography: the title leads, the body follows a beat behind.
      const eased = sm(o);
      return { ...c, o, cardY: (1 - eased) * 26, cardScale: 0.97 + eased * 0.03, titleY: (1 - eased) * 14, bodyO: cl((o - 0.35) / 0.65) };
    });

    // ambient tint: ochre while Nani is quiet, vermilion during SOS, sage after
    const och = 0.14 * cl((t - 16.55) / 0.15) * (1 - cl((t - 17.2) / 0.2));
    const sag = 0.18 * cl((t - 19.04) / 0.06) * (1 - cl((t - 19.25) / 0.15));
    let tint = { c: 'var(--ochre)', o: och };
    if (ver > tint.o) tint = { c: 'var(--vermilion)', o: ver };
    if (sag > tint.o) tint = { c: 'var(--sage)', o: sag };

    const hh = Math.floor(t), mm = Math.floor((t - hh) * 60);
    const clock = `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`;

    // Very end of the story: the map hands off to the closing on paper, no
    // hard cut — late enough that the final chapter gets its clean moment.
    const fade = cl((p - 0.965) / 0.035);

    return { t, tx, ty, s, k, ver, people, routes, chapters, tint, clock, fade, hint: p < 0.015 };
  });
</script>

<div class="lp">

  <!-- ═══ HERO ═══════════════════════════════════════════════════════════ -->
  <section class="lp-hero">
    <nav class="lp-nav" aria-label="Landing">
      <span class="lp-wordmark">Kinnect</span>
      <div class="lp-nav-actions">
        <button class="lp-nav-link" onclick={goStory}>How it works</button>
        <!-- Real links: middle-click / open-in-tab work, and App.svelte's
             anchor interception still gives them view transitions. -->
        <a class="lp-nav-link" href="#/login">Sign in</a>
        <a class="lp-nav-pill" href="#/register">Get the app</a>
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
        <a class="lp-cta lp-cta-primary" href="#/register">Create your family — free</a>
        <button class="lp-cta lp-cta-ghost" onclick={goStory}>See a day with one family ↓</button>
      </div>
    </div>

    <!-- The product, in miniature: a settled evening on the quiet map -->
    <div class="lp-hero-vignette" aria-hidden="true">
      <div class="lp-hv-head">
        <span class="lp-hv-dot"></span>
        <span class="lp-hv-verdict">Everyone&rsquo;s settled.</span>
        <span class="lp-hv-clock">19:30</span>
      </div>
      <div class="lp-hv-map">
        <div class="lp-hv-park"></div>
        <svg viewBox="0 0 340 260" class="lp-hv-route">
          <polyline points="56,224 56,156 150,156 150,118" fill="none" stroke="var(--member-2)"
            stroke-width="3" stroke-linecap="round" stroke-linejoin="round" stroke-dasharray="1 7" opacity="0.9" />
        </svg>
        <span class="lp-hv-place" style="left:252px; top:88px;">Home</span>
        <span class="lp-hv-place" style="left:34px; top:238px;">School</span>
        <div class="lp-hv-pebble" style="left:226px; top:120px; background:var(--ember);">P</div>
        <div class="lp-hv-pebble" style="left:252px; top:146px; background:var(--member-1);">A</div>
        <div class="lp-hv-pebble lp-hv-walking" style="left:150px; top:118px; background:var(--member-2);">M</div>
        <span class="lp-hv-tag">Meera · heading home</span>
      </div>
    </div>

    <div class="lp-scroll-cue"><span class="lp-cue-line" aria-hidden="true"></span>Scroll — a Tuesday with the Nair family, Bengaluru</div>
  </section>

  <!-- ═══ STORY (scroll-scrubbed) ════════════════════════════════════════ -->
  <section class="lp-story" bind:this={storyEl} aria-label="A day with the Nair family">
    <!-- The visual story is scroll-driven; this is its narrative for screen
         readers, so the section carries meaning without the animation. -->
    <p class="sr-only">
      A Tuesday with the Nair family in Bengaluru: everyone wakes at home.
      Meera walks to school and her pebble traces the route. Arjun drives to
      work — no pings, just where he is when you look. Through the middle of
      the day there is nothing to report, so Kinnect says nothing. At 16:40
      Nani's phone goes quiet — not an alarm, the page just warms — and a call
      settles it: she was napping. On her walk home from tuition Meera holds
      SOS; one sentence lands on every phone and Arjun, nine minutes away, is
      already moving. Twelve minutes later she's safe. By 19:30 everyone's
      settled — the day ends the way it started. Quiet.
    </p>
    <div class="lp-stage" aria-hidden="true">
      <div
        class="lp-cam"
        style:transform={`translate(${scene.tx}px, ${scene.ty}px) scale(${scene.s})`}
        style:filter={scene.ver > 0.01 ? `saturate(${1 - scene.ver})` : 'none'}
      >
        <div class="lp-grid-major" aria-hidden="true"></div>
        <div class="lp-grid-minor" aria-hidden="true"></div>

        <!-- Terrain: the city's fabric. Static SVG inside the camera layer —
             rasterised once, zero per-frame cost. Organic water/green (drawn
             paths, not border-radius ellipses), arterial roads cutting across
             the block grid, and low-contrast block clusters that thicken the
             neighbourhoods so the wide shots read as a place, not graph
             paper. -->
        <svg viewBox="0 0 2000 1400" width="2000" height="1400" class="lp-terrain" aria-hidden="true">
          <!-- city blocks — a breath of density around each neighbourhood -->
          <g class="lp-t-blocks">
            <rect x="1150" y="855" width="112" height="72" rx="3" />
            <rect x="1292" y="898" width="88" height="118" rx="3" />
            <rect x="1178" y="1002" width="132" height="78" rx="3" />
            <rect x="474" y="1058" width="118" height="88" rx="3" />
            <rect x="636" y="1148" width="98" height="70" rx="3" />
            <rect x="428" y="1206" width="88" height="108" rx="3" />
            <rect x="1382" y="298" width="108" height="78" rx="3" />
            <rect x="1502" y="432" width="88" height="60" rx="3" />
            <rect x="1298" y="418" width="78" height="98" rx="3" />
            <rect x="756" y="818" width="88" height="58" rx="3" />
            <rect x="878" y="702" width="68" height="88" rx="3" />
            <rect x="1602" y="598" width="118" height="88" rx="3" />
            <rect x="1698" y="758" width="88" height="68" rx="3" />
          </g>
          <!-- arterial roads — casing first, then the road -->
          <g class="lp-t-casing">
            <path d="M 140 1210 C 480 1108, 780 992, 1040 952 C 1300 912, 1450 640, 1560 470 C 1652 352, 1742 250, 1880 170" />
            <path d="M 1080 120 C 1112 400, 1074 700, 1112 1000 C 1128 1180, 1106 1290, 1120 1390" />
            <path d="M 150 1030 C 500 1006, 900 1062, 1300 1022 C 1600 992, 1800 1012, 1930 992" />
          </g>
          <g class="lp-t-roads">
            <path d="M 140 1210 C 480 1108, 780 992, 1040 952 C 1300 912, 1450 640, 1560 470 C 1652 352, 1742 250, 1880 170" />
            <path d="M 1080 120 C 1112 400, 1074 700, 1112 1000 C 1128 1180, 1106 1290, 1120 1390" />
            <path d="M 150 1030 C 500 1006, 900 1062, 1300 1022 C 1600 992, 1800 1012, 1930 992" />
          </g>
          <!-- water — a lake with an inlet and a southern point, not an egg -->
          <path class="lp-t-water" d="M 1594 208 C 1636 158, 1706 140, 1764 152 C 1788 157, 1794 176, 1822 170 C 1866 161, 1916 190, 1930 244 C 1941 288, 1918 316, 1926 348 C 1932 374, 1904 404, 1862 410 C 1820 416, 1792 396, 1758 408 C 1712 424, 1650 420, 1610 390 C 1576 364, 1568 330, 1580 300 C 1588 280, 1560 262, 1568 238 C 1573 224, 1582 218, 1594 208 Z" />
          <path class="lp-t-ripple" d="M 1648 260 q 58 -18 118 -4" />
          <path class="lp-t-ripple" d="M 1690 322 q 50 -14 102 -2" />
          <!-- green — the park with a bitten edge and a south-west lobe -->
          <path class="lp-t-green" d="M 626 544 C 662 490, 742 464, 806 476 C 848 484, 862 512, 896 520 C 926 528, 942 572, 928 608 C 918 634, 888 640, 884 664 C 879 692, 838 716, 794 716 C 762 716, 744 700, 712 708 C 668 719, 616 700, 598 662 C 584 632, 596 610, 588 588 C 582 570, 606 556, 626 544 Z" />
          <path class="lp-t-green" d="M 566 1066 C 596 1036, 654 1028, 690 1052 C 716 1069, 722 1096, 706 1118 C 688 1142, 640 1152, 602 1140 C 566 1128, 548 1094, 566 1066 Z" />
        </svg>

        <svg viewBox="0 0 2000 1400" width="2000" height="1400" class="lp-routes" aria-hidden="true">
          {#each scene.routes as r, i (i)}
            <polyline
              points={r.points} fill="none" stroke={r.color} stroke-width="3.5"
              stroke-linecap="round" stroke-linejoin="round"
              style:stroke-dasharray={r.dash}
              style:stroke-dashoffset={r.offset}
              style:opacity={r.opacity}
            />
          {/each}
        </svg>

        {#each PLACES as pl (pl.name)}
          <div class="lp-place" style:left={`${pl.x}px`} style:top={`${pl.y}px`} style:transform={`scale(${scene.k})`}>
            {#if !pl.area}
              <!-- anchor dot: the label points AT somewhere, not at nothing -->
              <span class="lp-place-dot" class:lp-dot-home={pl.center}></span>
            {/if}
            <span class="lp-place-label" class:lp-area={pl.area}
              style:transform={pl.area || pl.center ? `translate(-50%, calc(-50% + ${pl.dy || 0}px))` : `translate(${pl.dx}px, -50%)`}
            >{pl.name}</span>
          </div>
        {/each}

        {#each scene.people as per (per.initial)}
          <div class="lp-person lp-p-{per.initial}" class:lp-sos={per.sos} class:lp-self={per.self} class:lp-quiet-p={per.quiet}
            style:left={`${per.x}px`} style:top={`${per.y}px`} style:transform={`scale(${scene.k})`}>
            <div class="lp-person-inner" class:lp-moving={per.moving}>
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

      <div class="lp-tint" style:--tint-c={scene.tint.c} style:opacity={scene.tint.o} aria-hidden="true"></div>
      <div class="lp-stage-fade" style:opacity={scene.fade} aria-hidden="true"></div>

      <div class="lp-story-chrome" aria-hidden="true">
        <span class="lp-wordmark lp-wordmark-sm">Kinnect</span>
        <span class="lp-clock"><i class="lp-clock-dot"></i>{scene.clock} · Tuesday</span>
      </div>

      <div class="lp-hint" class:visible={scene.hint}>Keep scrolling — the day moves with you</div>

      {#each scene.chapters as c (c.time)}
        <div class="lp-chapter" style:opacity={c.o}
          style:transform={`translateY(${c.cardY}px) scale(${c.cardScale})`}
          style:pointer-events={c.o > 0.5 ? 'auto' : 'none'} aria-hidden={c.o < 0.1}>
          <div class="lp-chapter-card" class:lp-card-sos={c.sos} class:lp-card-sage={c.sage}>
            <span class="lp-chapter-time">{c.time}</span>
            <p class="lp-chapter-title" style:transform={`translateY(${c.titleY}px)`}>{c.title}</p>
            <p class="lp-chapter-body" style:opacity={c.bodyO}>{c.body}</p>
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- ═══ CLOSING ════════════════════════════════════════════════════════ -->
  <section class="lp-closing">
    <span class="lp-eyebrow reveal-scroll">That's the whole app</span>
    <h2 class="lp-closing-headline reveal-scroll">Everyone's settled.</h2>
    <p class="lp-closing-body reveal-scroll">
      One sentence, a few pebbles, and a hold-to-send SOS for the one evening in
      a thousand. Only people you invite can see anyone — ever.
    </p>
    <div class="lp-cta-row lp-cta-center reveal-scroll">
      <a class="lp-cta lp-cta-primary" href="#/register">Create your family — free</a>
      <a class="lp-cta lp-cta-ghost" href="#/register">Get the app</a>
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
    display: inline-flex; align-items: center;
    min-height: 44px; padding: 0 var(--space-1);
    border: none; background: transparent; cursor: pointer;
    color: var(--ink-2); font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    white-space: nowrap; text-decoration: none;
  }
  .lp-nav-link:hover { color: var(--ink); }
  .lp-nav-link:focus-visible, .lp-nav-pill:focus-visible, .lp-cta:focus-visible {
    outline: 2px solid var(--primary-400); outline-offset: 2px;
  }
  .lp-nav-pill {
    display: inline-flex; align-items: center; text-decoration: none;
    min-height: 44px; padding: 0 var(--space-4);
    border-radius: 999px;
    border: 1px solid color-mix(in oklch, var(--ink) 22%, transparent);
    background: var(--card); cursor: pointer;
    color: var(--ink); font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    white-space: nowrap;
    box-shadow: 0 1px 2px rgba(40, 30, 20, 0.08);
  }
  .lp-nav-pill:hover { border-color: color-mix(in oklch, var(--ink) 38%, transparent); }

  .lp-hero-body {
    flex: 1;
    display: flex; flex-direction: column; justify-content: center;
    gap: var(--space-6);
    max-width: 960px;
    padding: var(--space-12) 0;
  }
  .lp-eyebrow {
    font-size: var(--text-xs); font-weight: 600;
    letter-spacing: 0.12em; text-transform: uppercase;
    color: var(--ink-3);
  }
  /* an ember tick anchors the eyebrow — a mark, not a floating whisper */
  .lp-hero-body .lp-eyebrow { display: flex; align-items: center; gap: 12px; }
  .lp-hero-body .lp-eyebrow::before {
    content: ''; width: 26px; height: 2px; flex-shrink: 0;
    background: var(--ember); border-radius: 1px;
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
    display: inline-flex; align-items: center; justify-content: center;
    min-height: 52px; padding: 0 var(--space-5);
    border-radius: 999px; cursor: pointer;
    font-family: inherit; font-size: var(--text-base); font-weight: 600;
    white-space: nowrap; text-decoration: none;
    -webkit-tap-highlight-color: transparent;
  }
  .lp-cta-primary { border: none; background: var(--ember); color: var(--text-on-primary, #fff); }
  .lp-cta-primary:hover { background: var(--primary-600); }
  .lp-cta-ghost { border: 1px solid var(--hairline); background: transparent; color: var(--ink); }
  .lp-cta-ghost:hover { background: var(--surface-hover); }

  .lp-scroll-cue {
    display: flex; align-items: center; gap: var(--space-2);
    font-size: 13px; color: var(--ink-3);
  }
  .lp-cue-line {
    width: 28px; height: 1.5px; background: var(--ink-2);
    transform-origin: left center;
  }

  /* ── Hero entrance — the page arrives with intent ─────────────────────── */
  @media (prefers-reduced-motion: no-preference) {
    .lp-eyebrow, .lp-sub, .lp-cta-row, .lp-scroll-cue {
      opacity: 0;
      transform: translateY(14px);
      animation: lp-rise 640ms cubic-bezier(0.16, 1, 0.3, 1) both;
    }
    .lp-hero .lp-eyebrow { animation-delay: 80ms; }
    .lp-hero .lp-sub { animation-delay: 340ms; }
    .lp-hero .lp-cta-row { animation-delay: 460ms; }
    .lp-hero .lp-scroll-cue { animation-delay: 700ms; }
    /* the headline rises out of a clipped line box */
    .lp-headline {
      clip-path: inset(-4% -2% -8% -2%);
      opacity: 0;
      transform: translateY(0.35em);
      animation: lp-headline-rise 780ms cubic-bezier(0.16, 1, 0.3, 1) 160ms both;
    }
    .lp-cue-line { animation: lp-cue-sweep 2.6s ease-in-out 1.4s infinite; }
    .lp-hero-vignette {
      opacity: 0;
      animation: lp-hv-in 820ms cubic-bezier(0.16, 1, 0.3, 1) 520ms both;
    }
    /* the closing eyebrow/headline reuse .lp-eyebrow — but their reveal is
       scroll-driven via .reveal-scroll (global.css), so cancel the load-time
       rise there to avoid double animation */
    .lp-closing .lp-eyebrow, .lp-closing .lp-cta-row { animation: none; opacity: 1; transform: none; }
  }
  @keyframes lp-rise {
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes lp-headline-rise {
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes lp-cue-sweep {
    0%, 100% { transform: scaleX(1); opacity: 1; }
    50% { transform: scaleX(1.8); opacity: 0.45; }
  }
  @keyframes lp-hv-in {
    from { opacity: 0; transform: translateY(calc(-50% + 30px)) rotate(4.5deg); }
    to { opacity: 1; transform: translateY(-50%) rotate(2deg); }
  }

  /* ── Hero vignette — the product in miniature, floating right ─────────── */
  .lp-hero-vignette {
    display: none;
    position: absolute;
    right: clamp(24px, 5vw, 88px);
    top: 50%;
    width: min(400px, 30vw);
    background: var(--card);
    border: 1px solid var(--hairline);
    border-radius: 20px;
    box-shadow: var(--sh), 0 2px 8px rgba(40, 30, 20, 0.10);
    overflow: hidden;
    transform: translateY(-50%) rotate(2deg);
  }
  @media (min-width: 1080px) {
    .lp-hero-vignette { display: block; }
    .lp-hero-body { max-width: min(760px, 56vw); }
  }
  .lp-hv-head {
    display: flex; align-items: center; gap: 8px;
    padding: 14px 16px;
    border-bottom: 1px solid var(--hairline);
  }
  .lp-hv-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--sage); flex-shrink: 0; }
  .lp-hv-verdict {
    font-family: var(--font-serif); font-style: italic;
    font-size: 19px; color: var(--ink);
    flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .lp-hv-clock {
    font-size: 12px; font-weight: 600; color: var(--ink-3);
    font-variant-numeric: tabular-nums;
  }
  .lp-hv-map {
    position: relative; height: 260px;
    background:
      linear-gradient(90deg, var(--map-street) 0 2px, transparent 2px) 14px 0 / 76px 100%,
      linear-gradient(0deg, var(--map-street) 0 2px, transparent 2px) 0 10px / 100% 66px,
      linear-gradient(90deg, var(--map-street) 0 1px, transparent 1px) 4px 0 / 19px 100%,
      linear-gradient(0deg, var(--map-street) 0 1px, transparent 1px) 0 4px / 100% 17px,
      var(--map-base);
  }
  .lp-hv-park {
    position: absolute; left: -36px; top: 26px; width: 150px; height: 104px;
    background: var(--map-park);
    border-radius: 52% 48% 58% 42% / 55% 46% 54% 45%;
    box-shadow: inset 0 0 0 1.5px color-mix(in oklch, var(--ink) 7%, transparent);
  }
  .lp-hv-route { position: absolute; inset: 0; width: 100%; height: 100%; }
  .lp-hv-place {
    position: absolute;
    font-size: 10px; font-weight: 600; letter-spacing: 0.02em;
    color: var(--ink-2);
    padding: 1px 7px; border-radius: 999px;
    background: color-mix(in oklch, var(--card) 78%, transparent);
  }
  .lp-hv-pebble {
    position: absolute; box-sizing: border-box;
    width: 30px; height: 30px; margin: -15px 0 0 -15px;
    border-radius: 50%;
    border: 2px solid var(--card);
    box-shadow: 0 0 0 1.5px var(--sage), 0 2px 6px rgba(40, 30, 20, 0.25);
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-weight: 700; font-size: 11px;
  }
  .lp-hv-tag {
    position: absolute; left: 150px; top: 138px;
    transform: translateX(-50%);
    padding: 2px 8px; border-radius: 999px;
    background: var(--card);
    border: 1px solid var(--hairline);
    box-shadow: 0 1px 2px rgba(40, 30, 20, 0.14);
    font-size: 10.5px; font-weight: 600; color: var(--ink);
    white-space: nowrap;
  }

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
  /* City texture at street scale — thin, dense lines read as a map, not a
     spreadsheet. Avenues (3px) over lanes (1px). */
  .lp-grid-major {
    position: absolute; inset: 0;
    background-image:
      linear-gradient(90deg, var(--map-street) 0 3px, transparent 3px),
      linear-gradient(0deg, var(--map-street) 0 3px, transparent 3px);
    background-size: 150px 100%, 100% 130px;
    background-position: 60px 0, 0 40px;
  }
  .lp-grid-minor {
    position: absolute; inset: 0;
    background-image:
      linear-gradient(90deg, var(--map-street) 0 1px, transparent 1px),
      linear-gradient(0deg, var(--map-street) 0 1px, transparent 1px);
    background-size: 30px 100%, 100% 32px;
    background-position: 20px 0, 0 20px;
    opacity: 0.55;
  }
  .lp-terrain { position: absolute; left: 0; top: 0; pointer-events: none; }
  .lp-t-blocks rect { fill: color-mix(in oklch, var(--ink) 3.5%, transparent); }
  .lp-t-casing path {
    fill: none; stroke: color-mix(in oklch, var(--ink) 6%, transparent);
    stroke-width: 17; stroke-linecap: round;
  }
  .lp-t-roads path {
    fill: none; stroke: var(--map-street);
    stroke-width: 13; stroke-linecap: round;
  }
  .lp-t-water {
    fill: var(--map-water);
    stroke: color-mix(in oklch, var(--ink) 8%, transparent); stroke-width: 1.5;
  }
  .lp-t-ripple {
    fill: none; stroke: color-mix(in oklch, var(--ink) 10%, transparent);
    stroke-width: 1.5; stroke-linecap: round;
  }
  .lp-t-green {
    fill: var(--map-park);
    stroke: color-mix(in oklch, var(--ink) 8%, transparent); stroke-width: 1.5;
  }
  .lp-routes { position: absolute; left: 0; top: 0; overflow: visible; pointer-events: none; }

  .lp-place { position: absolute; width: 0; height: 0; transform-origin: 0 0; z-index: 2; }
  .lp-place-dot {
    position: absolute; left: -4px; top: -4px;
    width: 8px; height: 8px; border-radius: 50%;
    background: var(--card);
    box-shadow: 0 0 0 2px color-mix(in oklch, var(--ink) 30%, transparent);
  }
  .lp-dot-home {
    box-shadow: 0 0 0 2px var(--ember);
  }
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
  /* A quiet person is the story beat — she must not hide under "You" in the
     home cluster (later DOM order wins the z tie). */
  .lp-person.lp-quiet-p { z-index: 5; }
  .lp-person.lp-sos { z-index: 6; }
  .lp-person-inner {
    position: absolute; left: 0; top: 0;
    transform: translate(-50%, -22px);
    display: flex; flex-direction: column; align-items: center; gap: 6px;
  }
  .lp-pebble-wrap { position: relative; width: 40px; height: 40px; }
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
    width: 40px; height: 40px; border-radius: 50%;
    border: 2.5px solid var(--card);
    box-shadow: 0 0 0 2px var(--sage), 0 2px 8px rgba(40, 30, 20, 0.28);
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-weight: 700; font-size: 14px;
    transition: background 0.4s, box-shadow 0.4s;
  }
  .lp-pebble.lp-quiet {
    color: var(--ink); opacity: 0.9;
    box-shadow: 0 2px 8px rgba(40, 30, 20, 0.22);
  }
  /* In transit: the pebble lifts off the map a touch */
  .lp-person-inner { transition: transform 0.35s cubic-bezier(0.16, 1, 0.3, 1); }
  .lp-person-inner.lp-moving { transform: translate(-50%, -22px) scale(1.07); }
  .lp-person-inner.lp-moving .lp-pebble {
    box-shadow: 0 0 0 2px var(--sage), 0 8px 18px rgba(40, 30, 20, 0.32);
  }
  .lp-person.lp-sos .lp-pebble {
    box-shadow: 0 0 0 3px var(--card), 0 2px 8px rgba(40, 30, 20, 0.3);
  }
  .lp-tag {
    padding: 3px 9px; border-radius: 999px;
    background: var(--card);
    border: 1px solid var(--hairline);
    box-shadow: 0 1px 2px rgba(40, 30, 20, 0.14);
    font-size: 12px; font-weight: 600; color: var(--ink);
    white-space: nowrap;
    transition: background 0.4s;
  }
  .lp-tag-quiet { color: var(--ink-2); }
  .lp-tag-sos { background: var(--vermilion); border-color: transparent; color: #fff; font-weight: 700; }

  /* The home cluster: three pebbles a few px apart. Per-person tag placement
     keeps every name readable — Nani's rides above, the parents' tuck
     outward — instead of three tags stacking on one spot. */
  .lp-p-N .lp-person-inner { flex-direction: column-reverse; transform: translate(-50%, calc(-100% + 20px)); }
  .lp-p-A .lp-tag { transform: translateX(-16px); }
  .lp-p-M .lp-tag { transform: translateX(16px); }

  .lp-tint {
    position: absolute; inset: 0; pointer-events: none;
    background: radial-gradient(ellipse 90% 80% at 50% 45%,
      color-mix(in oklch, var(--tint-c) 35%, transparent) 0%,
      var(--tint-c) 78%);
  }
  .lp-stage-fade {
    position: absolute; inset: 0; pointer-events: none; z-index: 8;
    background: linear-gradient(to bottom, transparent 30%, var(--paper) 96%);
  }

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
    border: 1px solid var(--hairline);
    border-radius: 20px;
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
    .lp-sos-ring, .lp-pebble, .lp-tag, .lp-hint, .lp-person-inner { transition: none; }
    .lp-person-inner.lp-moving { transform: translate(-50%, -22px); }
  }

  /* Night mode now comes from the app-wide HEARTH NIGHT token block in
     tokens-hearth.css (:root[data-theme="dark"]) — --paper/--card/--ink and
     the map tokens all flip there, so the landing needs no scoped palette. */
</style>
