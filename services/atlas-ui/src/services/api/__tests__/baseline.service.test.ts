import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { baselineService } from '@/services/api/baseline.service';
import type { Tenant } from '@/types/models/tenant';

const mockTenant: Tenant = {
  id: '11111111-1111-1111-1111-111111111111',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

describe('baselineService', () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('restore', () => {
    it('POSTs JSON with tenant headers and no operator header', async () => {
      fetchMock.mockResolvedValue({ ok: true, status: 202, json: async () => ({}) });
      await baselineService.restore(mockTenant, {
        region: 'GMS',
        majorVersion: 83,
        minorVersion: 1,
        tenantId: mockTenant.id,
      });
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/data/baseline/restore',
        expect.objectContaining({ method: 'POST' }),
      );
      const call = fetchMock.mock.calls[0];
      if (!call) throw new Error('fetch was not called');
      const init = call[1] as RequestInit;
      const headers = init.headers as Headers;
      expect(headers.get('TENANT_ID')).toBe(mockTenant.id);
      expect(headers.get('REGION')).toBe('GMS');
      expect(headers.get('MAJOR_VERSION')).toBe('83');
      expect(headers.get('MINOR_VERSION')).toBe('1');
      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('X-Atlas-Operator')).toBeNull();
      expect(init.body).toBe(
        JSON.stringify({
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          tenantId: mockTenant.id,
        }),
      );
    });

    it('throws an Error with response.error body on 4xx', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 422,
        statusText: 'Unprocessable Entity',
        json: async () => ({ error: 'sha256 mismatch' }),
      });
      await expect(
        baselineService.restore(mockTenant, {
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          tenantId: mockTenant.id,
        }),
      ).rejects.toThrow(/sha256 mismatch/);
    });

    it('falls back to status when response body is not JSON', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => {
          throw new Error('not json');
        },
      });
      await expect(
        baselineService.restore(mockTenant, {
          region: 'GMS',
          majorVersion: 83,
          minorVersion: 1,
          tenantId: mockTenant.id,
        }),
      ).rejects.toThrow(/restore failed: 500/);
    });
  });

  describe('publish', () => {
    it('sets X-Atlas-Operator: 1 header', async () => {
      fetchMock.mockResolvedValue({ ok: true, status: 202, json: async () => ({}) });
      await baselineService.publish(mockTenant, 'GMS', 83, 1);
      const call = fetchMock.mock.calls[0];
      if (!call) throw new Error('fetch was not called');
      const init = call[1] as RequestInit;
      const headers = init.headers as Headers;
      expect(headers.get('X-Atlas-Operator')).toBe('1');
      expect(headers.get('TENANT_ID')).toBe(mockTenant.id);
      expect(init.body).toBe(JSON.stringify({ region: 'GMS', majorVersion: 83, minorVersion: 1 }));
    });

    it('throws response.error body on failure', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: async () => ({ error: 'missing region' }),
      });
      await expect(baselineService.publish(mockTenant, 'GMS', 83, 1)).rejects.toThrow(
        /missing region/,
      );
    });

    it('falls back to status when body is not JSON', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 503,
        statusText: 'Service Unavailable',
        json: async () => {
          throw new Error('not json');
        },
      });
      await expect(baselineService.publish(mockTenant, 'GMS', 83, 1)).rejects.toThrow(
        /publish failed: 503/,
      );
    });
  });
});
