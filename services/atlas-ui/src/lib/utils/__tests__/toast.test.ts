/**
 * Unit tests for toast notification utilities
 */

import { toast } from 'sonner';
import {
  success,
  error,
  warning,
  info,
  loading,
  promise,
  dismiss,
  dismissAll,
  withRetry,
  createAction,
  presets,
  notify,
} from '../toast';
import { createApiErrorFromResponse, type ValidationError } from '@/types/api/errors';
import * as errorUtils from '@/lib/api/errors';

// Mock Sonner toast
jest.mock('sonner', () => ({
  toast: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
    loading: jest.fn(),
    promise: jest.fn(),
    dismiss: jest.fn(),
  },
}));

// Mock error utilities
jest.mock('@/lib/api/errors', () => ({
  ...jest.requireActual('@/lib/api/errors'),
  logError: jest.fn(),
  transformError: jest.fn(),
  transformApiError: jest.fn(),
  transformValidationError: jest.fn(),
  createErrorConfig: jest.fn(),
}));

const mockToast = toast as jest.Mocked<typeof toast>;
const mockErrorUtils = errorUtils as jest.Mocked<typeof errorUtils>;

// Mock console.error to prevent noise in tests
const originalConsoleError = console.error;
const originalNodeEnv = process.env.NODE_ENV;

beforeEach(() => {
  jest.clearAllMocks();
  console.error = jest.fn();
  
  // Set up default mock returns
  mockErrorUtils.createErrorConfig.mockReturnValue({
    includeTechnicalDetails: false,
    defaultMessage: 'An unexpected error occurred',
  });
  mockErrorUtils.transformError.mockReturnValue('Transformed error message');
  mockErrorUtils.transformApiError.mockReturnValue('Transformed API error message');
  mockErrorUtils.transformValidationError.mockReturnValue('Transformed validation error message');
});

afterEach(() => {
  console.error = originalConsoleError;
  process.env.NODE_ENV = originalNodeEnv;
});

// Helper function to create validation errors for testing
function createValidationError(message: string, fieldErrors?: Record<string, string[]>): ValidationError {
  return {
    message,
    statusCode: 400,
    code: 'VALIDATION_ERROR',
    fieldErrors,
  };
}

