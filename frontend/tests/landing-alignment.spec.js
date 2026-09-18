// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Landing structure + alignment audit — Stitch "Kinnect Hearth" editorial
 * landing (frontend/src/pages/Landing.svelte).
 *
 * Covers:
 *   - nav section links present with anchor hrefs
 *   - auth CTAs route correctly (Sign In → #/login, Create/Start → #/register)
 *   - serif (Newsreader) hero headline with the exact design copy
 *   - the three "Dignity First" pillar headings
 *   - walk-with-me / emergency / circle sections exist, in document order
 *   - anchor links scroll without touching the hash router
 *   - 44px touch floor on the ember CTAs
 *   - no horizontal overflow at any viewport (incl. the 390px smoke)
 *
 * The landing is static marketing; the only network traffic is App.svelte's
 * session probe, mocked here with page.route() so no backend is required.
 * Service workers are blocked globally in playwright.config.js, so the mock
 * always intercepts.
 */

const VIEWPORTS = [
  { name: 'iPhone SE',   width: 375,  height: 667  },
  { name: 'iPhone 12',   width: 390,  height: 844  },
  { name: 'iPad',        width: 768,  height: 1024 },
  { name: 'Desktop',     width: 1280, height: 800  },
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
  // The landing is a lazy route chunk — wait for the hero mock figure.
  await page.waitForSelector('.hero-card', { timeout: 10000 });
}

for (const vp of VIEWPORTS) {
  test.describe(`Landing — ${vp.name} (${vp.width}×${vp.height})`, () => {
    test.use({ viewport: { width: vp.width, height: vp.height } });

    test.beforeEach(async ({ page }) => {
      await openLanding(page);
    });

    test('no horizontal overflow', async ({ page }) => {
      const { scrollW, vw } = await page.evaluate(() => ({
        scrollW: Math.max(
          document.documentElement.scrollWidth,
          document.body.scrollWidth
        ),
        vw: window.innerWidth,
      }));
      expect(scrollW, 'horizontal scroll detected').toBeLessThanOrEqual(vw + 2);
    });

    test('hero headline is the serif voice with the design copy', async ({ page }) => {
      const h1 = page.locator('h1.hero-title');
      await expect(h1).toBeVisible();
      await expect(h1).toContainText('Your family as a few pebbles');
      await expect(h1).toContainText('one honest sentence');
      await expect(h1).toContainText('and nothing else.');
      const fontFamily = await h1.evaluate((el) => getComputedStyle(el).fontFamily);
      expect(fontFamily, 'hero must speak in the Newsreader serif register').toMatch(/newsreader/i);
    });

    test('auth + hero CTAs carry the right routes', async ({ page }) => {
      // Sign In / Create your Circle (nav) — attribute checks work even when
      // the link is hidden at phone widths.
      await expect(page.locator('a.lp-signin')).toHaveAttribute('href', '#/login');
      await expect(page.locator('a.lp-create')).toHaveAttribute('href', '#/register');
      // Hero CTAs
      await expect(page.locator('a.hero-start')).toHaveAttribute('href', '#/register');
      await expect(page.locator('a.hero-start')).toHaveText(/Start your circle/);
      await expect(page.locator('a.hero-watch')).toHaveAttribute('href', '#walk-with-me');
      await expect(page.locator('a.hero-watch')).toContainText('Watch how it feels');
    });

    test('nav section links present with anchor hrefs', async ({ page }) => {
      const nav = page.locator('nav[aria-label="Landing sections"]');
      await expect(nav).toHaveCount(1);
      const links = [
        ['#philosophy',   'The Philosophy'],
        ['#walk-with-me', 'Walk With Me'],
        ['#emergency',    'Emergency Path'],
        ['#circle',       'Android & Web'],
      ];
      for (const [href, label] of links) {
        const a = nav.locator(`a[href="${href}"]`);
        await expect(a).toHaveCount(1);
        await expect(a).toHaveText(label);
      }
    });

    test('the three Dignity First pillars', async ({ page }) => {
      const pillars = page.locator('#philosophy .pillar h3');
      await expect(pillars).toHaveCount(3);
      await expect(pillars.nth(0)).toHaveText('One honest sentence beats a dashboard');
      await expect(pillars.nth(1)).toHaveText('Calm by default, loud only for SOS');
      await expect(pillars.nth(2)).toHaveText('Mutual care, never surveillance');
    });

    test('feature + invite sections exist in document order', async ({ page }) => {
      for (const id of ['philosophy', 'walk-with-me', 'emergency', 'circle']) {
        await expect(page.locator(`section[id="${id}"]`)).toHaveCount(1);
      }
      await expect(page.locator('#walk-with-me .session-card')).toHaveCount(1);
      await expect(page.locator('#emergency .emergency-card')).toHaveCount(1);
      await expect(page.locator('#circle .invite-card')).toHaveCount(1);
      await expect(page.locator('footer.lp-footer')).toHaveCount(1);

      const tops = await page.evaluate(() =>
        ['philosophy', 'walk-with-me', 'emergency', 'circle'].map((id) => {
          const el = document.getElementById(id);
          return el ? el.getBoundingClientRect().top + window.scrollY : -1;
        })
      );
      expect(tops[0]).toBeGreaterThan(0);
      expect(tops[1]).toBeGreaterThan(tops[0]);
      expect(tops[2]).toBeGreaterThan(tops[1]);
      expect(tops[3]).toBeGreaterThan(tops[2]);
    });

    test('ember CTAs meet the touch-target floor', async ({ page }) => {
      for (const sel of ['a.hero-start', 'a.lp-create']) {
        const box = await page.locator(sel).boundingBox();
        expect(box, `${sel} must be visible`).not.toBeNull();
        expect(box.height, `${sel} height ≥ 44px`).toBeGreaterThanOrEqual(44);
      }
    });

    test('"Watch how it feels" scrolls to walk-with-me, router untouched', async ({ page }) => {
      // Force reduced motion so scrollIntoView is instant and deterministic.
      await page.emulateMedia({ reducedMotion: 'reduce' });
      await page.locator('a.hero-watch').click();
      await page.waitForFunction(() => {
        const el = document.getElementById('walk-with-me');
        if (!el) return false;
        const r = el.getBoundingClientRect();
        return r.bottom > 0 && r.top < window.innerHeight * 0.6;
      }, undefined, { timeout: 5000 });
      expect(await page.evaluate(() => window.scrollY)).toBeGreaterThan(0);
      // Section scrolling must not disturb the hash router.
      expect(await page.evaluate(() => window.location.hash)).toBe('#/landing');
    });

    test('sticky nav stays pinned while scrolling', async ({ page }) => {
      await page.emulateMedia({ reducedMotion: 'reduce' });
      await page.evaluate(() => {
        document.getElementById('circle')?.scrollIntoView({ behavior: 'auto' });
      });
      await page.waitForTimeout(150);
      const box = await page.locator('header.lp-nav').boundingBox();
      expect(box).not.toBeNull();
      expect(Math.abs(box.y), 'nav must remain at the top edge').toBeLessThanOrEqual(1);
    });

    test('screenshot — hero', async ({ page }) => {
      await page.screenshot({
        path: `test-results/landing-hero-${vp.name.replace(/ /g, '-')}.png`,
        clip: { x: 0, y: 0, width: vp.width, height: Math.min(vp.height, 900) },
      });
    });
  });
}
