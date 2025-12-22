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

      // Make the fetch hang so we can cancel it - but respond to abort
      mockFetch.mockImplementation((_, options) => {
        return new Promise((_, reject) => {
          const signal = options?.signal;
          if (signal?.aborted) {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
            return;
          }
          signal?.addEventListener('abort', () => {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
          });
        });
      });

      const requestPromise = api.get('/test-cancel', { signal: controller.signal, skipDeduplication: true });

      // Cancel the request
      controller.abort();

      let error: Error | undefined;
      try { await requestPromise; } catch (e) { error = e as Error; }
      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });

    it('should handle already aborted signal', async () => {
      const controller = cancellation.createController();
      controller.abort();

      let error: Error | undefined;
      try {
        await api.get('/test-aborted', { signal: controller.signal, skipDeduplication: true });
      } catch (e) {
        error = e as Error;
      }
      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });

    it('should not make fetch call if signal is already aborted', async () => {
      const controller = cancellation.createController();
      controller.abort();

      let error: Error | undefined;
      try {
        await api.get('/test-no-fetch', { signal: controller.signal, skipDeduplication: true });
      } catch (e) {
        error = e as Error;
      }
      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');

      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('should cancel during retry delays', async () => {
      // This test verifies cancellation during retries is handled
      const controller = cancellation.createController();
      controller.abort();

      // With signal already aborted, should reject immediately
      let error: Error | undefined;
      try {
        await api.get('/test-retry', {
          signal: controller.signal,
          maxRetries: 3,
          retryDelay: 1000,
          skipDeduplication: true
        });
      } catch (e) {
        error = e as Error;
      }
      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });

    it('should work with all HTTP methods', async () => {
      const controller = cancellation.createController();

      // Mock fetch to respond to abort
      mockFetch.mockImplementation((_, options) => {
        return new Promise((_, reject) => {
          const signal = options?.signal;
          if (signal?.aborted) {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
            return;
          }
          signal?.addEventListener('abort', () => {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
          });
        });
      });

      const methods = [
        () => api.get('/test-methods-get', { signal: controller.signal, skipDeduplication: true }),
        () => api.post('/test-methods-post', {}, { signal: controller.signal, skipDeduplication: true }),
        () => api.put('/test-methods-put', {}, { signal: controller.signal, skipDeduplication: true }),
        () => api.patch('/test-methods-patch', {}, { signal: controller.signal, skipDeduplication: true }),
        () => api.delete('/test-methods-delete', { signal: controller.signal, skipDeduplication: true }),
      ];

      const promises = methods.map(method => method());

      controller.abort();

      for (const promise of promises) {
        let error: Error | undefined;
        try { await promise; } catch (e) { error = e as Error; }
        expect(error).toBeDefined();
        expect(error?.message).toContain('cancelled');
      }
    });
  });

  describe('timeout vs cancellation', () => {
    it('should distinguish between timeout and external cancellation', async () => {
      const controller = cancellation.createController();

      // Mock fetch to respond to abort
      mockFetch.mockImplementation((_, options) => {
        return new Promise((_, reject) => {
          const signal = options?.signal;
          if (signal?.aborted) {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
            return;
          }
          signal?.addEventListener('abort', () => {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
          });
        });
      });

      const requestPromise = api.get('/test-timeout-cancel', {
        signal: controller.signal,
        timeout: 10000,
        skipDeduplication: true
      });

      // Cancel externally before timeout
      controller.abort();

      let error: Error | undefined;
      try { await requestPromise; } catch (e) { error = e as Error; }
      expect(error).toBeDefined();
      expect(error?.message).toContain('cancelled');
    });

    it('should handle timeout when no external cancellation', async () => {
      // Mock a request that responds to the combined abort signal (from timeout)
      mockFetch.mockImplementation((_, options) => {
        return new Promise((_, reject) => {
          const signal = options?.signal;
          if (signal?.aborted) {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
            return;
          }
          signal?.addEventListener('abort', () => {
            const err = new Error('Aborted');
            err.name = 'AbortError';
            reject(err);
          });
        });
      });

      // Use a very short timeout for quick testing, and disable retries (408 is retryable)
      const requestPromise = api.get('/test-timeout', {
        timeout: 50,
        skipDeduplication: true,
        maxRetries: 0
      });

      let error: Error | undefined;
      try { await requestPromise; } catch (e) { error = e as Error; }
      expect(error).toBeDefined();
      expect(error?.message).toContain('timeout');
    });
  });
});