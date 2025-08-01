/**
 * React Query hooks for template management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Basic template operations (CRUD)
 * - Template searching and filtering by region, version
 * - Template validation and consistency checking
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 * - Template cloning and export functionality
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { templatesService } from '@/services/api/templates.service';
import type { Template, TemplateAttributes } from '@/types/models/template';
import type { ServiceOptions, QueryOptions, BatchResult } from '@/services/api/base.service';

// Query keys for consistent cache management
export const templateKeys = {
  all: ['templates'] as const,
  lists: () => [...templateKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...templateKeys.lists(), options] as const,
  details: () => [...templateKeys.all, 'detail'] as const,
  detail: (id: string) => [...templateKeys.details(), id] as const,
  
  // Specialized query keys for filtering
  byRegion: (region: string, options?: QueryOptions) => [...templateKeys.all, 'region', region, options] as const,
  byVersion: (majorVersion: number, minorVersion?: number, options?: QueryOptions) => 
    [...templateKeys.all, 'version', majorVersion, minorVersion, options] as const,
  byRegionAndVersion: (region: string, majorVersion: number, minorVersion?: number, options?: QueryOptions) => 
    [...templateKeys.all, 'regionVersion', region, majorVersion, minorVersion, options] as const,
    
  // Utility query keys
  validation: (id: string) => [...templateKeys.all, 'validation', id] as const,
  export: (format: string, options?: QueryOptions) => [...templateKeys.all, 'export', format, options] as const,
};

// ============================================================================
// BASIC TEMPLATE HOOKS
// ============================================================================

/**
 * Hook to fetch all templates with automatic sorting
 */
export function useTemplates(options?: QueryOptions): UseQueryResult<Template[], Error> {
  return useQuery({
    queryKey: templateKeys.list(options),
    queryFn: () => templatesService.getAll(options),
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific template by ID
 */
export function useTemplate(id: string, options?: ServiceOptions): UseQueryResult<Template, Error> {
  return useQuery({
    queryKey: templateKeys.detail(id),
    queryFn: () => templatesService.getById(id, options),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to check if a template exists
 */
export function useTemplateExists(id: string, options?: ServiceOptions): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: [...templateKeys.details(), id, 'exists'],
    queryFn: () => templatesService.exists(id, options),
    enabled: !!id,
    staleTime: 2 * 60 * 1000, // 2 minutes (shorter for existence checks)
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to create a new template
 */
export function useCreateTemplate(): UseMutationResult<Template, Error, TemplateAttributes> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (attributes: TemplateAttributes) => templatesService.create(attributes),
    onSuccess: (newTemplate) => {
      // Invalidate and refetch template lists
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      // Add the new template to the cache
      queryClient.setQueryData(templateKeys.detail(newTemplate.id), newTemplate);
      
      // Invalidate region-based queries that might include this template
      queryClient.invalidateQueries({ 
        queryKey: [...templateKeys.all, 'region', newTemplate.attributes.region] 
      });
      
      // Invalidate version-based queries
      queryClient.invalidateQueries({ 
        queryKey: [...templateKeys.all, 'version', newTemplate.attributes.majorVersion] 
      });
    },
    onError: (error) => {
      console.error('Failed to create template:', error);
    },
  });
}

/**
 * Hook to update an existing template
 */
export function useUpdateTemplate(): UseMutationResult<
  Template,
  Error,
  { id: string; updates: Partial<TemplateAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }) => templatesService.update(id, updates),
    onMutate: async ({ id, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: templateKeys.detail(id) });
      
      // Snapshot the previous value
      const previousTemplate = queryClient.getQueryData<Template>(templateKeys.detail(id));
      
      // Optimistically update the cache if we have previous data
      if (previousTemplate) {
        const optimisticTemplate: Template = {
          ...previousTemplate,
          attributes: { ...previousTemplate.attributes, ...updates },
        };
        queryClient.setQueryData(templateKeys.detail(id), optimisticTemplate);
      }
      
      return { previousTemplate };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousTemplate) {
        queryClient.setQueryData(templateKeys.detail(variables.id), context.previousTemplate);
      }
      console.error('Failed to update template:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: templateKeys.detail(variables.id) });
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      // Invalidate filtered queries if the updated fields might affect them
      if (data) {
        queryClient.invalidateQueries({ 
          queryKey: [...templateKeys.all, 'region', data.attributes.region] 
        });
        queryClient.invalidateQueries({ 
          queryKey: [...templateKeys.all, 'version', data.attributes.majorVersion] 
        });
      }
    },
  });
}

