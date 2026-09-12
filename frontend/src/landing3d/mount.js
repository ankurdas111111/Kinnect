/**
 * Mounts the React landing island into a Svelte-owned element.
 *
 * This is the ONLY bridge between the two frameworks. Everything React in this
 * app lives behind this module, which is dynamically imported by the landing
 * route and aliased away entirely for native builds (vite.config.js), so the
 * APK ships zero bytes of react, r3f, drei, postprocessing or gsap.
 */
export async function mountLanding(el, { onExit } = {}) {
  const [{ createRoot }, { createElement }, { default: App }] = await Promise.all([
    import('react-dom/client'),
    import('react'),
    import('./App.jsx'),
  ]);
  await import('./landing3d.css');

  const root = createRoot(el);
  root.render(createElement(App, { onExit }));
  return () => root.unmount();
}
