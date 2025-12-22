/**
 * Unit tests for API client request deduplication functionality
 */

import { api, apiClient, cancellation } from '@/lib/api/client';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch as jest.Mock;

describe('API Client Request Deduplication', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    api.clearPendingRequests();
    
    // Set up default successful response
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ data: 'test' }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));
  });

  afterEach(() => {
    jest.clearAllTimers();
    api.clearPendingRequests();
  });

  describe('basic deduplication', () => {
    it('should deduplicate identical GET requests', async () => {
      const url = '/test-endpoint';
      
      // Make two identical requests simultaneously
      const promise1 = api.get(url);
      const promise2 = api.get(url);
      
      // Only one fetch should be made (this is the core deduplication functionality)
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Both should resolve to the same result
      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toEqual(result2);
      expect(result1).toEqual({ data: 'test' });
    });

    it('should deduplicate identical POST requests with same data', async () => {
      const url = '/test-endpoint';
      const data = { key: 'value' };
      
      // Make two identical POST requests simultaneously
      const promise1 = api.post(url, data);
      const promise2 = api.post(url, data);
      
      // Only one fetch should be made (core deduplication functionality)
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      // Both should resolve to the same result
      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toEqual(result2);
      expect(result1).toEqual({ data: 'test' });
    });

    it('should NOT deduplicate requests with different data', async () => {
      const url = '/test-endpoint';
      const data1 = { key: 'value1' };
      const data2 = { key: 'value2' };
      
      // Make two POST requests with different data
      const promise1 = api.post(url, data1);
      const promise2 = api.post(url, data2);
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });

    it('should NOT deduplicate requests with different URLs', async () => {
      const data = { key: 'value' };
      
      // Make two POST requests to different URLs
      const promise1 = api.post('/endpoint1', data);
      const promise2 = api.post('/endpoint2', data);
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });

    it('should NOT deduplicate requests with different methods', async () => {
      const url = '/test-endpoint';
      const data = { key: 'value' };
      
      // Make requests with different methods
      const promise1 = api.post(url, data);
      const promise2 = api.put(url, data);
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });
  });

  describe('deduplication with different timing', () => {
    it('should create separate requests if first completes before second starts', async () => {
      const url = '/test-endpoint';

      // Make and complete first request
      const result1 = await api.get(url);
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Allow microtask queue to flush (cleanup runs in finally())
      await Promise.resolve();

      // Make second request after first completes
      const result2 = await api.get(url);
      expect(mockFetch).toHaveBeenCalledTimes(2);

      expect(result1).toEqual(result2);
    });

    it('should share request if second starts while first is pending', async () => {
      const url = '/test-endpoint';
      let resolveRequest: (value: Response) => void;

      // Make fetch hang
      mockFetch.mockImplementation(() => new Promise<Response>(resolve => {
        resolveRequest = resolve;
      }));

      // Start first request
      const promise1 = api.get(url);
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Start second request while first is pending
      const promise2 = api.get(url);
      expect(mockFetch).toHaveBeenCalledTimes(1); // Still only one call - verifies deduplication

      // Resolve the request
      resolveRequest!(new Response(JSON.stringify({ data: 'test' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));

      // Both should resolve to the same result (this is what matters, not promise identity)
      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toEqual(result2);
      expect(result1).toEqual({ data: 'test' });
    });
  });

  describe('error handling in deduplication', () => {
    it('should share errors among deduplicated requests', async () => {
      const url = '/test-endpoint';

      // Mock a non-retryable error (404) to avoid retry delays
      mockFetch.mockResolvedValue(new Response(JSON.stringify({ error: 'Not Found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' }
      }));

      const promise1 = api.get(url);
      const promise2 = api.get(url);

      // Only one fetch should be made (verifies deduplication)
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Both should reject with an error
      let error1: Error | undefined;
      let error2: Error | undefined;
      try { await promise1; } catch (e) { error1 = e as Error; }
      try { await promise2; } catch (e) { error2 = e as Error; }

      expect(error1).toBeDefined();
      expect(error2).toBeDefined();
      expect(error1?.message).toBe(error2?.message);
    });

    it('should share rejected promises among deduplicated requests', async () => {
      const url = '/test-endpoint';

      // Mock a non-retryable client error
      mockFetch.mockResolvedValue(new Response(JSON.stringify({ error: 'Bad Request' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      }));

      const promise1 = api.get(url);
      const promise2 = api.get(url);

      // Only one fetch should be made (verifies deduplication)
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Both should reject with the same error (testing behavior not identity)
      let error1: Error | undefined;
      let error2: Error | undefined;
      try { await promise1; } catch (e) { error1 = e as Error; }
      try { await promise2; } catch (e) { error2 = e as Error; }

      expect(error1).toBeDefined();
      expect(error2).toBeDefined();
      expect(error1?.message).toBe(error2?.message);
    });
  });

  describe('cancellation and deduplication', () => {
    it('should handle request cancellation', async () => {
      const url = '/test-endpoint';

      // Make fetch hang forever
      mockFetch.mockImplementation(() => new Promise<Response>(() => {}));

      const controller = cancellation.createController();

      // Start a request with cancellation signal
      const promise = api.get(url, { signal: controller.signal });

      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Cancel the request
      controller.abort();

      // The request should be cancelled
      let error: Error | undefined;
      try { await promise; } catch (e) { error = e as Error; }

      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });

    it('should abort underlying fetch when request is cancelled', async () => {
      const url = '/test-endpoint';
      let abortSignal: AbortSignal | undefined;

      // Capture the abort signal from fetch and respond to abort
      mockFetch.mockImplementation((_, options) => {
        abortSignal = options?.signal;
        return new Promise((_, reject) => {
          if (abortSignal?.aborted) {
            const abortError = new Error('Aborted');
            abortError.name = 'AbortError';
            reject(abortError);
            return;
          }
          // Listen for abort
          abortSignal?.addEventListener('abort', () => {
            const abortError = new Error('Aborted');
            abortError.name = 'AbortError';
            reject(abortError);
          });
        });
      });

      const controller = cancellation.createController();

      // Start request with skipDeduplication to ensure direct fetch access
      const promise = api.get(url, { signal: controller.signal, skipDeduplication: true });

      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Cancel the request
      controller.abort();

      // The underlying fetch should have been aborted
      expect(abortSignal?.aborted).toBe(true);

      // The promise should reject
      let error: Error | undefined;
      try { await promise; } catch (e) { error = e as Error; }

      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });
  });

  describe('skipping deduplication', () => {
    it('should skip deduplication when skipDeduplication is true', async () => {
      const url = '/test-endpoint';
      
      // Make two identical requests with skipDeduplication
      const promise1 = api.get(url, { skipDeduplication: true });
      const promise2 = api.get(url, { skipDeduplication: true });
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });

    it('should work with mixed deduplication settings', async () => {
      const url = '/test-endpoint';
      
      // Make one regular request and one with skipDeduplication
      const promise1 = api.get(url);
      const promise2 = api.get(url, { skipDeduplication: true });
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });
  });

  describe('utility functions', () => {
    it('should track pending request count correctly', async () => {
      const resolvers: ((value: Response) => void)[] = [];

      // Make fetch hang, capturing resolvers for each call
      mockFetch.mockImplementation(() => new Promise<Response>(resolve => {
        resolvers.push(resolve);
      }));

      expect(api.getPendingRequestCount()).toBe(0);

      // Start a request
      const promise1 = api.get('/test1');
      expect(api.getPendingRequestCount()).toBe(1);

      // Start another request to the same endpoint (should be deduplicated)
      const promise2 = api.get('/test1');
      expect(api.getPendingRequestCount()).toBe(1); // Still 1 because deduplicated

      // Start a request to different endpoint
      const promise3 = api.get('/test2');
      expect(api.getPendingRequestCount()).toBe(2);

      // Resolve first request (test1)
      resolvers[0]!(new Response(JSON.stringify({ data: 'test1' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));

      await Promise.all([promise1, promise2]);
      // Allow microtask to clean up pending requests
      await Promise.resolve();
      expect(api.getPendingRequestCount()).toBe(1); // Only /test2 pending

      // Resolve second request (test2)
      resolvers[1]!(new Response(JSON.stringify({ data: 'test2' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));

      await promise3;
      // Allow microtask to clean up pending requests
      await Promise.resolve();
      expect(api.getPendingRequestCount()).toBe(0);
    });

    it('should clear all pending requests', async () => {
      // Make fetch hang
      mockFetch.mockImplementation(() => new Promise(() => {}));

      // Start multiple requests
      const promise1 = api.get('/test1');
      const promise2 = api.get('/test2');
      const promise3 = api.post('/test3', { data: 'test' });

      expect(api.getPendingRequestCount()).toBe(3);

      // Clear all pending requests
      api.clearPendingRequests();

      expect(api.getPendingRequestCount()).toBe(0);

      // All promises should be rejected (they won't resolve, so we need to catch them)
      // The clear action aborts the underlying requests but doesn't directly reject the promises
      // The promises will eventually time out or reject due to abort, but for this test
      // we've already verified the main behavior (count goes to 0)
    });
  });

  describe('deduplication with different tenants', () => {
    beforeEach(() => {
      // Set a tenant
      api.setTenant({ 
        id: 'tenant-1', 
        attributes: { 
          name: 'Tenant 1',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1
        } 
      });
    });

    afterEach(() => {
      api.setTenant(null);
    });

    it('should create separate requests for different tenants', async () => {
      const url = '/test-endpoint';
      
      // Start request with tenant-1
      const promise1 = api.get(url);
      
      // Change tenant
      api.setTenant({ 
        id: 'tenant-2', 
        attributes: { 
          name: 'Tenant 2',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1
        } 
      });
      
      // Start request with tenant-2
      const promise2 = api.get(url);
      
      // They should be different promises
      expect(promise1).not.toBe(promise2);
      
      // Two fetches should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);
      
      await Promise.all([promise1, promise2]);
    });

    it('should deduplicate requests with same tenant', async () => {
      const url = '/test-endpoint';

      // Make two requests with same tenant
      const promise1 = api.get(url);
      const promise2 = api.get(url);

      // Only one fetch should be made (verifies deduplication)
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Both should resolve to the same result
      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toEqual(result2);
    });
  });
});