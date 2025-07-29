/**
 * Unit tests for API client request cancellation functionality
 */

import { api, cancellation } from '@/lib/api/client';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch as jest.Mock;

describe('API Client Request Cancellation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Set up default successful response
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ data: 'test' }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));
  });

  afterEach(() => {
    jest.clearAllTimers();
  });

  describe('cancellation utilities', () => {
    it('should create a new AbortController', () => {
      const controller = cancellation.createController();
      expect(controller).toBeInstanceOf(AbortController);
      expect(controller.signal.aborted).toBe(false);
    });

    it('should create a timeout controller that aborts after specified time', async () => {
      jest.useFakeTimers();
      
      const controller = cancellation.createTimeoutController(1000);
      expect(controller.signal.aborted).toBe(false);
      
      // Fast-forward time
      jest.advanceTimersByTime(1000);
      
      // Allow promises to resolve
      await jest.runOnlyPendingTimersAsync();
      
      expect(controller.signal.aborted).toBe(true);
      
      jest.useRealTimers();
    });

    it('should combine multiple signals correctly', () => {
      const controller1 = cancellation.createController();
      const controller2 = cancellation.createController();
      
      const combined = cancellation.combineSignals(controller1.signal, controller2.signal);
      expect(combined.signal.aborted).toBe(false);
      
      // Abort the first controller
      controller1.abort();
      expect(combined.signal.aborted).toBe(true);
    });

    it('should handle already aborted signals in combination', () => {
      const controller1 = cancellation.createController();
      const controller2 = cancellation.createController();
      
      controller1.abort();
      
      const combined = cancellation.combineSignals(controller1.signal, controller2.signal);
      expect(combined.signal.aborted).toBe(true);
    });

    it('should identify cancellation errors correctly', () => {
      const abortError = new Error('Request aborted');
      abortError.name = 'AbortError';
      
      const cancellationError = new Error('Request was cancelled');
      const networkError = new Error('Network error');
      
      expect(cancellation.isCancellationError(abortError)).toBe(true);
      expect(cancellation.isCancellationError(cancellationError)).toBe(true);
      expect(cancellation.isCancellationError(networkError)).toBe(false);
      expect(cancellation.isCancellationError('not an error')).toBe(false);
    });
  });

  describe('request cancellation', () => {
    it('should cancel a request using AbortController', async () => {
      const controller = cancellation.createController();
      
      // Make the fetch hang so we can cancel it
      mockFetch.mockImplementation(() => new Promise(() => {}));
      
      const requestPromise = api.get('/test', { signal: controller.signal });
      
      // Cancel the request
      controller.abort();
      
      await expect(requestPromise).rejects.toThrow('Request was cancelled');
    });

    it('should handle already aborted signal', async () => {
      const controller = cancellation.createController();
      controller.abort();
      
      await expect(api.get('/test', { signal: controller.signal }))
        .rejects.toThrow('Request was cancelled');
    });

    it('should not make fetch call if signal is already aborted', async () => {
      const controller = cancellation.createController();
      controller.abort();
      
      await expect(api.get('/test', { signal: controller.signal }))
        .rejects.toThrow('Request was cancelled');
      
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('should cancel during retry delays', async () => {
      jest.useFakeTimers();
      
      // Mock a server error that would trigger retries
      mockFetch.mockRejectedValue(new TypeError('Network error'));
      
      const controller = cancellation.createController();
      
      const requestPromise = api.get('/test', { 
        signal: controller.signal,
        maxRetries: 3,
        retryDelay: 1000 
      });
      
      // Let the first attempt fail
      await jest.runOnlyPendingTimersAsync();
      
      // Cancel during the retry delay
      controller.abort();
      
      // Advance timers to trigger the cancellation
      jest.advanceTimersByTime(100);
      await jest.runOnlyPendingTimersAsync();
      
      await expect(requestPromise).rejects.toThrow('Request was cancelled');
      
      jest.useRealTimers();
    });

    it('should work with all HTTP methods', async () => {
      const controller = cancellation.createController();
      
      mockFetch.mockImplementation(() => new Promise(() => {}));
      
      const methods = [
        () => api.get('/test', { signal: controller.signal }),
        () => api.post('/test', {}, { signal: controller.signal }),
        () => api.put('/test', {}, { signal: controller.signal }),
        () => api.patch('/test', {}, { signal: controller.signal }),
        () => api.delete('/test', { signal: controller.signal }),
      ];
      
      const promises = methods.map(method => method());
      
      controller.abort();
      
      for (const promise of promises) {
        await expect(promise).rejects.toThrow('Request was cancelled');
      }
    });
  });

  describe('timeout vs cancellation', () => {
    it('should distinguish between timeout and external cancellation', async () => {
      jest.useFakeTimers();
      
      // Mock a request that hangs
      mockFetch.mockImplementation(() => new Promise(() => {}));
      
      const controller = cancellation.createController();
      
      const requestPromise = api.get('/test', { 
        signal: controller.signal,
        timeout: 1000 
      });
      
      // Cancel externally before timeout
      controller.abort();
      
      await expect(requestPromise).rejects.toThrow('Request was cancelled');
      
      jest.useRealTimers();
    });

    it('should handle timeout when no external cancellation', async () => {
      jest.useFakeTimers();
      
      // Mock a request that hangs
      mockFetch.mockImplementation(() => new Promise(() => {}));
      
      const requestPromise = api.get('/test', { timeout: 1000 });
      
      // Let timeout occur
      jest.advanceTimersByTime(1000);
      await jest.runOnlyPendingTimersAsync();
      
      await expect(requestPromise).rejects.toThrow('Request timeout');
      
      jest.useRealTimers();
    });
  });
});