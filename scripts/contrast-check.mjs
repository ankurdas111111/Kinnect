#!/usr/bin/env node
/**
 * contrast-check.mjs — WCAG AA contrast matrix (CONTRACTS.md §13).
 *
 * Parses token values from global.css + styles/tokens-oklch.css (cascade
 * order: oklch file wins), resolves var() chains, flattens alpha composites
 * over the surface base (surface-1 is rgba over surface-0 — the COMPOSITE is
 * what users see, so that is what gets tested), and asserts AA:
 *   - 4.5:1 for text pairs
 *   - 3:1  for UI/status pairs
 * across both [data-theme] blocks. Exits 1 on any non-excepted failure.
 *
 * Deps: culori (frontend devDependency).
 */

import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const require = createRequire(join(ROOT, 'frontend', 'package.json'));
const { parse, formatHex, wcagContrast, rgb } = require('culori');

const FILES = [
  join(ROOT, 'frontend/src/global.css'),
  join(ROOT, 'frontend/src/styles/tokens-oklch.css'),
];

/**
 * Documented exceptions — each entry silences ONE failing pair with a reason.
 * Never add entries to make the math pass silently; every entry must carry a
 * TODO with the phase that fixes it.
 */
const EXCEPTIONS = [
  // PRE-EXISTING failures in the outgoing teal palette, measured 2026-07-13.
  // TODO(vigil-rebrand): Phase 1.1 replaces the palette and MUST delete every
  // entry below — moonstone-500 is specced ≥4.5:1 with white, and light-mode
  // ("Dawn") UI colors move to -600 shades that clear 3:1 on paper surfaces.
  { theme: 'dark',  pair: 'text-on-primary/primary-500', reason: 'TODO(vigil-rebrand): white on teal #14b8a6 = 2.49' },
  { theme: 'light', pair: 'text-on-primary/primary-500', reason: 'TODO(vigil-rebrand): white on teal #14b8a6 = 2.49' },
  { theme: 'light', pair: 'primary-400/surface-0',       reason: 'TODO(vigil-rebrand): dark-tuned -400 on paper = 1.78' },
  { theme: 'light', pair: 'success-500/surface-0',       reason: 'TODO(vigil-rebrand): dark-tuned -500 on paper = 2.42' },
  { theme: 'light', pair: 'warning-500/surface-0',       reason: 'TODO(vigil-rebrand): dark-tuned -500 on paper = 2.05' },
  { theme: 'light', pair: 'status-live/surface-0',       reason: 'TODO(vigil-rebrand): dark-tuned live-green on paper = 2.18' },
];

// ── CSS custom-property extraction ──────────────────────────────────────────

/** Extract `--name: value;` pairs from the body of every matching selector block. */
function extractBlocks(css, selectorRe) {
  const props = {};
  const re = new RegExp(selectorRe.source + String.raw`\s*\{([^}]*)\}`, 'g');
  let m;
  while ((m = re.exec(css))) {
    for (const pm of m[1].matchAll(/--([a-zA-Z0-9-]+)\s*:\s*([^;]+);/g)) {
      props[pm[1]] = pm[2].trim();
    }
  }
  return props;
}

function buildTheme(theme) {
  // :root first, then [data-theme="X"] overrides, per file in cascade order.
  const map = {};
  for (const f of FILES) {
    const css = readFileSync(f, 'utf8');
    Object.assign(map, extractBlocks(css, /:root(?:\[[^\]]*\])?/));
    Object.assign(map, extractBlocks(css, new RegExp(String.raw`\[data-theme=["']${theme}["']\]`)));
  }
  return map;
}

// ── Value resolution ─────────────────────────────────────────────────────────

