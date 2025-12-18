import type { NextConfig } from "next";

// Auto-detect container environment
const isContainer = 
  process.env.DOCKER_ENV === 'true' || 
  process.env.KUBERNETES_SERVICE_HOST !== undefined ||
  process.env.NODE_ENV === 'production';

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'maplestory.io',
        port: '',
        pathname: '/api/**',
      },
    ],
    minimumCacheTTL: 60 * 60 * 24, // 24 hours cache for character images
    formats: ['image/webp', 'image/avif'], // Modern formats for better compression
    deviceSizes: [128, 192, 256, 384, 512], // Common character image sizes
    imageSizes: [32, 48, 64, 96, 128, 192, 256], // Icon sizes
    dangerouslyAllowSVG: true, // Allow SVG fallbacks
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;",
    // Disable image optimization in containers to avoid 400 errors
    unoptimized: isContainer,
    // Alternative loader for containers
    ...(isContainer && {
      loader: 'custom' as const,
      loaderFile: './lib/image-loader.ts',
    }),
  },
  // Enable experimental features for better performance
  experimental: {
    optimizePackageImports: ['react-icons', '@tanstack/react-query'],
  },
  // Webpack optimizations for better bundle splitting
  webpack: (config, { dev, isServer }) => {
    if (!dev && !isServer) {
      // Optimize for character rendering components
      config.optimization.splitChunks.cacheGroups = {
        ...config.optimization.splitChunks.cacheGroups,
        character: {
          name: 'character-rendering',
          chunks: 'all',
          test: /[\\/]components[\\/]features[\\/]characters[\\/]/,
          priority: 30,
        },
        maplestory: {
          name: 'maplestory-api',
          chunks: 'all',
          test: /[\\/]services[\\/]api[\\/]maplestory/,
          priority: 25,
        },
      };
    }
    return config;
  },
};

export default nextConfig;
