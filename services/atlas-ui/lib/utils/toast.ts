/**
 * Toast notification utilities for Atlas UI
 * 
 * This module provides a consistent interface for displaying toast notifications
 * throughout the application using Sonner. It integrates with the error handling
 * system to automatically transform API errors into user-friendly messages.
 */

import { toast } from 'sonner';
import { 
  transformError, 
  transformApiError, 
  transformValidationError,
  logError,
  type ErrorContext,
  createErrorConfig
} from '@/lib/api/errors';
import {
  isApiError,
  isValidationError
} from '@/types/api/errors';

/**
 * Configuration options for toast notifications
 */
export interface ToastOptions {
  /** Duration in milliseconds (default: 4000) */
  duration?: number;
  /** Custom description text */
  description?: string;
  /** Custom action button */
  action?: {
    label: string;
    onClick: () => void;
  };
  /** Whether to include technical details (dev mode only) */
  includeTechnicalDetails?: boolean;
  /** Additional context for error handling */
  context?: ErrorContext;
}

/**
 * Default toast durations by type
 */
const DEFAULT_DURATIONS = {
  success: 4000,
  error: 6000,
  warning: 5000,
  info: 4000,
  loading: Infinity, // Loading toasts persist until dismissed
} as const;

/**
 * Display a success toast notification
 */
export const success = (
  message: string, 
  options: Omit<ToastOptions, 'context'> = {}
): string | number => {
  const { duration = DEFAULT_DURATIONS.success, description, action } = options;

  return toast.success(message, {
    duration,
    description,
    action,
  });
};

/**
 * Display an error toast notification with automatic error transformation
 */
export const error = (
  error: unknown, 
  options: ToastOptions = {}
): string | number => {
  const { 
    duration = DEFAULT_DURATIONS.error, 
    description, 
    action,
    includeTechnicalDetails = false,
    context = {}
  } = options;

  // Log the error for debugging
  logError(error, context);

  // Transform the error into a user-friendly message
  const errorConfig = createErrorConfig(context, { 
    includeTechnicalDetails 
  });

  let message: string;
  
  if (isValidationError(error)) {
    message = transformValidationError(error, errorConfig);
  } else if (isApiError(error)) {
    message = transformApiError(error, errorConfig);
  } else {
    message = transformError(error, errorConfig);
  }

  return toast.error(message, {
    duration,
    description,
    action,
  });
};

/**
 * Display a warning toast notification
 */
export const warning = (
  message: string, 
  options: Omit<ToastOptions, 'context'> = {}
): string | number => {
  const { duration = DEFAULT_DURATIONS.warning, description, action } = options;

  return toast.warning(message, {
    duration,
    description,
    action,
  });
};

/**
 * Display an info toast notification
 */
export const info = (
  message: string, 
  options: Omit<ToastOptions, 'context'> = {}
): string | number => {
  const { duration = DEFAULT_DURATIONS.info, description, action } = options;

  return toast.info(message, {
    duration,
    description,
    action,
  });
};

/**
 * Display a loading toast notification
 */
export const loading = (
  message: string, 
  options: Omit<ToastOptions, 'context'> = {}
): string | number => {
  const { duration = DEFAULT_DURATIONS.loading, description } = options;

  return toast.loading(message, {
    duration,
    description,
  });
};

/**
 * Display a promise-based toast that updates based on promise state
 */
export const promise = <T>(
  promise: Promise<T>,
  messages: {
    loading: string;
    success: string | ((data: T) => string);
    error: string | ((error: unknown) => string);
  },
  options: ToastOptions = {}
): Promise<T> => {
  const { context = {} } = options;

  toast.promise(promise, {
    loading: messages.loading,
    success: (data) => {
      return typeof messages.success === 'function' 
        ? messages.success(data) 
        : messages.success;
    },
    error: (error) => {
      // Log the error for debugging
      logError(error, context);

      if (typeof messages.error === 'function') {
        return messages.error(error);
      }

      // Transform error to user-friendly message if it's a string
      if (typeof messages.error === 'string') {
        return messages.error;
      }

      // Fallback to error transformation
      const errorConfig = createErrorConfig(context);
      return transformError(error, errorConfig);
    },
  });

  // Return the original promise to maintain proper typing
  return promise;
};

/**
 * Dismiss a specific toast by ID
 */
export const dismiss = (id: string | number): void => {
  toast.dismiss(id);
};

/**
 * Dismiss all active toasts
 */
export const dismissAll = (): void => {
  toast.dismiss();
};

/**
 * Create a toast notification with automatic retry functionality
 */
export const withRetry = (
  action: () => Promise<void>,
  messages: {
    loading: string;
    success: string;
    error: string;
  },
  options: ToastOptions & { maxRetries?: number } = {}
): Promise<void> => {
  const { maxRetries = 3, context = {} } = options;
  let retryCount = 0;

  const attemptAction = async (): Promise<void> => {
    try {
      await action();
      success(messages.success);
    } catch (err) {
      retryCount++;
      
      if (retryCount < maxRetries) {
        error(err, {
          ...options,
          context,
          action: {
            label: `Retry (${retryCount}/${maxRetries})`,
            onClick: () => attemptAction(),
          },
        });
        return;
      } else {
        error(err, {
          ...options,
          context,
        });
        return;
      }
    }
  };

  return attemptAction();
};

/**
 * Utility function to create action objects for toast notifications
 */
export const createAction = (
  label: string, 
  onClick: () => void
): ToastOptions['action'] => ({
  label,
  onClick,
});

/**
 * Preset toast configurations for common scenarios
 */
export const presets = {
  /**
   * Save operation presets
   */
  save: {
    loading: (item: string) => loading(`Saving ${item}...`),
    success: (item: string) => success(`${item} saved successfully`),
    error: (error: unknown, item: string) => 
      notify.error(error, { 
        context: { action: `saving ${item}` } 
      }),
  },

  /**
   * Delete operation presets
   */
  delete: {
    loading: (item: string) => loading(`Deleting ${item}...`),
    success: (item: string) => success(`${item} deleted successfully`),
    error: (error: unknown, item: string) => 
      notify.error(error, { 
        context: { action: `deleting ${item}` } 
      }),
  },

  /**
   * Load operation presets
   */
  load: {
    loading: (item: string) => loading(`Loading ${item}...`),
    error: (error: unknown, item: string) => 
      notify.error(error, { 
        context: { action: `loading ${item}` } 
      }),
  },

  /**
   * Authentication presets
   */
  auth: {
    signInSuccess: () => success('Welcome back!'),
    signOutSuccess: () => success('Signed out successfully'),
    signInError: (error: unknown) => 
      notify.error(error, { 
        context: { action: 'signing in' } 
      }),
  },

  /**
   * Form validation presets
   */
  validation: {
    error: (error: unknown, form: string) => 
      notify.error(error, { 
        context: { action: `validating ${form} form` } 
      }),
  },
} as const;

/**
 * Main toast notification interface
 * This is the primary export that provides a clean API for toast notifications
 */
export const notify = {
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
} as const;

/**
 * Default export for convenience
 */
export default notify;