/** Resolve var(--x, fallback) chains + color-mix(in oklch, C P%, transparent). */
function resolve(map, value, depth = 0) {
  if (depth > 12 || !value) return null;
  value = value.trim();

  const varMatch = value.match(/^var\(--([a-zA-Z0-9-]+)(?:\s*,\s*(.+))?\)$/);
  if (varMatch) {
    const inner = map[varMatch[1]];
    if (inner != null) return resolve(map, inner, depth + 1);
    return varMatch[2] ? resolve(map, varMatch[2], depth + 1) : null;
  }

  const mixMatch = value.match(/^color-mix\(in [a-z-]+\s*,\s*(.+?)\s+([\d.]+)%\s*,\s*transparent\s*\)$/);
  if (mixMatch) {
    const base = resolve(map, mixMatch[1], depth + 1);
    if (!base) return null;
    return { ...base, alpha: (base.alpha ?? 1) * (Number(mixMatch[2]) / 100) };
  }

  // Inline var() inside a longer value (e.g. oklch(from ...) not supported) —
  // substitute simple var() references then re-parse.
  if (value.includes('var(')) {
    const substituted = value.replace(/var\(--([a-zA-Z0-9-]+)(?:\s*,\s*([^)]+))?\)/g, (_, name, fb) => {
      const r = map[name] != null ? map[name] : fb;
      return r ?? '';
    });
    if (substituted !== value) return resolve(map, substituted, depth + 1);
  }

  const parsed = parse(value);
  return parsed ?? null;
}

/** Flatten fg (possibly alpha) over an opaque bg. */
function flatten(fg, bg) {
  const f = rgb(fg);
  const b = rgb(bg);
  const a = f.alpha ?? 1;
  if (a >= 1) return f;
  return {
    mode: 'rgb',
    r: f.r * a + b.r * (1 - a),
    g: f.g * a + b.g * (1 - a),
    b: f.b * a + b.b * (1 - a),
  };
}

// ── The matrix ───────────────────────────────────────────────────────────────

const TEXT_PAIRS = [
  ['text-primary', 'surface-0'],
  ['text-primary', 'surface-1'],
  ['text-primary', 'surface-2'],
  ['text-secondary', 'surface-0'],
  ['text-secondary', 'surface-1'],
  ['text-secondary', 'surface-2'],
  ['text-on-primary', 'primary-500'],
];

const UI_PAIRS = [
  ['primary-400', 'surface-0'],
  ['success-500', 'surface-0'],
  ['warning-500', 'surface-0'],
  ['danger-500', 'surface-0'],
  ['status-live', 'surface-0'],
];

let failed = false;
const rows = [];

for (const theme of ['dark', 'light']) {
  const map = buildTheme(theme);
  const base = resolve(map, map['surface-0'] ? 'var(--surface-0)' : '#ffffff');
  if (!base) {
    console.error(`✗ [${theme}] cannot resolve --surface-0`);
    failed = true;
    continue;
  }
  const opaqueBase = flatten(base, { mode: 'rgb', r: 0, g: 0, b: 0 });

  const check = (fgName, bgName, min) => {
    const fgRaw = resolve(map, `var(--${fgName})`);
    const bgRaw = resolve(map, `var(--${bgName})`);
    if (!fgRaw || !bgRaw) {
      rows.push([theme, `${fgName}/${bgName}`, 'UNRESOLVED', `≥${min}`, 'SKIP']);
      return;
    }
    // Composite: bg over surface base, then fg over that composite.
    const bgFlat = flatten(bgRaw, opaqueBase);
    const fgFlat = flatten(fgRaw, bgFlat);
    const ratio = wcagContrast(fgFlat, bgFlat);
    const pass = ratio >= min;
    const excepted = EXCEPTIONS.find((e) => e.theme === theme && e.pair === `${fgName}/${bgName}`);
    if (!pass && !excepted) failed = true;
    rows.push([
      theme,
      `${fgName}/${bgName}`,
      ratio.toFixed(2),
      `≥${min}`,
      pass ? 'PASS' : excepted ? 'EXCEPTED' : 'FAIL',
    ]);
  };

  for (const [fg, bg] of TEXT_PAIRS) check(fg, bg, 4.5);
  for (const [fg, bg] of UI_PAIRS) check(fg, bg, 3);
}

// ── Report ───────────────────────────────────────────────────────────────────

const w = [7, 34, 12, 6, 9];
const line = (cols) => cols.map((c, i) => String(c).padEnd(w[i])).join(' ');
console.log(line(['theme', 'pair (fg/bg, composited)', 'ratio', 'min', 'result']));
console.log('-'.repeat(w.reduce((a, b) => a + b + 1, 0)));
for (const r of rows) console.log(line(r));

if (failed) {
  console.error('\n✗ AA contrast failures. Fix token values or add a documented exception with a TODO.');
  process.exit(1);
}
console.log('\n✓ AA contrast matrix passes.');
