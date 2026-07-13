<script>
  import { run } from 'svelte/legacy';

  /**
   * GlobeCanvas v3 — Dot-Matrix Data Globe
   *
   * Inspired by cobe / GitHub Globe / Vercel globe.
   * Uses Spherical Fibonacci sampling for ~12 000 evenly-distributed dots,
   * per-dot Phong lighting, great-circle connection arcs with traveling
   * pulses, night-side city lights, and rich atmospheric glow.
   * Pure Canvas 2D — no WebGL.
   */
  import { onMount, onDestroy } from 'svelte';
  import { myLocation, otherUsers } from '../../lib/stores/map.js';
  import { resolveMemberColor } from '../../lib/getUserColor.js';

  /**
   * @typedef {Object} Props
   * @property {number} [size]
   */

  /** @type {Props} */
  let { size = 340 } = $props();

  let canvas = $state();
  let rafId;
  let t0 = 0, elapsed = 0, lastFrame = 0;

  let rotY = $state(0.38), rotX = $state(0.22);
  let isDragging = $state(false);
  let dragVelY = 0, dragVelX = 0;
  let lastPX = 0, lastPY = 0;
  let interacted = $state(false);
  let hasAutoFaced = $state(false);

  const AUTO_SPEED = 0.0018;
  const DEG = Math.PI / 180;

  let R  = $derived(size * 0.44);
  let CX = $derived(size / 2);
  let CY = $derived(size / 2);

  run(() => {
    if ($myLocation?.latitude != null && $myLocation?.longitude != null && !hasAutoFaced) {
      rotY = Math.PI / 2 - $myLocation.longitude * DEG;
      rotX = Math.atan(Math.sin($myLocation.latitude * DEG)) * 0.85;
      hasAutoFaced = true;
    }
  });

  // ── Light direction (view-space, upper-left) ──────────────────────────
  const LX = -0.48, LY = 0.44, LZ = 0.76;

  // ── Continent polygons (for land detection) ───────────────────────────
  const LAND = [
    [[37,10],[30,32],[22,36],[15,41],[11,44],[5,42],[0,42],[-5,40],
     [-10,38],[-18,35],[-27,33],[-35,19],[-34,26],[-22,33],[-14,37],
     [3,40],[11,44],[15,40],[11,16],[5,1],[5,-4],[10,-14],[15,-17],
     [25,-15],[32,-10],[37,10]],
    [[71,27],[62,26],[55,24],[50,27],[47,37],[42,41],[40,36],[36,27],
     [36,22],[40,20],[38,15],[44,12],[44,8],[43,-8],[47,-8],[48,0],
     [52,2],[57,6],[57,12],[57,20],[55,24],[60,27],[65,25],[68,22],[71,27]],
    [[72,28],[72,60],[72,100],[72,140],[66,173],[60,170],[55,162],
     [50,140],[45,130],[40,121],[35,120],[38,110],[42,100],[40,90],
     [35,80],[28,82],[22,72],[20,68],[25,62],[30,57],[35,50],[40,44],
     [43,40],[47,37],[50,27],[55,24],[60,27],[65,25],[68,22],[72,28]],
    [[28,82],[22,88],[20,87],[17,82],[13,80],[10,79],[8,80],[8,77],
     [10,76],[12,75],[15,74],[18,73],[20,73],[22,70],[24,68],[28,70],[28,82]],
    [[22,100],[15,100],[10,104],[5,103],[1,104],[-7,107],[-8,115],
     [-8,120],[-5,125],[2,110],[5,103],[10,100],[15,103],[22,100]],
    [[72,-85],[72,-120],[68,-167],[60,-162],[55,-130],[50,-124],[45,-124],
     [40,-124],[35,-120],[30,-110],[24,-105],[20,-87],[15,-83],[10,-83],
     [9,-77],[14,-84],[20,-88],[24,-107],[30,-110],[35,-120],[40,-124],
     [45,-76],[50,-60],[55,-66],[62,-68],[66,-80],[70,-78],[75,-100],
     [80,-80],[84,-65],[80,-50],[75,-65],[70,-76],[60,-65],[50,-56],
     [47,-53],[45,-66],[45,-76],[46,-84],[46,-90],[44,-93],[49,-97],
     [49,-117],[60,-140],[68,-167],[72,-85]],
    [[10,-74],[6,-52],[0,-49],[-5,-36],[-15,-38],[-23,-43],[-33,-52],
     [-40,-62],[-50,-65],[-55,-68],[-55,-70],[-50,-75],[-45,-73],
     [-40,-73],[-35,-72],[-30,-70],[-20,-70],[-15,-75],[-10,-78],
     [0,-80],[5,-77],[10,-74]],
    [[-20,114],[-25,113],[-32,116],[-37,139],[-38,145],[-38,148],
     [-33,152],[-27,153],[-20,149],[-18,146],[-15,145],[-12,141],
     [-13,136],[-15,130],[-14,127],[-16,122],[-20,118],[-20,114]],
    [[83,-45],[80,-25],[75,-18],[70,-23],[68,-28],[65,-38],[64,-52],
     [67,-55],[72,-60],[76,-65],[78,-70],[82,-50],[83,-45]],
    [[-13,50],[-18,44],[-25,43],[-26,45],[-24,48],[-20,48],[-13,50]],
    [[34,130],[36,136],[38,140],[40,141],[42,140],[40,141],[38,140],[36,136],[34,130]],
    [[58,-3],[55,-5],[52,-5],[51,1],[53,0],[55,-2],[57,-5],[58,-3]],
    [[-44,168],[-46,168],[-46,169],[-44,171],[-42,172],[-40,175],[-41,174],[-44,168]],
    [[66,-14],[64,-22],[63,-18],[63,-13],[65,-13],[66,-14]],
    [[-70,-180],[-70,-90],[-70,0],[-70,90],[-70,180],
     [-90,180],[-90,0],[-90,-180],[-70,-180]],
  ];

  // ── City lights [lat, lon, scale] ─────────────────────────────────────
  const CITIES = [
    [40.7,-74,1.4],[34.1,-118,1.3],[41.9,-87.6,1.2],[29.8,-95.4,1.0],
    [47.6,-122,0.9],[37.8,-122,1.0],[43.7,-79.4,1.0],[25.8,-80.2,0.9],
    [19.4,-99.1,1.2],[51.5,-0.1,1.3],[48.9,2.4,1.2],[52.5,13.4,1.1],
    [55.8,37.6,1.2],[40.4,-3.7,1.0],[41.9,12.5,1.0],[41.0,29.0,1.1],
    [35.7,139.7,1.5],[39.9,116.4,1.4],[31.2,121.5,1.3],[22.5,114.1,1.2],
    [1.4,103.8,1.2],[37.6,126.9,1.2],[28.6,77.2,1.3],[19.1,72.9,1.2],
    [23.1,113.3,1.1],[13.8,100.5,1.1],[25.2,55.3,1.0],[24.7,46.7,1.0],
    [30.0,31.2,1.1],[-26.2,28.1,1.0],[6.5,3.4,1.1],[-23.5,-46.6,1.2],
    [-34.6,-58.4,1.1],[-33.9,151.2,0.9],[-37.8,145.0,0.8],[6.2,106.8,1.1],
  ];

  // ── Point-in-polygon (ray casting) ────────────────────────────────────
  function pointInPoly(lat, lon, poly) {
    let inside = false;
    for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
      const [yi, xi] = poly[i], [yj, xj] = poly[j];
      if (((yi > lat) !== (yj > lat)) &&
          (lon < (xj - xi) * (lat - yi) / (yj - yi) + xi))
        inside = !inside;
    }
    return inside;
  }
  function isLand(lat, lon) {
    for (const poly of LAND) if (pointInPoly(lat, lon, poly)) return true;
    return false;
  }

  // ── Spherical Fibonacci dot grid ──────────────────────────────────────
  // Pre-computed once: {x,y,z} on unit sphere + land flag.
  const DOT_N = 12000;
  const GOLDEN = Math.PI * (3 - Math.sqrt(5));
  const DOTS = (() => {
    const arr = [];
    for (let i = 0; i < DOT_N; i++) {
      const y = 1 - (2 * i + 1) / DOT_N;
      const r = Math.sqrt(1 - y * y);
      const theta = GOLDEN * i;
      const x = Math.cos(theta) * r;
      const z = Math.sin(theta) * r;
      const lat = Math.asin(y) / DEG;
      const lon = Math.atan2(z, x) / DEG;
      arr.push({ x, y, z, land: isLand(lat, lon) });
    }
    return arr;
  })();

  // ── Star field ────────────────────────────────────────────────────────
  const STAR_COLS = ['rgba(220,235,255,','rgba(200,215,255,','rgba(255,248,210,','rgba(255,225,185,','rgba(245,245,255,'];
  const STARS = (() => {
    const s = []; let seed = 47831;
    const rnd = () => { seed ^= seed << 13; seed ^= seed >> 17; seed ^= seed << 5; return (seed >>> 0) / 0xffffffff; };
    for (let i = 0; i < 300; i++)
      s.push([rnd()*Math.PI*2, 1.06+rnd()*0.82, 0.18+rnd()*0.88, 0.14+rnd()*0.68,
              Math.floor(rnd()*5), rnd()*Math.PI*2, 0.4+rnd()*1.4]);
    return s;
  })();

  // ── Projection (for pins / arcs) ──────────────────────────────────────
  function latLonTo3D(lat, lon) {
    const phi = lat * DEG, lam = lon * DEG;
    return { x: Math.cos(phi)*Math.cos(lam), y: Math.sin(phi), z: Math.cos(phi)*Math.sin(lam) };
  }
  function project(lat, lon) {
    const p = latLonTo3D(lat, lon);
    const x1 = p.x * Math.cos(rotY) - p.z * Math.sin(rotY);
    const z1 = p.x * Math.sin(rotY) + p.z * Math.cos(rotY);
    const y2 = p.y * Math.cos(rotX) - z1 * Math.sin(rotX);
    const z2 = p.y * Math.sin(rotX) + z1 * Math.cos(rotX);
    return { sx: CX + x1 * R, sy: CY - y2 * R, depth: z2 };
  }

  // ── Great-circle arc renderer ─────────────────────────────────────────
  function drawArc(ctx, lat1, lon1, lat2, lon2, color, cRY, sRY, cRX, sRX) {
    const a = latLonTo3D(lat1, lon1), b = latLonTo3D(lat2, lon2);
    const dot = Math.max(-1, Math.min(1, a.x*b.x + a.y*b.y + a.z*b.z));
    const omega = Math.acos(dot);
    if (omega < 0.02) return; // too close
    const sinO = Math.sin(omega) || 0.001;
    const lift = omega * 0.15;
    const N = 48;
    const pts = [];
    for (let i = 0; i <= N; i++) {
      const t = i / N;
      const sa = Math.sin((1-t)*omega) / sinO;
      const sb = Math.sin(t*omega) / sinO;
      const h = 1 + lift * Math.sin(t * Math.PI);
      const x = (sa*a.x + sb*b.x) * h;
      const y = (sa*a.y + sb*b.y) * h;
      const z = (sa*a.z + sb*b.z) * h;
      const x1 = x*cRY - z*sRY;
      const z1 = x*sRY + z*cRY;
      const y2 = y*cRX - z1*sRX;
      const z2 = y*sRX + z1*cRX;
      pts.push({ sx: CX + x1*R, sy: CY - y2*R, depth: z2 });
    }

    // Draw arc line
    ctx.beginPath();
    let started = false;
    for (const p of pts) {
      if (p.depth < -0.08) { started = false; continue; }
      if (!started) { ctx.moveTo(p.sx, p.sy); started = true; }
      else ctx.lineTo(p.sx, p.sy);
    }
    ctx.strokeStyle = color; ctx.lineWidth = 1.2;
    ctx.globalAlpha = 0.50; ctx.stroke();
    ctx.lineWidth = 3.5; ctx.globalAlpha = 0.08; ctx.stroke(); // glow
    ctx.globalAlpha = 1;

    // Traveling pulse dot
    const tIdx = Math.floor(((elapsed * 0.28) % 1) * pts.length);
    const tp = pts[Math.min(tIdx, pts.length - 1)];
    if (tp.depth > 0) {
      const pg = ctx.createRadialGradient(tp.sx, tp.sy, 0, tp.sx, tp.sy, 7);
      pg.addColorStop(0, color); pg.addColorStop(1, 'transparent');
      ctx.globalAlpha = 0.45; ctx.fillStyle = pg;
      ctx.beginPath(); ctx.arc(tp.sx, tp.sy, 7, 0, Math.PI*2); ctx.fill();
      ctx.globalAlpha = 0.9; ctx.fillStyle = color;
      ctx.beginPath(); ctx.arc(tp.sx, tp.sy, 2, 0, Math.PI*2); ctx.fill();
      ctx.globalAlpha = 1;
    }
  }

  // ── Main draw ─────────────────────────────────────────────────────────
  function drawGlobe(ctx) {
    ctx.clearRect(0, 0, size, size);

    // ── Stars ─────────────────────────────────────────────────────────
    for (const [a, r, sz, al, ci, tp, ts] of STARS) {
      const sx = CX + Math.cos(a)*R*r, sy = CY + Math.sin(a)*R*r;
      if (sx < 0 || sx > size || sy < 0 || sy > size) continue;
      const tw = 0.72 + 0.28*Math.sin(elapsed*ts + tp);
      ctx.beginPath(); ctx.arc(sx, sy, sz*tw, 0, Math.PI*2);
      ctx.fillStyle = STAR_COLS[ci] + (al*tw) + ')'; ctx.fill();
    }

    // ── Atmosphere outer corona ───────────────────────────────────────
    const ao = ctx.createRadialGradient(CX, CY, R*0.82, CX, CY, R*1.72);
    ao.addColorStop(0,    'rgba(28,115,255,0.28)');
    ao.addColorStop(0.20, 'rgba(22,98,248,0.16)');
    ao.addColorStop(0.50, 'rgba(14,72,220,0.07)');
    ao.addColorStop(0.82, 'rgba(8,50,190,0.02)');
    ao.addColorStop(1,    'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R*1.72, 0, Math.PI*2);
    ctx.fillStyle = ao; ctx.fill();

    // ── Atmosphere inner limb ─────────────────────────────────────────
    const al = ctx.createRadialGradient(CX, CY, R*0.89, CX, CY, R*1.12);
    al.addColorStop(0,    'transparent');
    al.addColorStop(0.32, 'rgba(72,170,255,0.58)');
    al.addColorStop(0.70, 'rgba(50,145,248,0.30)');
    al.addColorStop(1,    'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R*1.12, 0, Math.PI*2);
    ctx.fillStyle = al; ctx.fill();

    // ── Dark ocean sphere ─────────────────────────────────────────────
    const oc = ctx.createRadialGradient(CX - R*0.20, CY - R*0.28, 0, CX, CY, R);
    oc.addColorStop(0,    '#163858');
    oc.addColorStop(0.18, '#102a44');
    oc.addColorStop(0.45, '#081a30');
    oc.addColorStop(0.75, '#04101f');
    oc.addColorStop(1,    '#020812');
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2);
    ctx.fillStyle = oc; ctx.fill();

    // ── Clip to sphere ────────────────────────────────────────────────
    ctx.save();
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2); ctx.clip();

    // ── Rotation matrix (pre-compute once) ────────────────────────────
    const cRY = Math.cos(rotY), sRY = Math.sin(rotY);
    const cRX = Math.cos(rotX), sRX = Math.sin(rotX);
    const dotScale = R / 155;

    // ── Dot grid — project & collect ──────────────────────────────────
    // We project all dots, then draw ocean and land in separate batches
    // for efficiency.
    const landProj = [];

    // Ocean dots (very faint, give sphere structure)
    ctx.fillStyle = '#3270a8';
    for (let i = 0; i < DOT_N; i++) {
      const gp = DOTS[i];
      const x1 = gp.x*cRY - gp.z*sRY;
      const z1 = gp.x*sRY + gp.z*cRY;
      const y2 = gp.y*cRX - z1*sRX;
      const z2 = gp.y*sRX + z1*cRX;
      if (z2 < 0.01) continue;

      const sx = CX + x1*R;
      const sy = CY - y2*R;

      // Per-dot Phong lighting (diffuse)
      const diffuse = Math.max(0, LX*x1 + LY*y2 + LZ*z2);
      const d = z2; // depth factor

      if (gp.land) {
        landProj.push({ sx, sy, d, diffuse });
      } else {
        const s = (0.42 + 0.38*d) * dotScale;
        ctx.globalAlpha = (0.018 + 0.055*d) * (0.3 + 0.7*diffuse);
        ctx.fillRect(sx - s*0.5, sy - s*0.5, s, s);
      }
    }
    ctx.globalAlpha = 1;

    // Land dots — glow pass (larger, dim)
    ctx.fillStyle = 'rgba(120,200,255,1)';
    for (const p of landProj) {
      const s = (2.2 + 2.0*p.d) * dotScale;
      ctx.globalAlpha = (0.04 + 0.10*p.d) * (0.15 + 0.85*p.diffuse);
      ctx.fillRect(p.sx - s*0.5, p.sy - s*0.5, s, s);
    }

    // Land dots — core pass (small, bright)
    ctx.fillStyle = '#c0ecff';
    for (const p of landProj) {
      const s = (0.6 + 0.9*p.d) * dotScale;
      ctx.globalAlpha = (0.22 + 0.72*p.d) * (0.12 + 0.88*p.diffuse);
      ctx.fillRect(p.sx - s*0.5, p.sy - s*0.5, s, s);
    }
    ctx.globalAlpha = 1;

    // ── Night overlay (before city lights) ────────────────────────────
    const ni = ctx.createRadialGradient(CX + R*0.50, CY + R*0.42, 0, CX + R*0.16, CY + R*0.12, R*1.18);
    ni.addColorStop(0,    'rgba(0,2,16,0.78)');
    ni.addColorStop(0.26, 'rgba(0,2,16,0.48)');
    ni.addColorStop(0.56, 'rgba(0,2,16,0.16)');
    ni.addColorStop(1,    'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2);
    ctx.fillStyle = ni; ctx.fill();

    // ── City lights (only night side) ─────────────────────────────────
    for (const [lat, lon, scale] of CITIES) {
      const p = project(lat, lon);
      if (p.depth < 0.04) continue;
      // Reconstruct view-space normal for lighting check
      const nx = (p.sx - CX) / R;
      const ny = (CY - p.sy) / R;
      const sd = LX*nx + LY*ny + LZ*p.depth;
      if (sd > 0.0) continue; // lit side
      const darkness = Math.min(1, -sd / 0.28);
      const ca = darkness * scale * 0.78;
      const cr = (1.2 + scale*1.4) * (R / 185);
      // Glow
      const gg = ctx.createRadialGradient(p.sx, p.sy, 0, p.sx, p.sy, cr*3.5);
      gg.addColorStop(0,   `rgba(255,198,100,${ca*0.55})`);
      gg.addColorStop(0.4, `rgba(255,168,60,${ca*0.22})`);
      gg.addColorStop(1,   'rgba(255,130,20,0)');
      ctx.beginPath(); ctx.arc(p.sx, p.sy, cr*3.5, 0, Math.PI*2);
      ctx.fillStyle = gg; ctx.fill();
      ctx.beginPath(); ctx.arc(p.sx, p.sy, cr, 0, Math.PI*2);
      ctx.fillStyle = `rgba(255,228,160,${ca})`; ctx.fill();
    }

    // ── Connection arcs ───────────────────────────────────────────────
    if ($myLocation?.latitude != null && $myLocation?.longitude != null) {
      for (const user of $otherUsers.values()) {
        if (!user.lat || !user.lng) continue;
        drawArc(ctx, $myLocation.latitude, $myLocation.longitude,
                user.lat, user.lng, resolveMemberColor(user.userId), cRY, sRY, cRX, sRX);
      }
    }

    // ── Family pins ───────────────────────────────────────────────────
    for (const user of $otherUsers.values()) {
      if (!user.lat || !user.lng) continue;
      const p = project(user.lat, user.lng);
      if (p.depth <= 0.02) continue;
      const col  = resolveMemberColor(user.userId);
      const ph   = (elapsed*1.4 + user.userId.charCodeAt(0)*0.3) % (Math.PI*2);
      const pulse = 7 + Math.sin(ph)*2;
      const pGr  = ctx.createRadialGradient(p.sx, p.sy, 0, p.sx, p.sy, pulse);
      pGr.addColorStop(0, 'rgba(200,185,255,0.45)'); pGr.addColorStop(1, 'transparent');
      ctx.beginPath(); ctx.arc(p.sx, p.sy, pulse, 0, Math.PI*2);
      ctx.fillStyle = pGr; ctx.fill();
      ctx.beginPath(); ctx.arc(p.sx, p.sy, 3, 0, Math.PI*2);
      ctx.fillStyle = col; ctx.fill();
      ctx.strokeStyle = 'rgba(255,255,255,0.5)'; ctx.lineWidth = 1;
      ctx.stroke();
    }

    // ── My location pin ───────────────────────────────────────────────
    if ($myLocation?.latitude != null && $myLocation?.longitude != null) {
      const p = project($myLocation.latitude, $myLocation.longitude);
      if (p.depth > 0.03) {
        // Expanding rings
        for (let i = 0; i < 2; i++) {
          const ph = ((elapsed*1.6 + i*Math.PI) % (Math.PI*2)) / (Math.PI*2);
          const rr = 9 + ph*16, aa = 0.55*(1-ph);
          ctx.beginPath(); ctx.arc(p.sx, p.sy, rr, 0, Math.PI*2);
          ctx.strokeStyle = `rgba(167,139,250,${aa})`; ctx.lineWidth = 1.5*(1-ph);
          ctx.stroke();
        }
        // Glow
        const gr = ctx.createRadialGradient(p.sx, p.sy, 0, p.sx, p.sy, 13);
        gr.addColorStop(0, 'rgba(196,181,253,0.70)');
        gr.addColorStop(0.4, 'rgba(167,139,250,0.30)');
        gr.addColorStop(1, 'transparent');
        ctx.beginPath(); ctx.arc(p.sx, p.sy, 13, 0, Math.PI*2);
        ctx.fillStyle = gr; ctx.fill();
        // Core
        ctx.beginPath(); ctx.arc(p.sx, p.sy, 4, 0, Math.PI*2);
        ctx.fillStyle = '#ddd6fe'; ctx.fill();
        ctx.strokeStyle = 'rgba(255,255,255,0.80)'; ctx.lineWidth = 1.5; ctx.stroke();
        // Label
        ctx.save();
        ctx.font = `bold ${Math.round(size*0.030)}px monospace`;
        ctx.textAlign = 'center';
        ctx.fillStyle = 'rgba(196,181,253,0.92)';
        ctx.fillText('YOU', p.sx, p.sy - 18);
        ctx.strokeStyle = 'rgba(196,181,253,0.35)'; ctx.lineWidth = 1;
        ctx.beginPath(); ctx.moveTo(p.sx, p.sy - 4); ctx.lineTo(p.sx, p.sy - 12); ctx.stroke();
        ctx.restore();
      }
    }

    // ── Sun highlight on ocean ────────────────────────────────────────
    const sh = ctx.createRadialGradient(CX - R*0.40, CY - R*0.38, 0, CX - R*0.05, CY - R*0.05, R*1.05);
    sh.addColorStop(0, 'rgba(255,250,230,0.12)');
    sh.addColorStop(0.22, 'rgba(255,242,198,0.06)');
    sh.addColorStop(0.55, 'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2);
    ctx.fillStyle = sh; ctx.fill();

    ctx.restore(); // end sphere clip

    // ── Specular glint ────────────────────────────────────────────────
    const sp = ctx.createRadialGradient(CX - R*0.38, CY - R*0.44, 0, CX - R*0.10, CY - R*0.14, R*0.72);
    sp.addColorStop(0, 'rgba(255,255,255,0.24)');
    sp.addColorStop(0.20, 'rgba(240,248,255,0.10)');
    sp.addColorStop(0.52, 'rgba(210,228,255,0.02)');
    sp.addColorStop(1, 'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2);
    ctx.fillStyle = sp; ctx.fill();

    // ── Limb darkening ────────────────────────────────────────────────
    const li = ctx.createRadialGradient(CX, CY, R*0.46, CX, CY, R);
    li.addColorStop(0, 'transparent');
    li.addColorStop(0.56, 'rgba(0,4,20,0.06)');
    li.addColorStop(0.80, 'rgba(0,4,20,0.50)');
    li.addColorStop(1, 'rgba(0,4,20,0.92)');
    ctx.beginPath(); ctx.arc(CX, CY, R, 0, Math.PI*2);
    ctx.fillStyle = li; ctx.fill();

    // ── Atmosphere halo ───────────────────────────────────────────────
    const ha = ctx.createRadialGradient(CX, CY, R*0.936, CX, CY, R*1.078);
    ha.addColorStop(0,    'transparent');
    ha.addColorStop(0.30, 'rgba(90,182,255,0.68)');
    ha.addColorStop(0.68, 'rgba(58,150,244,0.35)');
    ha.addColorStop(1,    'transparent');
    ctx.beginPath(); ctx.arc(CX, CY, R*1.078, 0, Math.PI*2);
    ctx.fillStyle = ha; ctx.fill();
  }

  // ── Animation loop ────────────────────────────────────────────────────
  function tick(ts) {
    rafId = requestAnimationFrame(tick);
    if (ts - lastFrame < 22) return;
    lastFrame = ts;
    if (!t0) t0 = ts;
    elapsed = (ts - t0) / 1000;

    if (!isDragging) {
      rotY += AUTO_SPEED + dragVelY;
      rotX += dragVelX;
      dragVelY *= 0.90;
      dragVelX *= 0.90;
    }
    rotX = Math.max(-0.68, Math.min(0.68, rotX));

    if (!canvas) return;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    if (canvas.width !== size*dpr || canvas.height !== size*dpr) {
      canvas.width = size*dpr; canvas.height = size*dpr;
    }
    const ctx = canvas.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    drawGlobe(ctx);
  }

  // ── Pointer ───────────────────────────────────────────────────────────
  function onPointerDown(e) {
    isDragging = true; interacted = true;
    lastPX = e.clientX; lastPY = e.clientY;
    e.currentTarget.setPointerCapture(e.pointerId);
  }
  function onPointerMove(e) {
    if (!isDragging) return;
    const dx = e.clientX - lastPX, dy = e.clientY - lastPY;
    dragVelY = dx * 0.013; dragVelX = -dy * 0.013;
    rotY += dragVelY;
    rotX = Math.max(-0.68, Math.min(0.68, rotX + dragVelX));
    lastPX = e.clientX; lastPY = e.clientY;
  }
  function onPointerUp() { isDragging = false; }

  onMount(() => {
    let started = false;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting && !started) {
        started = true;
        rafId = requestAnimationFrame(tick);
      }
    }, { threshold: 0.1 });
    if (canvas) observer.observe(canvas);
    return () => observer.disconnect();
  });
  onDestroy(() => { if (rafId) cancelAnimationFrame(rafId); });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="gc-wrap" style="--sz:{size}px">
  <canvas
    bind:this={canvas}
    width={size} height={size}
    class="gc-canvas"
    class:gc-dragging={isDragging}
    onpointerdown={onPointerDown}
    onpointermove={onPointerMove}
    onpointerup={onPointerUp}
    onpointercancel={onPointerUp}
    aria-label="Interactive Earth — drag to rotate"
  ></canvas>
  {#if !interacted}
    <span class="gc-hint" aria-hidden="true">drag to spin</span>
  {/if}
</div>

<style>
  .gc-wrap {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    user-select: none;
    -webkit-user-select: none;
  }
  .gc-canvas {
    width: var(--sz); height: var(--sz);
    display: block;
    border-radius: 50%;
    cursor: grab;
    filter:
      drop-shadow(0 0 28px rgba(40,120,255,0.38))
      drop-shadow(0 0 8px  rgba(20,75,210,0.48))
      drop-shadow(0 0 70px rgba(12,55,190,0.16));
    transition: filter 0.35s ease;
  }
  .gc-canvas:hover {
    filter:
      drop-shadow(0 0 38px rgba(55,138,255,0.54))
      drop-shadow(0 0 12px rgba(30,88,220,0.60))
      drop-shadow(0 0 88px rgba(18,65,200,0.24));
  }
  .gc-dragging { cursor: grabbing !important; }
  .gc-hint {
    font-size: 8px; font-weight: 700;
    color: rgba(255,255,255,0.14);
    letter-spacing: 0.10em; text-transform: uppercase;
    pointer-events: none;
    animation: gc-hint-fade 5.5s ease-in-out forwards;
  }
  @keyframes gc-hint-fade {
    0%{opacity:0} 12%{opacity:1} 78%{opacity:1} 100%{opacity:0}
  }
</style>
