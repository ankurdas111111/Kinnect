// @ts-check
import { test, expect } from '@playwright/test';

const VIEWPORTS = [
  { name: 'iPhone-SE',     width: 375,  height: 667  },
  { name: 'iPhone-14-Pro', width: 393,  height: 852  },
  { name: 'iPad',          width: 768,  height: 1024 },
  { name: 'Desktop',       width: 1280, height: 800  },
];

for (const vp of VIEWPORTS) {
  test(`full-page audit ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('http://localhost:5173/#/landing');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForSelector('.hero-card-wrap', { timeout: 10000 });
    await page.waitForTimeout(600); // let hero animate in

    // ── Scroll-reveal trigger ─────────────────────────────────────────────────
    // Gradually scroll to the bottom in steps so IntersectionObserver fires for
    // each section, then scroll back to the top for the final full-page screenshot.
    const totalHeight = await page.evaluate(() => document.body.scrollHeight);
    const step = vp.height;
    for (let y = 0; y <= totalHeight; y += step) {
      await page.evaluate((scrollY) => window.scrollTo({ top: scrollY, behavior: 'instant' }), y);
      await page.waitForTimeout(150); // let observer callbacks run
    }
    // Wait for all reveal-block elements to gain is-revealed class
    await page.waitForFunction(() => {
      const blocks = document.querySelectorAll('.reveal-block');
      return blocks.length === 0 || [...blocks].every(el => el.classList.contains('is-revealed'));
    }, { timeout: 5000 }).catch(() => {/* non-fatal — some blocks may be intentionally deferred */});

    // Scroll back to top for clean full-page screenshot
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
    await page.waitForTimeout(200);

    // ── Full-page screenshot ──────────────────────────────────────────────────
    await page.screenshot({
      path: `test-results/full-audit-${vp.name}.png`,
      fullPage: true,
    });

    // ── Overflow + layout audit ───────────────────────────────────────────────
    const report = await page.evaluate(() => {
      const sel = [
        // Hero
        '.hero', '.hero-content', '.hero-card-wrap', '.mockup-card',
        '.chip-safe', '.chip-alert', '.hero-title', '.hero-subtitle',
        '.hero-cta',
        // Stats
        '.stats-bar', '.stats-grid', '.stat-item',
        // Features
        '.features', '.features-grid', '.feature-cell',
        // How-it-works
        '.how-it-works', '.steps-track', '.step',
        // Demo
        '.demo-section', '.demo-frame', '.demo-tabs',
        // CTA
        '.cta-section', '.cta-inner',
      ];

      const vw = window.innerWidth;
      const results = {};

      for (const s of sel) {
        const el = document.querySelector(s);
        if (!el) { results[s] = null; continue; }
        const b = el.getBoundingClientRect();
        const cs = window.getComputedStyle(el);
        const rect = el.getBoundingClientRect();
        // getBoundingClientRect is relative to viewport; for full-page we add scrollY
        const absTop = rect.top + window.scrollY;
        const absBot = rect.bottom + window.scrollY;
        results[s] = {
          x: Math.round(b.x), y: Math.round(absTop),
          w: Math.round(b.width), h: Math.round(b.height),
          right: Math.round(b.right),
          bottom: Math.round(absBot),
          overflowsRight: b.right > vw + 2,
          overflowsLeft: b.x < -2,
          opacity: parseFloat(cs.opacity),
          display: cs.display,
          visible: cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity) > 0.05,
        };
      }
      return {
        vw,
        vh: window.innerHeight,
        scrollW: document.body.scrollWidth,
        totalH: document.body.scrollHeight,
        revealedCount: document.querySelectorAll('.reveal-block.is-revealed').length,
        totalReveal: document.querySelectorAll('.reveal-block').length,
        results,
      };
    });

    // Log overflows
    const overflows = Object.entries(report.results)
      .filter(([, v]) => v && (v.overflowsRight || v.overflowsLeft));

    if (overflows.length > 0) {
      console.log(`\n[${vp.name}] OVERFLOWS:`);
      for (const [s, v] of overflows) {
        console.log(`  ${s}: x=${v.x} right=${v.right} (vw=${report.vw})`);
      }
    }

    // Log invisible sections (opacity=0 after scroll-reveal)
    const invisible = Object.entries(report.results)
      .filter(([, v]) => v && v.h > 10 && !v.visible);
    if (invisible.length > 0) {
      console.log(`\n[${vp.name}] INVISIBLE (may be reveal-block not triggered):`);
      for (const [s, v] of invisible) {
        console.log(`  ${s}: opacity=${v.opacity} display=${v.display}`);
      }
    }

    console.log(`[${vp.name}] vw=${report.vw} scrollW=${report.scrollW} overflows=${overflows.length} revealed=${report.revealedCount}/${report.totalReveal}`);

    // Assert no horizontal overflow
    expect(report.scrollW, `[${vp.name}] horizontal scroll`).toBeLessThanOrEqual(report.vw + 2);
  });
}
