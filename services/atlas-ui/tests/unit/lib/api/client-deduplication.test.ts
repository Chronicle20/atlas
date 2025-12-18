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
      expect(mockFetch).toHaveBeenCalledTimes(1); // Still only one call
      
      // They should be the same promise
      expect(promise1).toBe(promise2);
      
      // Resolve the request
      resolveRequest!(new Response(JSON.stringify({ data: 'test' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toEqual(result2);
    });
  });

  describe('error handling in deduplication', () => {
    it('should share errors among deduplicated requests', async () => {
      const url = '/test-endpoint';
      
      // Mock a server error
      mockFetch.mockResolvedValue(new Response('Server Error', { status: 500 }));
      
      const promise1 = api.get(url);
      const promise2 = api.get(url);
      
      // They should be the same promise
      expect(promise1).toBe(promise2);
      
      // Both should reject with the same error
      await expect(promise1).rejects.toThrow();
      await expect(promise2).rejects.toThrow();
      
      // Only one fetch should be made
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('should share network errors among deduplicated requests', async () => {
      const url = '/test-endpoint';
      
      // Mock a network error
      mockFetch.mockRejectedValue(new TypeError('Network error'));
      
      const promise1 = api.get(url);
      const promise2 = api.get(url);
      
      // They should be the same promise
      expect(promise1).toBe(promise2);
      
      // Both should reject with the same error
      await expect(promise1).rejects.toThrow('Network error');
      await expect(promise2).rejects.toThrow('Network error');
      
      // Only one fetch should be made
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });

  describe('cancellation and deduplication', () => {
    it('should handle individual request cancellation in deduplicated group', async () => {
      const url = '/test-endpoint';
      let resolveRequest: (value: Response) => void;
      
      // Make fetch hang
      mockFetch.mockImplementation(() => new Promise<Response>(resolve => {
        resolveRequest = resolve;
      }));
      
      const controller1 = cancellation.createController();
      const controller2 = cancellation.createController();
      
      // Start two requests with different cancellation signals
      const promise1 = api.get(url, { signal: controller1.signal });
      const promise2 = api.get(url, { signal: controller2.signal });
      
      // They should be the same promise
      expect(promise1).toBe(promise2);
      
      // Cancel one request
      controller1.abort();
      
      // The first should be cancelled but the second should continue
      await expect(promise1).rejects.toThrow('Request was cancelled');
      
      // Resolve the underlying request
      resolveRequest!(new Response(JSON.stringify({ data: 'test' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      // The second should still resolve
      const result2 = await promise2;
      expect(result2).toEqual({ data: 'test' });
    });

    it('should cancel all requests when request count reaches zero', async () => {
      const url = '/test-endpoint';
      let abortSignal: AbortSignal | undefined;
      
      // Capture the abort signal from fetch
      mockFetch.mockImplementation((_, options) => {
        abortSignal = options?.signal;
        return new Promise(() => {}); // Hang forever
      });
      
      const controller1 = cancellation.createController();
      const controller2 = cancellation.createController();
      
      // Start two requests
      const promise1 = api.get(url, { signal: controller1.signal });
      const promise2 = api.get(url, { signal: controller2.signal });
      
      expect(promise1).toBe(promise2);
      
      // Cancel both requests
      controller1.abort();
      controller2.abort();
      
      // The underlying fetch should have been aborted
      expect(abortSignal?.aborted).toBe(true);
      
      await expect(promise1).rejects.toThrow('Request was cancelled');
      await expect(promise2).rejects.toThrow('Request was cancelled');
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
      let resolveRequest: (value: Response) => void;
      
      // Make fetch hang
      mockFetch.mockImplementation(() => new Promise<Response>(resolve => {
        resolveRequest = resolve;
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
      
      // Resolve first request
      resolveRequest!(new Response(JSON.stringify({ data: 'test1' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      await Promise.all([promise1, promise2]);
      expect(api.getPendingRequestCount()).toBe(1); // Only /test2 pending
      
      // Resolve second request
      resolveRequest!(new Response(JSON.stringify({ data: 'test2' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      await promise3;
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
      
      // All promises should be rejected
      await expect(promise1).rejects.toThrow();
      await expect(promise2).rejects.toThrow();
      await expect(promise3).rejects.toThrow();
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
      
      // They should be the same promise
      expect(promise1).toBe(promise2);
      
      // Only one fetch should be made
      expect(mockFetch).toHaveBeenCalledTimes(1);
      
      await Promise.all([promise1, promise2]);
    });
  });
});