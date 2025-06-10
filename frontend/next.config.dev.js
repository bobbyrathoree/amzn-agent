/** @type {import('next').NextConfig} */
const nextConfig = {
  // Development configuration - supports API routes
  experimental: {
    esmExternals: 'loose'
  },
  webpack: (config) => {
    // Ignore .map files to reduce noise in dev mode
    config.module.rules.push({
      test: /\.map$/,
      use: 'ignore-loader'
    });
    return config;
  }
}

module.exports = nextConfig