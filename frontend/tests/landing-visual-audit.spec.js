// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Landing visual audit — Stitch "Kinnect Hearth" editorial landing.
 *
 * Full-page screenshots + per-section overflow report at four widths, plus
 * two palette-law audits:
 *   - vermilion (var(--danger-500)) may only appear inside #emergency
 *   - real body copy never drops below 16px (mock figures excluded)
 *
 * Same conventions as the other UI specs: App.svelte's session probe is
 * mocked via page.route(); service workers are blocked globally in
 * playwright.config.js so mocks always intercept.
 */

const VIEWPORTS = [
  { name: 'iPhone-12',  width: 390,  height: 844  },
  { name: 'iPad',       width: 768,  height: 1024 },
  { name: 'Desktop',    width: 1280, height: 800  },
  { name: 'Desktop-XL', width: 1920, height: 1080 },
];

// One representative selector per landing region — all must exist in the DOM
// at every width (some are display:none on phones, which is fine).
const SECTIONS = [
  '.lp-nav', '.hero', '.hero-title', '.hero-cta-row', '.hero-card',
  '.verdict-card', '.standby-row',
  '#philosophy', '.pillar-grid', '.pillar',
  '#walk-with-me', '.session-card',
  '#emergency', '.emergency-card',
  '#circle', '.invite-card', '.join-form',
  '.lp-footer',
];

async function openLanding(page) {
  await page.route('**/api/**', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'unauthenticated' }),
    })
  );
  await page.goto('/#/landing');
  await page.waitForLoadState('domcontentloaded');
  await page.waitForSelector('.hero-card', { timeout: 10000 });
  await page.evaluate(() => document.fonts.ready);
}

for (const vp of VIEWPORTS) {
  test(`full-page audit ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await openLanding(page);

    // Walk the page once so layout/animations settle, then return to the top
    // for a clean full-page capture.
    const totalHeight = await page.evaluate(() => document.body.scrollHeight);
    for (let y = 0; y <= totalHeight; y += vp.height) {
      await page.evaluate((t) => window.scrollTo({ top: t, behavior: 'instant' }), y);
      await page.waitForTimeout(100);
    }
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
    await page.waitForTimeout(200);

    await page.screenshot({
      path: `test-results/landing-audit-${vp.name}.png`,
      fullPage: true,
    });

    // ── Presence + overflow report ────────────────────────────────────────
    const report = await page.evaluate((selectors) => {
      const vw = window.innerWidth;
      const results = {};
      for (const s of selectors) {
        const el = document.querySelector(s);
        if (!el) { results[s] = null; continue; }
        const b = el.getBoundingClientRect();
        const cs = window.getComputedStyle(el);
        results[s] = {
          x: Math.round(b.x), w: Math.round(b.width), right: Math.round(b.right),
          overflowsRight: b.right > vw + 2,
          overflowsLeft: b.x < -2,
          display: cs.display,
        };
      }
      return {
        vw,
        scrollW: Math.max(document.documentElement.scrollWidth, document.body.scrollWidth),
        results,
      };
    }, SECTIONS);

    const missing = SECTIONS.filter((s) => report.results[s] === null);
    expect(missing, `sections missing from DOM: ${missing.join(', ')}`).toEqual([]);

    const overflows = Object.entries(report.results)
      .filter(([, v]) => v && (v.overflowsRight || v.overflowsLeft));
    if (overflows.length > 0) {
      console.log(`\n[${vp.name}] OVERFLOWS:`);
      for (const [s, v] of overflows) {
        console.log(`  ${s}: x=${v.x} right=${v.right} (vw=${report.vw})`);
      }
    }
    console.log(`[${vp.name}] vw=${report.vw} scrollW=${report.scrollW} overflows=${overflows.length}`);

    expect(overflows.map(([s]) => s), `[${vp.name}] elements overflow the viewport`).toEqual([]);
    expect(report.scrollW, `[${vp.name}] horizontal scroll`).toBeLessThanOrEqual(report.vw + 2);
  });
}

test('vermilion is isolated to the emergency section', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openLanding(page);

  const audit = await page.evaluate(() => {
    // Resolve the SOS token to its computed rgb in the active theme.
    const probe = document.createElement('span');
    probe.style.color = 'var(--danger-500)';
    document.body.appendChild(probe);
    const vermilion = getComputedStyle(probe).color;
    probe.remove();

    const PROPS = [
      'color', 'backgroundColor',
      'borderTopColor', 'borderRightColor', 'borderBottomColor', 'borderLeftColor',
      'outlineColor', 'fill', 'stroke',
    ];
    const offenders = [];
    for (const el of document.querySelectorAll('.landing, .landing *')) {
      if (el.closest('#emergency')) continue;
      const cs = getComputedStyle(el);
      const hits = PROPS.filter((p) => cs[p] === vermilion);
      if (hits.length) {
        offenders.push(
          `<${el.tagName.toLowerCase()} class="${el.getAttribute('class') || ''}"> → ${hits.join(',')}`
        );
      }
    }

    // Class-name hygiene: no sos/vermilion/danger-styled classes outside the
    // emergency section (scoped svelte hashes are unaffected).
    const classOffenders = [];
    for (const el of document.querySelectorAll('.landing [class]')) {
      if (el.closest('#emergency')) continue;
      const cls = el.getAttribute('class') || '';
      if (/(vermilion|danger|sos-|\bsos\b)/i.test(cls)) classOffenders.push(cls);
    }
    return { vermilion, offenders, classOffenders };
  });

  expect(audit.offenders, 'vermilion computed colours outside #emergency').toEqual([]);
  expect(audit.classOffenders, 'SOS/vermilion class names outside #emergency').toEqual([]);
});

test('real body copy never drops below 16px', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openLanding(page);

  const tooSmall = await page.evaluate(() => {
    const offenders = [];
    for (const p of document.querySelectorAll('.landing p')) {
      // Product-mock figures intentionally use caption scales.
      if (p.closest('.mock-figure')) continue;
      const size = parseFloat(getComputedStyle(p).fontSize);
      if (size < 16) {
        offenders.push(`${(p.textContent || '').trim().slice(0, 48)}… @ ${size}px`);
      }
    }
    return offenders;
  });
  expect(tooSmall, 'paragraphs below the 16px floor').toEqual([]);
});
