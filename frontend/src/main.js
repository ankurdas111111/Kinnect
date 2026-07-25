import { mount } from 'svelte';
import App from './App.svelte';
// Self-hosted fonts (Fontsource): no fonts.googleapis.com runtime dependency,
// so Capacitor cold starts render branded type even fully offline.
// VIGIL display face: Bricolage Grotesque variable (wght axis, humanist warmth).
// Sora stays imported for ONE release as the --font-display fallback so slow
// connections never flash unbranded type — remove after visual QA signs off.
// Body: Nunito 400/600/700 (kept). Data: JetBrains Mono 500 (kept).
import '@fontsource-variable/bricolage-grotesque';
import '@fontsource/sora/600.css';
import '@fontsource/sora/700.css';
import '@fontsource/sora/800.css';
import '@fontsource/nunito/400.css';
import '@fontsource/nunito/600.css';
import '@fontsource/nunito/700.css';
import '@fontsource/jetbrains-mono/500.css';
import './global.css';
// OKLCH re-expression of the core color scales — MUST stay between global.css
// (defines the scales) and themes.css (named themes override --primary-*).
import './styles/tokens-oklch.css';
import './styles/components.css';
import './styles/themes.css';
// Daypart tint layer — theme-flavoring, so it sits between themes.css and
// tokens-fx.css (fx stays the last word on blur; the token sets never overlap).
import './styles/tokens-daypart.css';
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
