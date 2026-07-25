#!/usr/bin/env node
/**
 * check-three-async.mjs — guardrail: three.js must NEVER be statically imported.
 *
 * The 3D hero constellation is the single library that earned its bundle, and
 * only on desktop-web Landing at fx='full', idle-loaded post-LCP. If any page
 * chunk grows a STATIC `import ... from './three-xxxx.js'`, three would be pulled
 * onto the critical path (and into native bundles), silently blowing the budget.
 *
 * A dynamic `import('./three-xxxx.js')` is fine — that is the intended path.
 * This script fails only on a bare static import of the three chunk from a
 * chunk that is itself eagerly loaded.
 *
 * The heroConstellation module DOES statically import three (that is its whole
 * job), but heroConstellation is itself only ever reached via a dynamic import
 * from Landing — so it is an allowed async gateway, not a violation. We verify
 * that separately: the heroConstellation chunk must only be dynamic-imported.
 *
 * Run against dist/ after `vite build`. Zero dependencies.
 */

import { readFileSync, readdirSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const ASSETS = join(HERE, '..', 'dist', 'assets');

if (!existsSync(ASSETS)) {
  console.error('✗ dist/assets not found. Run `vite build` first.');
  process.exit(1);
}

const files = readdirSync(ASSETS).filter((f) => f.endsWith('.js'));
const threeChunks = files.filter((f) => /^three-[\w-]+\.js$/.test(f));

if (threeChunks.length === 0) {
  // No three chunk emitted at all (e.g. a native/capacitor build where the
  // module is aliased to the stub). That is a valid, passing state.
  console.log('✓ No three chunk emitted (native/stub build or tree-shaken away).');
  process.exit(0);
}

const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// Rollup may quote module specifiers with ', " or ` (backtick template) — allow all.
const Q = `["'\\\`]`;

/** True if `src` STATICALLY imports `target` (a `from"..."` or bare `import"..."`,
 *  never a dynamic `import(...)`). */
function staticallyImports(src, target) {
  const staticFrom = new RegExp(String.raw`from\s*${Q}[^"'\`]*${esc(target)}${Q}`);
  const bareImport = new RegExp(String.raw`(^|[;}\s])import\s*${Q}[^"'\`]*${esc(target)}${Q}`);
  return staticFrom.test(src) || bareImport.test(src);
}

/** True if `src` dynamically imports `target` (`import( ... "target" ... )`). */
function dynamicallyImports(src, target) {
  return new RegExp(String.raw`import\(\s*${Q}[^"'\`]*${esc(target)}${Q}`).test(src);
}

// The three chunk's designated static importer(s): the GL gateway module. These
// are allowed to statically import three ONLY IF they are themselves loaded
// exclusively via dynamic import.
const gatewayChunks = files.filter((f) => /^heroConstellation-[\w-]+\.js$/.test(f));

const violations = [];

// 1) No eagerly-loaded chunk may statically import three. A chunk is "eager"
//    unless it is a gateway that is only dynamic-imported elsewhere.
for (const f of files) {
  if (threeChunks.includes(f)) continue;
  const src = readFileSync(join(ASSETS, f), 'utf8');
  for (const chunk of threeChunks) {
    if (!staticallyImports(src, chunk)) continue;
    if (gatewayChunks.includes(f)) continue; // allowed gateway (verified in step 2)
    violations.push(`${f} statically imports ${chunk} (must be dynamic-only)`);
  }
}

// 2) Each gateway chunk must itself be reached ONLY via dynamic import — never
//    statically pulled onto an eager path (which would defeat the async gate).
for (const gw of gatewayChunks) {
  let staticallyPulled = false;
  let dynamicallyPulled = false;
  for (const f of files) {
    if (f === gw) continue;
    const src = readFileSync(join(ASSETS, f), 'utf8');
    if (staticallyImports(src, gw)) staticallyPulled = true;
    if (dynamicallyImports(src, gw)) dynamicallyPulled = true;
  }
  if (staticallyPulled) violations.push(`${gw} (which pulls three) is statically imported — must be dynamic-only`);
  else if (!dynamicallyPulled) violations.push(`${gw} is not reachable via dynamic import — unexpected chunking`);
}

if (violations.length) {
  console.error('✗ three.js is on an eager path — it must be dynamic-only (Landing hero):');
  for (const v of violations) console.error(`    ${v}`);
  process.exit(1);
}

console.log(`✓ three.js is async-only: ${threeChunks.join(', ')} reached only through the dynamic-imported gateway (${gatewayChunks.join(', ') || 'inlined'}).`);
