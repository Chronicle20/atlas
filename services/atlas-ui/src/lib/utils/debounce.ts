/**
 * Debouncing utilities for API calls and user interactions
 */

import React, { useCallback, useRef } from 'react';

/**
 * Debounce function for regular JavaScript functions
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: NodeJS.Timeout;
  
  return (...args: Parameters<T>) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => func(...args), delay);
  };
}

/**
 * Debounce function for async functions
 */
export function debounceAsync<T extends (...args: unknown[]) => Promise<unknown>>(
  func: T,
  delay: number
): (...args: Parameters<T>) => Promise<ReturnType<T>> {
  let timeoutId: NodeJS.Timeout;
  let latestResolve: ((value: ReturnType<T>) => void) | null = null;
  let latestReject: ((reason?: unknown) => void) | null = null;
  
  return (...args: Parameters<T>): Promise<ReturnType<T>> => {
    return new Promise((resolve, reject) => {
      // Cancel previous timeout and reject previous promise
      if (latestResolve && latestReject) {
        clearTimeout(timeoutId);
        latestReject(new Error('Debounced call cancelled'));
      }
      
      latestResolve = resolve;
      latestReject = reject;
      
      timeoutId = setTimeout(async () => {
        try {
          const result = await func(...args);
          resolve(result as ReturnType<T>);
        } catch (error) {
          reject(error);
        } finally {
          latestResolve = null;
          latestReject = null;
        }
      }, delay);
    });
  };
}

/**
 * React hook for debouncing values
 */
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value);
  
  React.useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);
    
    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);
  
  return debouncedValue;
}

/**
 * React hook for debouncing function calls
 */
export function useDebouncedCallback<T extends (...args: unknown[]) => unknown>(
  callback: T,
  delay: number,
  deps: React.DependencyList = []
): (...args: Parameters<T>) => void {
  const timeoutRef = useRef<NodeJS.Timeout | undefined>(undefined);
  
  return useCallback(
    (...args: Parameters<T>) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      
      timeoutRef.current = setTimeout(() => {
        callback(...args);
      }, delay);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [callback, delay, ...deps]
  );
}

/**
 * React hook for debouncing async function calls
 */
export function useDebouncedAsyncCallback<T extends (...args: unknown[]) => Promise<unknown>>(
  callback: T,
  delay: number,
  deps: React.DependencyList = []
): (...args: Parameters<T>) => Promise<ReturnType<T>> {
  const timeoutRef = useRef<NodeJS.Timeout | undefined>(undefined);
  const promiseRef = useRef<{
    resolve: (value: ReturnType<T>) => void;
    reject: (reason?: unknown) => void;
  } | null>(null);
  
  return useCallback(
    (...args: Parameters<T>): Promise<ReturnType<T>> => {
      return new Promise((resolve, reject) => {
        // Cancel previous timeout and reject previous promise
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        
        if (promiseRef.current) {
          promiseRef.current.reject(new Error('Debounced call cancelled'));
        }
        
        promiseRef.current = { resolve, reject };
        
        timeoutRef.current = setTimeout(async () => {
          try {
            const result = await callback(...args);
            if (promiseRef.current) {
              promiseRef.current.resolve(result as ReturnType<T>);
            }
          } catch (error) {
            if (promiseRef.current) {
              promiseRef.current.reject(error);
            }
          } finally {
            promiseRef.current = null;
          }
        }, delay);
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [callback, delay, ...deps]
  );
}

/**
 * Throttle function - limits execution to once per interval
 */
export function throttle<T extends (...args: unknown[]) => unknown>(
  func: T,
  interval: number
): (...args: Parameters<T>) => void {
  let lastCall = 0;
  
  return (...args: Parameters<T>) => {
    const now = Date.now();
    if (now - lastCall >= interval) {
      lastCall = now;
      func(...args);
    }
  };
}

/**
 * React hook for throttling function calls
 */
export function useThrottledCallback<T extends (...args: unknown[]) => unknown>(
  callback: T,
  interval: number,
  deps: React.DependencyList = []
): (...args: Parameters<T>) => void {
  const lastCallRef = useRef<number>(0);
  
  return useCallback(
    (...args: Parameters<T>) => {
      const now = Date.now();
      if (now - lastCallRef.current >= interval) {
        lastCallRef.current = now;
        callback(...args);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [callback, interval, ...deps]
  );
}