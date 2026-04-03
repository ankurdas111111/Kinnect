import { writable } from 'svelte/store';

/** Single active theme — Vaporwave only. */
export const THEMES = [
  {
    id: 'vapor',
    name: 'Vaporwave',
    desc: 'Y2K holographic',
    category: 'genz',
    colors: { bg: '#0d0b14', accent: '#a855f7', secondary: '#ec4899', text: '#f0e8ff' },
    animated: true,
  },
];

function createThemeStore() {
  const { subscribe, set } = writable('vapor');

  function applyTheme() {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', 'vapor');
    }
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('theme', 'vapor');
    }
    set('vapor');
  }

  return {
    subscribe,
    set: applyTheme,
    init() { applyTheme(); },
  };
}

export const themeStore = createThemeStore();
