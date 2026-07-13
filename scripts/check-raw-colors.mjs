#!/usr/bin/env node
/**
 * check-raw-colors.mjs — the no-raw-color guardrail (CONTRACTS.md §13).
 *
 * Scans frontend/src for raw hex colors and rgb()/rgba() literals in CSS,
 * <style> blocks, and JS color strings. Existing violations are grandfathered
 * per-file in scripts/raw-colors-grandfather.json; any file EXCEEDING its
 * grandfathered count (or any new file with violations) fails the check.
 * Fixing violations lowers the recorded count on --generate-grandfather runs;
 * CI runs the plain mode.
 *
 * Usage:
 *   node scripts/check-raw-colors.mjs                        # check (exit 1 on new violations)
 *   node scripts/check-raw-colors.mjs --generate-grandfather # (re)write the baseline
 *
 * Zero dependencies.
 */

import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs';
import { join, relative, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'frontend', 'src');
const GRANDFATHER_PATH = join(ROOT, 'scripts', 'raw-colors-grandfather.json');
const GENERATE = process.argv.includes('--generate-grandfather');

const EXTS = new Set(['.svelte', '.css', '.js']);

// Hex colors (#abc, #aabbcc, #aabbccdd) and rgb()/rgba() literals.
// oklch()/color-mix()/var() are the sanctioned forms and are NOT matched.
const HEX_RE = /#[0-9a-fA-F]{3,8}\b/g;
const RGB_RE = /\brgba?\s*\(/g;

// Lines that may legitimately carry raw colors:
//  - SVG data URIs / encoded fills (background-image svg payloads)
//  - explicit lint suspensions
const LINE_ALLOW_RE = /raw-color-ok|data:image\/svg|%23[0-9a-fA-F]{3,8}/;

/** Recursively list files under dir with allowed extensions. */
function listFiles(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    if (name === 'node_modules' || name.startsWith('.')) continue;
    const p = join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) out.push(...listFiles(p));
    else if (EXTS.has(name.slice(name.lastIndexOf('.')))) out.push(p);
  }
  return out;
}

/**
 * Count raw-color violations in one file.
 * - .css: whole file scanned
 * - .svelte: <style> blocks + inline style="" + JS color-ish strings
 * - .js: string literals that parse as colors (hex / rgb()) — catches canvas
 *   fillStyle and map-paint hexes
 * Hex inside id/route/url-ish strings is avoided by requiring color-length runs.
 */
function countViolations(path, text) {
  const lines = text.split('\n');
  let count = 0;
  const hits = [];
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (LINE_ALLOW_RE.test(line)) continue;
    const hex = line.match(HEX_RE) || [];
    // Only 3/4/6/8-digit runs are colors (avoid #12345 fragment ids).
    const hexColors = hex.filter((h) => [4, 5, 7, 9].includes(h.length));
    const rgb = line.match(RGB_RE) || [];
    const n = hexColors.length + rgb.length;
    if (n > 0) {
      count += n;
      if (hits.length < 5) hits.push(`${i + 1}: ${line.trim().slice(0, 90)}`);
    }
  }
  return { count, hits };
}

const files = listFiles(SRC);
const current = {};
for (const f of files) {
  const rel = relative(ROOT, f);
  const { count } = countViolations(f, readFileSync(f, 'utf8'));
  if (count > 0) current[rel] = count;
}

if (GENERATE) {
  const sorted = Object.fromEntries(Object.entries(current).sort(([a], [b]) => a.localeCompare(b)));
  writeFileSync(GRANDFATHER_PATH, JSON.stringify(sorted, null, 2) + '\n');
  const total = Object.values(sorted).reduce((a, b) => a + b, 0);
  console.log(`Grandfather baseline written: ${Object.keys(sorted).length} files, ${total} violations.`);
  process.exit(0);
}

if (!existsSync(GRANDFATHER_PATH)) {
  console.error('No grandfather baseline. Run: node scripts/check-raw-colors.mjs --generate-grandfather');
  process.exit(1);
}

const grandfather = JSON.parse(readFileSync(GRANDFATHER_PATH, 'utf8'));
let failed = false;
const report = [];

for (const [rel, count] of Object.entries(current)) {
  const allowed = grandfather[rel] ?? 0;
  if (count > allowed) {
    failed = true;
    const { hits } = countViolations(join(ROOT, rel), readFileSync(join(ROOT, rel), 'utf8'));
    report.push(`  ${rel} — ${count} raw colors (allowed ${allowed})`);
    for (const h of hits) report.push(`      ${h}`);
  }
}

// Celebrate shrinkage: files now under their baseline (informational).
const improved = Object.entries(grandfather).filter(([rel, allowed]) => (current[rel] ?? 0) < allowed);

if (failed) {
  console.error('✗ New raw-color violations (use CSS tokens / color-mix, or add `/* raw-color-ok */`):');
  for (const l of report) console.error(l);
  console.error('\nIf a violation is intentional, justify it in review — do not regenerate the baseline to bypass.');
  process.exit(1);
}

const totalNow = Object.values(current).reduce((a, b) => a + b, 0);
console.log(`✓ No new raw colors. Current total: ${totalNow}${improved.length ? ` (${improved.length} files improved vs baseline — regenerate to lock in)` : ''}`);
