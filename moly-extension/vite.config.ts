import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';

// Firefox build (MOLY_TARGET=firefox): same source, a different manifest, built into dist-firefox. Firefox has no side panel
// and no service workers: it takes sidebar_action and a background page script. The add-on ID is fixed by us (the listing and
// the native host manifest both name it); the browser gives each installation its own random moz-extension:// origin.
const FIREFOX = process.env.MOLY_TARGET === 'firefox';
const FIREFOX_ADDON_ID = 'moly@nireus79.github.io';

function toFirefoxManifest(manifest: any): any {
  const { side_panel, background, permissions, ...rest } = manifest;
  return {
    ...rest,
    permissions: permissions.filter((p: string) => p !== 'sidePanel'),
    background: { scripts: [background.service_worker], type: 'module' },
    sidebar_action: {
      default_title: 'Moly',
      default_panel: side_panel.default_path,
      default_icon: manifest.action.default_icon,
    },
    browser_specific_settings: {
      gecko: {
        id: FIREFOX_ADDON_ID,
        strict_min_version: '140.0',
        // Moly itself sends nothing to the developer; the text goes to the local app, which uses the AI provider the person chose.
        data_collection_permissions: { required: ['none'] },
      },
    },
  };
}

export default defineConfig({
  define: {
    'process.env': JSON.stringify({}),
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  plugins: [
    react(),
    {
      name: 'copy-manifest',
      apply: 'build',
      enforce: 'post',
      generateBundle() {
        let manifest = fs.readFileSync('manifest.json', 'utf-8');
        // Beta / sideload build (MOLY_BETA=1): a fixed public key gives the extension a fixed ID that the installers know,
        // without the store. The Web Store build must NOT carry this key: the store assigns its own ID.
        if (process.env.MOLY_BETA === '1') {
          const key = fs.readFileSync(path.resolve(__dirname, '../installer/beta-extension-key.pub'), 'utf-8').trim();
          manifest = JSON.stringify({ ...JSON.parse(manifest), key }, null, 2);
        }
        if (FIREFOX) manifest = JSON.stringify(toFirefoxManifest(JSON.parse(manifest)), null, 2);
        this.emitFile({
          type: 'asset',
          fileName: 'manifest.json',
          source: manifest,
        });
      },
    },
    {
      name: 'copy-html-files',
      apply: 'build',
      enforce: 'post',
      generateBundle() {
        const htmlFiles = [
          { src: 'src/sidebar/sidebar.html', dest: 'sidebar/sidebar.html' },
          { src: 'src/settings/settings.html', dest: 'settings/settings.html' },
        ];

        // The icons named in manifest.json must be in the package.
        [16, 32, 48, 128].forEach((size) => {
          this.emitFile({
            type: 'asset',
            fileName: `images/icon${size}.png`,
            source: fs.readFileSync(`images/icon${size}.png`),
          });
        });

        htmlFiles.forEach((file) => {
          try {
            const content = fs.readFileSync(file.src, 'utf-8');
            this.emitFile({
              type: 'asset',
              fileName: file.dest,
              source: content,
            });
          } catch (err) {
            console.warn(`Failed to copy ${file.src}:`, err);
          }
        });
      },
    },
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: FIREFOX ? 'dist-firefox' : 'dist',
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        passes: 2,
      },
      mangle: true,
      format: {
        comments: false,
      },
    },
    sourcemap: false,
    reportCompressedSize: true,
    lib: {
      entry: {
        background: path.resolve(__dirname, 'src/background.ts'),
        sidebar: path.resolve(__dirname, 'src/sidebar/index.ts'),
        settings: path.resolve(__dirname, 'src/settings/index.ts'),
      },
      name: 'Moly',
      formats: ['es'],
    },
    rollupOptions: {
      output: {
        entryFileNames: '[name].js',
        chunkFileNames: 'chunk-[name]-[hash].js',
        assetFileNames: '[name][extname]',
      },
    },
  },
  server: {
    port: 3000,
    strictPort: false,
  },
});
