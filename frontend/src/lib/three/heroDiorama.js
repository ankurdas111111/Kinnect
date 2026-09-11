/**
 * heroDiorama — ONE continuous WebGL world for the whole landing.
 *
 * The board, park, blocks and family tokens live in the scroll-story's own
 * coordinate system (the 2000×1400 story world). The hero is simply the
 * camera dwelling near Home at a table angle while the beat engine plays the
 * day in miniature; scrolling descends the same camera into the same world,
 * where the scrub takes over: tokens walk their routes (fat-line trails draw
 * on in 3D), tones re-light the scene, SOS pulses a ground ring under Meera.
 * Chapters, clock and hint stay DOM — narration above the world.
 *
 * Loaded lazily; the DOM scene remains the no-WebGL fallback.
 */
import * as THREE from 'three';
import { Line2 } from 'three/addons/lines/Line2.js';
import { LineMaterial } from 'three/addons/lines/LineMaterial.js';
import { LineGeometry } from 'three/addons/lines/LineGeometry.js';

// Hearth palettes resolved to sRGB (tokens are oklch, which THREE.Color
// can't parse — these are the same colours, pre-converted).
const PALETTES = {
  light: {
    board: 0xf2eee6, street: 0xfdfcf9, faint: 0xf7f4ec,
    /* park/block carry a touch more contrast vs the board so the diorama
       reads as a place at hero distance, not blank graph paper */
    park: 0xcfe0c2, block: 0xe0dacb,
    ink: 0x2a231c, tagBg: '#ffffff', tagText: '#2a231c', tagMut: '#837a70',
    ember: 0xb0532c, m1: 0x4a7ba6, m2: 0x8d5a8f, m3: 0x3f7d78,
    quiet: 0xc7c0b4, vermilion: 0xb2392e, sage: 0x47795b, ochre: 0xb9822f,
    keyLight: 0xfff6e8, ambient: 0xfdf6ec, fog: 0xf2eee6,
  },
  dark: {
    board: 0x262019, street: 0x352d24, faint: 0x2b241c,
    park: 0x2c3526, block: 0x2d261e,
    ink: 0xf2ede4, tagBg: '#332b22', tagText: '#f2ede4', tagMut: '#b5aa9c',
    ember: 0xe08a5a, m1: 0x6d9fc9, m2: 0xb07fb2, m3: 0x5da39d,
    quiet: 0x6b6459, vermilion: 0xd45a4a, sage: 0x6fa384, ochre: 0xcf9b45,
    keyLight: 0xffe0b8, ambient: 0x8a7a66, fog: 0x262019,
  },
};

// ── ONE coordinate system: the story world (2000×1400) → board units ──────
const W = 170, D = 119, F = 3.2;
const sx = (x) => (x / 2000 - 0.5) * W;   // story-world px → board x
const sz = (y) => (y / 1400 - 0.5) * D;   // story-world px → board z
const px = (p) => (p / 100 - 0.5) * W;    // percent helpers (beats/paint)
const pz = (p) => (p / 100 - 0.5) * D;

function roundRect(ctx, x, y, w, h, r) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

