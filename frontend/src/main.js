import { mount } from 'svelte';
import App from './App.svelte';
// Self-hosted fonts (Fontsource): no fonts.googleapis.com runtime dependency,
// so Capacitor cold starts render branded type even fully offline.
// HEARTH (Stitch "Kinnect Hearth"): Work Sans carries everything functional;
// Newsreader italic is the verdict voice only ("Everyone's settled.").
// Numbers are tabular in the sans face — "8 min · 1.2 km · 64%", not mono.
import '@fontsource/work-sans/400.css';
import '@fontsource/work-sans/500.css';
import '@fontsource/work-sans/600.css';
import '@fontsource/work-sans/700.css';
import '@fontsource/newsreader/400.css';
import '@fontsource/newsreader/400-italic.css';
import './global.css';
// OKLCH re-expression of the core color scales — MUST stay between global.css
// (defines the scales) and themes.css (named themes override --primary-*).
import './styles/tokens-oklch.css';
import './styles/components.css';
import './styles/themes.css';
// Daypart tint layer — theme-flavoring, so it sits between themes.css and
// tokens-fx.css (fx stays the last word on blur; the token sets never overlap).
import './styles/tokens-daypart.css';
// HEARTH redesign token layer — remaps the semantic tokens (surface/text/
// primary/…) onto the Hearth palette, so all 157 frontend files re-skin
// without component edits. After the theme/daypart sets, before fx (which
// stays the last word on blur).
import './styles/tokens-hearth.css';
// Modernization FX layer — MUST be last so [data-fx] calm-mode overrides win
// over theme blur values. Additive tokens (bento/tactile/color-mix) are safe.
import './styles/tokens-fx.css';

import { effects } from './lib/stores/effects.js';
import { initDaypart } from './lib/daypart.js';

// Pick the effects level (stored pref or device-capability default) before
// first paint so glass/blur render at the right budget from the start.
effects.init();
// Time-of-day ambient temperature (index.html pre-paints the attribute).
initDaypart();

const app = mount(App, { target: document.getElementById('app') });

export default app;
