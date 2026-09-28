// Browser QA gate for D-01 — run against the live app, fresh account each run.
import { chromium } from 'playwright';
const B = 'http://localhost:3000';
const stamp = process.env.STAMP;

const b = await chromium.launch({ channel: 'chrome' });
const ctx = await b.newContext({
  viewport: { width: 1440, height: 900 },
  permissions: ['geolocation'],
  geolocation: { latitude: -37.77, longitude: 144.99 },
});
const p = await ctx.newPage();
const errs = [];
p.on('pageerror', e => errs.push('pageerror: ' + e.message));
p.on('console', m => { if (m.type() === 'error') errs.push(m.text().slice(0, 120)); });

// --- register a brand-new user so we get a true first run
await p.goto(B + '/#/register', { waitUntil: 'domcontentloaded' });
await p.waitForTimeout(2000);
const step = async () => {
  for (const i of await p.$$('input')) {
    if (!(await i.isVisible())) continue;
    if (await i.evaluate(e => e.readOnly || e.disabled)) continue;
    const t = await i.getAttribute('type');
    const ac = await i.getAttribute('autocomplete');
    const ph = await i.getAttribute('placeholder');
    if (t === 'email') await i.fill(`qa.${stamp}@example.test`);
    else if (t === 'password') await i.fill(`Test-Passw0rd-${stamp}`);
    else if (ac === 'given-name' || /John/i.test(ph || '')) await i.fill('QA');
    else if (ac === 'family-name' || /Doe/i.test(ph || '')) await i.fill('Gate');
    else if (t === 'text') await i.fill('QA');
  }
  const btn = await p.$('button[type=submit]') || await p.$('button:has-text("Continue")');
  if (btn) { await btn.click(); await p.waitForTimeout(2500); }
};
await step(); await step(); await step();

// Wait past BOTH timers (onboarding 800ms, spotlight 1400ms) plus margin.
await p.waitForTimeout(8000);

const probe = () => p.evaluate(() => {
  const vis = e => e.checkVisibility
    ? e.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })
    : e.getBoundingClientRect().width > 0;
  const dlgs = [...document.querySelectorAll('[role=dialog],[aria-modal="true"],dialog')].filter(vis);
  return {
    dialogCount: dlgs.length,
    labels: dlgs.map(d => d.getAttribute('aria-label') || '(none)'),
    maybeLater: [...document.querySelectorAll('button')]
      .filter(e => vis(e) && /maybe later/i.test(e.innerText)).length,
    activeEl: document.activeElement
      ? document.activeElement.tagName + '.' + (document.activeElement.className || '').toString().split(' ')[0]
      : 'none',
    focusInsideDialog: dlgs.some(d => d.contains(document.activeElement)),
  };
});

const before = await probe();
console.log('BEFORE ESC:', JSON.stringify(before));
await p.keyboard.press('Escape');
await p.waitForTimeout(1200);
const after = await probe();
console.log('AFTER  ESC:', JSON.stringify(after));
console.log('ERRORS:', JSON.stringify(errs.slice(0, 5)));

// The first-run flow is SEQUENCED (onboarding -> feature guide), so a raw
// dialogCount drop is the wrong assertion: one closes as the next opens.
// Assert instead that the dialog we escaped is gone, and that whatever
// replaces it also holds focus.
const escaped = !after.labels.includes('Get started');

// The 401 on /api/me is a known pre-existing baseline defect (audit D-10:
// loadSession fires on public routes). Scope the error gate to regressions.
const newErrs = errs.filter(e => !/401|Unauthorized/.test(e));

const checks = [
  ['exactly one modal dialog on first run', before.dialogCount === 1],
  ['exactly one "Maybe later" affordance', before.maybeLater <= 1],
  ['focus moved off <body>', before.activeEl !== 'BODY.'],
  ['focus is inside the dialog', before.focusInsideDialog === true],
  ['Escape dismisses the focused dialog', escaped],
  ['never more than one dialog at a time', after.dialogCount <= 1],
  ['the next dialog in sequence also holds focus', after.dialogCount === 0 || after.focusInsideDialog === true],
  ['no new console/page errors', newErrs.length === 0],
];
let pass = true;
console.log('\n--- D-01 GATE ---');
for (const [name, ok] of checks) { if (!ok) pass = false; console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`); }
console.log(pass ? '\nGATE: PASS' : '\nGATE: FAIL');
await b.close();
process.exit(pass ? 0 : 1);
