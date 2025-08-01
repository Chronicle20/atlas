/**
 * Tests for template React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import * as templateService from '@/services/api/templates.service';
import {
  useTemplates,
  useTemplate,
  useTemplateExists,
  useCreateTemplate,
  useUpdateTemplate,
  useDeleteTemplate,
  useCreateTemplatesBatch,
  useUpdateTemplatesBatch,
  useDeleteTemplatesBatch,
  useTemplatesByRegion,
  useTemplatesByVersion,
  useTemplatesByRegionAndVersion,
  useValidateTemplate,
  useExportTemplates,
  useCloneTemplate,
  useInvalidateTemplates,
  useTemplateCacheStats,
  templateKeys,
} from './useTemplates';
import type { Template, TemplateAttributes } from '@/types/models/template';

// Mock the templates service
jest.mock('@/services/api/templates.service');

const mockTemplatesService = templateService.templatesService as jest.Mocked<typeof templateService.templatesService>;

// Sample test data
const mockTemplate: Template = {
  id: '1',
  attributes: {
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: {
      templates: [{
        jobIndex: 0,
        subJobIndex: 0,
        gender: 0,
        mapId: 100000000,
        faces: [20000, 20001],
        hairs: [30000, 30001],
        hairColors: [0, 1],
        skinColors: [0, 1],
        tops: [1040000],
        bottoms: [1060000],
        shoes: [1072000],
        weapons: [1302000],
        items: [],
        skills: [1000, 1001]
      }]
    },
    npcs: [{
      npcId: 9000000,
      impl: 'NpcShop'
    }],
    socket: {
      handlers: [{
        opCode: '0x01',
        validator: 'loginValidator',
        handler: 'loginHandler',
        options: {}
      }],
      writers: [{
        opCode: '0x02',
        writer: 'loginWriter',
        options: {}
      }]
    },
    worlds: [{
      name: 'Scania',
      flag: 'normal',
      serverMessage: 'Welcome',
      eventMessage: 'Event',
      whyAmIRecommended: 'Popular'
    }]
  }
};

const mockTemplateAttributes: TemplateAttributes = mockTemplate.attributes;

// Test wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
  
  TestWrapper.displayName = 'TestWrapper';
  
  return TestWrapper;
}

describe('useTemplates hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Keys', () => {
    it('should generate correct query keys', () => {
      expect(templateKeys.all).toEqual(['templates']);
      expect(templateKeys.lists()).toEqual(['templates', 'list']);
      expect(templateKeys.list({ page: 1 })).toEqual(['templates', 'list', { page: 1 }]);
      expect(templateKeys.detail('1')).toEqual(['templates', 'detail', '1']);
      expect(templateKeys.byRegion('GMS')).toEqual(['templates', 'region', 'GMS', undefined]);
      expect(templateKeys.byVersion(83, 1)).toEqual(['templates', 'version', 83, 1, undefined]);
      expect(templateKeys.validation('1')).toEqual(['templates', 'validation', '1']);
    });
  });

  describe('useTemplates', () => {
    it('should fetch all templates', async () => {
      const mockTemplates = [mockTemplate];
      mockTemplatesService.getAll.mockResolvedValue(mockTemplates);

      const { result } = renderHook(() => useTemplates(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplates);
      expect(mockTemplatesService.getAll).toHaveBeenCalledWith(undefined);
    });

    it('should pass options to service', async () => {
      const options = { page: 1, limit: 10 };
      mockTemplatesService.getAll.mockResolvedValue([]);

      const { result } = renderHook(() => useTemplates(options), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockTemplatesService.getAll).toHaveBeenCalledWith(options);
    });
  });

  describe('useTemplate', () => {
    it('should fetch template by ID', async () => {
      mockTemplatesService.getById.mockResolvedValue(mockTemplate);

      const { result } = renderHook(() => useTemplate('1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplate);
      expect(mockTemplatesService.getById).toHaveBeenCalledWith('1', undefined);
    });

    it('should not fetch when ID is empty', () => {
      renderHook(() => useTemplate(''), {
        wrapper: createWrapper(),
      });

      // The main assertion is that the service wasn't called due to enabled: false
      expect(mockTemplatesService.getById).not.toHaveBeenCalled();
    });
  });

  describe('useTemplateExists', () => {
    it('should check if template exists', async () => {
      mockTemplatesService.exists.mockResolvedValue(true);

      const { result } = renderHook(() => useTemplateExists('1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBe(true);
      expect(mockTemplatesService.exists).toHaveBeenCalledWith('1', undefined);
    });
  });

  describe('useCreateTemplate', () => {
    it('should create new template', async () => {
      const queryClient = new QueryClient();
      const invalidateQueriesSpy = jest.spyOn(queryClient, 'invalidateQueries');
      const setQueryDataSpy = jest.spyOn(queryClient, 'setQueryData');

      mockTemplatesService.create.mockResolvedValue(mockTemplate);

      const wrapper = ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );

      const { result } = renderHook(() => useCreateTemplate(), { wrapper });

      result.current.mutate(mockTemplateAttributes);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplate);
      expect(mockTemplatesService.create).toHaveBeenCalledWith(mockTemplateAttributes);
      expect(setQueryDataSpy).toHaveBeenCalledWith(templateKeys.detail('1'), mockTemplate);
      expect(invalidateQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.lists() });
    });
  });

  describe('useUpdateTemplate', () => {
    it('should update template with optimistic updates', async () => {
      const queryClient = new QueryClient();
      const cancelQueriesSpy = jest.spyOn(queryClient, 'cancelQueries');
      const getQueryDataSpy = jest.spyOn(queryClient, 'getQueryData').mockReturnValue(mockTemplate);
      const setQueryDataSpy = jest.spyOn(queryClient, 'setQueryData');

      const updatedTemplate = { ...mockTemplate, attributes: { ...mockTemplate.attributes, region: 'EMS' } };
      mockTemplatesService.update.mockResolvedValue(updatedTemplate);

      const wrapper = ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );

      const { result } = renderHook(() => useUpdateTemplate(), { wrapper });

      result.current.mutate({ id: '1', updates: { region: 'EMS' } });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(cancelQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.detail('1') });
      expect(getQueryDataSpy).toHaveBeenCalledWith(templateKeys.detail('1'));
      expect(setQueryDataSpy).toHaveBeenCalledWith(
        templateKeys.detail('1'),
        expect.objectContaining({
          attributes: expect.objectContaining({ region: 'EMS' })
        })
      );
    });
  });

  describe('useDeleteTemplate', () => {
    it('should delete template with optimistic updates', async () => {
      const queryClient = new QueryClient();
      const cancelQueriesSpy = jest.spyOn(queryClient, 'cancelQueries');
      const removeQueriesSpy = jest.spyOn(queryClient, 'removeQueries');

      mockTemplatesService.delete.mockResolvedValue();

      const wrapper = ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );

      const { result } = renderHook(() => useDeleteTemplate(), { wrapper });

      result.current.mutate({ id: '1' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(cancelQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.detail('1') });
      expect(removeQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.detail('1') });
      expect(mockTemplatesService.delete).toHaveBeenCalledWith('1');
    });
  });

  describe('useTemplatesByRegion', () => {
    it('should fetch templates by region', async () => {
      const mockTemplates = [mockTemplate];
      mockTemplatesService.getByRegion.mockResolvedValue(mockTemplates);

      const { result } = renderHook(() => useTemplatesByRegion('GMS'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplates);
      expect(mockTemplatesService.getByRegion).toHaveBeenCalledWith('GMS', undefined);
    });

    it('should not fetch when region is empty', () => {
      renderHook(() => useTemplatesByRegion(''), {
        wrapper: createWrapper(),
      });

      // The main assertion is that the service wasn't called due to enabled: false
      expect(mockTemplatesService.getByRegion).not.toHaveBeenCalled();
    });
  });

  describe('useTemplatesByVersion', () => {
    it('should fetch templates by version', async () => {
      const mockTemplates = [mockTemplate];
      mockTemplatesService.getByVersion.mockResolvedValue(mockTemplates);

      const { result } = renderHook(() => useTemplatesByVersion(83, 1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplates);
      expect(mockTemplatesService.getByVersion).toHaveBeenCalledWith(83, 1, undefined);
    });
  });

  describe('useTemplatesByRegionAndVersion', () => {
    it('should fetch templates by region and version', async () => {
      const mockTemplates = [mockTemplate];
      mockTemplatesService.getByRegionAndVersion.mockResolvedValue(mockTemplates);

      const { result } = renderHook(() => useTemplatesByRegionAndVersion('GMS', 83, 1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockTemplates);
      expect(mockTemplatesService.getByRegionAndVersion).toHaveBeenCalledWith('GMS', 83, 1, undefined);
    });
  });

  describe('useValidateTemplate', () => {
    it('should validate template consistency', async () => {
      const validationResult = { isValid: true, errors: [] };
      mockTemplatesService.validateTemplateConsistency.mockResolvedValue(validationResult);

      const { result } = renderHook(() => useValidateTemplate('1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(validationResult);
      expect(mockTemplatesService.validateTemplateConsistency).toHaveBeenCalledWith('1');
    });
  });

  describe('useExportTemplates', () => {
    it('should export templates', async () => {
      const mockBlob = new Blob(['test'], { type: 'application/json' });
      mockTemplatesService.export.mockResolvedValue(mockBlob);

      const { result } = renderHook(() => useExportTemplates(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ format: 'json' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockBlob);
      expect(mockTemplatesService.export).toHaveBeenCalledWith('json', undefined);
    });
  });

  describe('useCloneTemplate', () => {
    it('should clone template attributes', () => {
      const clonedAttributes: TemplateAttributes = {
        ...mockTemplateAttributes,
        region: '',
        majorVersion: 0,
        minorVersion: 0
      };
      mockTemplatesService.cloneTemplate.mockReturnValue(clonedAttributes);

      const { result } = renderHook(() => useCloneTemplate(), {
        wrapper: createWrapper(),
      });

      const cloned = result.current(mockTemplate);

      expect(cloned).toEqual(clonedAttributes);
      expect(mockTemplatesService.cloneTemplate).toHaveBeenCalledWith(mockTemplate);
    });
  });

  describe('Batch Operations', () => {
    describe('useCreateTemplatesBatch', () => {
      it('should create templates in batch', async () => {
        const batchResult = { successes: [mockTemplate], failures: [] };
        mockTemplatesService.createBatch.mockResolvedValue(batchResult);

        const { result } = renderHook(() => useCreateTemplatesBatch(), {
          wrapper: createWrapper(),
        });

        result.current.mutate({ templates: [mockTemplateAttributes] });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(batchResult);
        expect(mockTemplatesService.createBatch).toHaveBeenCalledWith([mockTemplateAttributes], undefined);
      });
    });

    describe('useUpdateTemplatesBatch', () => {
      it('should update templates in batch', async () => {
        const batchResult = { successes: [mockTemplate], failures: [] };
        mockTemplatesService.updateBatch.mockResolvedValue(batchResult);

        const { result } = renderHook(() => useUpdateTemplatesBatch(), {
          wrapper: createWrapper(),
        });

        const updates = [{ id: '1', data: { region: 'EMS' } }];
        result.current.mutate({ updates });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(batchResult);
        expect(mockTemplatesService.updateBatch).toHaveBeenCalledWith(updates, undefined);
      });
    });

    describe('useDeleteTemplatesBatch', () => {
      it('should delete templates in batch', async () => {
        const batchResult = { successes: ['1'], failures: [] };
        mockTemplatesService.deleteBatch.mockResolvedValue(batchResult);

        const { result } = renderHook(() => useDeleteTemplatesBatch(), {
          wrapper: createWrapper(),
        });

        result.current.mutate({ ids: ['1'] });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(batchResult);
        expect(mockTemplatesService.deleteBatch).toHaveBeenCalledWith(['1'], undefined);
      });
    });
  });

  describe('Cache Management', () => {
    describe('useInvalidateTemplates', () => {
      it('should provide cache invalidation functions', () => {
        const queryClient = new QueryClient();
        const invalidateQueriesSpy = jest.spyOn(queryClient, 'invalidateQueries');

        const wrapper = ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={queryClient}>
            {children}
          </QueryClientProvider>
        );

        const { result } = renderHook(() => useInvalidateTemplates(), { wrapper });

        result.current.invalidateAll();
        result.current.invalidateTemplate('1');
        result.current.invalidateByRegion('GMS');

        expect(invalidateQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.all });
        expect(invalidateQueriesSpy).toHaveBeenCalledWith({ queryKey: templateKeys.detail('1') });
        expect(invalidateQueriesSpy).toHaveBeenCalledWith({ 
          queryKey: [...templateKeys.all, 'region', 'GMS'] 
        });
      });
    });

    describe('useTemplateCacheStats', () => {
      it('should fetch cache statistics', async () => {
        const cacheStats = { hits: 10, misses: 2, size: 5 };
        mockTemplatesService.getCacheStats.mockReturnValue(cacheStats);

        const { result } = renderHook(() => useTemplateCacheStats(), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(cacheStats);
        expect(mockTemplatesService.getCacheStats).toHaveBeenCalled();
      });
    });
  });
});