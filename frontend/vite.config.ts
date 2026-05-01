import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ArcoResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    Components({
      dts: false,
      resolvers: [
        ArcoResolver({
          importStyle: 'css',
        }),
      ],
    }),
  ],
  server: {
    host: '127.0.0.1',
    port: 4173,
    proxy: {
      '/__api': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/__api/, ''),
      },
    },
  },
  build: {
    outDir: '../static/admin_spa',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined
          }

          if (id.includes('vue-router')) {
            return 'vue-router'
          }

          if (id.includes('pinia')) {
            return 'pinia'
          }

          if (id.includes('/vue/')) {
            return 'vue'
          }

          return 'vendor'
        },
      },
    },
  }
})
