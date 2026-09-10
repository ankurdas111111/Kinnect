/**
 * heroDiorama — the landing hero's quiet map as a real WebGL diorama.
 *
 * A warm paper board (map texture: grid, arterials, dotted route, painted
 * place names), an extruded park and low city blocks, and the family as
 * standing stone tokens with real lighting and soft shadows. The Svelte beat
 * engine drives everything: tokens glide (with a walking hop) between the
 * day's places, tags follow as billboarded sprites, tones re-light the scene,
 * SOS pulses a ground ring. Camera: oblique table view with idle drift and
 * pointer parallax, descending to top-down as the visitor scrolls into the
 * scroll-story (setFlat 1 → 0 mirrors the DOM world's --flat).
 *
 * Loaded lazily (dynamic import) so three.js never touches the initial
 * bundle; the DOM scene remains as the no-WebGL fallback.
 */
import * as THREE from 'three';

// Hearth palettes resolved to sRGB (tokens are oklch, which THREE.Color
// can't parse — these are the same colours, pre-converted).
const PALETTES = {
  light: {
    board: 0xf2eee6, street: 0xfdfcf9, faint: 0xf7f4ec,
    park: 0xd9e6d0, block: 0xe8e3d8, edge: 0xd8d2c4,
    ink: 0x2a231c, tagBg: '#ffffff', tagText: '#2a231c', tagMut: '#837a70',
    ember: 0xb0532c, m1: 0x4a7ba6, m2: 0x8d5a8f, m3: 0x3f7d78,
    quiet: 0xc7c0b4, vermilion: 0xb2392e, sage: 0x47795b, ochre: 0xb9822f,
    keyLight: 0xfff6e8, ambient: 0xfdf6ec, fog: 0xf2eee6,
  },
  dark: {
    board: 0x262019, street: 0x352d24, faint: 0x2b241c,
    park: 0x2c3526, block: 0x2d261e, edge: 0x1d1812,
    ink: 0xf2ede4, tagBg: '#332b22', tagText: '#f2ede4', tagMut: '#b5aa9c',
    ember: 0xe08a5a, m1: 0x6d9fc9, m2: 0xb07fb2, m3: 0x5da39d,
    quiet: 0x6b6459, vermilion: 0xd45a4a, sage: 0x6fa384, ochre: 0xcf9b45,
    keyLight: 0xffe0b8, ambient: 0x8a7a66, fog: 0x262019,
  },
};

