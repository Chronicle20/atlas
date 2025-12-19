/**
 * Custom image loader for containerized environments
 * This bypasses Next.js image optimization to avoid 400 errors in Docker containers
 */

export interface ImageLoaderProps {
  src: string;
  width: number;
  quality?: number;
}

export default function imageLoader({ src }: ImageLoaderProps): string {
  // For external URLs (like maplestory.io), return as-is to avoid optimization issues
  if (src.startsWith('http://') || src.startsWith('https://')) {
    return src;
  }
  
  // For local SVG files, return as-is since they don't need optimization
  if (src.endsWith('.svg')) {
    return src;
  }
  
  // For other local files, we could implement custom logic here
  // For now, return as-is to avoid optimization issues in containers
  return src;
}