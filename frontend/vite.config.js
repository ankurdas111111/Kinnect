import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import react from '@vitejs/plugin-react';
import { compression } from 'vite-plugin-compression2';
import { fileURLToPath } from 'node:url';

const isCapacitorTarget = process.env.VITE_TARGET === 'capacitor';
const capacitorStubPath = fileURLToPath(new URL('./src/lib/capacitor-stub.js', import.meta.url));
// The landing page is web-only: the native app redirects away from it
// (lib/entryPoint.js). These aliases make that a BUILD-TIME guarantee too, so
// three.js, React and R3F are never emitted into the APK/IPA at all.
//
// NOTE: this previously pointed at three/heroConstellation.js, a file that no
// longer exists — the alias silently matched nothing and three.js was shipping
// in native builds. Keep these paths in step with the real module names.
const heroDioramaPath = fileURLToPath(new URL('./src/lib/three/heroDiorama.js', import.meta.url));
const landingMountPath = fileURLToPath(new URL('./src/landing3d/mount.js', import.meta.url));

// Emit .br + .gz siblings next to every compressible build asset. The Go
// static handler serves them with Content-Encoding (Render has no edge
// compression and the runtime gzip middleware skips ServeFile responses).
const compressInclude = /\.(js|css|html|svg|json|map|wasm|txt|xml)$/;

export default defineConfig({
  plugins: [
    svelte(),
    // JSX only — the landing3d island. Scoped so the plugin never touches the
    // Svelte components or plain .js modules that make up the rest of the app.
    react({ include: /\.(jsx|tsx)$/ }),
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
      ? [
          // Native builds: swap the three.js hero module for the capacitor stub
          // BEFORE resolution, so three is never pulled into the graph (0 bytes
          // in the APK/IPA). The regex matches the WHOLE specifier (any number of
          // leading ../) exactly as written in Landing's dynamic import, plus any
          // absolute resolution of the same file — so the replacement is the full
          // stub path, never a mangled fragment.
          { find: /^(?:\.\.\/)+lib\/three\/heroDiorama(?:\.js)?$/, replacement: capacitorStubPath },
          { find: heroDioramaPath, replacement: capacitorStubPath },
          // The React landing island: keeps react, r3f, drei, postprocessing
          // and gsap out of the native bundle entirely.
          { find: /^(?:\.\.\/)+landing3d\/mount(?:\.js)?$/, replacement: capacitorStubPath },
          { find: landingMountPath, replacement: capacitorStubPath },
        ]
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
          // three.js — Landing hero constellation only, dynamically imported.
          // Pinned into an async 'three' chunk; deliberately NOT added to the
          // inject-critical-preloads plugin (must stay off the critical path).
          if (id.includes('node_modules/three')) return 'three';
          // React + R3F + drei + postprocessing + GSAP power the landing only.
          // One async chunk, loaded when /#/landing mounts and never before, so
          // the app's initial route budget is untouched.
          if (
            id.includes('node_modules/react') ||
            id.includes('node_modules/react-dom') ||
            id.includes('node_modules/scheduler') ||
            id.includes('node_modules/@react-three') ||
            id.includes('node_modules/postprocessing') ||
            id.includes('node_modules/gsap') ||
            id.includes('node_modules/@gsap') ||
            id.includes('node_modules/maath') ||
            id.includes('node_modules/zustand') ||
            id.includes('/src/landing3d/')
          ) return 'landing3d';
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
