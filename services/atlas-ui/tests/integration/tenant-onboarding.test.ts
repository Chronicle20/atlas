/**
 * @jest-environment jsdom
 *
 * Integration tests for Tenant Onboarding Flow
 * Tests the complete flow from template selection to tenant creation
 */

import {
  onboardingService,
  TemplateNotFoundError,
  TenantCreationError,
  ConfigurationCreationError,
} from '@/services/api/onboarding.service';
import { tenantsService } from '@/services/api/tenants.service';
import { templatesService } from '@/services/api/templates.service';
import type { Template } from '@/types/models/template';

// Mock the services
jest.mock('@/services/api/tenants.service', () => ({
  tenantsService: {
    createTenant: jest.fn(),
    createTenantConfiguration: jest.fn(),
  },
}));

jest.mock('@/services/api/templates.service', () => ({
  templatesService: {
    getByRegionAndVersion: jest.fn(),
    getTemplateOptions: jest.fn(),
  },
}));

const mockTenantsService = tenantsService as jest.Mocked<typeof tenantsService>;
const mockTemplatesService = templatesService as jest.Mocked<typeof templatesService>;

describe('Tenant Onboarding Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const mockTemplate: Template = {
    id: 'template-gms-83',
    attributes: {
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
      usesPin: true,
      characters: {
        templates: [
          {
            jobIndex: 0,
            subJobIndex: 0,
            gender: 0,
            mapId: 10000,
            faces: [20000, 20001],
            hairs: [30000, 30001],
            hairColors: [0, 1],
            skinColors: [0, 1],
            tops: [1040000],
            bottoms: [1060000],
            shoes: [1070000],
            weapons: [1302000],
            items: [],
            skills: [],
          },
        ],
      },
      npcs: [
        { npcId: 9010000, impl: 'MapleTV' },
        { npcId: 9201000, impl: 'JobAdvancement' },
      ],
      socket: {
        handlers: [
          { opCode: '0x01', validator: 'NoOpValidator', handler: 'LoginHandle', options: {} },
        ],
        writers: [
          { opCode: '0x01', writer: 'LoginWriter', options: {} },
        ],
      },
      worlds: [
        { name: 'Scania', flag: 'normal', serverMessage: 'Welcome!', eventMessage: '', whyAmIRecommended: 'Original world' },
      ],
    },
  };

  const mockTenant = {
    id: 'tenant-new-123',
    attributes: {
      name: 'My New Tenant',
      region: 'GMS',
      majorVersion: 83,
      minorVersion: 1,
    },
  };

  const mockTenantConfig = {
    id: 'tenant-new-123',
    attributes: {
      ...mockTemplate.attributes,
    },
  };

  describe('Full Onboarding Flow', () => {
    describe('from template grid (template already loaded)', () => {
      it('should create tenant and configuration in sequence', async () => {
        mockTenantsService.createTenant.mockResolvedValue(mockTenant);
        mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

        const result = await onboardingService.onboardTenant('My New Tenant', mockTemplate);

        // Verify sequence: tenant created first
        expect(mockTenantsService.createTenant).toHaveBeenCalledWith({
          name: 'My New Tenant',
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
        });

        // Then configuration created with full template data
        expect(mockTenantsService.createTenantConfiguration).toHaveBeenCalledWith(
          expect.objectContaining({
            region: 'GMS',
            majorVersion: 83,
            minorVersion: 1,
            usesPin: true,
            socket: expect.objectContaining({
              handlers: expect.any(Array),
              writers: expect.any(Array),
            }),
            npcs: expect.any(Array),
            worlds: expect.any(Array),
          })
        );

        // Verify result contains both
        expect(result.tenant.id).toBe('tenant-new-123');
        expect(result.config.id).toBe('tenant-new-123');
      });

      it('should include character templates in configuration', async () => {
        mockTenantsService.createTenant.mockResolvedValue(mockTenant);
        mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

        await onboardingService.onboardTenant('My New Tenant', mockTemplate);

        expect(mockTenantsService.createTenantConfiguration).toHaveBeenCalledWith(
          expect.objectContaining({
            characters: expect.objectContaining({
              templates: expect.arrayContaining([
                expect.objectContaining({
                  jobIndex: 0,
                  mapId: 10000,
                }),
              ]),
            }),
          })
        );
      });
    });

    describe('from tenant switcher (need to fetch template)', () => {
      it('should fetch template then create tenant and configuration', async () => {
        mockTemplatesService.getByRegionAndVersion.mockResolvedValue([mockTemplate]);
        mockTenantsService.createTenant.mockResolvedValue(mockTenant);
        mockTenantsService.createTenantConfiguration.mockResolvedValue(mockTenantConfig);

        const result = await onboardingService.onboardTenantByVersion(
          'My New Tenant',
          'GMS',
          83,
          1
        );

        // Verify template fetch
        expect(mockTemplatesService.getByRegionAndVersion).toHaveBeenCalledWith('GMS', 83, 1);

        // Verify tenant creation
        expect(mockTenantsService.createTenant).toHaveBeenCalled();
        expect(mockTenantsService.createTenantConfiguration).toHaveBeenCalled();

        expect(result.tenant.id).toBe('tenant-new-123');
      });
    });
  });

  describe('Error Recovery Scenarios', () => {
    describe('template not found', () => {
      it('should throw TemplateNotFoundError without creating tenant', async () => {
        mockTemplatesService.getByRegionAndVersion.mockResolvedValue([]);

        await expect(
          onboardingService.onboardTenantByVersion('My Tenant', 'INVALID', 999, 1)
        ).rejects.toThrow(TemplateNotFoundError);

        // Tenant should NOT be created
        expect(mockTenantsService.createTenant).not.toHaveBeenCalled();
        expect(mockTenantsService.createTenantConfiguration).not.toHaveBeenCalled();
      });
    });

    describe('tenant creation failure', () => {
      it('should throw TenantCreationError without creating configuration', async () => {
        mockTenantsService.createTenant.mockRejectedValue(new Error('Database error'));

        await expect(
          onboardingService.onboardTenant('My Tenant', mockTemplate)
        ).rejects.toThrow(TenantCreationError);

        // Configuration should NOT be created
        expect(mockTenantsService.createTenantConfiguration).not.toHaveBeenCalled();
      });
    });

    describe('configuration creation failure (partial failure)', () => {
      it('should throw ConfigurationCreationError with tenant ID for recovery', async () => {
        mockTenantsService.createTenant.mockResolvedValue(mockTenant);
        mockTenantsService.createTenantConfiguration.mockRejectedValue(
          new Error('Configuration validation failed')
        );

        try {
          await onboardingService.onboardTenant('My Tenant', mockTemplate);
          fail('Expected ConfigurationCreationError to be thrown');
        } catch (error) {
          expect(error).toBeInstanceOf(ConfigurationCreationError);

          // Should include tenant ID for manual recovery
          const configError = error as ConfigurationCreationError;
          expect(configError.tenantId).toBe('tenant-new-123');
          expect(configError.message).toContain('tenant-new-123');
        }
      });
    });
  });

  describe('Template Options (Sparse Fieldsets)', () => {
    it('should fetch lightweight template options for dropdowns', async () => {
      const mockOptions = [
        { id: '1', attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } },
        { id: '2', attributes: { region: 'GMS', majorVersion: 95, minorVersion: 1 } },
        { id: '3', attributes: { region: 'JMS', majorVersion: 185, minorVersion: 1 } },
      ];
      mockTemplatesService.getTemplateOptions.mockResolvedValue(mockOptions);

      const options = await templatesService.getTemplateOptions();

      expect(options).toHaveLength(3);
      expect(options[0].attributes).not.toHaveProperty('socket');
      expect(options[0].attributes).not.toHaveProperty('npcs');
      expect(options[0].attributes).toHaveProperty('region');
      expect(options[0].attributes).toHaveProperty('majorVersion');
    });
  });

  describe('Data Integrity', () => {
    it('should preserve all template data in configuration', async () => {
      const fullTemplate: Template = {
        id: 'full-template',
        attributes: {
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          usesPin: false,
          characters: {
            templates: [
              {
                jobIndex: 1,
                subJobIndex: 0,
                gender: 1,
                mapId: 20000,
                faces: [21000],
                hairs: [31000],
                hairColors: [2],
                skinColors: [3],
                tops: [1041000],
                bottoms: [1061000],
                shoes: [1071000],
                weapons: [1302001],
                items: [2000000],
                skills: [1000001],
              },
            ],
          },
          npcs: [
            { npcId: 1234567, impl: 'CustomNpc' },
          ],
          socket: {
            handlers: [
              { opCode: '0xFF', validator: 'CustomValidator', handler: 'CustomHandler', options: { key: 'value' } },
            ],
            writers: [
              { opCode: '0xFE', writer: 'CustomWriter', options: {} },
            ],
          },
          worlds: [
            { name: 'CustomWorld', flag: 'event', serverMessage: 'Custom!', eventMessage: 'Event!', whyAmIRecommended: 'Special' },
          ],
        },
      };

      mockTenantsService.createTenant.mockResolvedValue(mockTenant);
      mockTenantsService.createTenantConfiguration.mockResolvedValue({
        id: 'tenant-new-123',
        attributes: fullTemplate.attributes,
      });

      await onboardingService.onboardTenant('Data Test', fullTemplate);

      const configCall = mockTenantsService.createTenantConfiguration.mock.calls[0][0];

      // Verify all data is preserved
      expect(configCall.usesPin).toBe(false);
      expect(configCall.characters.templates[0].jobIndex).toBe(1);
      expect(configCall.characters.templates[0].skills).toEqual([1000001]);
      expect(configCall.npcs[0].impl).toBe('CustomNpc');
      expect(configCall.socket.handlers[0].options).toEqual({ key: 'value' });
      expect(configCall.worlds[0].flag).toBe('event');
    });
  });
});
