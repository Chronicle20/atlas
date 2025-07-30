/**
 * React Query hooks for tenant management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Basic tenant operations (CRUD)
 * - Tenant configuration operations (CRUD)
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { tenantsService, type TenantBasic, type TenantBasicAttributes, type TenantConfig, type TenantConfigAttributes } from '@/services/api/tenants.service';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const tenantKeys = {
  all: ['tenants'] as const,
  basic: () => [...tenantKeys.all, 'basic'] as const,
  basicLists: () => [...tenantKeys.basic(), 'list'] as const,
  basicList: (options?: QueryOptions) => [...tenantKeys.basicLists(), options] as const,
  basicDetails: () => [...tenantKeys.basic(), 'detail'] as const,
  basicDetail: (id: string) => [...tenantKeys.basicDetails(), id] as const,
  
  configs: () => [...tenantKeys.all, 'configs'] as const,
  configLists: () => [...tenantKeys.configs(), 'list'] as const,
  configList: (options?: QueryOptions) => [...tenantKeys.configLists(), options] as const,
  configDetails: () => [...tenantKeys.configs(), 'detail'] as const,
  configDetail: (id: string) => [...tenantKeys.configDetails(), id] as const,
};

// ============================================================================
// BASIC TENANT HOOKS
// ============================================================================

/**
 * Hook to fetch all basic tenants
 */
export function useTenants(options?: QueryOptions): UseQueryResult<TenantBasic[], Error> {
  return useQuery({
    queryKey: tenantKeys.basicList(options),
    queryFn: () => tenantsService.getAllTenants(options),
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific tenant by ID
 */
export function useTenant(id: string, options?: ServiceOptions): UseQueryResult<TenantBasic, Error> {
  return useQuery({
    queryKey: tenantKeys.basicDetail(id),
    queryFn: () => tenantsService.getTenantById(id, options),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to create a new tenant
 */
export function useCreateTenant(): UseMutationResult<TenantBasic, Error, TenantBasicAttributes> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (attributes: TenantBasicAttributes) => tenantsService.createTenant(attributes),
    onSuccess: (newTenant) => {
      // Invalidate and refetch tenant lists
      queryClient.invalidateQueries({ queryKey: tenantKeys.basicLists() });
      
      // Add the new tenant to the cache
      queryClient.setQueryData(tenantKeys.basicDetail(newTenant.id), newTenant);
    },
    onError: (error) => {
      console.error('Failed to create tenant:', error);
    },
  });
}

/**
 * Hook to update an existing tenant
 */
export function useUpdateTenant(): UseMutationResult<
  TenantBasic,
  Error,
  { tenant: TenantBasic; updates: Partial<TenantBasicAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, updates }) => tenantsService.updateTenant(tenant, updates),
    onMutate: async ({ tenant, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: tenantKeys.basicDetail(tenant.id) });
      
      // Snapshot the previous value
      const previousTenant = queryClient.getQueryData<TenantBasic>(tenantKeys.basicDetail(tenant.id));
      
      // Optimistically update the cache
      const optimisticTenant: TenantBasic = {
        ...tenant,
        attributes: { ...tenant.attributes, ...updates },
      };
      queryClient.setQueryData(tenantKeys.basicDetail(tenant.id), optimisticTenant);
      
      return { previousTenant };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousTenant) {
        queryClient.setQueryData(tenantKeys.basicDetail(variables.tenant.id), context.previousTenant);
      }
      console.error('Failed to update tenant:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: tenantKeys.basicDetail(variables.tenant.id) });
      queryClient.invalidateQueries({ queryKey: tenantKeys.basicLists() });
    },
  });
}

/**
 * Hook to delete a tenant
 */
