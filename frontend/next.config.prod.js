/** @type {import('next').NextConfig} */
const nextConfig = {
  // Production configuration - static export for S3/CloudFront deployment
  output: 'export',
  trailingSlash: true,
  images: {
    unoptimized: true
  },
  distDir: 'out',
  experimental: {
    esmExternals: 'loose'
  },
  env: {
    NEXT_PUBLIC_BUILD_MODE: 'production'
  },
  webpack: (config, { dev, isServer }) => {
    // Exclude API routes from production builds
    if (!dev && isServer) {
      config.module.rules.push({
        test: /src\/app\/api\/.*\.ts$/,
        use: 'ignore-loader'
      });
    }
    return config;
  }
}

module.exports = nextConfig