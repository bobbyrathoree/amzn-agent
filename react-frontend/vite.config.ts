import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        // NOTE: This is the old us-east-1 production endpoint - replace with us-west-2 endpoint after deployment
        target: 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
        configure: (proxy, _options) => {
          proxy.on('error', (err, _req, _res) => {
            console.log('proxy error', err);
          });
        },
      },
      '/development': {
        // Dev environment API - maps to us-west-2 dev endpoint
        target: 'https://qzhjqg9q72.execute-api.us-west-2.amazonaws.com/dev',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/development/, ''),
        configure: (proxy, _options) => {
          proxy.on('error', (err, _req, _res) => {
            console.log('proxy error', err);
          });
        },
      }
    }
  }
})
