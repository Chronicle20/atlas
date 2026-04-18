/**
 * Simplified test for API client retry functionality
 */

import { isRetryableError } from '../errors';
import { createApiErrorFromResponse } from '@/types/api/errors';

describe('Retry Logic', () => {
  describe('isRetryableError function', () => {
    it('should identify retryable server errors', () => {
      const error503 = createApiErrorFromResponse(503, 'Service Unavailable');
      const error429 = createApiErrorFromResponse(429, 'Too Many Requests');
      const error500 = createApiErrorFromResponse(500, 'Internal Server Error');
      const error502 = createApiErrorFromResponse(502, 'Bad Gateway');
      const error504 = createApiErrorFromResponse(504, 'Gateway Timeout');

      expect(isRetryableError(error503)).toBe(true);
      expect(isRetryableError(error429)).toBe(true);
      expect(isRetryableError(error500)).toBe(true);
      expect(isRetryableError(error502)).toBe(true);
      expect(isRetryableError(error504)).toBe(true);
    });

    it('should not retry client errors', () => {
      const error404 = createApiErrorFromResponse(404, 'Not Found');
      const error400 = createApiErrorFromResponse(400, 'Bad Request');
      const error401 = createApiErrorFromResponse(401, 'Unauthorized');
      const error403 = createApiErrorFromResponse(403, 'Forbidden');

      expect(isRetryableError(error404)).toBe(false);
      expect(isRetryableError(error400)).toBe(false);
      expect(isRetryableError(error401)).toBe(false);
      expect(isRetryableError(error403)).toBe(false);
    });

    it('should handle network errors as retryable', () => {
      const networkError = createApiErrorFromResponse(0, 'Network Error');
      expect(isRetryableError(networkError)).toBe(true);
    });

    it('should not retry non-error objects', () => {
      expect(isRetryableError('string error')).toBe(false);
      expect(isRetryableError(null)).toBe(false);
      expect(isRetryableError(undefined)).toBe(false);
      expect(isRetryableError({})).toBe(false);
    });
  });

  describe('createApiErrorFromResponse', () => {
    it('should preserve status codes for server errors', () => {
      const error503 = createApiErrorFromResponse(503, 'Service Unavailable');
      const error502 = createApiErrorFromResponse(502, 'Bad Gateway');
      const error504 = createApiErrorFromResponse(504, 'Gateway Timeout');

      expect(error503.statusCode).toBe(503);
      expect(error502.statusCode).toBe(502);
      expect(error504.statusCode).toBe(504);
    });

    it('should set correct error codes', () => {
      const error503 = createApiErrorFromResponse(503, 'Service Unavailable');
      const error404 = createApiErrorFromResponse(404, 'Not Found');
      const error401 = createApiErrorFromResponse(401, 'Unauthorized');

      expect(error503.code).toBe('SERVER_ERROR');
      expect(error404.code).toBe('NOT_FOUND');
      expect(error401.code).toBe('AUTHENTICATION_ERROR');
    });
  });
});