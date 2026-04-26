import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'node:url';

const isCapacitorTarget = process.env.VITE_TARGET === 'capacitor';
const capacitorStubPath = fileURLToPath(new URL('./src/lib/capacitor-stub.js', import.meta.url));

export default defineConfig({
  plugins: [
    svelte(),
    // Ensure the maplibre chunk gets a <link rel="modulepreload"> in the built
    // HTML. Vite's built-in modulepreload injection already handles this for
    // statically-imported chunks; this plugin is a no-op safety net for when
    // Vite does not automatically detect it (e.g. if the import becomes deeply
    // dynamic). It checks first so it never duplicates what Vite already added.
    {
      name: 'inject-maplibre-preload',
      transformIndexHtml(html, ctx) {
        if (!ctx.bundle) return html;
        const maplibreChunk = Object.values(ctx.bundle).find(
          chunk => chunk.type === 'chunk' && chunk.name === 'maplibre'
        );
        if (!maplibreChunk) return html;
        // maplibreChunk.fileName may already include "assets/" prefix
        const fileName = maplibreChunk.fileName.replace(/^assets\//, '');
        const assetPath = `/assets/${fileName}`;
        // Skip if Vite already injected a modulepreload for this chunk
        if (html.includes(assetPath)) return html;
        const tag = `<link rel="modulepreload" href="${assetPath}" crossorigin />`;
        return html.replace('</head>', `${tag}\n</head>`);
      }
    }
  ],
  cacheDir: process.env.VITE_CACHE_DIR || 'node_modules/.vite',
  resolve: {
    alias: isCapacitorTarget
      ? {}
      : {
          '@capacitor/app': capacitorStubPath,
          '@capacitor/geolocation': capacitorStubPath,
          '@capacitor/device': capacitorStubPath,
          '@capacitor/network': capacitorStubPath,
          '@capacitor/share': capacitorStubPath,
          '@capacitor/haptics': capacitorStubPath,
          '@capacitor/local-notifications': capacitorStubPath,
          '@capacitor/core': capacitorStubPath,
          '@capacitor-community/background-geolocation': capacitorStubPath
        }
  },
  root: '.',
  publicDir: 'public',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    minify: 'esbuild',
    cssCodeSplit: false,
    rollupOptions: {
      // Background geolocation is a native-only Capacitor plugin with no JS dist.
      // It is injected by the native shell at runtime — mark as external so Rollup
      // doesn't try to bundle it (applies for both web and capacitor builds).
      external: ['@capacitor-community/background-geolocation'],
      output: {
        manualChunks(id) {
          if (id.includes('tesseract.js')) return 'tesseract';
          if (id.includes('maplibre-gl')) return 'maplibre';
          if (id.includes('socket.io-client')) return 'socket';
          if (id.includes('node_modules/svelte')) return 'svelte-runtime';
          if (id.includes('/pages/LiveViewer')) return 'page-live';
          if (id.includes('/pages/WatchViewer')) return 'page-watch';
          if (id.includes('/pages/SecretChatViewer')) return 'page-m';
          if (id.includes('/pages/Monitoring')) return 'page-monitoring';
        }
      },
      treeshake: true
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:3001',
      '/ws': {
        target: 'http://localhost:3001',
        ws: true
      },
      '/health': 'http://localhost:3001'
    }
  }
});
