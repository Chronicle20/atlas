import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BaselinesPage } from '@/pages/BaselinesPage';
import type { CanonicalSelection } from '@/lib/headers';
import type { Baseline } from '@/services/api/baseline.service';

// The picker has its own tests; stub it with a button that selects GMS 83.1.
vi.mock('@/components/features/baselines/BaselineTargetPicker', () => ({
  BaselineTargetPicker: ({ onChange }: { onChange: (sel: CanonicalSelection | null) => void }) => (
    <button onClick={() => onChange({ region: 'GMS', majorVersion: 83, minorVersion: 1 })}>
      pick-gms-83
    </button>
  ),
}));

// Mutable fixture state, reset per test.
let baselines: Baseline[] = [];
let wzStatus: { fileCount: number; totalBytes: number; updatedAt: string | null } | undefined;
let dataStatus:
  | { documentCount: number; updatedAt: string | null; baselineRestoredAt: string | null; baselineSha256: string | null }
  | undefined;
const uploadMutate = vi.fn();
const processMutate = vi.fn();
const publishMutate = vi.fn();

vi.mock('@/lib/hooks/api/useCanonicalData', () => ({
  useBaselines: () => ({ data: baselines, isLoading: false, isError: false, error: null }),
  useCanonicalWzInputStatus: () => ({ data: wzStatus }),
  useCanonicalDataStatus: () => ({ data: dataStatus }),
  useUploadCanonicalWz: () => ({ mutate: uploadMutate, isPending: false }),
  useRunCanonicalProcessing: () => ({ mutate: processMutate, isPending: false }),
  usePublishCanonicalBaseline: () => ({ mutate: publishMutate, isPending: false }),
}));

beforeEach(() => {
  baselines = [];
  wzStatus = undefined;
  dataStatus = undefined;
  uploadMutate.mockClear();
  processMutate.mockClear();
  publishMutate.mockClear();
});

const sampleBaseline: Baseline = {
  region: 'GMS',
  majorVersion: 83,
  minorVersion: 1,
  sha256: 'a'.repeat(64),
  publishedAt: '2026-07-04T12:34:56Z',
  sizeBytes: 123456789,
};

describe('BaselinesPage', () => {
  it('renders the empty state when no baselines are published', () => {
    render(<BaselinesPage />);
    expect(screen.getByText(/no canonical baselines published yet/i)).toBeInTheDocument();
  });

  it('renders baseline rows with truncated sha and an em dash for a blank sha', () => {
    baselines = [sampleBaseline, { ...sampleBaseline, region: 'JMS', majorVersion: 185, sha256: '', sizeBytes: 1024 }];
    render(<BaselinesPage />);
    expect(screen.getByText('GMS')).toBeInTheDocument();
    expect(screen.getByText('83.1')).toBeInTheDocument();
    expect(screen.getByText(`${'a'.repeat(12)}…`)).toBeInTheDocument();
    // The workflow badges also render em dashes while nothing is selected,
    // so assert at-least-one rather than exactly-one.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    // 123456789 bytes -> value >= 10 in MB -> zero decimals.
    expect(screen.getByText('118 MB')).toBeInTheDocument();
    expect(screen.getByText('1 KB')).toBeInTheDocument();
  });

  it('disables all workflow rows until a selection exists', () => {
    render(<BaselinesPage />);
    expect(screen.getByRole('button', { name: /upload/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /process data/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeDisabled();
  });

  it('enables upload after selection; process stays disabled with 0 wz files', () => {
    wzStatus = { fileCount: 0, totalBytes: 0, updatedAt: null };
    dataStatus = { documentCount: 0, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    expect(screen.getByRole('button', { name: /upload/i })).toBeEnabled();
    expect(screen.getByRole('button', { name: /process data/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeDisabled();
  });

  it('enables process with wz files and publish with documents', () => {
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    expect(screen.getByRole('button', { name: /process data/i })).toBeEnabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeEnabled();
  });

  it('publishes immediately when the selection has no existing baseline', () => {
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    fireEvent.click(screen.getByRole('button', { name: /publish baseline/i }));
    expect(publishMutate).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/replace the shared canonical baseline/i)).not.toBeInTheDocument();
  });

  it('requires confirmation when re-publishing over an existing baseline', () => {
    baselines = [sampleBaseline];
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    fireEvent.click(screen.getByRole('button', { name: /publish baseline/i }));
    expect(publishMutate).not.toHaveBeenCalled();
    expect(screen.getByText(/this will replace the shared canonical baseline for GMS v83\.1/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /replace baseline/i }));
    expect(publishMutate).toHaveBeenCalledTimes(1);
  });
});
