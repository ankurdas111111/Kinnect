/**
 * epActions.js — small Svelte actions shared by Emergency Profile children.
 * Extracted verbatim from EmergencyProfile.svelte:192–200.
 */

/**
 * autoResize — grow a <textarea> to fit its content, no scrollbar.
 * @param {HTMLTextAreaElement} node
 */
export function autoResize(node) {
  function resize() {
    node.style.height = 'auto';
    node.style.height = node.scrollHeight + 'px';
  }
  node.addEventListener('input', resize);
  resize();
  return { destroy() { node.removeEventListener('input', resize); } };
}
