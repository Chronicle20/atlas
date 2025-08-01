/**
 * Tests for useConversations React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { 
  useConversations, 
  useConversation, 
  useConversationExists,
  useConversationByNpc, 
  useConversationSearch,
  useConversationsByNpc,
  useConversationStateConsistency,
  useConversationExport,
  useCreateConversation,
  useUpdateConversation,
  usePatchConversation,
  useDeleteConversation,
  conversationKeys 
} from '../useConversations';
import { conversationsService } from '@/services/api/conversations.service';
import type { Conversation, ConversationAttributes } from '@/types/models/conversation';
import type { Tenant } from '@/types/models/tenant';

// Mock the conversationsService
jest.mock('@/services/api/conversations.service');
const mockConversationsService = conversationsService as jest.Mocked<typeof conversationsService>;

// Mock data
const mockTenant: Tenant = {
  id: 'tenant-1',
  type: 'tenants',
  attributes: {
    id: 'tenant-1',
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  }
};

const mockConversationAttributes: ConversationAttributes = {
  npcId: 100,
  startState: 'welcome',
  states: [
    {
      id: 'welcome',
      type: 'dialogue',
      dialogue: {
        text: 'Welcome to my shop!',
        choices: [
          {
            text: 'Show me your wares',
            nextState: 'shop'
          },
          {
            text: 'Goodbye',
            nextState: null
          }
        ]
      }
    },
    {
      id: 'shop',
      type: 'listSelection',
      listSelection: {
        title: 'Choose an item',
        choices: [
          {
            text: 'Potion - 100 mesos',
            nextState: 'purchase'
          }
        ]
      }
    },
    {
      id: 'purchase',
      type: 'dialogue',
      dialogue: {
        text: 'Thank you for your purchase!',
        choices: [
          {
            text: 'You\'re welcome',
            nextState: null
          }
        ]
      }
    }
  ]
};

const mockConversation: Conversation = {
  id: 'conversation-1',
  type: 'conversations',
  attributes: mockConversationAttributes
};

const mockConversations: Conversation[] = [
  mockConversation,
  {
    id: 'conversation-2',
    type: 'conversations',
    attributes: {
      ...mockConversationAttributes,
      npcId: 101,
      startState: 'greeting'
    }
  }
];

const mockStateConsistencyResult = {
  isValid: true,
  errors: []
};

const mockExportBlob = new Blob(['test data'], { type: 'application/json' });

// Test wrapper component
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

describe('useConversations hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  // ============================================================================
  // QUERY HOOKS TESTS
  // ============================================================================

  describe('useConversations', () => {
    it('should fetch all conversations successfully', async () => {
      mockConversationsService.getAll.mockResolvedValue(mockConversations);

      const { result } = renderHook(() => useConversations(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockConversations);
      expect(mockConversationsService.getAll).toHaveBeenCalledWith(undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      mockConversationsService.getAll.mockResolvedValue(mockConversations);

      const { result } = renderHook(() => useConversations(null as any), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockConversationsService.getAll).not.toHaveBeenCalled();
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Failed to fetch conversations');
      mockConversationsService.getAll.mockRejectedValue(error);

      const { result } = renderHook(() => useConversations(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toEqual(error);
    });
  });

  describe('useConversation', () => {
    it('should fetch specific conversation successfully', async () => {
      mockConversationsService.getById.mockResolvedValue(mockConversation);

      const { result } = renderHook(() => useConversation(mockTenant, 'conversation-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockConversation);
      expect(mockConversationsService.getById).toHaveBeenCalledWith('conversation-1', undefined);
    });
  });

  describe('useConversationExists', () => {
    it('should check if conversation exists successfully', async () => {
      mockConversationsService.exists.mockResolvedValue(true);

      const { result } = renderHook(() => useConversationExists(mockTenant, 'conversation-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBe(true);
      expect(mockConversationsService.exists).toHaveBeenCalledWith('conversation-1', undefined);
    });
  });

  describe('useConversationByNpc', () => {
    it('should fetch conversation by NPC ID successfully', async () => {
      mockConversationsService.getByNpcId.mockResolvedValue(mockConversation);

      const { result } = renderHook(() => useConversationByNpc(mockTenant, 100), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockConversation);
      expect(mockConversationsService.getByNpcId).toHaveBeenCalledWith(100, undefined);
    });

    it('should return null when no conversation exists for NPC', async () => {
      mockConversationsService.getByNpcId.mockResolvedValue(null);

      const { result } = renderHook(() => useConversationByNpc(mockTenant, 999), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBeNull();
    });
  });

  describe('useConversationSearch', () => {
    it('should search conversations by text successfully', async () => {
      mockConversationsService.searchByText.mockResolvedValue([mockConversation]);

      const { result } = renderHook(() => useConversationSearch(mockTenant, 'shop'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([mockConversation]);
      expect(mockConversationsService.searchByText).toHaveBeenCalledWith('shop', undefined);
    });

    it('should not search when search text is too short', () => {
      const { result } = renderHook(() => useConversationSearch(mockTenant, 'ab'), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockConversationsService.searchByText).not.toHaveBeenCalled();
    });
  });

  describe('useCreateConversation', () => {
    it('should create conversation successfully', async () => {
      mockConversationsService.create.mockResolvedValue(mockConversation);

      const { result } = renderHook(() => useCreateConversation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        conversationAttributes: mockConversationAttributes,
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockConversation);
      expect(mockConversationsService.create).toHaveBeenCalledWith(mockConversationAttributes, undefined);
    });

    it('should handle create conversation errors', async () => {
      const error = new Error('Failed to create conversation');
      mockConversationsService.create.mockRejectedValue(error);

      const { result } = renderHook(() => useCreateConversation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        conversationAttributes: mockConversationAttributes,
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toEqual(error);
    });
  });

  describe('useUpdateConversation', () => {
    it('should update conversation successfully', async () => {
      mockConversationsService.update.mockResolvedValue(mockConversation);

      const { result } = renderHook(() => useUpdateConversation(), {
        wrapper: createWrapper(),
      });

      const updates = { npcId: 102 };
      
      result.current.mutate({
        id: 'conversation-1',
        conversationAttributes: updates,
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockConversation);
      expect(mockConversationsService.update).toHaveBeenCalledWith('conversation-1', updates, undefined);
    });
  });

  describe('useDeleteConversation', () => {
    it('should delete conversation successfully', async () => {
      mockConversationsService.delete.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteConversation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        id: 'conversation-1',
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockConversationsService.delete).toHaveBeenCalledWith('conversation-1', undefined);
    });
  });

  // ============================================================================
  // QUERY KEY TESTS
  // ============================================================================

  describe('conversationKeys', () => {
    it('should generate correct query keys', () => {
      expect(conversationKeys.all).toEqual(['conversations']);
      expect(conversationKeys.lists()).toEqual(['conversations', 'list']);
      expect(conversationKeys.list(mockTenant)).toEqual(['conversations', 'list', 'tenant-1', undefined]);
      expect(conversationKeys.details()).toEqual(['conversations', 'detail']);
      expect(conversationKeys.detail(mockTenant, 'conversation-1')).toEqual(['conversations', 'detail', 'tenant-1', 'conversation-1']);
      expect(conversationKeys.byNpc()).toEqual(['conversations', 'byNpc']);
      expect(conversationKeys.npcConversation(mockTenant, 100)).toEqual(['conversations', 'byNpc', 'tenant-1', 100]);
      expect(conversationKeys.searches()).toEqual(['conversations', 'search']);
      expect(conversationKeys.search(mockTenant, 'test')).toEqual(['conversations', 'search', 'tenant-1', 'test']);
      expect(conversationKeys.validation()).toEqual(['conversations', 'validation']);
      expect(conversationKeys.stateConsistency(mockTenant, 'conversation-1')).toEqual(['conversations', 'validation', 'tenant-1', 'conversation-1']);
      expect(conversationKeys.exports()).toEqual(['conversations', 'export']);
      expect(conversationKeys.export(mockTenant, 'json')).toEqual(['conversations', 'export', 'tenant-1', 'json']);
    });

    it('should handle null tenant in query keys', () => {
      expect(conversationKeys.list(null)).toEqual(['conversations', 'list', 'no-tenant', undefined]);
      expect(conversationKeys.detail(null, 'conversation-1')).toEqual(['conversations', 'detail', 'no-tenant', 'conversation-1']);
      expect(conversationKeys.npcConversation(null, 100)).toEqual(['conversations', 'byNpc', 'no-tenant', 100]);
    });
  });
});