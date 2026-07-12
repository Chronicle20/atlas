import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query';
import { seedService, type DataStatus, type WzInputStatus } from '@/services/api/seed.service';
import { baselineService, type Baseline } from '@/services/api/baseline.service';
import type { CanonicalSelection } from '@/lib/headers';
import { showWzUploadErrorToast } from '@/lib/hooks/api/useSeed';

const canonicalWzInputKey = (sel: CanonicalSelection) =>
  ['canonical', 'wzInput', sel.region, sel.majorVersion, sel.minorVersion] as const;
const canonicalDataStatusKey = (sel: CanonicalSelection) =>
  ['canonical', 'dataStatus', sel.region, sel.majorVersion, sel.minorVersion] as const;
export const baselinesKey = ['baselines'] as const;

export function useCanonicalWzInputStatus(sel: CanonicalSelection | null): UseQueryResult<WzInputStatus, Error> {
  return useQuery({
    queryKey: sel ? canonicalWzInputKey(sel) : ['canonical', 'wzInput', 'none'],
    queryFn: () => seedService.getCanonicalWzInputStatus(sel!),
    enabled: !!sel,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useCanonicalDataStatus(sel: CanonicalSelection | null): UseQueryResult<DataStatus, Error> {
  return useQuery({
    queryKey: sel ? canonicalDataStatusKey(sel) : ['canonical', 'dataStatus', 'none'],
    queryFn: () => seedService.getCanonicalDataStatus(sel!),
    enabled: !!sel,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useUploadCanonicalWz(sel: CanonicalSelection | null): UseMutationResult<void, Error, File> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => {
      if (!sel) {
        throw new Error('useUploadCanonicalWz: no region/version selected');
      }
      return seedService.uploadCanonicalWzFiles(sel, file);
    },
    onSuccess: () => {
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalWzInputKey(sel) });
    },
    onError: showWzUploadErrorToast,
  });
}

export function useRunCanonicalProcessing(sel: CanonicalSelection | null): UseMutationResult<void, Error, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => {
      if (!sel) {
        throw new Error('useRunCanonicalProcessing: no region/version selected');
      }
      return seedService.runCanonicalDataProcessing(sel);
    },
    onSuccess: () => {
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalDataStatusKey(sel) });
    },
  });
}

export function useBaselines(): UseQueryResult<Baseline[], Error> {
  return useQuery({
    queryKey: baselinesKey,
    queryFn: () => baselineService.listBaselines(),
  });
}

export function usePublishCanonicalBaseline(sel: CanonicalSelection | null): UseMutationResult<void, Error, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => {
      if (!sel) {
        throw new Error('usePublishCanonicalBaseline: no region/version selected');
      }
      return baselineService.publish(sel);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: baselinesKey });
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalDataStatusKey(sel) });
    },
  });
}
