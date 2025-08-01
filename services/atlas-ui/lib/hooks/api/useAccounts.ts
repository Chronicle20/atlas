/**
 * React Query hooks for account management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Account retrieval operations with tenant context
 * - Account session management (terminate sessions)
 * - Account statistics and analytics
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { accountsService, type Account, type AccountAttributes, type AccountQueryOptions } from '@/services/api/accounts.service';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const accountKeys = {
  all: ['accounts'] as const,
  lists: () => [...accountKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: AccountQueryOptions) => [...accountKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  details: () => [...accountKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, id: string) => [...accountKeys.details(), tenant?.id || 'no-tenant', id] as const,
  
  // Specialized queries
  searches: () => [...accountKeys.all, 'search'] as const,
  search: (tenant: Tenant | null, pattern: string) => [...accountKeys.searches(), tenant?.id || 'no-tenant', pattern] as const,
  loggedIn: (tenant: Tenant | null) => [...accountKeys.all, 'loggedIn', tenant?.id || 'no-tenant'] as const,
  banned: (tenant: Tenant | null) => [...accountKeys.all, 'banned', tenant?.id || 'no-tenant'] as const,
  stats: (tenant: Tenant | null) => [...accountKeys.all, 'stats', tenant?.id || 'no-tenant'] as const,
};

// ============================================================================
// ACCOUNT QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all accounts for a tenant with optional filtering
 */
