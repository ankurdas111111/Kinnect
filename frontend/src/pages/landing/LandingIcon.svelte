<script>
  /**
   * Inline 24px-grid icon set for the landing — hand-drawn equivalents of
   * the design's Material Symbols so the page ships no icon font. Stroke
   * icons inherit currentColor; all decorative (aria-hidden).
   */
  let { name, size = 20 } = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="1.8"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
  focusable="false"
>
  {#if name === 'arrow-forward'}
    <path d="M4.5 12h15" /><path d="m13.5 6 6 6-6 6" />
  {:else if name === 'play'}
    <circle cx="12" cy="12" r="8.5" />
    <path d="M10.2 8.8l5 3.2-5 3.2z" fill="currentColor" stroke="none" />
  {:else if name === 'cottage'}
    <path d="m3.5 11.5 8.5-7 8.5 7" />
    <path d="M6 10.2V20h12v-9.8" />
    <path d="M10 20v-5.5h4V20" />
  {:else if name === 'walk'}
    <circle cx="13.6" cy="4.6" r="1.9" />
    <path d="M13.2 8 10.6 9.9l-1.1 3.6" />
    <path d="m13.2 8 1.4 3.6 2.9 1.4" />
    <path d="m11.9 12.9-3 7.1" />
    <path d="m13.9 13.4 1.4 2.4.6 4.2" />
  {:else if name === 'shield-check'}
    <path d="M12 3.5 5.5 6v5c0 4.3 2.8 7.4 6.5 8.8 3.7-1.4 6.5-4.5 6.5-8.8V6Z" />
    <path d="m9.3 11.7 1.9 1.9 3.5-4.1" />
  {:else if name === 'chevron-left'}
    <path d="m14.5 6-6 6 6 6" />
  {:else if name === 'quote'}
    <path
      d="M5 16.5c-.8-.9-1.3-2-1.3-3.4C3.7 9.9 5.6 7.6 8.4 6.5l.7 1.3c-1.7.8-2.7 2-2.9 3.4.3-.1.6-.2 1-.2 1.5 0 2.6 1.1 2.6 2.6s-1.2 2.9-2.8 2.9c-.7 0-1.4-.3-2-.9Z"
      fill="currentColor" stroke="none" />
    <path
      d="M13.6 16.5c-.8-.9-1.3-2-1.3-3.4 0-3.2 1.9-5.5 4.7-6.6l.7 1.3c-1.7.8-2.7 2-2.9 3.4.3-.1.6-.2 1-.2 1.5 0 2.6 1.1 2.6 2.6s-1.2 2.9-2.8 2.9c-.7 0-1.4-.3-2-.9Z"
      fill="currentColor" stroke="none" />
  {:else if name === 'bell-paused'}
    <path d="M18 16H6c1-1.4 1.5-2.6 1.5-5.2A4.5 4.5 0 0 1 12 6.3a4.5 4.5 0 0 1 4.5 4.5c0 2.6.5 3.8 1.5 5.2Z" />
    <path d="M10.4 19a1.7 1.7 0 0 0 3.2 0" />
    <path d="M10.8 9.8v3.2" /><path d="M13.2 9.8v3.2" />
  {:else if name === 'lock-open'}
    <rect x="5" y="10.5" width="14" height="9" rx="2.2" />
    <path d="M8.5 10.5V8a3.5 3.5 0 0 1 6.8-1.2" />
    <path d="M12 14v2.2" />
  {:else if name === 'check-circle'}
    <circle cx="12" cy="12" r="8.5" />
    <path d="m8.4 12.2 2.4 2.4 4.8-5.2" />
  {:else if name === 'heart'}
    <path
      d="M12 19.6C7 15.9 4.3 13 4.3 9.9 4.3 7.7 6 6 8.1 6c1.5 0 2.9.8 3.9 2.2C13 6.8 14.4 6 15.9 6 18 6 19.7 7.7 19.7 9.9c0 3.1-2.7 6-7.7 9.7Z"
      fill="currentColor" stroke="none" />
  {:else if name === 'sos-star'}
    <path d="M12 3.8v16.4" /><path d="m4.9 7.9 14.2 8.2" /><path d="M19.1 7.9 4.9 16.1" />
  {:else if name === 'sos-cross'}
    <path d="M9.6 3.8h4.8v5.8h5.8v4.8h-5.8v5.8H9.6v-5.8H3.8V9.6h5.8Z" fill="currentColor" stroke="none" />
  {:else if name === 'radial-hold'}
    <circle cx="12" cy="12" r="3" />
    <path d="M12 3a9 9 0 0 1 9 9" />
    <path d="M21 12a9 9 0 0 1-9 9" opacity="0.45" />
    <path d="M3 12a9 9 0 0 1 9-9" opacity="0.45" />
  {:else if name === 'volume'}
    <path d="M4.5 9.5v5h3l4 3.5v-12l-4 3.5Z" />
    <path d="M15 9a4.2 4.2 0 0 1 0 6" />
    <path d="M17.5 6.6a7.6 7.6 0 0 1 0 10.8" />
  {:else if name === 'lock'}
    <rect x="5" y="10.5" width="14" height="9" rx="2.2" />
    <path d="M8.5 10.5V8a3.5 3.5 0 0 1 7 0v2.5" />
    <path d="M12 14v2.2" />
  {:else if name === 'cloud-off'}
    <path d="M8.6 5.6A5.5 5.5 0 0 1 17.3 9.9 3.8 3.8 0 0 1 19.9 14" />
    <path d="M5.2 7.7A5.5 5.5 0 0 0 6.7 18h9.1" />
    <path d="m4 4 16 16" />
  {:else if name === 'android'}
    <path d="M5 16.5a7 7 0 0 1 14 0Z" />
    <path d="m7.4 6.9 1.5 2.6" /><path d="m16.6 6.9-1.5 2.6" />
    <path d="M9.4 13.6h.01" stroke-width="2.4" /><path d="M14.6 13.6h.01" stroke-width="2.4" />
  {/if}
</svg>
