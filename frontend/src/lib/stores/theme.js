import { writable } from 'svelte/store';

/**
 * VIGIL identity — two faces of one brand.
 * `id` doubles as the data-theme attribute value; global.css +
 * styles/tokens-oklch.css carry the actual token blocks.
 */
export const THEMES = [
  {
    id: 'dark',
    name: 'Vigil',
    desc: 'Night-sky indigo',
    category: 'core',
    colors: { bg: '#0b0d16', accent: '#5863d3', secondary: '#90a6f7', text: '#f5f3ef' }, // raw-color-ok — swatch preview literals
    animated: false,
  },
  {
    id: 'light',
    name: 'Dawn',
    desc: 'Warm paper',
    category: 'core',
    colors: { bg: '#f9f6f2', accent: '#5863d3', secondary: '#474dad', text: '#1f212b' }, // raw-color-ok — swatch preview literals
    animated: false,
  },
];

/** Retired pre-VIGIL theme ids → nearest VIGIL face. Keep in sync with the
 *  pre-paint migration in index.html — both sides must agree or returning
 *  users get a flash of unstyled theme on first load. */
const LEGACY = { vapor: 'dark', aurora: 'dark', midnight: 'dark', 'deep-ocean': 'dark', bloom: 'dark' };

function normalize(id) {
  if (id && LEGACY[id]) return LEGACY[id];
  return THEMES.some((t) => t.id === id) ? id : null;
}

function createThemeStore() {
  const { subscribe, set } = writable('dark');

  function apply(id) {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', id);
    }
    try {
      localStorage.setItem('theme', id);
    } catch (e) { /* private mode */ }
    set(id);
  }

  return {
    subscribe,
    /** @param {string} id — 'dark' | 'light' (legacy ids migrate) */
    set(id) {
      apply(normalize(id) || 'dark');
    },
    init() {
      let stored = null;
      try {
        stored = normalize(localStorage.getItem('theme'));
      } catch (e) { /* private mode */ }
      const osLight =
        typeof window !== 'undefined' &&
        window.matchMedia?.('(prefers-color-scheme: light)').matches;
      apply(stored || (osLight ? 'light' : 'dark'));
    },
  };
}

export const themeStore = createThemeStore();
