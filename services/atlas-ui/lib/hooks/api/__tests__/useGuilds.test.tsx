/**
 * Tests for guild React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import {
  useGuilds,
  useGuild,
  useGuildsByWorld,
  useGuildSearch,
  useGuildsWithSpace,
  useGuildRankings,
  guildKeys,
} from '../useGuilds';
import type { Guild, GuildAttributes, GuildMember } from '@/types/models/guild';
import type { Tenant } from '@/types/models/tenant';

// Mock the guilds service
jest.mock('@/services/api/guilds.service', () => ({
  guildsService: {
    getAll: jest.fn(),
    getById: jest.fn(),
    getByWorld: jest.fn(),
    search: jest.fn(),
    getWithSpace: jest.fn(),
    getRankings: jest.fn(),
    exists: jest.fn(),
    getMemberCount: jest.fn(),
  },
}));

import { guildsService } from '@/services/api/guilds.service';

// Test data
const mockTenant: Tenant = {
  id: 'tenant-1',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

const mockGuildMember: GuildMember = {
  characterId: 1,
  name: 'TestPlayer',
  jobId: 100,
  level: 50,
  title: 1,
  online: true,
  allianceTitle: 0,
};

const mockGuild: Guild = {
  id: 'guild-1',
  attributes: {
    worldId: 1,
    name: 'TestGuild',
    notice: 'Welcome to our guild!',
    points: 1000,
    capacity: 100,
    logo: 1,
    logoColor: 0,
    logoBackground: 0,
    logoBackgroundColor: 0,
    leaderId: 1,
    members: [mockGuildMember],
    titles: [{ name: 'Master', index: 1 }],
  },
};

const mockGuilds: Guild[] = [mockGuild];

// Test wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  TestWrapper.displayName = 'TestWrapper';
  return TestWrapper;
}

describe('useGuilds', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Hooks', () => {
    it('should fetch all guilds successfully', async () => {
      (guildsService.getAll as jest.Mock).mockResolvedValue(mockGuilds);

      const { result } = renderHook(() => useGuilds(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuilds);
      });

      expect(guildsService.getAll).toHaveBeenCalledWith(mockTenant, undefined);
    });

    it('should fetch guild by ID successfully', async () => {
      (guildsService.getById as jest.Mock).mockResolvedValue(mockGuild);

      const { result } = renderHook(() => useGuild(mockTenant, 'guild-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuild);
      });

      expect(guildsService.getById).toHaveBeenCalledWith(mockTenant, 'guild-1', undefined);
    });

    it('should fetch guilds by world ID successfully', async () => {
      (guildsService.getByWorld as jest.Mock).mockResolvedValue(mockGuilds);

      const { result } = renderHook(() => useGuildsByWorld(mockTenant, 1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuilds);
      });

      expect(guildsService.getByWorld).toHaveBeenCalledWith(mockTenant, 1, undefined);
    });

    it('should search guilds successfully', async () => {
      (guildsService.search as jest.Mock).mockResolvedValue(mockGuilds);

      const { result } = renderHook(() => useGuildSearch(mockTenant, 'Test'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuilds);
      });

      expect(guildsService.search).toHaveBeenCalledWith(mockTenant, 'Test', undefined, undefined);
    });

    it('should fetch guilds with space successfully', async () => {
      (guildsService.getWithSpace as jest.Mock).mockResolvedValue(mockGuilds);

      const { result } = renderHook(() => useGuildsWithSpace(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuilds);
      });

      expect(guildsService.getWithSpace).toHaveBeenCalledWith(mockTenant, undefined, undefined);
    });

    it('should fetch guild rankings successfully', async () => {
      (guildsService.getRankings as jest.Mock).mockResolvedValue(mockGuilds);

      const { result } = renderHook(() => useGuildRankings(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
        expect(result.current.data).toEqual(mockGuilds);
      });

      expect(guildsService.getRankings).toHaveBeenCalledWith(mockTenant, undefined, 50, undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      const { result } = renderHook(() => useGuilds(null as Tenant | null), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.data).toBeUndefined();
      expect(guildsService.getAll).not.toHaveBeenCalled();
    });

    it('should not fetch guild when guildId is not provided', () => {
      const { result } = renderHook(() => useGuild(mockTenant, ''), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.data).toBeUndefined();
      expect(guildsService.getById).not.toHaveBeenCalled();
    });
  });

  describe('Query Keys', () => {
    it('should generate correct query keys', () => {
      expect(guildKeys.all).toEqual(['guilds']);
      expect(guildKeys.lists()).toEqual(['guilds', 'list']);
      expect(guildKeys.list(mockTenant)).toEqual(['guilds', 'list', 'tenant-1', undefined]);
      expect(guildKeys.detail(mockTenant, 'guild-1')).toEqual(['guilds', 'detail', 'tenant-1', 'guild-1']);
      expect(guildKeys.byWorld(mockTenant, 1)).toEqual(['guilds', 'list', 'tenant-1', 'world', 1]);
      expect(guildKeys.search(mockTenant, 'test')).toEqual(['guilds', 'search', 'tenant-1', 'test', undefined]);
      expect(guildKeys.rankings(mockTenant, 1, 25)).toEqual(['guilds', 'list', 'tenant-1', 'rankings', 1, 25]);
    });
  });
});