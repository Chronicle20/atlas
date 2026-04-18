/**
 * @jest-environment jsdom
 */

import { templatesService } from './templates.service';
import { api } from '@/lib/api/client';
import type { Template, TemplateAttributes } from '@/types/models/template';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  api: {
    getList: jest.fn(),
    getOne: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
    upload: jest.fn(),
    download: jest.fn(),
    clearCacheByPattern: jest.fn(),
    getCacheStats: jest.fn(),
  },
}));

const mockApi = api as jest.Mocked<typeof api>;

describe('TemplatesService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const mockTemplateAttributes: TemplateAttributes = {
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: {
      templates: [
        {
          jobIndex: 0,
          subJobIndex: 0,
          gender: 0,
          mapId: 100000000,
          faces: [20000, 20001],
          hairs: [30000, 30001],
          hairColors: [0, 1],
          skinColors: [0, 1],
          tops: [1040000, 1040001],
          bottoms: [1060000, 1060001],
          shoes: [1072000, 1072001],
          weapons: [1302000, 1302001],
          items: [2000000, 2000001],
          skills: [1000, 1001],
        }
      ]
    },
    npcs: [
      {
        npcId: 9000000,
        impl: 'TestNpc'
      }
    ],
    socket: {
      handlers: [
        {
          opCode: '0x01',
          validator: 'TestValidator',
          handler: 'TestHandler',
          options: {}
        }
      ],
      writers: [
        {
          opCode: '0x02',
          writer: 'TestWriter',
          options: {}
        }
      ]
    },
    worlds: [
      {
        name: 'Scania',
        flag: 'scania',
        serverMessage: 'Welcome to Scania!',
        eventMessage: 'Event active!',
        whyAmIRecommended: 'Popular server'
      }
    ]
  };

  const mockTemplate: Template = {
    id: 'template-1',
    attributes: mockTemplateAttributes
  };

  const mockApiResponseData = [
    {
      id: 'template-1',
      attributes: {
        ...mockTemplateAttributes,
        socket: {
          handlers: [
            { opCode: '0x0A', validator: 'TestValidator2', handler: 'TestHandler2', options: {} },
            { opCode: '0x01', validator: 'TestValidator', handler: 'TestHandler', options: {} }
          ],
          writers: [
            { opCode: '0x0B', writer: 'TestWriter2', options: {} },
            { opCode: '0x02', writer: 'TestWriter', options: {} }
          ]
        }
      }
    }
  ];

  describe('getAll', () => {
    it('should fetch and sort templates correctly', async () => {
      mockApi.getList.mockResolvedValue(mockApiResponseData);

      const result = await templatesService.getAll();

      expect(mockApi.getList).toHaveBeenCalledWith('/api/configurations/templates', expect.any(Object));
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('template-1');
      
      // Check that handlers and writers are sorted by opCode
      expect(result[0].attributes.socket.handlers[0].opCode).toBe('0x01');
      expect(result[0].attributes.socket.handlers[1].opCode).toBe('0x0A');
      expect(result[0].attributes.socket.writers[0].opCode).toBe('0x02');
      expect(result[0].attributes.socket.writers[1].opCode).toBe('0x0B');
    });

    it('should handle query options', async () => {
      mockApi.getList.mockResolvedValue([]);

      await templatesService.getAll({ 
        search: 'test',
        filters: { region: 'GMS' }
      });

      expect(mockApi.getList).toHaveBeenCalledWith(
        '/api/configurations/templates?search=test&filter%5Bregion%5D=GMS',
        expect.any(Object)
      );
    });
  });

  describe('getById', () => {
    it('should fetch single template and apply transformations', async () => {
      mockApi.getOne.mockResolvedValue(mockApiResponseData[0]);

      const result = await templatesService.getById('template-1');

      expect(mockApi.getOne).toHaveBeenCalledWith('/api/configurations/templates/template-1', expect.any(Object));
      expect(result.id).toBe('template-1');
      expect(result.attributes.socket.handlers).toHaveLength(2);
    });
  });

  describe('create', () => {
    it('should create template with validation', async () => {
      mockApi.post.mockResolvedValue(mockTemplate);

      const result = await templatesService.create(mockTemplateAttributes);

      expect(mockApi.post).toHaveBeenCalledWith(
        '/api/configurations/templates',
        {
          data: {
            type: 'templates',
            attributes: mockTemplateAttributes
          }
        },
        expect.objectContaining({ validate: true })
      );
      expect(result).toEqual(mockTemplate);
    });

    it('should fail validation for invalid data', async () => {
      const invalidAttributes = {
        ...mockTemplateAttributes,
        region: '', // Invalid - empty region
        majorVersion: -1, // Invalid - negative version
      };

      await expect(templatesService.create(invalidAttributes)).rejects.toThrow('Validation failed');
    });

    it('should validate character template structure', async () => {
      const invalidAttributes = {
        ...mockTemplateAttributes,
        characters: {
          templates: [
            {
              ...mockTemplateAttributes.characters.templates[0],
              jobIndex: -1, // Invalid - negative job index
              gender: 2, // Invalid - must be 0 or 1
            }
          ]
        }
      };

      await expect(templatesService.create(invalidAttributes)).rejects.toThrow('Validation failed');
    });

    it('should validate NPC structure', async () => {
      const invalidAttributes = {
        ...mockTemplateAttributes,
        npcs: [
          {
            npcId: 0, // Invalid - must be positive
            impl: '', // Invalid - empty string
          }
        ]
      };

      await expect(templatesService.create(invalidAttributes)).rejects.toThrow('Validation failed');
    });

    it('should validate socket handlers structure', async () => {
      const invalidAttributes = {
        ...mockTemplateAttributes,
        socket: {
          ...mockTemplateAttributes.socket,
          handlers: [
            {
              opCode: '', // Invalid - empty string
              validator: '',
              handler: '',
              options: {}
            }
          ]
        }
      };

      await expect(templatesService.create(invalidAttributes)).rejects.toThrow('Validation failed');
    });
  });

  describe('update', () => {
    it('should update template with validation', async () => {
      const updates = { region: 'EMS', majorVersion: 84 };
      mockApi.put.mockResolvedValue({ ...mockTemplate, attributes: { ...mockTemplate.attributes, ...updates } });

      const result = await templatesService.update('template-1', updates, { validate: false });

      expect(mockApi.put).toHaveBeenCalledWith(
        '/api/configurations/templates/template-1',
        {
          data: {
            type: 'templates',
            attributes: updates
          }
        },
        expect.objectContaining({ validate: false })
      );
      expect(result.attributes.region).toBe('EMS');
      expect(result.attributes.majorVersion).toBe(84);
    });
  });

  describe('delete', () => {
    it('should delete template', async () => {
      mockApi.delete.mockResolvedValue(undefined);

      await templatesService.delete('template-1');

      expect(mockApi.delete).toHaveBeenCalledWith('/api/configurations/templates/template-1', expect.any(Object));
    });
  });

  describe('cloneTemplate', () => {
    it('should clone template and reset version fields', () => {
      const result = templatesService.cloneTemplate(mockTemplate);

      expect(result).toEqual({
        ...mockTemplateAttributes,
        region: '',
        majorVersion: 0,
        minorVersion: 0,
      });
    });

    it('should create deep copy without affecting original', () => {
      const result = templatesService.cloneTemplate(mockTemplate);
      
      // Modify the cloned result
      result.characters.templates[0].jobIndex = 999;
      
      // Original should be unchanged
      expect(mockTemplate.attributes.characters.templates[0].jobIndex).toBe(0);
    });
  });

  describe('getByRegion', () => {
    it('should filter templates by region', async () => {
      mockApi.getList.mockResolvedValue([]);

      await templatesService.getByRegion('GMS');

      expect(mockApi.getList).toHaveBeenCalledWith(
        '/api/configurations/templates?filter%5Bregion%5D=GMS',
        expect.any(Object)
      );
    });
  });

  describe('getByVersion', () => {
    it('should filter templates by major version only', async () => {
      mockApi.getList.mockResolvedValue([]);

      await templatesService.getByVersion(83);

      expect(mockApi.getList).toHaveBeenCalledWith(
        '/api/configurations/templates?filter%5BmajorVersion%5D=83',
        expect.any(Object)
      );
    });

    it('should filter templates by major and minor version', async () => {
      mockApi.getList.mockResolvedValue([]);

      await templatesService.getByVersion(83, 1);

      expect(mockApi.getList).toHaveBeenCalledWith(
        '/api/configurations/templates?filter%5BmajorVersion%5D=83&filter%5BminorVersion%5D=1',
        expect.any(Object)
      );
    });
  });

  describe('validateTemplateConsistency', () => {
    it('should validate template without issues', async () => {
      mockApi.getOne.mockResolvedValue(mockTemplate);

      const result = await templatesService.validateTemplateConsistency('template-1');

      expect(result.isValid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('should detect templates with no character customization options', async () => {
      const templateWithIssues = {
        ...mockTemplate,
        attributes: {
          ...mockTemplate.attributes,
          characters: {
            templates: [
              {
                ...mockTemplate.attributes.characters.templates[0],
                faces: [], // No faces available
                hairs: [], // No hairs available
              }
            ]
          }
        }
      };

      mockApi.getOne.mockResolvedValue(templateWithIssues);

      const result = await templatesService.validateTemplateConsistency('template-1');

      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Character template 0: No faces defined');
      expect(result.errors).toContain('Character template 0: No hairs defined');
    });

    it('should detect duplicate NPC IDs', async () => {
      const templateWithDuplicateNpcs = {
        ...mockTemplate,
        attributes: {
          ...mockTemplate.attributes,
          npcs: [
            { npcId: 9000000, impl: 'TestNpc1' },
            { npcId: 9000000, impl: 'TestNpc2' }, // Duplicate ID
          ]
        }
      };

      mockApi.getOne.mockResolvedValue(templateWithDuplicateNpcs);

      const result = await templatesService.validateTemplateConsistency('template-1');

      expect(result.isValid).toBe(false);
      expect(result.errors.some(error => error.includes('Duplicate NPC IDs found'))).toBe(true);
    });

    it('should detect duplicate handler opCodes', async () => {
      const templateWithDuplicateHandlers = {
        ...mockTemplate,
        attributes: {
          ...mockTemplate.attributes,
          socket: {
            ...mockTemplate.attributes.socket,
            handlers: [
              { opCode: '0x01', validator: 'TestValidator1', handler: 'TestHandler1', options: {} },
              { opCode: '0x01', validator: 'TestValidator2', handler: 'TestHandler2', options: {} }, // Duplicate opCode
            ]
          }
        }
      };

      mockApi.getOne.mockResolvedValue(templateWithDuplicateHandlers);

      const result = await templatesService.validateTemplateConsistency('template-1');

      expect(result.isValid).toBe(false);
      expect(result.errors.some(error => error.includes('Duplicate handler opCodes found'))).toBe(true);
    });
  });

  describe('export', () => {
    it('should export templates as JSON', async () => {
      mockApi.getList.mockResolvedValue([mockTemplate]);

      const result = await templatesService.export('json');

      expect(result.type).toBe('application/json');
      
      // Mock the Blob text method for testing
      const mockText = jest.fn().mockResolvedValue(JSON.stringify([mockTemplate]));
      (result as any).text = mockText;
      
      const content = await result.text();
      const parsedContent = JSON.parse(content);
      expect(parsedContent).toHaveLength(1);
      expect(parsedContent[0].id).toBe('template-1');
    });

    it('should export templates as CSV', async () => {
      mockApi.getList.mockResolvedValue([mockTemplate]);

      const result = await templatesService.export('csv');

      expect(result.type).toBe('text/csv');
      
      // Mock the Blob text method for testing
      const csvContent = 'ID,Region,Major Version,Minor Version,Uses Pin,Character Templates Count,NPCs Count,Handlers Count,Writers Count,Worlds Count\ntemplate-1,GMS,83,1,false,1,1,1,1,1';
      const mockText = jest.fn().mockResolvedValue(csvContent);
      (result as any).text = mockText;
      
      const content = await result.text();
      const lines = content.split('\n');
      expect(lines[0]).toContain('ID,Region,Major Version');
      expect(lines[1]).toContain('template-1,GMS,83,1');
    });
  });

  describe('batch operations', () => {
    it('should handle batch create', async () => {
      const templates = [mockTemplateAttributes];
      mockApi.post.mockResolvedValue(mockTemplate);

      const result = await templatesService.createBatch(templates);

      expect(result.successCount).toBe(1);
      expect(result.failureCount).toBe(0);
      expect(result.successes).toHaveLength(1);
    });

    it('should handle batch update', async () => {
      const updates = [{ id: 'template-1', data: { region: 'EMS' } }];
      mockApi.put.mockResolvedValue({ ...mockTemplate, attributes: { ...mockTemplate.attributes, region: 'EMS' } });

      const result = await templatesService.updateBatch(updates, { validate: false });

      expect(result.successCount).toBe(1);
      expect(result.failureCount).toBe(0);
      expect(result.successes).toHaveLength(1);
    });

    it('should handle batch delete', async () => {
      const ids = ['template-1'];
      mockApi.delete.mockResolvedValue(undefined);

      const result = await templatesService.deleteBatch(ids);

      expect(result.successCount).toBe(1);
      expect(result.failureCount).toBe(0);
      expect(result.successes).toContain('template-1');
    });
  });

  describe('cache management', () => {
    it('should clear cache', () => {
      templatesService.clearCache();
      expect(mockApi.clearCacheByPattern).toHaveBeenCalled();
    });

    it('should get cache stats', () => {
      mockApi.getCacheStats.mockReturnValue({ entries: [] });
      
      const stats = templatesService.getCacheStats();
      
      expect(mockApi.getCacheStats).toHaveBeenCalled();
      expect(stats).toBeDefined();
    });
  });
});