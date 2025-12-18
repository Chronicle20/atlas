/**
 * Image utilities for handling different environments and optimization strategies
 */

/**
 * Checks if we're running in a containerized environment
 */
export function isContainerEnvironment(): boolean {
  return process.env.DOCKER_ENV === 'true' || 
         process.env.KUBERNETES_SERVICE_HOST !== undefined ||
         process.env.NODE_ENV === 'production';
}

/**
 * Determines if images should be unoptimized based on environment
 * This helps avoid 400 errors in containerized environments
 */
export function shouldUseUnoptimizedImages(): boolean {
  // Always use unoptimized in containers
  if (isContainerEnvironment()) {
    return true;
  }
  
  // Use unoptimized for external images by default to avoid issues
  return process.env.DISABLE_IMAGE_OPTIMIZATION === 'true';
}

/**
 * Determines if a specific image source should be unoptimized
 */
export function shouldUnoptimizeImageSrc(src: string): boolean {
  // Always unoptimize external URLs to avoid optimization issues
  if (src.startsWith('http://') || src.startsWith('https://')) {
    return true;
  }
  
  // Always unoptimize SVG files
  if (src.endsWith('.svg')) {
    return true;
  }
  
  // Use global setting for other images
  return shouldUseUnoptimizedImages();
}

/**
 * Gets image loading strategy based on environment
 */
export function getImageLoadingStrategy(): 'lazy' | 'eager' {
  // In containers, use eager loading to avoid lazy loading issues
  if (isContainerEnvironment()) {
    return 'eager';
  }
  
  return 'lazy';
}