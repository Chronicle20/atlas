import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { seedService } from '@/services/api/seed.service';
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

/**
 * Fixture matching the exact shape libs/atlas-seeder's
 * GET /<prefix>/seed/status handler emits: a flat JSON object (NOT a
 * JSON:API envelope) with a `subdomains` map keyed by Subdomain.Name().
 */
function seedStatusBody(subdomains: Record<string, number>) {
  return {
    groupName: 'group',
    subdomains: Object.fromEntries(
      Object.entries(subdomains).map(([k, c]) => [k, { count: c, updatedAt: null }]),
    ),
    updatedAt: null,
    catalogRevision: 'rev-abc',
    tenantSeededRevision: 'rev-abc',
    tenantSeededAt: '2026-05-27T16:35:00Z',
  };
}

describe('seedService status projections', () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('sends tenant headers and does NOT request JSON:API for /seed/status', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'monster-drop': 0 }),
    });
    await seedService.getDropsSeedStatus(mockTenant);
    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error('fetch was not called');
    expect(call[0]).toBe('/api/drops/seed/status');
    const init = call[1] as RequestInit;
    const headers = init.headers as Headers;
    expect(headers.get('TENANT_ID')).toBe(mockTenant.id);
    expect(headers.get('REGION')).toBe('GMS');
    expect(headers.get('MAJOR_VERSION')).toBe('83');
    expect(headers.get('MINOR_VERSION')).toBe('1');
    // The seeder lib emits plain `application/json` — explicitly do NOT
    // request a JSON:API envelope, which would have caused the previous
    // fetcher to dereference `body.data.attributes` and crash.
    expect(headers.get('Accept')).toBeNull();
  });

  it('projects drops subdomain map to monster/continent/reactor fields', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () =>
        seedStatusBody({
          'monster-drop': 22640,
          'continent-drop': 4,
          'reactor-drop': 1116,
        }),
    });
    const s = await seedService.getDropsSeedStatus(mockTenant);
    expect(s.monsterDropCount).toBe(22640);
    expect(s.continentDropCount).toBe(4);
    expect(s.reactorDropCount).toBe(1116);
    expect(s.updatedAt).toBe('2026-05-27T16:35:00Z');
  });

  it('returns 0 for missing subdomains rather than NaN/undefined', async () => {
    // Regression for the bug that prompted this rewrite: the old
    // fetcher dereferenced body.data.attributes which is undefined on
    // the new server response, throwing and rendering the badge as "—"
    // even after a successful seed.
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({}), // empty map
    });
    const s = await seedService.getDropsSeedStatus(mockTenant);
    expect(s.monsterDropCount).toBe(0);
    expect(s.continentDropCount).toBe(0);
    expect(s.reactorDropCount).toBe(0);
  });

  it('projects gachapons subdomain map to gachapon/item/globalItem fields', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () =>
        seedStatusBody({
          gachapons: 12,
          items: 1643,
          globalItems: 44,
        }),
    });
    const s = await seedService.getGachaponsSeedStatus(mockTenant);
    expect(s.gachaponCount).toBe(12);
    expect(s.itemCount).toBe(1643);
    expect(s.globalItemCount).toBe(44);
  });

  it('reads npc-conversations from npc.conversation key', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'npc.conversation': 462 }),
    });
    const s = await seedService.getNpcConversationsSeedStatus(mockTenant);
    expect(s.conversationCount).toBe(462);
  });

  it('reads quest-conversations from quest.conversation key', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'quest.conversation': 214 }),
    });
    const s = await seedService.getQuestConversationsSeedStatus(mockTenant);
    expect(s.conversationCount).toBe(214);
  });

  it('reads npc-shops + auxiliary commodities from subdomain map', async () => {
    // ShopSubdomain implements seeder.SubdomainAuxiliary, so the
    // status response carries a "commodities" entry alongside the
    // primary "npc-shops" entry. The UI projects both.
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'npc-shops': 99, commodities: 3194 }),
    });
    const s = await seedService.getNpcShopsSeedStatus(mockTenant);
    expect(s.shopCount).toBe(99);
    expect(s.commodityCount).toBe(3194);
  });

  it('defaults commodity count to 0 when auxiliary key is absent', async () => {
    // Backward-compat case: a deployment running an older atlas-npc-shops
    // image that does not yet implement SubdomainAuxiliary returns
    // only the primary subdomain. The projection must not blow up.
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'npc-shops': 99 }),
    });
    const s = await seedService.getNpcShopsSeedStatus(mockTenant);
    expect(s.shopCount).toBe(99);
    expect(s.commodityCount).toBe(0);
  });

  it('reads portal-actions from portal-actions key', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'portal-actions': 81 }),
    });
    const s = await seedService.getPortalScriptsSeedStatus(mockTenant);
    expect(s.scriptCount).toBe(81);
  });

  it('reads reactor-actions from reactor-actions key', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => seedStatusBody({ 'reactor-actions': 13 }),
    });
    const s = await seedService.getReactorScriptsSeedStatus(mockTenant);
    expect(s.scriptCount).toBe(13);
  });

  it('sums onUserEnter + onFirstUserEnter for map-action scripts', async () => {
    // map-actions exposes two seeder subdomains, but the UI displays a
    // single scriptCount; the projector must sum them so the badge
    // shows the full file count.
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () =>
        seedStatusBody({
          onUserEnter: 8,
          onFirstUserEnter: 1,
        }),
    });
    const s = await seedService.getMapActionScriptsSeedStatus(mockTenant);
    expect(s.scriptCount).toBe(9);
  });

  it('falls back to updatedAt when tenantSeededAt is null', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        ...seedStatusBody({ 'monster-drop': 1 }),
        tenantSeededAt: null,
        updatedAt: '2026-05-27T17:00:00Z',
      }),
    });
    const s = await seedService.getDropsSeedStatus(mockTenant);
    expect(s.updatedAt).toBe('2026-05-27T17:00:00Z');
  });

  it('throws on non-2xx', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });
    await expect(seedService.getDropsSeedStatus(mockTenant)).rejects.toThrow(/500/);
  });
});