export function useAccounts(
  tenant: Tenant, 
  options?: AccountQueryOptions
): UseQueryResult<Account[], Error> {
  return useQuery({
    queryKey: accountKeys.list(tenant, options),
    queryFn: () => accountsService.getAllAccounts(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000, // 2 minutes (accounts change more frequently than tenants)
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to fetch a specific account by ID within a tenant
 */
export function useAccount(
  tenant: Tenant, 
  id: string, 
  options?: ServiceOptions
): UseQueryResult<Account, Error> {
  return useQuery({
    queryKey: accountKeys.detail(tenant, id),
    queryFn: () => accountsService.getAccountById(tenant, id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to check if an account exists within a tenant
 */
export function useAccountExists(
  tenant: Tenant, 
  id: string, 
  options?: ServiceOptions
): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: [...accountKeys.detail(tenant, id), 'exists'],
    queryFn: () => accountsService.accountExists(tenant, id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 1 * 60 * 1000, // 1 minute for existence checks
    gcTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to search accounts by name pattern within a tenant
 */
export function useAccountSearch(
  tenant: Tenant, 
  namePattern: string, 
  options?: ServiceOptions
): UseQueryResult<Account[], Error> {
  return useQuery({
    queryKey: accountKeys.search(tenant, namePattern),
    queryFn: () => accountsService.searchAccountsByName(tenant, namePattern, options),
    enabled: !!tenant?.id && !!namePattern && namePattern.length > 0,
    staleTime: 1 * 60 * 1000, // Search results can be more stale
    gcTime: 3 * 60 * 1000,
  });
}

/**
 * Hook to fetch logged-in accounts for a tenant
 */
export function useLoggedInAccounts(
  tenant: Tenant, 
  options?: ServiceOptions
): UseQueryResult<Account[], Error> {
  return useQuery({
    queryKey: accountKeys.loggedIn(tenant),
    queryFn: () => accountsService.getLoggedInAccounts(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 30 * 1000, // 30 seconds (login status changes frequently)
    gcTime: 2 * 60 * 1000,
    refetchInterval: 60 * 1000, // Auto-refresh every minute
  });
}

/**
 * Hook to fetch banned accounts for a tenant
 */
export function useBannedAccounts(
  tenant: Tenant, 
  options?: ServiceOptions
): UseQueryResult<Account[], Error> {
  return useQuery({
    queryKey: accountKeys.banned(tenant),
    queryFn: () => accountsService.getBannedAccounts(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 5 * 60 * 1000, // 5 minutes (ban status changes less frequently)
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch account statistics for a tenant
 */
export function useAccountStats(
  tenant: Tenant, 
  options?: ServiceOptions
): UseQueryResult<{
  total: number;
  loggedIn: number;
  banned: number;
  totalCharacterSlots: number;
  averageCharacterSlots: number;
}, Error> {
  return useQuery({
    queryKey: accountKeys.stats(tenant),
    queryFn: () => accountsService.getAccountStats(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

// ============================================================================
// ACCOUNT MUTATION HOOKS
// ============================================================================

/**
 * Hook to terminate a single account session
 */
export function useTerminateAccountSession(): UseMutationResult<
  void,
  Error,
  { tenant: Tenant; accountId: string; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, accountId, options }) => 
      accountsService.terminateAccountSession(tenant, accountId, options),
    onMutate: async ({ tenant, accountId }) => {
      // Cancel any outgoing refetches for this account
      await queryClient.cancelQueries({ queryKey: accountKeys.detail(tenant, accountId) });
      
      // Snapshot the previous account data
      const previousAccount = queryClient.getQueryData<Account>(accountKeys.detail(tenant, accountId));
      
      // Optimistically update the account to show as logged out
      if (previousAccount) {
        const optimisticAccount: Account = {
          ...previousAccount,
          attributes: {
            ...previousAccount.attributes,
            loggedIn: 0,
          },
        };
        queryClient.setQueryData(accountKeys.detail(tenant, accountId), optimisticAccount);
      }
      
      return { previousAccount };
    },
    onError: (error, { tenant, accountId }, context) => {
      // Revert optimistic update on error
      if (context?.previousAccount) {
        queryClient.setQueryData(accountKeys.detail(tenant, accountId), context.previousAccount);
      }
      console.error('Failed to terminate account session:', error);
    },
    onSettled: (data, error, { tenant, accountId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: accountKeys.detail(tenant, accountId) });
      queryClient.invalidateQueries({ queryKey: accountKeys.loggedIn(tenant) });
      queryClient.invalidateQueries({ queryKey: accountKeys.stats(tenant) });
      queryClient.invalidateQueries({ queryKey: accountKeys.lists() });
    },
  });
}

/**
 * Hook to terminate multiple account sessions in batch
 */
export function useTerminateMultipleSessions(): UseMutationResult<
  { successful: string[]; failed: Array<{ id: string; error: string }> },
  Error,
  { tenant: Tenant; accountIds: string[]; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, accountIds, options }) => 
      accountsService.terminateMultipleSessions(tenant, accountIds, options),
    onMutate: async ({ tenant, accountIds }) => {
      // Cancel outgoing refetches for affected accounts
      const cancelPromises = accountIds.map(accountId =>
        queryClient.cancelQueries({ queryKey: accountKeys.detail(tenant, accountId) })
      );
      await Promise.all(cancelPromises);
      
      // Snapshot previous account data
      const previousAccounts = new Map<string, Account>();
      
      // Optimistically update all accounts to show as logged out
      accountIds.forEach(accountId => {
        const previousAccount = queryClient.getQueryData<Account>(accountKeys.detail(tenant, accountId));
        if (previousAccount) {
          previousAccounts.set(accountId, previousAccount);
          
          const optimisticAccount: Account = {
            ...previousAccount,
            attributes: {
              ...previousAccount.attributes,
              loggedIn: 0,
            },
          };
          queryClient.setQueryData(accountKeys.detail(tenant, accountId), optimisticAccount);
        }
      });
      
      return { previousAccounts };
    },
    onError: (error, { tenant, accountIds }, context) => {
      // Revert optimistic updates on error
      if (context?.previousAccounts) {
        accountIds.forEach(accountId => {
          const previousAccount = context.previousAccounts.get(accountId);
          if (previousAccount) {
            queryClient.setQueryData(accountKeys.detail(tenant, accountId), previousAccount);
          }
        });
      }
      console.error('Failed to terminate multiple account sessions:', error);
    },
    onSettled: (data, error, { tenant, accountIds }) => {
      // Invalidate and refetch relevant queries
      accountIds.forEach(accountId => {
        queryClient.invalidateQueries({ queryKey: accountKeys.detail(tenant, accountId) });
      });
      queryClient.invalidateQueries({ queryKey: accountKeys.loggedIn(tenant) });
      queryClient.invalidateQueries({ queryKey: accountKeys.stats(tenant) });
      queryClient.invalidateQueries({ queryKey: accountKeys.lists() });
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate account-related queries for a specific tenant
 */
export function useInvalidateAccounts() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Invalidate all account queries
     */
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: accountKeys.all }),
    
    /**
     * Invalidate all account lists for a tenant
     */
    invalidateLists: () => 
      queryClient.invalidateQueries({ queryKey: accountKeys.lists() }),
    
    /**
     * Invalidate specific account list for a tenant
     */
    invalidateList: (tenant: Tenant, options?: AccountQueryOptions) =>
      queryClient.invalidateQueries({ queryKey: accountKeys.list(tenant, options) }),
    
    /**
     * Invalidate specific account details
     */
    invalidateAccount: (tenant: Tenant, id: string) =>
      queryClient.invalidateQueries({ queryKey: accountKeys.detail(tenant, id) }),
    
    /**
     * Invalidate logged-in accounts for a tenant
     */
    invalidateLoggedIn: (tenant: Tenant) =>
      queryClient.invalidateQueries({ queryKey: accountKeys.loggedIn(tenant) }),
    
    /**
     * Invalidate banned accounts for a tenant
     */
    invalidateBanned: (tenant: Tenant) =>
      queryClient.invalidateQueries({ queryKey: accountKeys.banned(tenant) }),
    
    /**
     * Invalidate account statistics for a tenant
     */
    invalidateStats: (tenant: Tenant) =>
      queryClient.invalidateQueries({ queryKey: accountKeys.stats(tenant) }),
    
    /**
     * Invalidate all account-related queries for a specific tenant
     */
    invalidateAllForTenant: (tenant: Tenant) => {
      queryClient.invalidateQueries({ queryKey: [...accountKeys.all, tenant.id] });
    },
  };
}

/**
 * Hook to prefetch account data for performance optimization
 */
export function usePrefetchAccounts() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Prefetch all accounts for a tenant
     */
    prefetchAccounts: (tenant: Tenant, options?: AccountQueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: accountKeys.list(tenant, options),
        queryFn: () => accountsService.getAllAccounts(tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
    
    /**
     * Prefetch specific account
     */
    prefetchAccount: (tenant: Tenant, id: string, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: accountKeys.detail(tenant, id),
        queryFn: () => accountsService.getAccountById(tenant, id, options),
        staleTime: 2 * 60 * 1000,
      }),
    
    /**
     * Prefetch logged-in accounts
     */
    prefetchLoggedIn: (tenant: Tenant, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: accountKeys.loggedIn(tenant),
        queryFn: () => accountsService.getLoggedInAccounts(tenant, options),
        staleTime: 30 * 1000,
      }),
    
    /**
     * Prefetch account statistics
     */
    prefetchStats: (tenant: Tenant, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: accountKeys.stats(tenant),
        queryFn: () => accountsService.getAccountStats(tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
  };
}

// Export types for external use
export type { Account, AccountAttributes, AccountQueryOptions };