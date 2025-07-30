/**
 * Tests for useTenants React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';

import {
  useTenants,
  useTenant,
  useCreateTenant,
  useUpdateTenant,
  useDeleteTenant,
  useTenantConfigurations,
  useTenantConfiguration,
  useCreateTenantConfiguration,
  useUpdateTenantConfiguration,
  useCreateFromTemplate,
  tenantKeys,
  type TenantBasic,
  type TenantBasicAttributes,
  type TenantConfig,
  type TenantConfigAttributes,
} from '../useTenants';

// Mock the tenants service
jest.mock('@/services/api/tenants.service', () => ({
  tenantsService: {
    getAllTenants: jest.fn(),
    getTenantById: jest.fn(),
    createTenant: jest.fn(),
    updateTenant: jest.fn(),
    deleteTenant: jest.fn(),
    getAllTenantConfigurations: jest.fn(),
    getTenantConfigurationById: jest.fn(),
    createTenantConfiguration: jest.fn(),
    updateTenantConfiguration: jest.fn(),
    createTenantFromTemplate: jest.fn(),
  },
}));

import { tenantsService } from '@/services/api/tenants.service';

// Test data
const mockTenant: TenantBasic = {
  id: 'test-tenant-1',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

const mockTenantAttributes: TenantBasicAttributes = {
  name: 'New Tenant',
  region: 'EMS',
  majorVersion: 84,
  minorVersion: 0,
};

const mockTenantConfig: TenantConfig = {
  id: 'test-config-1',
  attributes: {
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [] },
    npcs: [],
    socket: { handlers: [], writers: [] },
    worlds: [],
  },
};

// Test wrapper
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useTenants hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query key factories', () => {
    it('should generate correct query keys', () => {
      expect(tenantKeys.all).toEqual(['tenants']);
      expect(tenantKeys.basic()).toEqual(['tenants', 'basic']);
      expect(tenantKeys.basicDetail('123')).toEqual(['tenants', 'basic', 'detail', '123']);
      expect(tenantKeys.configs()).toEqual(['tenants', 'configs']);
      expect(tenantKeys.configDetail('456')).toEqual(['tenants', 'configs', 'detail', '456']);
    });
  });

  describe('Basic tenant hooks', () => {
    describe('useTenants', () => {
      it('should fetch all tenants successfully', async () => {
        const mockTenants = [mockTenant];
        (tenantsService.getAllTenants as jest.Mock).mockResolvedValue(mockTenants);

        const { result } = renderHook(() => useTenants(), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockTenants);
        expect(tenantsService.getAllTenants).toHaveBeenCalledWith(undefined);
      });

      it('should pass options to service', async () => {
        const options = { signal: new AbortController().signal };
        (tenantsService.getAllTenants as jest.Mock).mockResolvedValue([]);

        renderHook(() => useTenants(options), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(tenantsService.getAllTenants).toHaveBeenCalledWith(options);
        });
      });
    });

    describe('useTenant', () => {
      it('should fetch tenant by ID successfully', async () => {
        (tenantsService.getTenantById as jest.Mock).mockResolvedValue(mockTenant);

        const { result } = renderHook(() => useTenant('test-tenant-1'), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockTenant);
        expect(tenantsService.getTenantById).toHaveBeenCalledWith('test-tenant-1', undefined);
      });

      it('should not run query when ID is empty', () => {
        const { result } = renderHook(() => useTenant(''), {
          wrapper: createWrapper(),
        });

        expect(result.current.isFetching).toBe(false);
        expect(tenantsService.getTenantById).not.toHaveBeenCalled();
      });
    });

    describe('useCreateTenant', () => {
      it('should create tenant successfully', async () => {
        (tenantsService.createTenant as jest.Mock).mockResolvedValue(mockTenant);

        const { result } = renderHook(() => useCreateTenant(), {
          wrapper: createWrapper(),
        });

        result.current.mutate(mockTenantAttributes);

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockTenant);
        expect(tenantsService.createTenant).toHaveBeenCalledWith(mockTenantAttributes);
      });

      it('should handle creation errors', async () => {
        const error = new Error('Creation failed');
        (tenantsService.createTenant as jest.Mock).mockRejectedValue(error);
        const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

        const { result } = renderHook(() => useCreateTenant(), {
          wrapper: createWrapper(),
        });

        result.current.mutate(mockTenantAttributes);

        await waitFor(() => {
          expect(result.current.isError).toBe(true);
        });

        expect(result.current.error).toEqual(error);
        expect(consoleSpy).toHaveBeenCalledWith('Failed to create tenant:', error);
        
        consoleSpy.mockRestore();
      });
    });

    describe('useUpdateTenant', () => {
      it('should update tenant successfully', async () => {
        const updatedTenant = { ...mockTenant, attributes: { ...mockTenant.attributes, name: 'Updated' } };
        (tenantsService.updateTenant as jest.Mock).mockResolvedValue(updatedTenant);

        const { result } = renderHook(() => useUpdateTenant(), {
          wrapper: createWrapper(),
        });

        const updates = { name: 'Updated' };
        result.current.mutate({ tenant: mockTenant, updates });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(updatedTenant);
        expect(tenantsService.updateTenant).toHaveBeenCalledWith(mockTenant, updates);
      });
    });

    describe('useDeleteTenant', () => {
      it('should delete tenant successfully', async () => {
        (tenantsService.deleteTenant as jest.Mock).mockResolvedValue(undefined);

        const { result } = renderHook(() => useDeleteTenant(), {
          wrapper: createWrapper(),
        });

        result.current.mutate({ id: 'test-tenant-1' });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(tenantsService.deleteTenant).toHaveBeenCalledWith('test-tenant-1');
      });
    });
  });

  describe('Tenant configuration hooks', () => {
    describe('useTenantConfigurations', () => {
      it('should fetch all configurations successfully', async () => {
        const mockConfigs = [mockTenantConfig];
        (tenantsService.getAllTenantConfigurations as jest.Mock).mockResolvedValue(mockConfigs);

        const { result } = renderHook(() => useTenantConfigurations(), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockConfigs);
        expect(tenantsService.getAllTenantConfigurations).toHaveBeenCalledWith(undefined);
      });
    });

    describe('useTenantConfiguration', () => {
      it('should fetch configuration by ID successfully', async () => {
        (tenantsService.getTenantConfigurationById as jest.Mock).mockResolvedValue(mockTenantConfig);

        const { result } = renderHook(() => useTenantConfiguration('test-config-1'), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockTenantConfig);
        expect(tenantsService.getTenantConfigurationById).toHaveBeenCalledWith('test-config-1', undefined);
      });
    });

    describe('useCreateTenantConfiguration', () => {
      it('should create configuration successfully', async () => {
        (tenantsService.createTenantConfiguration as jest.Mock).mockResolvedValue(mockTenantConfig);

        const { result } = renderHook(() => useCreateTenantConfiguration(), {
          wrapper: createWrapper(),
        });

        result.current.mutate(mockTenantConfig.attributes);

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(mockTenantConfig);
        expect(tenantsService.createTenantConfiguration).toHaveBeenCalledWith(mockTenantConfig.attributes);
      });
    });

    describe('useUpdateTenantConfiguration', () => {
      it('should update configuration successfully', async () => {
        const updatedConfig = { ...mockTenantConfig, attributes: { ...mockTenantConfig.attributes, usesPin: true } };
        (tenantsService.updateTenantConfiguration as jest.Mock).mockResolvedValue(updatedConfig);

        const { result } = renderHook(() => useUpdateTenantConfiguration(), {
          wrapper: createWrapper(),
        });

        const updates = { usesPin: true };
        result.current.mutate({ tenant: mockTenantConfig, updates });

        await waitFor(() => {
          expect(result.current.isSuccess).toBe(true);
        });

        expect(result.current.data).toEqual(updatedConfig);
        expect(tenantsService.updateTenantConfiguration).toHaveBeenCalledWith(mockTenantConfig, updates);
      });
    });
  });

  describe('Utility hooks', () => {
    describe('useCreateFromTemplate', () => {
      it('should create from template', () => {
        const templateAttributes = { ...mockTenantConfig.attributes };
        (tenantsService.createTenantFromTemplate as jest.Mock).mockReturnValue(templateAttributes);

        const { result } = renderHook(() => useCreateFromTemplate());

        const template = { attributes: mockTenantConfig.attributes };
        const createdAttributes = result.current(template);

        expect(createdAttributes).toEqual(templateAttributes);
        expect(tenantsService.createTenantFromTemplate).toHaveBeenCalledWith(template);
      });
    });
  });
});