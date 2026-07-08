import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { compression } from 'vite-plugin-compression2';
import { fileURLToPath } from 'node:url';

const isCapacitorTarget = process.env.VITE_TARGET === 'capacitor';
const capacitorStubPath = fileURLToPath(new URL('./src/lib/capacitor-stub.js', import.meta.url));

// Emit .br + .gz siblings next to every compressible build asset. The Go
// static handler serves them with Content-Encoding (Render has no edge
// compression and the runtime gzip middleware skips ServeFile responses).
const compressInclude = /\.(js|css|html|svg|json|map|wasm|txt|xml)$/;

export default defineConfig({
  plugins: [
    svelte(),
    compression({ include: compressInclude, algorithms: ['brotliCompress'], threshold: 1024 }),
    compression({ include: compressInclude, algorithms: ['gzip'], threshold: 1024 }),
    // Ensure the maplibre chunk gets a <link rel="modulepreload"> in the built
    // HTML. Vite's built-in modulepreload injection already handles this for
    // statically-imported chunks; this plugin is a no-op safety net for when
    // Vite does not automatically detect it (e.g. if the import becomes deeply
    // dynamic). It checks first so it never duplicates what Vite already added.
    {
      name: 'inject-critical-preloads',
      transformIndexHtml(html, ctx) {
        if (!ctx.bundle) return html;
        const chunks = Object.values(ctx.bundle);
        let result = html;
        for (const chunkName of ['maplibre']) {
          const chunk = chunks.find(c => c.type === 'chunk' && c.name === chunkName);
          if (!chunk) continue;
          const assetPath = `/assets/${chunk.fileName.replace(/^assets\//, '')}`;
          if (result.includes(assetPath)) continue;
          result = result.replace('</head>', `<link rel="modulepreload" href="${assetPath}" crossorigin />\n</head>`);
        }
        return result;
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
    cssCodeSplit: true,
    rollupOptions: {
      // Background geolocation is a native-only Capacitor plugin with no JS dist.
      // It is injected by the native shell at runtime — mark as external so Rollup
      // doesn't try to bundle it (applies for both web and capacitor builds).
      external: ['@capacitor-community/background-geolocation'],
      output: {
        manualChunks(id) {
          // tesseract.js is dynamically imported by lib/rideImport.js (OCR)
          if (id.includes('tesseract.js')) return 'tesseract';
          if (id.includes('maplibre-gl')) return 'maplibre';
          if (id.includes('node_modules/svelte')) return 'svelte-runtime';
          // Crypto is loaded only by secret chat — keep it in its own chunk
          if (id.includes('/lib/crypto')) return 'lib-crypto';
          // Secret chat surface (viewer + panel + message + gate) in one async chunk
          if (id.includes('/pages/SecretChatViewer')) return 'page-m';
          if (
            id.includes('/components/SecretChatPanel') ||
            id.includes('/components/SecretChatMessage') ||
            id.includes('/components/SecretChatGate') ||
            id.includes('/components/SecretChatCompose') ||
            id.includes('/components/SecretChatInlineDecrypt')
          ) return 'page-m';
          if (id.includes('/pages/LiveViewer')) return 'page-live';
          if (id.includes('/pages/WatchViewer')) return 'page-watch';
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