/**
 * Hook to partially update template (PATCH)
 */
export function usePatchTemplate(): UseMutationResult<
  Template,
  Error,
  { id: string; updates: Partial<TemplateAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }) => templatesService.patch(id, updates),
    onSuccess: (updatedTemplate) => {
      // Update cache with the new template data
      queryClient.setQueryData(templateKeys.detail(updatedTemplate.id), updatedTemplate);
      
      // Invalidate lists to reflect changes
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
    },
    onError: (error) => {
      console.error('Failed to patch template:', error);
    },
  });
}

/**
 * Hook to delete a template
 */
export function useDeleteTemplate(): UseMutationResult<void, Error, { id: string }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }) => templatesService.delete(id),
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: templateKeys.detail(id) });
      
      // Snapshot the previous value
      const previousTemplate = queryClient.getQueryData<Template>(templateKeys.detail(id));
      
      // Optimistically remove from cache
      queryClient.removeQueries({ queryKey: templateKeys.detail(id) });
      
      return { previousTemplate };
    },
    onError: (error, variables, context) => {
      // Restore the template to cache on error
      if (context?.previousTemplate) {
        queryClient.setQueryData(templateKeys.detail(variables.id), context.previousTemplate);
      }
      console.error('Failed to delete template:', error);
    },
    onSettled: () => {
      // Invalidate template lists
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
    },
  });
}

// ============================================================================
// BATCH OPERATION HOOKS
// ============================================================================

/**
 * Hook to create multiple templates in batch
 */
export function useCreateTemplatesBatch(): UseMutationResult<
  BatchResult<Template>,
  Error,
  { templates: TemplateAttributes[]; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ templates, options }) => templatesService.createBatch(templates, options),
    onSuccess: (result) => {
      // Invalidate template lists
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      // Add successful templates to individual caches
      result.successes.forEach(template => {
        queryClient.setQueryData(templateKeys.detail(template.id), template);
      });
    },
    onError: (error) => {
      console.error('Failed to create templates batch:', error);
    },
  });
}

/**
 * Hook to update multiple templates in batch
 */
export function useUpdateTemplatesBatch(): UseMutationResult<
  BatchResult<Template>,
  Error,
  { updates: Array<{ id: string; data: Partial<TemplateAttributes> }>; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ updates, options }) => templatesService.updateBatch(updates, options),
    onSuccess: (result) => {
      // Update caches for successful updates
      result.successes.forEach(template => {
        queryClient.setQueryData(templateKeys.detail(template.id), template);
      });
      
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
    },
    onError: (error) => {
      console.error('Failed to update templates batch:', error);
    },
  });
}

/**
 * Hook to delete multiple templates in batch
 */
export function useDeleteTemplatesBatch(): UseMutationResult<
  BatchResult<string>,
  Error,
  { ids: string[]; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ ids, options }) => templatesService.deleteBatch(ids, options),
    onMutate: async ({ ids }) => {
      // Cancel queries for all templates being deleted
      const cancelPromises = ids.map(id => 
        queryClient.cancelQueries({ queryKey: templateKeys.detail(id) })
      );
      await Promise.all(cancelPromises);
      
      // Snapshot previous values
      const previousTemplates = ids.map(id => ({
        id,
        template: queryClient.getQueryData<Template>(templateKeys.detail(id))
      }));
      
      // Optimistically remove from caches
      ids.forEach(id => {
        queryClient.removeQueries({ queryKey: templateKeys.detail(id) });
      });
      
      return { previousTemplates };
    },
    onError: (error, variables, context) => {
      // Restore templates to cache on error
      if (context?.previousTemplates) {
        context.previousTemplates.forEach(({ id, template }) => {
          if (template) {
            queryClient.setQueryData(templateKeys.detail(id), template);
          }
        });
      }
      console.error('Failed to delete templates batch:', error);
    },
    onSettled: () => {
      // Invalidate template lists
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
    },
  });
}

