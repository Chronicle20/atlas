/**
 * React Query hooks for service configuration management
 *
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Login service configurations
 * - Channel service configurations
 * - Drops service configurations
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import {
  servicesService,
  type Service,
  type CreateServiceInput,
  type UpdateServiceInput,
} from '@/services/api/services.service';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';
import { api } from '@/lib/api/client';

// Query keys for consistent cache management
export const serviceKeys = {
  all: ['services'] as const,
  lists: () => [...serviceKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...serviceKeys.lists(), options] as const,
  details: () => [...serviceKeys.all, 'detail'] as const,
  detail: (id: string) => [...serviceKeys.details(), id] as const,
};

// ============================================================================
// QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all services
 */
export function useServices(options?: QueryOptions): UseQueryResult<Service[], Error> {
  return useQuery({
    queryKey: serviceKeys.list(options),
    queryFn: () => servicesService.getAllServices({ ...options, useCache: false }),
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific service by ID
 */
export function useService(id: string, options?: ServiceOptions): UseQueryResult<Service, Error> {
  return useQuery({
    queryKey: serviceKeys.detail(id),
    queryFn: () => servicesService.getServiceById(id, { ...options, useCache: false }),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

// ============================================================================
// MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new service
 */
export function useCreateService(): UseMutationResult<Service, Error, CreateServiceInput> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateServiceInput) => servicesService.createService(input),
    retry: false, // Don't retry on failure to prevent loops
    onSuccess: (newService) => {
      // Clear API client cache for services
      api.clearCacheByPattern('services');

      // Invalidate and refetch service lists
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() });

      // Add the new service to the cache
      queryClient.setQueryData(serviceKeys.detail(newService.id), newService);
    },
    onError: (error) => {
      console.error('Failed to create service:', error);
    },
  });
}

/**
 * Hook to update an existing service
 */
export function useUpdateService(): UseMutationResult<
  Service,
  Error,
  { id: string; input: UpdateServiceInput }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, input }) => servicesService.updateService(id, input),
    retry: false, // Don't retry on failure
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: serviceKeys.detail(id) });

      // Snapshot the previous value for rollback
      const previousService = queryClient.getQueryData<Service>(serviceKeys.detail(id));

      // Note: We don't do optimistic updates for service updates because
      // the type system makes it complex to merge attributes correctly.
      // The UI will show a loading state instead.

      return { previousService };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousService) {
        queryClient.setQueryData(serviceKeys.detail(variables.id), context.previousService);
      }
      console.error('Failed to update service:', error);
    },
    onSettled: (data, error, variables) => {
      // Clear API client cache for services
      api.clearCacheByPattern('services');

      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: serviceKeys.detail(variables.id) });
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() });
    },
  });
}

/**
 * Hook to delete a service
 */
export function useDeleteService(): UseMutationResult<void, Error, { id: string }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }) => servicesService.deleteService(id),
    retry: false, // Don't retry on failure
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: serviceKeys.detail(id) });

      // Snapshot the previous value
      const previousService = queryClient.getQueryData<Service>(serviceKeys.detail(id));

      // Optimistically remove from cache
      queryClient.removeQueries({ queryKey: serviceKeys.detail(id) });

      // Also update the list cache optimistically
      const previousList = queryClient.getQueryData<Service[]>(serviceKeys.lists());

      return { previousService, previousList };
    },
    onError: (error, variables, context) => {
      // Restore the service to cache on error
      if (context?.previousService) {
        queryClient.setQueryData(serviceKeys.detail(variables.id), context.previousService);
      }
      console.error('Failed to delete service:', error);
    },
    onSettled: () => {
      // Clear API client cache for services
      api.clearCacheByPattern('services');

      // Invalidate service lists
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() });
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate service-related queries
 */
export function useInvalidateServices() {
  const queryClient = useQueryClient();

  return {
    invalidateAll: () => {
      api.clearCacheByPattern('services');
      queryClient.invalidateQueries({ queryKey: serviceKeys.all });
    },
    invalidateLists: () => {
      api.clearCacheByPattern('services');
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() });
    },
    invalidateService: (id: string) => {
      api.clearCacheByPattern('services');
      queryClient.invalidateQueries({ queryKey: serviceKeys.detail(id) });
    },
  };
}

/**
 * Hook to prefetch a service (useful for hover states)
 */
export function usePrefetchService() {
  const queryClient = useQueryClient();

  return (id: string) => {
    queryClient.prefetchQuery({
      queryKey: serviceKeys.detail(id),
      queryFn: () => servicesService.getServiceById(id),
      staleTime: 5 * 60 * 1000,
    });
  };
}

// Export types for external use
export type {
  Service,
  CreateServiceInput,
  UpdateServiceInput,
};
