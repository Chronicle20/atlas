/**
 * Test suite for ConversationsService
 */
import { conversationsService } from '../conversations.service';
import type { ConversationAttributes } from '@/types/models/conversation';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  api: {
    getList: jest.fn(),
    getOne: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
    clearCacheByPattern: jest.fn(),
    getCacheStats: jest.fn(),
  },
}));

describe('ConversationsService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('validation', () => {
    it('should validate required fields', () => {
      const invalidData = {};
      const errors = (conversationsService as any).validate(invalidData);
      
      expect(errors).toHaveLength(3); // npcId, startState, states
      expect(errors[0].field).toBe('npcId');
      expect(errors[1].field).toBe('startState');
      expect(errors[2].field).toBe('states');
    });

    it('should validate npcId type', () => {
      const invalidData = {
        npcId: 'invalid',
        startState: 'start',
        states: [{ id: 'start', type: 'dialogue' }]
      };
      const errors = (conversationsService as any).validate(invalidData);
      
      expect(errors.some(e => e.field === 'npcId')).toBeTruthy();
    });

    it('should validate state references', () => {
      const invalidData: ConversationAttributes = {
        npcId: 1,
        startState: 'nonexistent',
        states: [{ id: 'start', type: 'dialogue' as const }]
      };
      const errors = (conversationsService as any).validate(invalidData);
      
      expect(errors.some(e => e.field === 'startState' && e.message.includes('must exist'))).toBeTruthy();
    });

    it('should pass validation for valid data', () => {
      const validData: ConversationAttributes = {
        npcId: 1,
        startState: 'start',
        states: [
          {
            id: 'start',
            type: 'dialogue',
            dialogue: {
              dialogueType: 'sendOk',
              text: 'Hello!',
              choices: []
            }
          }
        ]
      };
      const errors = (conversationsService as any).validate(validData);
      
      expect(errors).toHaveLength(0);
    });
  });

  describe('transformRequest', () => {
    it('should wrap conversation attributes in API format', () => {
      const input: ConversationAttributes = {
        npcId: 1,
        startState: 'start',
        states: []
      };
      
      const result = (conversationsService as any).transformRequest(input);
      
      expect(result).toEqual({
        data: {
          type: 'conversations',
          attributes: input
        }
      });
    });

    it('should include id for updates', () => {
      const input = {
        id: 'test-id',
        npcId: 1,
        startState: 'start',
        states: []
      };
      
      const result = (conversationsService as any).transformRequest(input);
      
      expect(result.data.id).toBe('test-id');
    });
  });

  describe('transformResponse', () => {
    it('should extract data from API response wrapper', () => {
      const apiResponse = {
        data: {
          id: 'test-id',
          type: 'conversations',
          attributes: {
            npcId: 1,
            startState: 'start',
            states: []
          }
        }
      };
      
      const result = (conversationsService as any).transformResponse(apiResponse);
      
      expect(result).toEqual(apiResponse.data);
    });

    it('should return data as-is if not wrapped', () => {
      const directData = {
        id: 'test-id',
        type: 'conversations',
        attributes: {
          npcId: 1,
          startState: 'start',
          states: []
        }
      };
      
      const result = (conversationsService as any).transformResponse(directData);
      
      expect(result).toEqual(directData);
    });
  });

  describe('validateStateConsistency', () => {
    const mockConversation = {
      id: 'test-id',
      type: 'conversations',
      attributes: {
        npcId: 1,
        startState: 'start',
        states: [
          {
            id: 'start',
            type: 'dialogue' as const,
            dialogue: {
              dialogueType: 'sendNext' as const,
              text: 'Welcome!',
              choices: [
                { text: 'Continue', nextState: 'middle', context: {} }
              ]
            }
          },
          {
            id: 'middle',
            type: 'dialogue' as const,
            dialogue: {
              dialogueType: 'sendOk' as const,
              text: 'Good!',
              choices: []
            }
          },
          {
            id: 'unreachable',
            type: 'dialogue' as const,
            dialogue: {
              dialogueType: 'sendOk' as const,
              text: 'Never reached',
              choices: []
            }
          }
        ]
      }
    };

    beforeEach(() => {
      (require('@/lib/api/client').api.getOne as jest.Mock).mockResolvedValue(mockConversation);
    });

    it('should detect unreachable states', async () => {
      const result = await conversationsService.validateStateConsistency('test-id');
      
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain("State 'unreachable' is unreachable");
    });

    it('should detect invalid state references', async () => {
      const conversationWithInvalidRef = {
        ...mockConversation,
        attributes: {
          ...mockConversation.attributes,
          states: [
            {
              id: 'start',
              type: 'dialogue' as const,
              dialogue: {
                dialogueType: 'sendNext' as const,
                text: 'Welcome!',
                choices: [
                  { text: 'Continue', nextState: 'nonexistent', context: {} }
                ]
              }
            }
          ]
        }
      };

      (require('@/lib/api/client').api.getOne as jest.Mock).mockResolvedValue(conversationWithInvalidRef);
      
      const result = await conversationsService.validateStateConsistency('test-id');
      
      expect(result.isValid).toBe(false);
      expect(result.errors.some(error => error.includes('nonexistent'))).toBe(true);
    });
  });
});