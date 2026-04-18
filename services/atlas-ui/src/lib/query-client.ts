import { QueryClient } from '@tanstack/react-query';

/**
 * Default configuration for React Query client
 */
const queryClientConfig = {
  defaultOptions: {
    queries: {
      // Time before data is considered stale (5 minutes)
      staleTime: 5 * 60 * 1000,
      // Time before inactive queries are garbage collected (10 minutes)
      gcTime: 10 * 60 * 1000,
      // Retry failed requests up to 3 times
      retry: 3,
      // Retry with exponential backoff
      retryDelay: (attemptIndex: number) => Math.min(1000 * 2 ** attemptIndex, 30000),
      // Don't refetch on window focus by default
      refetchOnWindowFocus: false,
      // Don't refetch on reconnect by default
      refetchOnReconnect: true,
      // Don't refetch on mount if data exists and is not stale
      refetchOnMount: true,
    },
    mutations: {
      // Retry failed mutations once
      retry: 1,
      // Retry with 1 second delay
      retryDelay: 1000,
    },
  },
};

/**
 * Creates a new QueryClient instance with default configuration
 */
export function createQueryClient(): QueryClient {
  return new QueryClient(queryClientConfig);
}

/**
 * Global query client instance for use in client-side components
 */
export const queryClient = createQueryClient();