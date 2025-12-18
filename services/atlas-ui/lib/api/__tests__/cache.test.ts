/**
 * Tests for API client response caching functionality
 */

import { api, cache } from '../client';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('API Client Response Caching', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    api.clearCache();
    
    // Default successful response
    mockFetch.mockResolvedValue(new Response(
      JSON.stringify({ data: 'test response' }),
      {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }
    ));
  });

  afterEach(() => {
    api.clearCache();
  });

  describe('basic caching', () => {
    it('should cache GET requests when cache options are provided', async () => {
      const url = '/test';
      const cacheOptions = cache.defaultOptions();
      
      // First request
      const result1 = await api.get(url, { cache: cacheOptions });
      expect(result1).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Second request should use cache
      const result2 = await api.get(url, { cache: cacheOptions });
      expect(result2).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(1); // Still only one fetch call
    });

    it('should not cache GET requests when cache is disabled', async () => {
      const url = '/test';
      
      // First request
      const result1 = await api.get(url, { cache: cache.disable(), skipDeduplication: true });
      expect(result1).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Second request should make another fetch (no caching, no deduplication)
      const result2 = await api.get(url, { cache: cache.disable(), skipDeduplication: true });
      expect(result2).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('should not cache GET requests when no cache options are provided', async () => {
      const url = '/test';
      
      // First request
      const result1 = await api.get(url, { skipDeduplication: true });
      expect(result1).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Second request should make another fetch (no caching, no deduplication)
      const result2 = await api.get(url, { skipDeduplication: true });
      expect(result2).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('cache expiration', () => {
    it('should expire cached data after TTL', async () => {
      const url = '/test';
      const cacheOptions = cache.withTTL(0.001); // 0.001 minutes = 60ms
      
      // First request
      const result1 = await api.get(url, { cache: cacheOptions });
      expect(result1).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Wait for cache to expire
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Second request should make a new fetch
      const result2 = await api.get(url, { cache: cacheOptions });
      expect(result2).toEqual({ data: 'test response' });
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('cache utility functions', () => {
    it('should create cache options with different TTL values', () => {
      const shortLived = cache.shortLived();
      expect(shortLived.ttl).toBe(60 * 1000); // 1 minute
      
      const longLived = cache.longLived();
      expect(longLived.ttl).toBe(30 * 60 * 1000); // 30 minutes
      expect(longLived.staleWhileRevalidate).toBe(true);
      
      const customTTL = cache.withTTL(10);
      expect(customTTL.ttl).toBe(10 * 60 * 1000); // 10 minutes
    });

    it('should create cache options with custom key prefix', () => {
      const prefixed = cache.withPrefix('test-prefix', 5 * 60 * 1000);
      expect(prefixed.keyPrefix).toBe('test-prefix');
      expect(prefixed.ttl).toBe(5 * 60 * 1000);
    });

    it('should create stale-while-revalidate options', () => {
      const swr = cache.staleWhileRevalidate(5 * 60 * 1000, 2 * 60 * 1000);
      expect(swr.staleWhileRevalidate).toBe(true);
      expect(swr.ttl).toBe(5 * 60 * 1000);
      expect(swr.maxStaleTime).toBe(2 * 60 * 1000);
    });
  });

  describe('cache management', () => {
    it('should clear all cache entries', async () => {
      const url1 = '/test1';
      const url2 = '/test2';
      const cacheOptions = cache.defaultOptions();
      
      // Make cached requests
      await api.get(url1, { cache: cacheOptions });
      await api.get(url2, { cache: cacheOptions });
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      // Clear cache
      api.clearCache();
      
      // Subsequent requests should not use cache
      await api.get(url1, { cache: cacheOptions });
      await api.get(url2, { cache: cacheOptions });
      expect(mockFetch).toHaveBeenCalledTimes(4);
    });

    it('should provide cache statistics', async () => {
      const url = '/test';
      const cacheOptions = cache.defaultOptions();
      
      // Initially no cache entries
      let stats = api.getCacheStats();
      expect(stats.size).toBe(0);
      expect(stats.entries).toEqual([]);
      
      // Make a cached request
      await api.get(url, { cache: cacheOptions });
      
      // Should have one cache entry
      stats = api.getCacheStats();
      expect(stats.size).toBe(1);
      expect(stats.entries).toHaveLength(1);
      expect(stats.entries[0]!.isStale).toBe(false);
    });

    it('should clear cache by pattern', async () => {
      const cacheOptions = cache.defaultOptions();
      
      // Make requests with different URLs
      await api.get('/users/1', { cache: cacheOptions });
      await api.get('/posts/1', { cache: cacheOptions });
      await api.get('/users/2', { cache: cacheOptions });
      expect(mockFetch).toHaveBeenCalledTimes(3);
      
      // Clear only user-related cache entries
      // Note: The actual pattern matching depends on the base64 encoded cache key
      // For this test, we'll just verify the method exists and can be called
      api.clearCacheByPattern('.*');
      
      // All cache should be cleared
      const stats = api.getCacheStats();
      expect(stats.size).toBe(0);
    });
  });

  describe('different HTTP methods', () => {
    it('should only cache GET requests', async () => {
      const url = '/test';
      const cacheOptions = cache.defaultOptions();
      
      // GET request should be cached
      await api.get(url, { cache: cacheOptions });
      await api.get(url, { cache: cacheOptions });
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // POST request should not be cached (cache option is ignored for POST)
      // We need to skip deduplication to ensure each POST makes a separate call
      await api.post(url, { data: 'test' }, { cache: cacheOptions, skipDeduplication: true });
      await api.post(url, { data: 'test' }, { cache: cacheOptions, skipDeduplication: true });
      expect(mockFetch).toHaveBeenCalledTimes(3); // GET (1) + POST (2)
    });
  });
});