import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SetupPage } from '@/pages/SetupPage';

const mockTenant = {
  id: '11111111-1111-1111-1111-111111111111',
  attributes: { name: 'Test Tenant', region: 'GMS', majorVersion: 83, minorVersion: 1 },
};

const idleMutation = { mutate: vi.fn(), isPending: false };
const emptyStatus = { data: undefined };

// Mutable per-test data-status so individual tests can flip documentCount.
let dataStatusData: { documentCount: number; updatedAt: string | null; baselineRestoredAt: string | null; baselineSha256: string | null } = {
  documentCount: 0,
  updatedAt: null,
  baselineRestoredAt: null,
  baselineSha256: null,
};

vi.mock('@/context/tenant-context', () => ({
  useTenant: () => ({ activeTenant: mockTenant }),
}));

vi.mock('@/lib/hooks/api/useBaseline', () => ({
  useRestoreBaseline: () => idleMutation,
}));

vi.mock('@/lib/hooks/api/useSeed', () => ({
  useSeedDrops: () => idleMutation,
  useSeedGachapons: () => idleMutation,
  useSeedNpcConversations: () => idleMutation,
  useSeedQuestConversations: () => idleMutation,
  useSeedNpcShops: () => idleMutation,
  useSeedPortalScripts: () => idleMutation,
  useSeedReactorScripts: () => idleMutation,
  useSeedMapActionScripts: () => idleMutation,
  useUploadWzFiles: () => idleMutation,
  useRunDataProcessing: () => idleMutation,
  useWzInputStatus: () => ({ data: { fileCount: 2, totalBytes: 1024, updatedAt: null } }),
  useDataStatus: () => ({ data: dataStatusData }),
  useDropsSeedStatus: () => emptyStatus,
  useGachaponsSeedStatus: () => emptyStatus,
  useNpcConversationsSeedStatus: () => emptyStatus,
  useQuestConversationsSeedStatus: () => emptyStatus,
  useNpcShopsSeedStatus: () => emptyStatus,
  usePortalScriptsSeedStatus: () => emptyStatus,
  useReactorScriptsSeedStatus: () => emptyStatus,
  useMapActionScriptsSeedStatus: () => emptyStatus,
  showWzUploadErrorToast: vi.fn(),
}));

describe('SetupPage (tenant-only)', () => {
  it('is titled Setup and has no scope toggle and no publish row', () => {
    render(<SetupPage />);
    expect(screen.getByRole('heading', { name: 'Setup' })).toBeInTheDocument();
    expect(screen.queryByTestId('scope-toggle')).not.toBeInTheDocument();
    expect(screen.queryByText(/Publish Canonical Baseline/i)).not.toBeInTheDocument();
  });

  it('shows the restore row when the tenant document count is 0', () => {
    dataStatusData = { documentCount: 0, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<SetupPage />);
    expect(screen.getByText(/Restore Canonical Baseline/i)).toBeInTheDocument();
  });

  it('hides the restore row when documents exist', () => {
    dataStatusData = { documentCount: 5, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<SetupPage />);
    expect(screen.queryByText(/Restore Canonical Baseline/i)).not.toBeInTheDocument();
  });

  it('renders all eight seed rows', () => {
    render(<SetupPage />);
    for (const label of [
      'Monster & Reactor Drops',
      'Gachapons',
      'NPC Conversations',
      'Quest Conversations',
      'NPC Shops',
      'Portal Scripts',
      'Reactor Scripts',
      'Map Action Scripts',
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });
});
