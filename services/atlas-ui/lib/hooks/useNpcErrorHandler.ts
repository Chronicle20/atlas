/**
 * Custom hook for handling NPC-related errors with user-friendly notifications
 */

import { useCallback, useEffect, useRef, useMemo } from 'react';
import { toast } from 'sonner';
import { errorLogger } from '@/services/errorLogger';

interface NpcErrorHandlerOptions {
  showToasts?: boolean;
  logErrors?: boolean;
  onError?: (error: NpcError) => void;
  maxToastsPerMinute?: number;
}

interface NpcError {
  npcId: number;
  type: 'network' | 'api' | 'parsing' | 'rendering' | 'unknown';
  message: string;
  originalError: Error | undefined;
  retryable: boolean;
  userFriendlyMessage: string;
}

const DEFAULT_OPTIONS: Required<NpcErrorHandlerOptions> = {
  showToasts: true,
  logErrors: true,
  onError: () => {},
  maxToastsPerMinute: 3,
};

/**
 * Hook for handling NPC-related errors with context-aware messages and notifications
 */
export function useNpcErrorHandler(options: NpcErrorHandlerOptions = {}) {
  const config = useMemo(() => ({ ...DEFAULT_OPTIONS, ...options }), [options]);
  const toastCountRef = useRef(0);
  const lastToastResetRef = useRef(Date.now());

  // Reset toast count every minute
  useEffect(() => {
    const interval = setInterval(() => {
      toastCountRef.current = 0;
      lastToastResetRef.current = Date.now();
    }, 60000);

    return () => clearInterval(interval);
  }, []);

  /**
   * Classifies an error and provides user-friendly messaging
   */
  const classifyError = useCallback((error: Error | string, npcId: number): NpcError => {
    const errorMessage = typeof error === 'string' ? error : error.message;
    const errorLower = errorMessage.toLowerCase();

    let type: NpcError['type'] = 'unknown';
    let retryable = true;
    let userFriendlyMessage = 'An unexpected error occurred while loading NPC data.';

    // Network errors
    if (errorLower.includes('fetch') || 
        errorLower.includes('network') || 
        errorLower.includes('connection') ||
        errorLower.includes('timeout')) {
      type = 'network';
      userFriendlyMessage = 'Unable to connect to the MapleStory API. Please check your internet connection and try again.';
      retryable = true;
    }
    // API errors
    else if (errorLower.includes('404') || errorLower.includes('not found')) {
      type = 'api';
      userFriendlyMessage = `NPC ${npcId} data is not available in the MapleStory database.`;
      retryable = false;
    }
    else if (errorLower.includes('500') || errorLower.includes('internal server')) {
      type = 'api';
      userFriendlyMessage = 'The MapleStory API is temporarily unavailable. Please try again later.';
      retryable = true;
    }
    else if (errorLower.includes('403') || errorLower.includes('unauthorized')) {
      type = 'api';
      userFriendlyMessage = 'Access to NPC data is currently restricted. Please try again later.';
      retryable = false;
    }
    else if (errorLower.includes('429') || errorLower.includes('rate limit')) {
      type = 'api';
      userFriendlyMessage = 'Too many requests to the API. Please wait a moment and try again.';
      retryable = true;
    }
    // Parsing errors
    else if (errorLower.includes('json') || errorLower.includes('parse')) {
      type = 'parsing';
      userFriendlyMessage = 'Received invalid data from the MapleStory API. Please try again.';
      retryable = true;
    }
    // Rendering errors
    else if (errorLower.includes('render') || errorLower.includes('component')) {
      type = 'rendering';
      userFriendlyMessage = 'There was an error displaying the NPC card. Please refresh the page.';
      retryable = true;
    }

    return {
      npcId,
      type,
      message: errorMessage,
      originalError: typeof error === 'object' ? error : undefined,
      retryable,
      userFriendlyMessage,
    };
  }, []);

  /**
   * Shows a toast notification if within rate limits
   */
  const showToastIfAllowed = useCallback((npcError: NpcError) => {
    if (!config.showToasts) return;

    // Check rate limiting
    if (toastCountRef.current >= config.maxToastsPerMinute) {
      return;
    }

    toastCountRef.current++;

    const toastOptions = {
      duration: npcError.retryable ? 4000 : 6000,
      action: npcError.retryable ? {
        label: 'Retry',
        onClick: () => window.location.reload(),
      } : undefined,
    };

    switch (npcError.type) {
      case 'network':
        toast.error(npcError.userFriendlyMessage, {
          ...toastOptions,
          description: 'Check your connection and try again.',
        });
        break;
      case 'api':
        if (npcError.retryable) {
          toast.warning(npcError.userFriendlyMessage, {
            ...toastOptions,
            description: 'This is usually temporary.',
          });
        } else {
          toast.info(npcError.userFriendlyMessage, {
            ...toastOptions,
            description: 'This NPC may not exist in the database.',
          });
        }
        break;
      case 'parsing':
      case 'rendering':
        toast.error(npcError.userFriendlyMessage, toastOptions);
        break;
      default:
        toast.error(npcError.userFriendlyMessage, toastOptions);
    }
  }, [config.showToasts, config.maxToastsPerMinute]);

  /**
   * Main error handling function
   */
  const handleError = useCallback((error: Error | string, npcId: number, context?: Record<string, unknown>) => {
    const npcError = classifyError(error, npcId);

    // Log error if enabled
    if (config.logErrors) {
      const errorToLog = npcError.originalError || new Error(npcError.message);
      errorLogger.logError(errorToLog, undefined, {
        userId: npcId.toString(), // Use userId field for NPC ID
        url: `npc_error_${npcError.type}`,
        ...context,
      });
    }

    // Show toast notification
    showToastIfAllowed(npcError);

    // Call custom error handler
    config.onError(npcError);

    return npcError;
  }, [classifyError, config, showToastIfAllowed]);

  /**
   * Handle multiple errors (for batch operations)
   */
  const handleErrors = useCallback((errors: Array<{ error: Error | string; npcId: number; context?: Record<string, unknown> }>) => {
    const processedErrors = errors.map(({ error, npcId, context }) => 
      handleError(error, npcId, context)
    );

    // Show summary toast for multiple errors
    if (config.showToasts && processedErrors.length > 1 && toastCountRef.current < config.maxToastsPerMinute) {
      const retryableCount = processedErrors.filter(e => e.retryable).length;
      const nonRetryableCount = processedErrors.length - retryableCount;

      let summaryMessage = `Failed to load ${processedErrors.length} NPC${processedErrors.length > 1 ? 's' : ''}`;
      
      if (retryableCount > 0 && nonRetryableCount > 0) {
        summaryMessage += ` (${retryableCount} temporary, ${nonRetryableCount} permanent)`;
      } else if (retryableCount > 0) {
        summaryMessage += ' (temporary issues)';
      } else {
        summaryMessage += ' (data not available)';
      }

      toast.error(summaryMessage, {
        duration: 5000,
        action: retryableCount > 0 ? {
          label: 'Retry All',
          onClick: () => window.location.reload(),
        } : undefined,
      });

      toastCountRef.current++;
    }

    return processedErrors;
  }, [handleError, config.showToasts, config.maxToastsPerMinute]);

  /**
   * Create error handler for specific NPC
   */
  const createNpcErrorHandler = useCallback((npcId: number) => {
    return (error: Error | string, context?: Record<string, unknown>) => 
      handleError(error, npcId, context);
  }, [handleError]);

  /**
   * Reset toast counter (useful for manual resets)
   */
  const resetToastCounter = useCallback(() => {
    toastCountRef.current = 0;
    lastToastResetRef.current = Date.now();
  }, []);

  return {
    handleError,
    handleErrors,
    createNpcErrorHandler,
    classifyError,
    resetToastCounter,
  };
}

// Export types for external use
export type { NpcError, NpcErrorHandlerOptions };