// scene-space mapping: beat coords are % of a 1600×900 world
const W = 170, D = 96, F = 3.2; // F: board oversize past the world
const px = (p) => (p / 100 - 0.5) * W;
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
  renderer.shadowMap.type = THREE.PCFSoftShadowMap;

  const scene = new THREE.Scene();
  scene.fog = new THREE.Fog(pal.fog, 190, 420);

  const camera = new THREE.PerspectiveCamera(38, 1, 1, 800);
  const CAM_TARGET = new THREE.Vector3(34, 0, 2);

  // ── lights ────────────────────────────────────────────────────────────
  const ambient = new THREE.AmbientLight(pal.ambient, 1.15);
  const key = new THREE.DirectionalLight(pal.keyLight, 1.9);
  key.position.set(-70, 120, 60);
  key.castShadow = true;
  key.shadow.mapSize.set(2048, 2048);
  const sc = key.shadow.camera;
  sc.left = -140; sc.right = 140; sc.top = 140; sc.bottom = -140; sc.far = 400;
  key.shadow.bias = -0.0004;
  key.shadow.radius = 7;
  const sosLight = new THREE.PointLight(0xd45a4a, 0, 90, 2);
  scene.add(ambient, key, sosLight);

  // ── the board: map painted onto a big ground plane ────────────────────
  const mapCanvas = document.createElement('canvas');
  mapCanvas.width = 4096; mapCanvas.height = 2320;
  const mapTex = new THREE.CanvasTexture(mapCanvas);
  mapTex.anisotropy = Math.min(8, renderer.capabilities.getMaxAnisotropy());
  mapTex.colorSpace = THREE.SRGBColorSpace;

  function hex(n) { return '#' + n.toString(16).padStart(6, '0'); }

  function paintMap() {
    const c = mapCanvas.getContext('2d');
    const cw = mapCanvas.width, ch = mapCanvas.height;
    // world % -> texture px: the canvas spans F world-widths, world centred
    const gx = (p) => (0.5 + (p / 100 - 0.5) / F) * cw;
    const gy = (p) => (0.5 + (p / 100 - 0.5) / F) * ch;
    c.fillStyle = hex(pal.board); c.fillRect(0, 0, cw, ch);
    // fine lanes
    c.strokeStyle = hex(pal.faint); c.lineWidth = 1.5;
    for (let x = 0; x < cw; x += 42) { c.beginPath(); c.moveTo(x, 0); c.lineTo(x, ch); c.stroke(); }
    for (let y = 0; y < ch; y += 44) { c.beginPath(); c.moveTo(0, y); c.lineTo(cw, y); c.stroke(); }
    // avenues
    c.strokeStyle = hex(pal.street); c.lineWidth = 4;
    for (let x = 52; x < cw; x += 210) { c.beginPath(); c.moveTo(x, 0); c.lineTo(x, ch); c.stroke(); }
    for (let y = 38; y < ch; y += 186) { c.beginPath(); c.moveTo(0, y); c.lineTo(cw, y); c.stroke(); }
    // arterials
    c.strokeStyle = hex(pal.street); c.lineCap = 'round'; c.lineWidth = 34;
    c.beginPath(); c.moveTo(0, gy(88));
    c.bezierCurveTo(gx(24), gy(71), gx(50), gy(72), gx(66), gy(60));
    c.bezierCurveTo(gx(82), gy(49), gx(100), gy(24), gx(150), gy(-20)); c.stroke();
    c.beginPath(); c.moveTo(gx(69), 0);
    c.bezierCurveTo(gx(72.5), gy(28), gx(70.5), gy(60), gx(73), gy(160)); c.stroke();
    // Meera's dotted route (school → home)
    c.strokeStyle = hex(pal.m2); c.lineWidth = 7; c.setLineDash([2, 20]); c.lineCap = 'round';
    c.beginPath(); c.moveTo(gx(62), gy(68.5)); c.lineTo(gx(65.2), gy(60));
    c.lineTo(gx(65.2), gy(52)); c.lineTo(gx(76), gy(52)); c.lineTo(gx(78.5), gy(47.5)); c.stroke();
    c.setLineDash([]);
    // painted place names + anchor rings
    const places = [['Home', 74.6, 40.5], ['School', 62, 72.5], ['Office', 88, 21.5], ['Market', 62, 34]];
    c.font = '600 27px "Instrument Sans", system-ui, sans-serif';
    c.textAlign = 'center';
    for (const [name, x, y] of places) {
      c.strokeStyle = hex(pal.ink); c.globalAlpha = 0.35; c.lineWidth = 3;
      c.beginPath(); c.arc(gx(x), gy(y) + 16, 5, 0, Math.PI * 2); c.stroke();
      c.globalAlpha = 0.5; c.fillStyle = hex(pal.ink);
      c.fillText(name, gx(x), gy(y) - 6);
      c.globalAlpha = 1;
    }
    // neighbourhood names, whisper-quiet
    c.font = '600 21px "Instrument Sans", system-ui, sans-serif';
    c.globalAlpha = 0.28;
    c.fillText('K O R A M A N G A L A', gx(84), gy(74));
    c.fillText('J A Y A N A G A R', gx(30), gy(88));
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

  // ── park: extruded organic blob + a grove ─────────────────────────────
  function blob(cx, cz, rx, rz, wobble = 0.22, seed = 1) {
    const s = new THREE.Shape();
    const N = 14;
    for (let i = 0; i <= N; i++) {
      const a = (i / N) * Math.PI * 2;
      const w = 1 + wobble * Math.sin(a * 3 + seed) * Math.cos(a * 2 - seed);
      const x = cx + Math.cos(a) * rx * w;
      const y = cz + Math.sin(a) * rz * w;
      if (i === 0) s.moveTo(x, y); else s.lineTo(x, y);
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
  const park = extrude(blob(px(56), pz(77), 15, 9, 0.2, 2.1), 1.5);
  const grove = extrude(blob(px(44), pz(30), 8, 5.4, 0.24, 4.7), 1.1);
  scene.add(park, grove);

  // ── low city blocks — the town the family lives in ────────────────────
  const blockMat = new THREE.MeshStandardMaterial({ color: pal.block, roughness: 0.92 });
  const blocks = new THREE.Group();
  const BLOCKS = [
    [80, 66, 7, 5, 2.4], [86, 61, 5, 6, 3.4], [82, 74, 8, 5, 1.8],
    [30, 74, 7, 5, 2.6], [38, 80, 6, 4, 3.8], [26, 84, 5, 6, 2.2],
    [86, 30, 6, 5, 3.0], [92, 38, 5, 4, 2.0], [81, 34, 4.6, 6, 4.2],
    [48, 56, 5, 4, 1.6], [55, 46, 4, 5, 2.8], [90, 52, 7, 5, 2.4],
    [42, 38, 5, 4.4, 2.0], [58, 24, 6, 4, 3.2],
  ];
  for (const [x, z, w, d, h] of BLOCKS) {
    const b = new THREE.Mesh(new THREE.BoxGeometry(w, h, d), blockMat);
    b.position.set(px(x), h / 2, pz(z));
    b.castShadow = true; b.receiveShadow = true;
    blocks.add(b);
  }
  scene.add(blocks);

  // ── the family: standing stone tokens + billboarded tags ──────────────
  function makeTagTexture(text, kind) {
    const c = document.createElement('canvas');
    c.width = 512; c.height = 112;
    const x = c.getContext('2d');
    x.clearRect(0, 0, 512, 112);
    x.font = '600 44px "Instrument Sans", system-ui, sans-serif';
    const tw = Math.min(470, x.measureText(text).width + 56);
    const bg = kind === 'sos' ? hex(pal.vermilion) : pal.tagBg;
    const fg = kind === 'sos' ? '#ffffff' : kind === 'quiet' ? pal.tagMut : pal.tagText;
    x.fillStyle = bg;
    roundRect(x, 256 - tw / 2, 8, tw, 88, 44);
    x.fill();
    x.strokeStyle = 'rgba(42,35,28,0.14)'; x.lineWidth = 2; x.stroke();
    x.fillStyle = fg;
    x.textAlign = 'center'; x.textBaseline = 'middle';
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

  const cast = {};
  const CAST_DEF = [
    ['P', 'ember'], ['A', 'm1'], ['M', 'm2'], ['N', 'm3'],
  ];
  for (const [k, colorKey] of CAST_DEF) {
    const g = new THREE.Group();
    const stone = new THREE.Mesh(stoneGeo, new THREE.MeshStandardMaterial({ color: pal[colorKey], roughness: 0.55 }));
    stone.scale.set(1, 0.82, 1);
    stone.position.y = 2.7;
    stone.castShadow = true;
    const ring = new THREE.Mesh(ringGeo, new THREE.MeshStandardMaterial({ color: 0xffffff, roughness: 0.6 }));
    ring.rotation.x = Math.PI / 2;
    ring.position.y = 2.7;
    ring.scale.set(1, 1, 0.6);
    const letter = new THREE.Sprite(new THREE.SpriteMaterial({ map: makeLetterTexture(k), depthWrite: false }));
    letter.scale.set(3.4, 3.4, 1);
    letter.position.y = 3.1;
    const tag = new THREE.Sprite(new THREE.SpriteMaterial({ map: makeTagTexture('', 'live'), depthWrite: false }));
    tag.scale.set(19, 4.2, 1);
    tag.position.y = { P: 9.4, A: 13.2, M: 11.4, N: 13.2 }[k];
    const sosRing = new THREE.Mesh(sosRingGeo, new THREE.MeshBasicMaterial({ color: pal.vermilion, transparent: true, opacity: 0 }));
    sosRing.rotation.x = -Math.PI / 2;
    sosRing.position.y = 0.25;
    g.add(stone, ring, letter, tag, sosRing);
    scene.add(g);
    cast[k] = {
      group: g, stone, ring, letter, tag, sosRing, colorKey,
      cur: new THREE.Vector3(), target: new THREE.Vector3(),
      tagText: '', tagKind: 'live', sos: false, quiet: false, dropAt: 0,
    };
  }

  // ── state fed from Svelte ─────────────────────────────────────────────
  let flat = 1;               // 1 = table view, 0 = top-down (story handoff)
  let ptrX = 0, ptrY = 0;     // -1..1 pointer parallax
  let tone = null;            // null | quiet | sos | safe
  const start = performance.now();
  const DROP_STAGGER = { P: 0.9, A: 1.05, M: 1.2, N: 1.35 };

  function applyBeat(hb) {
    tone = hb.tone;
    for (const k of Object.keys(cast)) {
      const m = cast[k];
      const [bx, by, status] = hb[k];
      m.target.set(px(bx), 0, pz(by));
      m.sos = status === 'SOS';
      m.quiet = status === 'quiet';
      const names = { P: 'You', A: 'Arjun', M: 'Meera', N: 'Nani' };
      const text = `${names[k]} · ${status}`;
      const kind = m.sos ? 'sos' : m.quiet ? 'quiet' : 'live';
      if (text !== m.tagText || kind !== m.tagKind) {
        m.tag.material.map?.dispose();
        m.tag.material.map = makeTagTexture(text, kind);
        m.tag.material.needsUpdate = true;
        m.tagText = text; m.tagKind = kind;
      }
      const targetColor = m.sos ? pal.vermilion : m.quiet ? pal.quiet : pal[m.colorKey];
      m.stone.material.color.setHex(targetColor);
    }
  }

  // initial placement = first applied beat, teleport (no glide from origin)
  let firstBeat = true;

  // ── camera + loop ─────────────────────────────────────────────────────
  const toneColors = {
    null: { amb: null, keyI: 1.9 },
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

    // entrance envelopes
    const boardIn = Math.min(1, t / 0.9);
    board.material.opacity = boardIn;
    board.material.transparent = boardIn < 1;
    const parkIn = Math.max(0.001, Math.min(1, (t - 0.25) / 0.7));
    park.scale.set(1, 1, parkIn);        // extrude depth axis after rotation
    grove.scale.set(1, 1, Math.max(0.001, Math.min(1, (t - 0.4) / 0.7)));
    blocks.children.forEach((b, i) => {
      const bi = Math.max(0.001, Math.min(1, (t - 0.35 - i * 0.045) / 0.55));
      b.scale.y = bi;
    });

    // tokens: drop in, then glide with a walking hop
    for (const k of Object.keys(cast)) {
      const m = cast[k];
      if (firstBeat) m.cur.copy(m.target);
      const dropT = Math.max(0, Math.min(1, (t - DROP_STAGGER[k]) / 0.7));
      const dropY = (1 - dropT) ** 2 * 46 - Math.sin(dropT * Math.PI) * 2.2;
      const dist = m.cur.distanceTo(m.target);
      const kSmooth = 1 - Math.exp(-dt * 2.1);
      m.cur.lerp(m.target, kSmooth);
      const walking = dist > 0.8;
      const hop = walking ? Math.abs(Math.sin(t * 7)) * 0.9 : 0;
      m.group.position.set(m.cur.x, Math.max(0, dropY) + hop, m.cur.z);
      m.group.visible = dropT > 0.001;
      // idle breath
      const breathe = 1 + (walking ? 0 : Math.sin(t * 1.3 + DROP_STAGGER[k] * 5) * 0.02);
      m.stone.scale.set(breathe, 0.82 * breathe, breathe);
      // sos ring pulse
      if (m.sos) {
        const p = (t % 1.15) / 1.15;
        m.sosRing.scale.setScalar(0.6 + p * 1.1);
        m.sosRing.material.opacity = (1 - p) * 0.75;
      } else if (m.sosRing.material.opacity > 0) {
        m.sosRing.material.opacity = Math.max(0, m.sosRing.material.opacity - dt * 3);
      }
    }
    firstBeat = false;

    // tone lighting
    const tc = toneColors[tone] || toneColors.null;
    const targetAmb = tc.tint ? baseAmb.clone().lerp(tc.tint, 0.34) : baseAmb;
    curAmb.lerp(targetAmb, 1 - Math.exp(-dt * 3));
    ambient.color.copy(curAmb);
    key.intensity += ((tc.keyI ?? 1.9) - key.intensity) * (1 - Math.exp(-dt * 3));
    if (tone === 'sos') {
      const mm = cast.M;
      sosLight.position.set(mm.cur.x, 16, mm.cur.z);
      sosLight.intensity += (140 - sosLight.intensity) * (1 - Math.exp(-dt * 4));
    } else {
      sosLight.intensity += (0 - sosLight.intensity) * (1 - Math.exp(-dt * 4));
    }

    // camera: oblique ←→ top-down, idle drift, pointer parallax
    const polar = THREE.MathUtils.lerp(0.34, 1.02, flat) + ptrY * 0.05 * flat;
    const azim = -0.42 + Math.sin(t * 0.07) * 0.045 + ptrX * 0.075 * flat;
    const radius = THREE.MathUtils.lerp(215, 178, flat);
    camera.position.set(
      CAM_TARGET.x + radius * Math.sin(polar) * Math.sin(azim),
      CAM_TARGET.y + radius * Math.cos(polar),
      CAM_TARGET.z + radius * Math.sin(polar) * Math.cos(azim)
    );
    camera.lookAt(CAM_TARGET);

    renderer.render(scene, camera);
    if (reduced && t > 3.4) stopLoop(); // settle, then hold still
  }

  // Render only while the hero is on screen — offscreen frames are pure
  // waste and can starve the story's scrub loop on weak GPUs.
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
  const io = new IntersectionObserver((es) => {
    if (es[0].isIntersecting) startLoop(); else stopLoop();
  }, { threshold: 0.01 });
  io.observe(canvas);

  function resize() {
    const w = canvas.clientWidth || canvas.parentElement.clientWidth;
    const h = canvas.clientHeight || canvas.parentElement.clientHeight;
    if (!w || !h) return;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }
  const ro = new ResizeObserver(resize);
  ro.observe(canvas.parentElement);
  resize();

  // theme swaps repaint the whole palette
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
      m.tagText = ''; // force tag re-render in new palette
      m.stone.material.color.setHex(m.sos ? pal.vermilion : m.quiet ? pal.quiet : pal[m.colorKey]);
    }
  }

  paintMap();
  // repaint once real fonts are ready so painted names use Instrument Sans
  if (document.fonts?.ready) document.fonts.ready.then(() => !disposed && paintMap());

  startLoop();

  return {
    setBeat: applyBeat,
    setPointer(x, y) { ptrX = x; ptrY = y; },
    setFlat(f) { flat = f; },
    destroy() {
      disposed = true;
      renderer.setAnimationLoop(null);
      io.disconnect(); ro.disconnect(); mo.disconnect();
      scene.traverse((o) => {
        o.geometry?.dispose?.();
        const mats = Array.isArray(o.material) ? o.material : o.material ? [o.material] : [];
        for (const m of mats) { m.map?.dispose?.(); m.dispose?.(); }
      });
      renderer.dispose();
    },
  };
}
