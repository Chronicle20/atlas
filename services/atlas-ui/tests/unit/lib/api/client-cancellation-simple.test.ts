/**
 * Simple test to verify AbortController support is working
 */

import { api, cancellation } from '@/lib/api/client';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch as jest.Mock;

describe('API Client Request Cancellation - Simple Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('cancellation utilities', () => {
    it('should create a new AbortController', () => {
      const controller = cancellation.createController();
      expect(controller).toBeInstanceOf(AbortController);
      expect(controller.signal.aborted).toBe(false);
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

  describe('signal passing', () => {
    it('should pass signal to API requests', async () => {
      const controller = cancellation.createController();
      
      // Mock successful response
      mockFetch.mockResolvedValue(new Response(JSON.stringify({ data: 'test' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      await api.get('/test', { signal: controller.signal });
      
      // Verify fetch was called with the signal
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/test'),
        expect.objectContaining({
          signal: controller.signal,
        })
      );
    });

    it('should accept signal option for all HTTP methods', async () => {
      const controller = cancellation.createController();
      
      // Mock successful response
      mockFetch.mockResolvedValue(new Response(JSON.stringify({ data: 'test' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
      
      // Test all HTTP methods accept signal option
      await api.get('/test', { signal: controller.signal });
      await api.post('/test', {}, { signal: controller.signal });
      await api.put('/test', {}, { signal: controller.signal });
      await api.patch('/test', {}, { signal: controller.signal });
      await api.delete('/test', { signal: controller.signal });
      
      // Verify all calls were made
      expect(mockFetch).toHaveBeenCalledTimes(5);
      
      // Verify all calls included the signal
      mockFetch.mock.calls.forEach(call => {
        expect(call[1]).toEqual(expect.objectContaining({
          signal: controller.signal,
        }));
      });
    });
  });
});