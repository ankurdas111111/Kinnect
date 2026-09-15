/**
 * Instance transforms for the small stuff that turns a block of boxes into a
 * city: windows, rooftop water tanks, trees, street lamps.
 *
 * All of it is generated once, deterministically, and drawn with InstancedMesh
 * — roughly 1,100 windows in ONE draw call. Doing this as individual meshes is
 * what would actually kill the frame rate.
 */
import { BUILDINGS, PARK, ROADS } from './world.js';

/** Deterministic PRNG: the skyline must not reshuffle on every render. */
function rng(seed) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

const WIN_W = 1.5, WIN_H = 2.0;
const FLOOR_H = 4.0;          // metres per storey
const SILL = 2.2;             // ground-floor offset

/**
 * Windows, laid out per storey on all four faces.
 * Returns [{ p:[x,y,z], ry, lit }] — ry rotates the quad to face outward.
 */
export function buildWindows() {
  const rand = rng(20260915);
  const out = [];
  BUILDINGS.forEach((b) => {
    const floors = Math.max(1, Math.floor((b.h - SILL) / FLOOR_H));
    const faces = [
      { ry: 0,            len: b.w, off: b.d / 2 },
      { ry: Math.PI,      len: b.w, off: b.d / 2 },
      { ry: Math.PI / 2,  len: b.d, off: b.w / 2 },
      { ry: -Math.PI / 2, len: b.d, off: b.w / 2 },
    ];
    faces.forEach((f, fi) => {
      const cols = Math.max(1, Math.floor(f.len / 3.4));
      const span = (cols - 1) * 3.4;
      for (let c = 0; c < cols; c++) {
        for (let fl = 0; fl < floors; fl++) {
          // A fully-lit building looks like a spreadsheet; a real one is patchy.
          if (rand() < 0.28) continue;
          const u = -span / 2 + c * 3.4;
          const y = SILL + fl * FLOOR_H;
          // local offset, then rotate into the building's own rotation
          let lx, lz;
          if (fi === 0)      { lx = u;         lz = f.off + 0.06; }
          else if (fi === 1) { lx = u;         lz = -f.off - 0.06; }
          else if (fi === 2) { lx = f.off + 0.06;  lz = u; }
          else               { lx = -f.off - 0.06; lz = u; }
          const cos = Math.cos(b.rot), sin = Math.sin(b.rot);
          out.push({
            p: [b.x + lx * cos - lz * sin, y, b.z + lx * sin + lz * cos],
            ry: b.rot + f.ry,
            lit: rand() < 0.62,
          });
        }
      }
    });
  });
  return out;
}

/** Rooftop water tanks — the thing every Bengaluru roof actually has. */
export function buildTanks() {
  const rand = rng(77);
  const out = [];
  BUILDINGS.forEach((b) => {
    const n = 1 + Math.floor(rand() * 2);
    for (let i = 0; i < n; i++) {
      const ox = (rand() - 0.5) * (b.w - 4);
      const oz = (rand() - 0.5) * (b.d - 4);
      const cos = Math.cos(b.rot), sin = Math.sin(b.rot);
      out.push({
        p: [b.x + ox * cos - oz * sin, b.h + 1.5, b.z + ox * sin + oz * cos],
        s: 0.8 + rand() * 0.5,
      });
    }
  });
  return out;
}

/** Trees, scattered inside the park polygon. */
export function buildTrees() {
  const rand = rng(404);
  const xs = PARK.map(p => p[0]), zs = PARK.map(p => p[1]);
  const minX = Math.min(...xs), maxX = Math.max(...xs);
  const minZ = Math.min(...zs), maxZ = Math.max(...zs);
  const inside = (x, z) => {
    let hit = false;
    for (let i = 0, j = PARK.length - 1; i < PARK.length; j = i++) {
      const [xi, zi] = PARK[i], [xj, zj] = PARK[j];
      if ((zi > z) !== (zj > z) && x < ((xj - xi) * (z - zi)) / (zj - zi) + xi) hit = !hit;
    }
    return hit;
  };
  const out = [];
  let guard = 0;
  while (out.length < 34 && guard++ < 900) {
    const x = minX + rand() * (maxX - minX);
    const z = minZ + rand() * (maxZ - minZ);
    if (!inside(x, z)) continue;
    out.push({ p: [x, 0, z], s: 0.75 + rand() * 0.6 });
  }
  return out;
}

/** Street lamps along the arterial roads — emissive at night. */
export function buildLamps() {
  const out = [];
  ROADS.forEach((r) => {
    for (let i = 0; i < r.pts.length - 1; i++) {
      const [x1, z1] = r.pts[i], [x2, z2] = r.pts[i + 1];
      const steps = 4;
      for (let s = 1; s < steps; s++) {
        const t = s / steps;
        const x = x1 + (x2 - x1) * t, z = z1 + (z2 - z1) * t;
        const dx = x2 - x1, dz = z2 - z1;
        const L = Math.hypot(dx, dz) || 1;
        const nx = -dz / L * (r.w / 2 + 2.2), nz = dx / L * (r.w / 2 + 2.2);
        out.push({ p: [x + nx, 6.5, z + nz] });
        out.push({ p: [x - nx, 6.5, z - nz] });
      }
    }
  });
  return out;
}