// ============================================================================
// SPECIALIZED QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch templates by region
 */
export function useTemplatesByRegion(region: string, options?: QueryOptions): UseQueryResult<Template[], Error> {
  return useQuery({
    queryKey: templateKeys.byRegion(region, options),
    queryFn: () => templatesService.getByRegion(region, options),
    enabled: !!region,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch templates by version
 */
export function useTemplatesByVersion(
  majorVersion: number, 
  minorVersion?: number, 
  options?: QueryOptions
): UseQueryResult<Template[], Error> {
  return useQuery({
    queryKey: templateKeys.byVersion(majorVersion, minorVersion, options),
    queryFn: () => templatesService.getByVersion(majorVersion, minorVersion, options),
    enabled: majorVersion >= 0,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch templates by region and version combination
 */
export function useTemplatesByRegionAndVersion(
  region: string,
  majorVersion: number,
  minorVersion?: number,
  options?: QueryOptions
): UseQueryResult<Template[], Error> {
  return useQuery({
    queryKey: templateKeys.byRegionAndVersion(region, majorVersion, minorVersion, options),
    queryFn: () => templatesService.getByRegionAndVersion(region, majorVersion, minorVersion, options),
    enabled: !!region && majorVersion >= 0,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

// ============================================================================
// UTILITY AND VALIDATION HOOKS
// ============================================================================

/**
 * Hook to validate template consistency
 */
export function useValidateTemplate(templateId: string): UseQueryResult<
  { isValid: boolean; errors: string[] }, 
  Error
> {
  return useQuery({
    queryKey: templateKeys.validation(templateId),
    queryFn: () => templatesService.validateTemplateConsistency(templateId),
    enabled: !!templateId,
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to export templates
 */
export function useExportTemplates(): UseMutationResult<
  Blob,
  Error,
  { format?: 'json' | 'csv'; options?: QueryOptions }
> {
  return useMutation({
    mutationFn: ({ format = 'json', options }) => templatesService.export(format, options),
    onError: (error) => {
      console.error('Failed to export templates:', error);
    },
  });
}

/**
 * Hook to clone template attributes
 */
export function useCloneTemplate() {
  return (template: Template): TemplateAttributes => {
    return templatesService.cloneTemplate(template);
  };
}

// ============================================================================
// CACHE MANAGEMENT HOOKS
// ============================================================================

/**
 * Hook to invalidate template-related queries
 */
export function useInvalidateTemplates() {
  const queryClient = useQueryClient();
  
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: templateKeys.all }),
    invalidateLists: () => queryClient.invalidateQueries({ queryKey: templateKeys.lists() }),
    invalidateTemplate: (id: string) => queryClient.invalidateQueries({ queryKey: templateKeys.detail(id) }),
    invalidateByRegion: (region: string) => queryClient.invalidateQueries({ 
      queryKey: [...templateKeys.all, 'region', region] 
    }),
    invalidateByVersion: (majorVersion: number, minorVersion?: number) => queryClient.invalidateQueries({ 
      queryKey: [...templateKeys.all, 'version', majorVersion, minorVersion] 
    }),
    clearCache: () => {
      templatesService.clearCache();
      queryClient.invalidateQueries({ queryKey: templateKeys.all });
    },
  };
}

/**
 * Hook to get template cache statistics
 */
export function useTemplateCacheStats() {
  return useQuery({
    queryKey: [...templateKeys.all, 'cacheStats'],
    queryFn: () => templatesService.getCacheStats(),
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 60 * 1000, // 1 minute
  });
}

// Export types for external use
export type { Template, TemplateAttributes };