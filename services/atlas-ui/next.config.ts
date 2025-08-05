import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  images: {
    domains: ['maplestory.io'],
    minimumCacheTTL: 60 * 60 * 24, // 24 hours cache for character images
  },
};

export default nextConfig;
