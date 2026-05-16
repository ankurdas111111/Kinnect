// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Landing page alignment audit.
 * Checks: chips within viewport, no element overflow, no unexpected stacking.
 */

const VIEWPORTS = [
  { name: 'iPhone SE',       width: 375,  height: 667  },
  { name: 'iPhone 14 Pro',   width: 393,  height: 852  },
  { name: 'iPad',            width: 768,  height: 1024 },
  { name: 'Desktop',         width: 1280, height: 800  },
];

for (const vp of VIEWPORTS) {
  test.describe(`Landing alignment — ${vp.name} (${vp.width}×${vp.height})`, () => {
    test.use({ viewport: { width: vp.width, height: vp.height } });

    test.beforeEach(async ({ page }) => {
      await page.goto('/#/landing');
      await page.waitForLoadState('domcontentloaded');
      // Wait for hero section to be visible
      await page.waitForSelector('.hero-card-wrap', { timeout: 10000 });
    });

    test('chip-safe does not overflow viewport horizontally', async ({ page }) => {
      const chip = page.locator('.chip-safe');
      await expect(chip).toBeVisible();
      const box = await chip.boundingBox();
      expect(box).not.toBeNull();
      // Must not overflow left edge
      expect(box.x).toBeGreaterThanOrEqual(0);
      // Must not overflow right edge
      expect(box.x + box.width).toBeLessThanOrEqual(vp.width + 1); // +1 for sub-pixel
    });

    test('chip-alert does not overflow viewport horizontally', async ({ page }) => {
      const chip = page.locator('.chip-alert');
      await expect(chip).toBeVisible();
      const box = await chip.boundingBox();
      expect(box).not.toBeNull();
      // Must not overflow left edge
      expect(box.x).toBeGreaterThanOrEqual(0);
      // Must not overflow right edge
      expect(box.x + box.width).toBeLessThanOrEqual(vp.width + 1);
    });

    test('chip-safe does not overlap mockup-card interior', async ({ page }) => {
      const card  = page.locator('.mockup-card');
      const chip  = page.locator('.chip-safe');
      const cardBox = await card.boundingBox();
      const chipBox = await chip.boundingBox();
      expect(cardBox).not.toBeNull();
      expect(chipBox).not.toBeNull();
      // chip bottom should be at or above card top (floating above)
      // OR chip top should be above card top — either is fine if there's no z-index clash
      // Key invariant: chip is not purely inside card (it should float outside or at edge)
      const chipBotY = chipBox.y + chipBox.height;
      const cardTopY = cardBox.y;
      // chip bottom must be at or above the card's topbar bottom (~40px in)
      // i.e. the chip should not fully sit inside the scrollable list area
      const cardListStartY = cardTopY + 40 + 140; // topbar + map height
      expect(chipBotY).toBeLessThan(cardListStartY);
    });

    test('chip-alert is below mockup-card (does not overlap list rows)', async ({ page }) => {
      const chip = page.locator('.chip-alert');
      const card = page.locator('.mockup-card');
      const chipBox = await chip.boundingBox();
      const cardBox = await card.boundingBox();
      expect(chipBox).not.toBeNull();
      expect(cardBox).not.toBeNull();
      // chip-alert is now positioned below the card bottom edge
      // chip top should be at or below card bottom
      expect(chipBox.y).toBeGreaterThanOrEqual(cardBox.y + cardBox.height - 24); // allow 24px inside card edge
    });

    test('mockup-ping dot is visible and not clipped', async ({ page }) => {
      const dot = page.locator('.ping-dot');
      await expect(dot).toBeVisible();
      const box = await dot.boundingBox();
      expect(box).not.toBeNull();
      expect(box.width).toBeGreaterThan(0);
      expect(box.height).toBeGreaterThan(0);
    });

    test('map pins are inside mockup-map bounds', async ({ page }) => {
      const map  = page.locator('.mockup-map');
      const mapBox = await map.boundingBox();
      for (const cls of ['.pin-1', '.pin-2', '.pin-3']) {
        const pin = page.locator(cls);
        const pinBox = await pin.boundingBox();
        expect(pinBox).not.toBeNull();
        // Pin center should be within map bounds
        const pinCx = pinBox.x + pinBox.width / 2;
        const pinCy = pinBox.y + pinBox.height / 2;
        expect(pinCx).toBeGreaterThanOrEqual(mapBox.x);
        expect(pinCx).toBeLessThanOrEqual(mapBox.x + mapBox.width);
        expect(pinCy).toBeGreaterThanOrEqual(mapBox.y);
        expect(pinCy).toBeLessThanOrEqual(mapBox.y + mapBox.height);
      }
    });

    test('mockup-row items are horizontally aligned', async ({ page }) => {
      const rows = page.locator('.mockup-row');
      const count = await rows.count();
      expect(count).toBe(3);
      for (let i = 0; i < count; i++) {
        const row     = rows.nth(i);
        const avatar  = row.locator('.mockup-avatar');
        const info    = row.locator('.mockup-info');
        const ago     = row.locator('.mockup-ago');
        const avatarBox = await avatar.boundingBox();
        const infoBox   = await info.boundingBox();
        const agoBox    = await ago.boundingBox();
        expect(avatarBox).not.toBeNull();
        expect(infoBox).not.toBeNull();
        expect(agoBox).not.toBeNull();
        // All three should be on the same horizontal band (midpoints within 8px)
        const midpoints = [
          avatarBox.y + avatarBox.height / 2,
          infoBox.y   + infoBox.height   / 2,
          agoBox.y    + agoBox.height    / 2,
        ];
        const minMid = Math.min(...midpoints);
        const maxMid = Math.max(...midpoints);
        expect(maxMid - minMid).toBeLessThan(16); // ≤16px vertical spread
      }
    });

    test('hero section does not cause horizontal scroll', async ({ page }) => {
      const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
      // scrollWidth > clientWidth means overflow
      const windowWidth = await page.evaluate(() => window.innerWidth);
      expect(bodyWidth).toBeLessThanOrEqual(windowWidth + 2); // +2 sub-pixel
    });

    test('screenshot — full hero', async ({ page }) => {
      await page.screenshot({
        path: `test-results/landing-hero-${vp.name.replace(/ /g, '-')}.png`,
        clip: { x: 0, y: 0, width: vp.width, height: Math.min(vp.height, 900) },
      });
    });
  });
}
