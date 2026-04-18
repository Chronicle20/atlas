/**
 * Returns the default image loading strategy. Plain <img> supports
 * native lazy loading; we default to "lazy" everywhere.
 */
export function getImageLoadingStrategy(): "lazy" | "eager" {
  return "lazy";
}
