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
            // eslint-disable-next-line no-console
            console.log('proxy error', err);
          });
          proxy.on('proxyReq', (proxyReq, req, _res) => {
            // eslint-disable-next-line no-console
            console.log('Sending Request to the Target:', req.method, req.url);
            // eslint-disable-next-line no-console
            console.log('Request Headers:', req.headers);
            // eslint-disable-next-line no-console
            console.log('Proxy Request Headers:', proxyReq.getHeaders());
          });
          proxy.on('proxyRes', (proxyRes, req, _res) => {
            // eslint-disable-next-line no-console
            console.log('Received Response from the Target:', proxyRes.statusCode, req.url);
            // eslint-disable-next-line no-console
            console.log('Response Headers:', proxyRes.headers);
          });
        },
      }
    }
  }
})