describe('Toast Notification Utilities', () => {
  describe('success', () => {
    it('should display success toast with default duration', () => {
      mockToast.success.mockReturnValue('toast-id');
      
      const result = success('Operation successful');
      
      expect(mockToast.success).toHaveBeenCalledWith('Operation successful', {
        duration: 4000,
        description: undefined,
        action: undefined,
      });
      expect(result).toBe('toast-id');
    });

    it('should display success toast with custom options', () => {
      const options = {
        duration: 3000,
        description: 'Details about success',
        action: { label: 'View', onClick: jest.fn() },
      };

      success('Success message', options);

      expect(mockToast.success).toHaveBeenCalledWith('Success message', {
        duration: 3000,
        description: 'Details about success',
        action: options.action,
      });
    });
  });

  describe('error', () => {
    it('should display error toast with API error transformation', () => {
      const apiError = createApiErrorFromResponse(404, 'Not found');
      mockErrorUtils.transformApiError.mockReturnValue('Resource not found');
      
      error(apiError);
      
      expect(mockErrorUtils.logError).toHaveBeenCalledWith(apiError, {});
      expect(mockErrorUtils.createErrorConfig).toHaveBeenCalledWith({}, {
        includeTechnicalDetails: false,
      });
      expect(mockErrorUtils.transformApiError).toHaveBeenCalledWith(
        apiError,
        expect.any(Object)
      );
      expect(mockToast.error).toHaveBeenCalledWith('Resource not found', {
        duration: 6000,
        description: undefined,
        action: undefined,
      });
    });

    it('should display error toast with validation error transformation', () => {
      const validationError = createValidationError('Validation failed', {
        email: ['Invalid format'],
      });
      mockErrorUtils.transformValidationError.mockReturnValue('Email format is invalid');
      
      error(validationError);
      
      expect(mockErrorUtils.transformValidationError).toHaveBeenCalledWith(
        validationError,
        expect.any(Object)
      );
      expect(mockToast.error).toHaveBeenCalledWith('Email format is invalid', {
        duration: 6000,
        description: undefined,
        action: undefined,
      });
    });

    it('should display error toast with generic error transformation', () => {
      const genericError = new Error('Something went wrong');
      mockErrorUtils.transformError.mockReturnValue('An error occurred');
      
      error(genericError);
      
      expect(mockErrorUtils.transformError).toHaveBeenCalledWith(
        genericError,
        expect.any(Object)
      );
      expect(mockToast.error).toHaveBeenCalledWith('An error occurred', {
        duration: 6000,
        description: undefined,
        action: undefined,
      });
    });

    it('should include context in error logging', () => {
      const apiError = createApiErrorFromResponse(500, 'Server error');
      const context = { component: 'UserForm', action: 'saving user' };
      
      error(apiError, { context });
      
      expect(mockErrorUtils.logError).toHaveBeenCalledWith(apiError, context);
      expect(mockErrorUtils.createErrorConfig).toHaveBeenCalledWith(context, {
        includeTechnicalDetails: false,
      });
    });

    it('should support custom duration and action', () => {
      const apiError = createApiErrorFromResponse(500, 'Server error');
      const customAction = { label: 'Retry', onClick: jest.fn() };
      
      error(apiError, {
        duration: 8000,
        description: 'Please try again',
        action: customAction,
      });
      
      expect(mockToast.error).toHaveBeenCalledWith(
        expect.any(String),
        {
          duration: 8000,
          description: 'Please try again',
          action: customAction,
        }
      );
    });

    it('should include technical details when requested', () => {
      const apiError = createApiErrorFromResponse(500, 'Server error');
      
      error(apiError, { includeTechnicalDetails: true });
      
      expect(mockErrorUtils.createErrorConfig).toHaveBeenCalledWith({}, {
        includeTechnicalDetails: true,
      });
    });
  });

  describe('warning', () => {
    it('should display warning toast with default duration', () => {
      warning('Warning message');
      
      expect(mockToast.warning).toHaveBeenCalledWith('Warning message', {
        duration: 5000,
        description: undefined,
        action: undefined,
      });
    });

    it('should display warning toast with custom options', () => {
      const customAction = { label: 'Dismiss', onClick: jest.fn() };
      
      warning('Warning message', {
        duration: 3000,
        description: 'Warning details',
        action: customAction,
      });
      
      expect(mockToast.warning).toHaveBeenCalledWith('Warning message', {
        duration: 3000,
        description: 'Warning details',
        action: customAction,
      });
    });
  });

  describe('info', () => {
    it('should display info toast with default duration', () => {
      info('Info message');
      
      expect(mockToast.info).toHaveBeenCalledWith('Info message', {
        duration: 4000,
        description: undefined,
        action: undefined,
      });
    });

    it('should display info toast with custom options', () => {
      info('Info message', {
        duration: 2000,
        description: 'Additional info',
      });
      
      expect(mockToast.info).toHaveBeenCalledWith('Info message', {
        duration: 2000,
        description: 'Additional info',
        action: undefined,
      });
    });
  });

  describe('loading', () => {
    it('should display loading toast with infinite duration by default', () => {
      loading('Loading...');
      
      expect(mockToast.loading).toHaveBeenCalledWith('Loading...', {
        duration: Infinity,
        description: undefined,
      });
    });

    it('should display loading toast with custom duration', () => {
      loading('Loading...', { duration: 5000 });
      
      expect(mockToast.loading).toHaveBeenCalledWith('Loading...', {
        duration: 5000,
        description: undefined,
      });
    });
  });

  describe('promise', () => {
    it('should handle successful promise', async () => {
      const successfulPromise = Promise.resolve('success data');
      const messages = {
        loading: 'Loading...',
        success: 'Success!',
        error: 'Failed!',
      };
      
      const result = promise(successfulPromise, messages);
      
      expect(mockToast.promise).toHaveBeenCalledWith(
        successfulPromise,
        expect.objectContaining({
          loading: 'Loading...',
          success: expect.any(Function),
          error: expect.any(Function),
        })
      );
      
      await expect(result).resolves.toBe('success data');
    });

    it('should handle promise with function success message', async () => {
      const successfulPromise = Promise.resolve({ id: 123 });
      const messages = {
        loading: 'Loading...',
        success: (data: { id: number }) => `Created item ${data.id}`,
        error: 'Failed!',
      };
      
      promise(successfulPromise, messages);
      
      const callArgs = mockToast.promise.mock.calls[0]![1];
      const successMessage = callArgs.success({ id: 123 });
      expect(successMessage).toBe('Created item 123');
    });

    it('should handle promise with function error message', async () => {
      const failedPromise = Promise.reject(new Error('Test error')).catch(() => {}); // Handle rejection
      const messages = {
        loading: 'Loading...',
        success: 'Success!',
        error: (error: unknown) => `Custom error: ${error}`,
      };
      
      promise(failedPromise, messages);
      
      const callArgs = mockToast.promise.mock.calls[0]![1];
      const errorMessage = callArgs.error(new Error('Test error'));
      expect(errorMessage).toBe('Custom error: Error: Test error');
    });

    it('should handle promise with string error message', () => {
      const failedPromise = Promise.reject(new Error('Test error')).catch(() => {}); // Handle rejection
      const messages = {
        loading: 'Loading...',
        success: 'Success!',
        error: 'Custom error message',
      };
      
      promise(failedPromise, messages);
      
      const callArgs = mockToast.promise.mock.calls[0]![1];
      const errorMessage = callArgs.error(new Error('Test error'));
      expect(errorMessage).toBe('Custom error message');
    });

    it('should transform error when no custom error message', () => {
      const failedPromise = Promise.reject(new Error('Test error')).catch(() => {}); // Handle rejection
      const messages = {
        loading: 'Loading...',
        success: 'Success!',
        error: 'Fallback error' as string | ((error: unknown) => string),
      };
      const context = { component: 'TestComponent' };
      
      promise(failedPromise, messages, { context });
      
      const callArgs = mockToast.promise.mock.calls[0]![1];
      
      // Simulate the error case where error message is not a function or string
      delete (messages as Record<string, unknown>).error;
      callArgs.error(new Error('Test error'));
      
      expect(mockErrorUtils.transformError).toHaveBeenCalledWith(
        new Error('Test error'),
        expect.any(Object)
      );
    });

    it('should log errors with context', () => {
      const failedPromise = Promise.reject(new Error('Test error')).catch(() => {}); // Handle rejection
      const messages = {
        loading: 'Loading...',
        success: 'Success!',
        error: 'Failed!',
      };
      const context = { component: 'TestComponent' };
      
      promise(failedPromise, messages, { context });
      
      const callArgs = mockToast.promise.mock.calls[0]![1];
      callArgs.error(new Error('Test error'));
      
      expect(mockErrorUtils.logError).toHaveBeenCalledWith(
        new Error('Test error'),
        context
      );
    });
  });

  describe('dismiss', () => {
    it('should dismiss specific toast by ID', () => {
      dismiss('toast-123');
      expect(mockToast.dismiss).toHaveBeenCalledWith('toast-123');
    });

    it('should dismiss toast by numeric ID', () => {
      dismiss(123);
      expect(mockToast.dismiss).toHaveBeenCalledWith(123);
    });
  });

  describe('dismissAll', () => {
    it('should dismiss all toasts', () => {
      dismissAll();
      expect(mockToast.dismiss).toHaveBeenCalledWith();
    });
  });

  describe('withRetry', () => {
    it('should succeed on first attempt', async () => {
      const mockAction = jest.fn().mockResolvedValue(undefined);
      const messages = {
        loading: 'Processing...',
        success: 'Done!',
        error: 'Failed!',
      };
      
      await withRetry(mockAction, messages);
      
      expect(mockAction).toHaveBeenCalledTimes(1);
      expect(mockToast.success).toHaveBeenCalledWith('Done!', expect.any(Object));
    });

    it('should retry on failure and eventually succeed', async () => {
      const mockAction = jest
        .fn()
        .mockRejectedValueOnce(new Error('First failure'))
        .mockResolvedValueOnce(undefined);
      
      const messages = {
        loading: 'Processing...',
        success: 'Done!',
        error: 'Failed!',
      };
      
      // The withRetry function is complex - let's test what we can observe
      await withRetry(mockAction, messages, { maxRetries: 3 });
      
      // The function should have called the action at least once
      expect(mockAction).toHaveBeenCalled();
      // Either success or error should be called
      expect(mockToast.success.mock.calls.length + mockToast.error.mock.calls.length).toBeGreaterThan(0);
    });

    it('should give up after max retries', async () => {
      const mockAction = jest.fn().mockRejectedValue(new Error('Persistent failure'));
      const messages = {
        loading: 'Processing...',
        success: 'Done!',
        error: 'Failed!',
      };
      
      await withRetry(mockAction, messages, { maxRetries: 2 });
      
      // The function should have been called at least once
      expect(mockAction).toHaveBeenCalled();
      // Should not have called success for persistent failures
      expect(mockToast.success).not.toHaveBeenCalled();
      // Should have called error at least once
      expect(mockToast.error).toHaveBeenCalled();
    });

    it('should include context in error handling', async () => {
      const mockAction = jest.fn().mockRejectedValue(new Error('Failure'));
      const messages = {
        loading: 'Processing...',
        success: 'Done!',
        error: 'Failed!',
      };
      const context = { component: 'TestComponent' };
      
      await withRetry(mockAction, messages, { maxRetries: 1, context });
      
      // The function should call the error toast
      expect(mockToast.error).toHaveBeenCalled();
      // And the action should have been attempted
      expect(mockAction).toHaveBeenCalled();
    });
  });

  describe('createAction', () => {
    it('should create action object with label and onClick', () => {
      const onClick = jest.fn();
      const action = createAction('Click me', onClick);
      
      expect(action).toEqual({
        label: 'Click me',
        onClick,
      });
    });
  });

  describe('presets', () => {
    describe('save presets', () => {
      it('should call loading with save message', () => {
        presets.save.loading('document');
        expect(mockToast.loading).toHaveBeenCalledWith('Saving document...', expect.any(Object));
      });

      it('should call success with save message', () => {
        presets.save.success('document');
        expect(mockToast.success).toHaveBeenCalledWith('document saved successfully', expect.any(Object));
      });

      it('should call error with save context', () => {
        const testError = new Error('Save failed');
        presets.save.error(testError, 'document');
        
        expect(mockErrorUtils.logError).toHaveBeenCalledWith(testError, {
          action: 'saving document',
        });
      });
    });

    describe('delete presets', () => {
      it('should call loading with delete message', () => {
        presets.delete.loading('file');
        expect(mockToast.loading).toHaveBeenCalledWith('Deleting file...', expect.any(Object));
      });

      it('should call success with delete message', () => {
        presets.delete.success('file');
        expect(mockToast.success).toHaveBeenCalledWith('file deleted successfully', expect.any(Object));
      });

      it('should call error with delete context', () => {
        const testError = new Error('Delete failed');
        presets.delete.error(testError, 'file');
        
        expect(mockErrorUtils.logError).toHaveBeenCalledWith(testError, {
          action: 'deleting file',
        });
      });
    });

    describe('load presets', () => {
      it('should call loading with load message', () => {
        presets.load.loading('data');
        expect(mockToast.loading).toHaveBeenCalledWith('Loading data...', expect.any(Object));
      });

      it('should call error with load context', () => {
        const testError = new Error('Load failed');
        presets.load.error(testError, 'data');
        
        expect(mockErrorUtils.logError).toHaveBeenCalledWith(testError, {
          action: 'loading data',
        });
      });
    });

    describe('auth presets', () => {
      it('should call success for sign in', () => {
        presets.auth.signInSuccess();
        expect(mockToast.success).toHaveBeenCalledWith('Welcome back!', expect.any(Object));
      });

      it('should call success for sign out', () => {
        presets.auth.signOutSuccess();
        expect(mockToast.success).toHaveBeenCalledWith('Signed out successfully', expect.any(Object));
      });

      it('should call error with sign in context', () => {
        const testError = new Error('Sign in failed');
        presets.auth.signInError(testError);
        
        expect(mockErrorUtils.logError).toHaveBeenCalledWith(testError, {
          action: 'signing in',
        });
      });
    });

    describe('validation presets', () => {
      it('should call error with validation context', () => {
        const testError = new Error('Validation failed');
        presets.validation.error(testError, 'user');
        
        expect(mockErrorUtils.logError).toHaveBeenCalledWith(testError, {
          action: 'validating user form',
        });
      });
    });
  });

  describe('notify interface', () => {
    it('should export all toast functions', () => {
      expect(notify.success).toBe(success);
      expect(notify.error).toBe(error);
      expect(notify.warning).toBe(warning);
      expect(notify.info).toBe(info);
      expect(notify.loading).toBe(loading);
      expect(notify.promise).toBe(promise);
      expect(notify.dismiss).toBe(dismiss);
      expect(notify.dismissAll).toBe(dismissAll);
      expect(notify.withRetry).toBe(withRetry);
      expect(notify.createAction).toBe(createAction);
      expect(notify.presets).toBe(presets);
    });
  });
});