import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'copy-manifest',
      apply: 'build',
      enforce: 'post',
      generateBundle() {
        const manifest = fs.readFileSync('manifest.json', 'utf-8');
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
          { src: 'src/popup/popup.html', dest: 'popup/popup.html' },
          { src: 'src/sidebar/sidebar.html', dest: 'sidebar/sidebar.html' },
          { src: 'src/settings/settings.html', dest: 'settings/settings.html' },
        ];

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
    outDir: 'dist',
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
        background: path.resolve(__dirname, 'src/background/serviceWorker.ts'),
        content: path.resolve(__dirname, 'src/content/contentScript.ts'),
        popup: path.resolve(__dirname, 'src/popup/index.ts'),
        sidebar: path.resolve(__dirname, 'src/sidebar/index.ts'),
        settings: path.resolve(__dirname, 'src/settings/index.ts'),
      },
      name: 'Moly',
      formats: ['es'],
    },
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            if (id.includes('zustand')) {
              return 'zustand';
            }
            if (id.includes('react')) {
              return 'react-vendor';
            }
            return 'vendor';
          }
        },
      },
    },
  },
  server: {
    port: 3000,
    strictPort: false,
  },
});
