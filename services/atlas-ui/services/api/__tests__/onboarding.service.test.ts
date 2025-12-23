/**
 * @jest-environment jsdom
 */

import {
  onboardingService,
  TemplateNotFoundError,
  TenantCreationError,
  ConfigurationCreationError,
} from '../onboarding.service';
import { tenantsService } from '../tenants.service';
import { templatesService } from '../templates.service';
import type { Template } from '@/types/models/template';

// Mock the tenants service
jest.mock('../tenants.service', () => ({
  tenantsService: {
    createTenant: jest.fn(),
    createTenantConfiguration: jest.fn(),
  },
}));

// Mock the templates service
jest.mock('../templates.service', () => ({
  templatesService: {
    getByRegionAndVersion: jest.fn(),
  },
}));

const mockTenantsService = tenantsService as jest.Mocked<typeof tenantsService>;
const mockTemplatesService = templatesService as jest.Mocked<typeof templatesService>;

describe('OnboardingService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const mockTemplate: Template = {
    id: 'template-123',
    attributes: {
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      usesPin: true,
      characters: {
        templates: [],
      },
      npcs: [],
      socket: {
        handlers: [],
        writers: [],
      },
      worlds: [],
    },
  };

  const mockTenant = {
    id: 'tenant-456',
    attributes: {
      name: 'Test Tenant',
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
    },
  };

  const mockTenantConfig = {
    id: 'tenant-456',
    attributes: {
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      usesPin: true,
      characters: { templates: [] },
      npcs: [],
      socket: { handlers: [], writers: [] },
      worlds: [],
    },
  };

  describe('onboardTenant', () => {
    it('should create tenant and configuration successfully', async () => {
      mockTenantsService.createTenant.mockResolvedValue(mockTenant);
      mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

      const result = await onboardingService.onboardTenant('Test Tenant', mockTemplate);

      expect(mockTenantsService.createTenant).toHaveBeenCalledWith({
        name: 'Test Tenant',
        region: 'GMS',
        majorVersion: 83,
        minorVersion: 1,
      });

      expect(mockTenantsService.createTenantConfiguration).toHaveBeenCalledWith(
        'tenant-456', // Tenant ID is passed first
        {
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          usesPin: true,
          characters: { templates: [] },
          npcs: [],
          socket: { handlers: [], writers: [] },
          worlds: [],
        }
      );

      expect(result.tenant).toEqual(mockTenant);
      expect(result.config).toEqual(mockTenantConfig);
    });

    it('should throw TenantCreationError when tenant creation fails', async () => {
      const error = new Error('Network error');
      mockTenantsService.createTenant.mockRejectedValue(error);

      await expect(
        onboardingService.onboardTenant('Test Tenant', mockTemplate)
      ).rejects.toThrow(TenantCreationError);

      expect(mockTenantsService.createTenantConfiguration).not.toHaveBeenCalled();
    });

    it('should throw ConfigurationCreationError when config creation fails', async () => {
      mockTenantsService.createTenant.mockResolvedValue(mockTenant);
      mockTenantsService.createTenantConfiguration.mockRejectedValue(new Error('Config error'));

      try {
        await onboardingService.onboardTenant('Test Tenant', mockTemplate);
        fail('Expected ConfigurationCreationError to be thrown');
      } catch (error) {
        expect(error).toBeInstanceOf(ConfigurationCreationError);
        expect((error as ConfigurationCreationError).tenantId).toBe('tenant-456');
        expect((error as ConfigurationCreationError).message).toContain('tenant-456');
      }
    });
  });

  describe('onboardTenantByVersion', () => {
    it('should fetch template and create tenant successfully', async () => {
      mockTemplatesService.getByRegionAndVersion.mockResolvedValue([mockTemplate]);
      mockTenantsService.createTenant.mockResolvedValue(mockTenant);
      mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

      const result = await onboardingService.onboardTenantByVersion(
        'Test Tenant',
        'GMS',
        83,
        1
      );

      expect(mockTemplatesService.getByRegionAndVersion).toHaveBeenCalledWith('GMS', 83, 1);
      expect(result.tenant).toEqual(mockTenant);
      expect(result.config).toEqual(mockTenantConfig);
    });

    it('should throw TemplateNotFoundError when no template matches', async () => {
      mockTemplatesService.getByRegionAndVersion.mockResolvedValue([]);

      await expect(
        onboardingService.onboardTenantByVersion('Test Tenant', 'GMS', 999, 1)
      ).rejects.toThrow(TemplateNotFoundError);

      expect(mockTenantsService.createTenant).not.toHaveBeenCalled();
    });

    it('should use first matching template when multiple exist', async () => {
      const template2: Template = {
        ...mockTemplate,
        id: 'template-789',
      };
      mockTemplatesService.getByRegionAndVersion.mockResolvedValue([mockTemplate, template2]);
      mockTenantsService.createTenant.mockResolvedValue(mockTenant);
      mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

      await onboardingService.onboardTenantByVersion('Test Tenant', 'GMS', 83, 1);

      // Should use the first template
      expect(mockTenantsService.createTenant).toHaveBeenCalledWith(
        expect.objectContaining({
          region: mockTemplate.attributes.region,
        })
      );
    });
  });

  describe('Error classes', () => {
    describe('TemplateNotFoundError', () => {
      it('should have correct name and message', () => {
        const error = new TemplateNotFoundError('GMS', 83, 1);

        expect(error.name).toBe('TemplateNotFoundError');
        expect(error.message).toContain('GMS');
        expect(error.message).toContain('83');
        expect(error.message).toContain('1');
      });
    });

    describe('TenantCreationError', () => {
      it('should store original error', () => {
        const originalError = new Error('Original');
        const error = new TenantCreationError('Failed', originalError);

        expect(error.name).toBe('TenantCreationError');
        expect(error.originalError).toBe(originalError);
      });
    });

    describe('ConfigurationCreationError', () => {
      it('should store tenant ID and original error', () => {
        const originalError = new Error('Original');
        const error = new ConfigurationCreationError('Failed', 'tenant-123', originalError);

        expect(error.name).toBe('ConfigurationCreationError');
        expect(error.tenantId).toBe('tenant-123');
        expect(error.originalError).toBe(originalError);
      });
    });
  });
});
