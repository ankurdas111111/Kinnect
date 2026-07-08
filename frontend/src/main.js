import { mount } from 'svelte';
import App from './App.svelte';
// Self-hosted fonts (Fontsource): no fonts.googleapis.com runtime dependency,
// so Capacitor cold starts render branded type even fully offline.
// Weights match what the UI uses: Sora 600/700/800, Nunito 400/600/700, JetBrains Mono 500.
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
// Modernization FX layer — MUST be last so [data-fx] calm-mode overrides win
// over theme blur values. Additive tokens (bento/tactile/color-mix) are safe.
import './styles/tokens-fx.css';

import { effects } from './lib/stores/effects.js';

// Pick the effects level (stored pref or device-capability default) before
// first paint so glass/blur render at the right budget from the start.
effects.init();

const app = mount(App, { target: document.getElementById('app') });

export default app;
