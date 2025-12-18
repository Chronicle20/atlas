/**
 * Service Worker for Character Image Caching
 * Provides offline support for character images with intelligent cache management
 */

const CACHE_NAME = 'atlas-character-images-v1';
const MAPLESTORY_API_ORIGIN = 'https://maplestory.io';
const MAX_CACHE_AGE = 7 * 24 * 60 * 60 * 1000; // 7 days
const MAX_CACHE_SIZE = 100; // Maximum number of cached images

// Install event - set up the cache
self.addEventListener('install', (event) => {
  console.log('[SW] Character cache service worker installing');
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[SW] Character cache service worker activated');
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((cacheName) => cacheName.startsWith('atlas-character-images-'))
          .filter((cacheName) => cacheName !== CACHE_NAME)
          .map((cacheName) => {
            console.log('[SW] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          })
      );
    }).then(() => {
      return self.clients.claim();
    })
  );
});

// Fetch event - intercept requests to MapleStory API
self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  
  // Only handle character image requests
  if (url.origin === MAPLESTORY_API_ORIGIN && url.pathname.includes('/character/')) {
    event.respondWith(handleCharacterImageRequest(event.request));
  }
});

/**
 * Handle character image requests with cache-first strategy
 */
async function handleCharacterImageRequest(request) {
  try {
    const cache = await caches.open(CACHE_NAME);
    const cacheKey = getCacheKey(request.url);
    
    // Try to get from cache first
    const cachedResponse = await cache.match(cacheKey);
    
    if (cachedResponse) {
      // Check if cache is still fresh
      const cacheDate = cachedResponse.headers.get('sw-cache-date');
      if (cacheDate && Date.now() - parseInt(cacheDate) < MAX_CACHE_AGE) {
        console.log('[SW] Serving character image from cache:', request.url);
        return cachedResponse;
      }
    }
    
    // Fetch from network
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // Clone the response for caching
      const responseToCache = networkResponse.clone();
      
      // Add cache metadata
      const headers = new Headers(responseToCache.headers);
      headers.set('sw-cache-date', Date.now().toString());
      
      const cachedResponse = new Response(responseToCache.body, {
        status: responseToCache.status,
        statusText: responseToCache.statusText,
        headers: headers,
      });
      
      // Cache the response
      await cache.put(cacheKey, cachedResponse);
      
      // Clean up old cache entries
      await cleanupCache(cache);
      
      console.log('[SW] Cached character image:', request.url);
    }
    
    return networkResponse;
    
  } catch (error) {
    console.error('[SW] Error handling character image request:', error);
    
    // Try to serve from cache as fallback
    const cache = await caches.open(CACHE_NAME);
    const cachedResponse = await cache.match(getCacheKey(request.url));
    
    if (cachedResponse) {
      console.log('[SW] Serving stale character image from cache:', request.url);
      return cachedResponse;
    }
    
    // Return a basic error response
    return new Response('Character image unavailable', {
      status: 503,
      statusText: 'Service Unavailable',
    });
  }
}

/**
 * Generate a consistent cache key for character images
 */
function getCacheKey(url) {
  // Remove dynamic parameters that don't affect the image
  const urlObj = new URL(url);
  const pathname = urlObj.pathname;
  const searchParams = new URLSearchParams(urlObj.search);
  
  // Keep only essential parameters
  const essentialParams = ['resize', 'renderMode', 'flipX'];
  const filteredParams = new URLSearchParams();
  
  essentialParams.forEach(param => {
    if (searchParams.has(param)) {
      filteredParams.set(param, searchParams.get(param));
    }
  });
  
  return `${pathname}${filteredParams.toString() ? '?' + filteredParams.toString() : ''}`;
}

/**
 * Clean up old cache entries to maintain size limit
 */
async function cleanupCache(cache) {
  const keys = await cache.keys();
  
  if (keys.length <= MAX_CACHE_SIZE) {
    return;
  }
  
  // Get cache entries with timestamps
  const entries = await Promise.all(
    keys.map(async (key) => {
      const response = await cache.match(key);
      const cacheDate = response.headers.get('sw-cache-date');
      return {
        key,
        timestamp: cacheDate ? parseInt(cacheDate) : 0,
      };
    })
  );
  
  // Sort by timestamp (oldest first)
  entries.sort((a, b) => a.timestamp - b.timestamp);
  
  // Delete oldest entries to get under the limit
  const entriesToDelete = entries.slice(0, entries.length - MAX_CACHE_SIZE + 10); // Delete 10 extra for buffer
  
  await Promise.all(
    entriesToDelete.map(entry => {
      console.log('[SW] Deleting old cache entry:', entry.key.url);
      return cache.delete(entry.key);
    })
  );
}

/**
 * Message handler for cache management from main thread
 */
self.addEventListener('message', (event) => {
  if (event.data && event.data.type) {
    switch (event.data.type) {
      case 'CLEAR_CHARACTER_CACHE':
        clearCharacterCache().then(() => {
          event.ports[0].postMessage({ success: true });
        }).catch((error) => {
          event.ports[0].postMessage({ success: false, error: error.message });
        });
        break;
        
      case 'GET_CACHE_STATS':
        getCacheStats().then((stats) => {
          event.ports[0].postMessage({ success: true, stats });
        }).catch((error) => {
          event.ports[0].postMessage({ success: false, error: error.message });
        });
        break;
        
      case 'PRELOAD_IMAGES':
        preloadImages(event.data.urls).then(() => {
          event.ports[0].postMessage({ success: true });
        }).catch((error) => {
          event.ports[0].postMessage({ success: false, error: error.message });
        });
        break;
    }
  }
});

/**
 * Clear all character image cache
 */
async function clearCharacterCache() {
  const cache = await caches.open(CACHE_NAME);
  const keys = await cache.keys();
  await Promise.all(keys.map(key => cache.delete(key)));
  console.log('[SW] Cleared character image cache');
}

/**
 * Get cache statistics
 */
async function getCacheStats() {
  const cache = await caches.open(CACHE_NAME);
  const keys = await cache.keys();
  
  let totalSize = 0;
  const entries = await Promise.all(
    keys.map(async (key) => {
      const response = await cache.match(key);
      const cacheDate = response.headers.get('sw-cache-date');
      const blob = await response.blob();
      totalSize += blob.size;
      
      return {
        url: key.url,
        size: blob.size,
        timestamp: cacheDate ? parseInt(cacheDate) : 0,
      };
    })
  );
  
  return {
    totalEntries: keys.length,
    totalSize,
    entries: entries.sort((a, b) => b.timestamp - a.timestamp), // Newest first
  };
}

/**
 * Preload character images
 */
async function preloadImages(urls) {
  const cache = await caches.open(CACHE_NAME);
  
  const preloadPromises = urls.map(async (url) => {
    try {
      const cacheKey = getCacheKey(url);
      const cached = await cache.match(cacheKey);
      
      if (cached) {
        console.log('[SW] Image already cached:', url);
        return;
      }
      
      const response = await fetch(url);
      if (response.ok) {
        const headers = new Headers(response.headers);
        headers.set('sw-cache-date', Date.now().toString());
        
        const cachedResponse = new Response(response.body, {
          status: response.status,
          statusText: response.statusText,
          headers: headers,
        });
        
        await cache.put(cacheKey, cachedResponse);
        console.log('[SW] Preloaded character image:', url);
      }
    } catch (error) {
      console.error('[SW] Failed to preload image:', url, error);
    }
  });
  
  await Promise.allSettled(preloadPromises);
  await cleanupCache(cache);
}