export function initHeroDiorama(canvas, { reduced = false } = {}) {
  const theme = () =>
    document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
  let pal = PALETTES[theme()];

  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.shadowMap.enabled = true;
  renderer.shadowMap.type = THREE.PCFShadowMap;

  const scene = new THREE.Scene();
  /* Far edge sized for the portrait story pull-back (radius up to ~300) —
     at 460 the whole board fogged out on phones. */
  scene.fog = new THREE.Fog(pal.fog, 230, 620);

  const camera = new THREE.PerspectiveCamera(38, 1, 1, 900);

  // ── lights ────────────────────────────────────────────────────────────
  const ambient = new THREE.AmbientLight(pal.ambient, 1.15);
  const key = new THREE.DirectionalLight(pal.keyLight, 1.9);
  key.position.set(-70, 130, 60);
  key.castShadow = true;
  key.shadow.mapSize.set(2048, 2048);
  const sc = key.shadow.camera;
  sc.left = -160; sc.right = 160; sc.top = 160; sc.bottom = -160; sc.far = 420;
  key.shadow.bias = -0.0004;
  key.shadow.radius = 7;
  const sosLight = new THREE.PointLight(0xd45a4a, 0, 110, 2);
  scene.add(ambient, key, sosLight);

  // ── the board: the story's map painted in its own coordinates ─────────
  const mapCanvas = document.createElement('canvas');
  mapCanvas.width = 4096; mapCanvas.height = 2867; // matches W:D
  const mapTex = new THREE.CanvasTexture(mapCanvas);
  mapTex.anisotropy = Math.min(8, renderer.capabilities.getMaxAnisotropy());
  mapTex.colorSpace = THREE.SRGBColorSpace;

  const hex = (n) => '#' + n.toString(16).padStart(6, '0');

  function paintMap() {
    const c = mapCanvas.getContext('2d');
    const cw = mapCanvas.width, ch = mapCanvas.height;
    const gx = (p) => (0.5 + (p / 100 - 0.5) / F) * cw;   // story % → texture px
    const gy = (p) => (0.5 + (p / 100 - 0.5) / F) * ch;
    c.fillStyle = hex(pal.board); c.fillRect(0, 0, cw, ch);
    c.strokeStyle = hex(pal.faint); c.lineWidth = 1.5;
    for (let x = 0; x < cw; x += 42) { c.beginPath(); c.moveTo(x, 0); c.lineTo(x, ch); c.stroke(); }
    for (let y = 0; y < ch; y += 44) { c.beginPath(); c.moveTo(0, y); c.lineTo(cw, y); c.stroke(); }
    c.strokeStyle = hex(pal.street); c.lineWidth = 4;
    for (let x = 52; x < cw; x += 210) { c.beginPath(); c.moveTo(x, 0); c.lineTo(x, ch); c.stroke(); }
    for (let y = 38; y < ch; y += 186) { c.beginPath(); c.moveTo(0, y); c.lineTo(cw, y); c.stroke(); }
    // arterials sweeping the town
    c.strokeStyle = hex(pal.street); c.lineCap = 'round'; c.lineWidth = 34;
    c.beginPath(); c.moveTo(0, gy(92));
    c.bezierCurveTo(gx(26), gy(80), gx(46), gy(76), gx(58), gy(64));
    c.bezierCurveTo(gx(70), gy(52), gx(84), gy(30), gx(130), gy(-30)); c.stroke();
    c.beginPath(); c.moveTo(gx(56), -80);
    c.bezierCurveTo(gx(58), gy(30), gx(55.5), gy(62), gx(58), gy(150)); c.stroke();
    // Meera's usual way to school, painted faint on the map itself
    c.strokeStyle = hex(pal.m2); c.lineWidth = 7; c.setLineDash([2, 20]); c.lineCap = 'round';
    c.globalAlpha = 0.4;
    c.beginPath(); c.moveTo(gx(51.7), gy(65.9)); c.lineTo(gx(45), gy(65.9));
    c.lineTo(gx(45), gy(71.4)); c.lineTo(gx(28), gy(71.4)); c.stroke();
    c.setLineDash([]); c.globalAlpha = 1;
    // place names + anchor rings — story coordinates
    const places = [
      ['Home', 50, 61.4], ['School', 28, 69.3], ['Office', 68, 29.3],
      ['Market', 59, 52.2], ['Tuition', 37, 80.8],
    ];
    c.font = '600 27px "Instrument Sans", system-ui, sans-serif';
    c.textAlign = 'center';
    for (const [name, x, y] of places) {
      c.strokeStyle = hex(pal.ink); c.globalAlpha = 0.35; c.lineWidth = 3;
      c.beginPath(); c.arc(gx(x), gy(y) + 16, 5, 0, Math.PI * 2); c.stroke();
      c.globalAlpha = 0.5; c.fillStyle = hex(pal.ink);
      c.fillText(name, gx(x), gy(y) - 6);
      c.globalAlpha = 1;
    }
    c.font = '600 21px "Instrument Sans", system-ui, sans-serif';
    c.globalAlpha = 0.28;
    c.fillText('K O R A M A N G A L A', gx(64), gy(70));
    c.fillText('J A Y A N A G A R', gx(27), gy(84));
    c.fillText('I N D I R A N A G A R', gx(69), gy(22));
    c.fillText('C U B B O N   P A R K', gx(37), gy(46.5));
    c.globalAlpha = 1;
    mapTex.needsUpdate = true;
  }

  const board = new THREE.Mesh(
    new THREE.PlaneGeometry(W * F, D * F),
    new THREE.MeshStandardMaterial({ map: mapTex, roughness: 0.96, metalness: 0 })
  );
  board.rotation.x = -Math.PI / 2;
  board.receiveShadow = true;
  scene.add(board);

  // ── park + grove (extruded), city blocks ──────────────────────────────
  function blob(cx, cz, rx, rz, wobble = 0.22, seed = 1) {
    const s = new THREE.Shape();
    const N = 14;
    for (let i = 0; i <= N; i++) {
      const a = (i / N) * Math.PI * 2;
      const w = 1 + wobble * Math.sin(a * 3 + seed) * Math.cos(a * 2 - seed);
      s[i === 0 ? 'moveTo' : 'lineTo'](cx + Math.cos(a) * rx * w, cz + Math.sin(a) * rz * w);
    }
    s.closePath();
    return s;
  }
  const parkMat = new THREE.MeshStandardMaterial({ color: pal.park, roughness: 0.9 });
  function extrude(shape, h) {
    const g = new THREE.ExtrudeGeometry(shape, { depth: h, bevelEnabled: true, bevelSize: 0.7, bevelThickness: 0.5, bevelSegments: 2 });
    const m = new THREE.Mesh(g, parkMat);
    m.rotation.x = -Math.PI / 2;
    m.castShadow = true; m.receiveShadow = true;
    return m;
  }
  const park = extrude(blob(px(37), pz(40), 16, 9.4, 0.2, 2.1), 1.5);   // Cubbon Park
  const grove = extrude(blob(px(27), pz(88), 8, 5, 0.24, 4.7), 1.1);    // Jayanagar green
  scene.add(park, grove);

  const blockMat = new THREE.MeshStandardMaterial({ color: pal.block, roughness: 0.92 });
  const blocks = new THREE.Group();
  const BLOCKS = [
    [60, 66, 7, 5, 2.4], [65, 62, 5, 6, 3.4], [61, 74, 8, 5, 1.8],
    [30, 76, 7, 5, 2.6], [36, 82, 6, 4, 3.8], [24, 80, 5, 6, 2.2],
    [66, 26, 6, 5, 3.0], [72, 33, 5, 4, 2.0], [62, 30, 4.6, 6, 4.2],
    [46, 55, 5, 4, 1.6], [54, 44, 4, 5, 2.8], [70, 52, 7, 5, 2.4],
    [44, 34, 5, 4.4, 2.0], [52, 84, 6, 4, 3.2], [40, 60, 4.4, 4, 1.8],
  ];
  for (const [x, z, w, d, h] of BLOCKS) {
    const b = new THREE.Mesh(new THREE.BoxGeometry(w, h, d), blockMat);
    b.position.set(px(x), h / 2, pz(z));
    b.castShadow = true; b.receiveShadow = true;
    blocks.add(b);
  }
  scene.add(blocks);

  // ── the family tokens ─────────────────────────────────────────────────
  function makeTagTexture(text, kind) {
    const c = document.createElement('canvas');
    c.width = 512; c.height = 112;
    const x = c.getContext('2d');
    x.font = '600 44px "Instrument Sans", system-ui, sans-serif';
    const tw = Math.min(470, x.measureText(text).width + 56);
    const bg = kind === 'sos' ? hex(pal.vermilion) : pal.tagBg;
    const fg = kind === 'sos' ? '#ffffff' : kind === 'quiet' ? pal.tagMut : pal.tagText;
    x.fillStyle = bg;
    roundRect(x, 256 - tw / 2, 8, tw, 88, 44);
    x.fill();
    x.strokeStyle = 'rgba(42,35,28,0.14)'; x.lineWidth = 2; x.stroke();
    x.fillStyle = fg; x.textAlign = 'center'; x.textBaseline = 'middle';
    x.fillText(text, 256, 56);
    const t = new THREE.CanvasTexture(c);
    t.colorSpace = THREE.SRGBColorSpace;
    return t;
  }
  function makeLetterTexture(letter) {
    const c = document.createElement('canvas');
    c.width = 128; c.height = 128;
    const x = c.getContext('2d');
    x.font = '700 84px "Instrument Sans", system-ui, sans-serif';
    x.textAlign = 'center'; x.textBaseline = 'middle';
    x.fillStyle = '#ffffff';
    x.fillText(letter, 64, 70);
    const t = new THREE.CanvasTexture(c);
    t.colorSpace = THREE.SRGBColorSpace;
    return t;
  }

  const stoneGeo = new THREE.SphereGeometry(3.1, 28, 20);
  const ringGeo = new THREE.TorusGeometry(3.15, 0.42, 12, 36);
  const sosRingGeo = new THREE.TorusGeometry(5.4, 0.5, 10, 40);
  const NAMES = { P: 'You', A: 'Arjun', M: 'Meera', N: 'Nani' };

  const cast = {};
  for (const [k, colorKey] of [['P', 'ember'], ['A', 'm1'], ['M', 'm2'], ['N', 'm3']]) {
    const g = new THREE.Group();
    const stone = new THREE.Mesh(stoneGeo, new THREE.MeshStandardMaterial({ color: pal[colorKey], roughness: 0.55 }));
    stone.scale.set(1, 0.82, 1);
    stone.position.y = 2.7;
    stone.castShadow = true;
    const ring = new THREE.Mesh(ringGeo, new THREE.MeshStandardMaterial({ color: 0xffffff, roughness: 0.6 }));
    ring.rotation.x = Math.PI / 2; ring.position.y = 2.7; ring.scale.set(1, 1, 0.6);
    const letter = new THREE.Sprite(new THREE.SpriteMaterial({ map: makeLetterTexture(k), depthWrite: false, transparent: true }));
    letter.scale.set(3.4, 3.4, 1); letter.position.y = 3.1;
    const tag = new THREE.Sprite(new THREE.SpriteMaterial({ map: makeTagTexture(`${NAMES[k]} · home`, 'live'), depthWrite: false, transparent: true }));
    tag.scale.set(19, 4.2, 1);
    // lateral world offsets so tags fan out around a cluster from ANY camera
    // angle (pure height stagger collapses at top-down)
    const TAG_POS = { P: [0, 7.5, 7], A: [-9, 8.5, 2.5], M: [9, 8.5, 2.5], N: [0, 11, -6.5] };
    tag.position.set(...TAG_POS[k]);
    const sosRing = new THREE.Mesh(sosRingGeo, new THREE.MeshBasicMaterial({ color: pal.vermilion, transparent: true, opacity: 0 }));
    sosRing.rotation.x = -Math.PI / 2; sosRing.position.y = 0.25;
    g.add(stone, ring, letter, tag, sosRing);
    scene.add(g);
    cast[k] = {
      group: g, stone, ring, letter, tag, sosRing, colorKey,
      cur: new THREE.Vector3(), target: new THREE.Vector3(),
      tagText: '', tagKind: '', sos: false, quiet: false,
    };
  }
  function setMember(k, wx, wz, label, kind) {
    const m = cast[k];
    m.target.set(wx, 0, wz);
    m.sos = kind === 'sos';
    m.quiet = kind === 'quiet';
    if (label !== m.tagText || kind !== m.tagKind) {
      m.tag.material.map?.dispose();
      m.tag.material.map = makeTagTexture(label, kind);
      m.tag.material.needsUpdate = true;
      m.tagText = label; m.tagKind = kind;
    }
    m.stone.material.color.setHex(m.sos ? pal.vermilion : m.quiet ? pal.quiet : pal[m.colorKey]);
  }

  // ── story route trails: fat lines that draw on with the scrub ─────────
  const lineMats = [];
  let routes = [];
  function setRoutes(defs) {
    for (const r of routes) { scene.remove(r.line); r.line.geometry.dispose(); }
    routes = [];
    for (const def of defs) {
      const pts = [];
      for (let i = 0; i < def.pts.length - 1; i++) {
        const [ax, ay] = def.pts[i], [bx, by] = def.pts[i + 1];
        const segN = Math.max(2, Math.round(Math.hypot(bx - ax, by - ay) / 18));
        for (let j = 0; j < segN; j++) {
          const u = j / segN;
          pts.push(sx(ax + (bx - ax) * u), 1.1, sz(ay + (by - ay) * u));
        }
      }
      const [lx, ly] = def.pts[def.pts.length - 1];
      pts.push(sx(lx), 1.1, sz(ly));
      const geo = new LineGeometry();
      geo.setPositions(pts);
      const mat = new LineMaterial({
        color: def.color, linewidth: 3.5, transparent: true, opacity: 0,
        worldUnits: false, depthWrite: false,
      });
      lineMats.push(mat);
      const line = new Line2(geo, mat);
      line.computeLineDistances();
      scene.add(line);
      routes.push({ line, mat, segs: pts.length / 3 - 1 });
    }
    resize(); // set line material resolutions
  }

  // ── mode + camera state ───────────────────────────────────────────────
  let mode = 'hero';               // 'hero' | 'story'
  let flat = 1;                    // hero: 1 table view → 0 top-down
  let ptrX = 0, ptrY = 0;
  let tone = null;                 // hero tone key
  let storyTone = { key: null, strength: 0 };
  let storyCam = [1000, 900, 1.5];
  const HERO_TARGET = new THREE.Vector3(26, 0, 12);
  const camPos = new THREE.Vector3();
  const camTgt = new THREE.Vector3();
  let camSnap = true;

  const start = performance.now();
  const DROP_STAGGER = { P: 0.9, A: 1.05, M: 1.2, N: 1.35 };

  function applyBeat(hb) {
    if (mode !== 'hero') return;
    tone = hb.tone;
    for (const k of Object.keys(cast)) {
      const [bx, by, status] = hb[k];
      const kind = status === 'SOS' ? 'sos' : status === 'quiet' ? 'quiet' : 'live';
      setMember(k, px(bx), pz(by), `${NAMES[k]} · ${status}`, kind);
    }
  }

  function applyStory({ people, routes: rStates, tint, cam }) {
    mode = 'story';
    storyCam = cam;
    storyTone = tint;
    for (const per of people) {
      const k = per.initial;
      if (!cast[k]) continue;
      const kind = per.sos ? 'sos' : per.quiet ? 'quiet' : 'live';
      setMember(k, sx(per.x), sz(per.y), per.label, kind);
    }
    for (let i = 0; i < routes.length && i < rStates.length; i++) {
      const r = routes[i], st = rStates[i];
      r.line.geometry.instanceCount = Math.max(0, Math.round(r.segs * st.f));
      r.mat.opacity = st.o * 0.9;
      r.line.visible = st.o > 0.01;
    }
  }
  function setHeroMode() {
    if (mode === 'hero') return;
    mode = 'hero';
    for (const r of routes) { r.mat.opacity = 0; r.line.visible = false; }
  }

  let firstBeat = true;
  const toneColors = {
    null: { keyI: 1.9 },
    quiet: { tint: new THREE.Color(0xd9a54a), keyI: 1.65 },
    sos: { tint: new THREE.Color(0xd45a4a), keyI: 1.5 },
    safe: { tint: new THREE.Color(0x6fa384), keyI: 1.8 },
  };
  const baseAmb = new THREE.Color(pal.ambient);
  const curAmb = new THREE.Color(pal.ambient);

  let last = performance.now();
  let disposed = false;

  function frame(now) {
    const dt = Math.min(0.05, (now - last) / 1000);
    last = now;
    const t = (now - start) / 1000;

    // entrance envelopes (first load only)
    const boardIn = Math.min(1, t / 0.9);
    board.material.opacity = boardIn;
    board.material.transparent = boardIn < 1;
    park.scale.set(1, 1, Math.max(0.001, Math.min(1, (t - 0.25) / 0.7)));
    grove.scale.set(1, 1, Math.max(0.001, Math.min(1, (t - 0.4) / 0.7)));
    blocks.children.forEach((b, i) => {
      b.scale.y = Math.max(0.001, Math.min(1, (t - 0.35 - i * 0.045) / 0.55));
    });

    for (const k of Object.keys(cast)) {
      const m = cast[k];
      if (firstBeat) m.cur.copy(m.target);
      const dropT = Math.max(0, Math.min(1, (t - DROP_STAGGER[k]) / 0.7));
      const dropY = (1 - dropT) ** 2 * 46 - Math.sin(dropT * Math.PI) * 2.2;
      const dist = m.cur.distanceTo(m.target);
      // story scrub needs a snappier chase than the hero's leisurely glide
      const kSm = 1 - Math.exp(-dt * (mode === 'story' ? 6 : 2.1));
      m.cur.lerp(m.target, kSm);
      const walking = dist > 0.8;
      const hop = walking ? Math.abs(Math.sin(t * 7)) * 0.9 : 0;
      m.group.position.set(m.cur.x, Math.max(0, dropY) + hop, m.cur.z);
      m.group.visible = dropT > 0.001;
      const breathe = 1 + (walking ? 0 : Math.sin(t * 1.3 + DROP_STAGGER[k] * 5) * 0.02);
      m.stone.scale.set(breathe, 0.82 * breathe, breathe);
      m.tag.material.opacity = m.quiet ? 0.55 : 1;
      m.letter.material.opacity = m.quiet ? 0.7 : 1;
      if (m.sos) {
        const p = (t % 1.15) / 1.15;
        m.sosRing.scale.setScalar(0.6 + p * 1.1);
        m.sosRing.material.opacity = (1 - p) * 0.75;
      } else if (m.sosRing.material.opacity > 0) {
        m.sosRing.material.opacity = Math.max(0, m.sosRing.material.opacity - dt * 3);
      }
    }
    firstBeat = false;

    // tone lighting: hero beats (full strength) or story tint (scrub-driven)
    const tKey = mode === 'hero' ? tone : storyTone.key;
    const tStr = mode === 'hero' ? 1 : storyTone.strength;
    const tc = toneColors[tKey] || toneColors.null;
    const targetAmb = tc.tint ? baseAmb.clone().lerp(tc.tint, 0.34 * tStr) : baseAmb;
    curAmb.lerp(targetAmb, 1 - Math.exp(-dt * 3));
    ambient.color.copy(curAmb);
    const keyTarget = tc.keyI != null ? THREE.MathUtils.lerp(1.9, tc.keyI, tStr) : 1.9;
    key.intensity += (keyTarget - key.intensity) * (1 - Math.exp(-dt * 3));
    const sosOn = (mode === 'hero' && tone === 'sos') ||
      (mode === 'story' && storyTone.key === 'sos' && storyTone.strength > 0.4);
    if (sosOn) {
      const mm = cast.M;
      sosLight.position.set(mm.cur.x, 16, mm.cur.z);
      sosLight.intensity += (150 - sosLight.intensity) * (1 - Math.exp(-dt * 4));
    } else {
      sosLight.intensity += (0 - sosLight.intensity) * (1 - Math.exp(-dt * 4));
    }

    // camera: hero table view ←→ story scrub view, always smoothed
    let wantPosX, wantPosY, wantPosZ, tgt;
    if (mode === 'hero') {
      const polar = THREE.MathUtils.lerp(0.34, 1.02, flat) + ptrY * 0.05 * flat;
      const azim = -0.42 + Math.sin(t * 0.07) * 0.045 + ptrX * 0.075 * flat;
      const radius = THREE.MathUtils.lerp(215, 178, flat);
      // Wide viewports: aim left of the tokens so the inhabited neighbourhood
      // sits right of the headline instead of hiding behind it.
      tgt = camera.aspect > 1.1
        ? new THREE.Vector3(-14, 0, HERO_TARGET.z)
        : HERO_TARGET;
      wantPosX = tgt.x + radius * Math.sin(polar) * Math.sin(azim);
      wantPosY = radius * Math.cos(polar);
      wantPosZ = tgt.z + radius * Math.sin(polar) * Math.cos(azim);
    } else {
      const [cx, cy, s] = storyCam;
      tgt = new THREE.Vector3(sx(cx), 0, sz(cy));
      // Story keyframes were framed for ~16:10; portrait's narrower horizontal
      // FOV cropped family members at the viewport edge — pull back to keep
      // the framed WIDTH roughly constant across aspects.
      const aspectComp = Math.min(1.5, Math.max(1, Math.pow(1.45 / camera.aspect, 0.5)));
      const radius = (150 / s) * aspectComp;
      const polar = 0.32 + ptrY * 0.02;
      const azim = -0.18 + ptrX * 0.03;
      wantPosX = tgt.x + radius * Math.sin(polar) * Math.sin(azim);
      wantPosY = radius * Math.cos(polar);
      wantPosZ = tgt.z + radius * Math.sin(polar) * Math.cos(azim);
    }
    const ck = camSnap ? 1 : 1 - Math.exp(-dt * 6);
    camPos.lerp(new THREE.Vector3(wantPosX, wantPosY, wantPosZ), ck);
    camTgt.lerp(tgt, ck);
    camSnap = false;
    camera.position.copy(camPos);
    camera.lookAt(camTgt);

    renderer.render(scene, camera);
    if (reduced && t > 3.4) stopLoop();
  }

  // render only while the landing world is active/visible
  let running = false;
  let settled = false;
  function startLoop() {
    if (running || disposed || (reduced && settled)) return;
    running = true; last = performance.now();
    renderer.setAnimationLoop(frame);
  }
  function stopLoop() {
    if (reduced) settled = true;
    running = false;
    renderer.setAnimationLoop(null);
  }

  function resize() {
    const w = canvas.clientWidth || canvas.parentElement.clientWidth;
    const h = canvas.clientHeight || canvas.parentElement.clientHeight;
    if (!w || !h) return;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    for (const m of lineMats) m.resolution.set(w, h);
  }
  const ro = new ResizeObserver(resize);
  ro.observe(canvas.parentElement);
  resize();

  const mo = new MutationObserver(() => setTheme());
  mo.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
  function setTheme() {
    pal = PALETTES[theme()];
    paintMap();
    parkMat.color.setHex(pal.park);
    blockMat.color.setHex(pal.block);
    baseAmb.setHex(pal.ambient);
    key.color.setHex(pal.keyLight);
    scene.fog.color.setHex(pal.fog);
    for (const k of Object.keys(cast)) {
      const m = cast[k];
      m.tagText = '';
      m.stone.material.color.setHex(m.sos ? pal.vermilion : m.quiet ? pal.quiet : pal[m.colorKey]);
      m.sosRing.material.color.setHex(pal.vermilion);
    }
  }

  paintMap();
  if (document.fonts?.ready) document.fonts.ready.then(() => !disposed && paintMap());
  startLoop();

  return {
    setBeat: applyBeat,
    setStory: applyStory,
    setHeroMode,
    setRoutes,
    setPointer(x, y) { ptrX = x; ptrY = y; },
    setFlat(f) { flat = f; },
    setActive(on) { if (on) startLoop(); else stopLoop(); },
    destroy() {
      disposed = true;
      renderer.setAnimationLoop(null);
      ro.disconnect(); mo.disconnect();
      scene.traverse((o) => {
        o.geometry?.dispose?.();
        const mats = Array.isArray(o.material) ? o.material : o.material ? [o.material] : [];
        for (const m of mats) { m.map?.dispose?.(); m.dispose?.(); }
      });
      renderer.dispose();
    },
  };
}