export function useDeleteTenant(): UseMutationResult<void, Error, { id: string; tenant?: TenantBasic }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }) => tenantsService.deleteTenant(id),
    onMutate: async ({ id, tenant }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: tenantKeys.basicDetail(id) });
      
      // Snapshot the previous value
      const previousTenant = queryClient.getQueryData<TenantBasic>(tenantKeys.basicDetail(id));
      
      // Optimistically remove from cache
      queryClient.removeQueries({ queryKey: tenantKeys.basicDetail(id) });
      
      return { previousTenant };
    },
    onError: (error, variables, context) => {
      // Restore the tenant to cache on error
      if (context?.previousTenant) {
        queryClient.setQueryData(tenantKeys.basicDetail(variables.id), context.previousTenant);
      }
      console.error('Failed to delete tenant:', error);
    },
    onSettled: () => {
      // Invalidate tenant lists
      queryClient.invalidateQueries({ queryKey: tenantKeys.basicLists() });
    },
  });
}

// ============================================================================
// TENANT CONFIGURATION HOOKS
// ============================================================================

/**
 * Hook to fetch all tenant configurations
 */
export function useTenantConfigurations(options?: QueryOptions): UseQueryResult<TenantConfig[], Error> {
  return useQuery({
    queryKey: tenantKeys.configList(options),
    queryFn: () => tenantsService.getAllTenantConfigurations(options),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch a specific tenant configuration by ID
 */
export function useTenantConfiguration(id: string, options?: ServiceOptions): UseQueryResult<TenantConfig, Error> {
  return useQuery({
    queryKey: tenantKeys.configDetail(id),
    queryFn: () => tenantsService.getTenantConfigurationById(id, options),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to create a new tenant configuration
 */
export function useCreateTenantConfiguration(): UseMutationResult<TenantConfig, Error, TenantConfigAttributes> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (attributes: TenantConfigAttributes) => tenantsService.createTenantConfiguration(attributes),
    onSuccess: (newConfig) => {
      // Invalidate and refetch configuration lists
      queryClient.invalidateQueries({ queryKey: tenantKeys.configLists() });
      
      // Add the new configuration to the cache
      queryClient.setQueryData(tenantKeys.configDetail(newConfig.id), newConfig);
    },
    onError: (error) => {
      console.error('Failed to create tenant configuration:', error);
    },
  });
}

/**
 * Hook to update an existing tenant configuration
 */
export function useUpdateTenantConfiguration(): UseMutationResult<
  TenantConfig,
  Error,
  { tenant: TenantConfig; updates: Partial<TenantConfigAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, updates }) => tenantsService.updateTenantConfiguration(tenant, updates),
    onMutate: async ({ tenant, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: tenantKeys.configDetail(tenant.id) });
      
      // Snapshot the previous value
      const previousConfig = queryClient.getQueryData<TenantConfig>(tenantKeys.configDetail(tenant.id));
      
      // Optimistically update the cache
      const optimisticConfig: TenantConfig = {
        ...tenant,
        attributes: { ...tenant.attributes, ...updates },
      };
      queryClient.setQueryData(tenantKeys.configDetail(tenant.id), optimisticConfig);
      
      return { previousConfig };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousConfig) {
        queryClient.setQueryData(tenantKeys.configDetail(variables.tenant.id), context.previousConfig);
      }
      console.error('Failed to update tenant configuration:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: tenantKeys.configDetail(variables.tenant.id) });
      queryClient.invalidateQueries({ queryKey: tenantKeys.configLists() });
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to create tenant configuration from template
 */
export function useCreateFromTemplate() {
  return (template: { attributes: TenantConfigAttributes }): TenantConfigAttributes => {
    return tenantsService.createTenantFromTemplate(template);
  };
}

/**
 * Hook to invalidate tenant-related queries
 */
export function useInvalidateTenants() {
  const queryClient = useQueryClient();
  
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: tenantKeys.all }),
    invalidateBasic: () => queryClient.invalidateQueries({ queryKey: tenantKeys.basic() }),
    invalidateConfigs: () => queryClient.invalidateQueries({ queryKey: tenantKeys.configs() }),
    invalidateTenant: (id: string) => queryClient.invalidateQueries({ queryKey: tenantKeys.basicDetail(id) }),
    invalidateConfig: (id: string) => queryClient.invalidateQueries({ queryKey: tenantKeys.configDetail(id) }),
  };
}

// Export types for external use
export type { TenantBasic, TenantBasicAttributes, TenantConfig, TenantConfigAttributes };