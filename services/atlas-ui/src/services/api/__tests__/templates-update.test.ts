import { beforeEach, describe, expect, it, vi } from "vitest";

const patch = vi.fn();
const put = vi.fn();
const getOne = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    patch: (...args: unknown[]) => patch(...args),
    put: (...args: unknown[]) => put(...args),
    getOne: (...args: unknown[]) => getOne(...args),
    get: vi.fn(),
    getList: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
    setTenant: vi.fn(),
  },
}));

import { templatesService } from "@/services/api/templates.service";
import type { TemplateAttributes } from "@/types/models/template";

function fullAttributes(): TemplateAttributes {
  return {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [] },
    npcs: [],
    worlds: [],
    socket: {
      handlers: [],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    },
  } as TemplateAttributes;
}

describe("templatesService.update", () => {
  beforeEach(() => {
    patch
      .mockReset()
      .mockResolvedValue({ data: { id: "t1", attributes: fullAttributes() } });
    put.mockReset();
    getOne.mockReset();
  });

  // atlas-configurations registers PATCH only on /configurations/templates/{id}
  // (templates/resource.go:29); there is no MethodPut route anywhere in the
  // service, so a PUT there could only ever 405.
  it("issues a PATCH, never a PUT", async () => {
    await templatesService.update("t1", fullAttributes());
    expect(patch).toHaveBeenCalledTimes(1);
    expect(put).not.toHaveBeenCalled();
    expect(patch.mock.calls[0]![0]).toBe("/api/configurations/templates/t1");
  });

  it("sends the whole attribute document in a JSON:API envelope", async () => {
    await templatesService.update("t1", fullAttributes());
    const body = patch.mock.calls[0]![1] as {
      data: { id: string; type: string; attributes: TemplateAttributes };
    };
    expect(body.data.type).toBe("templates");
    expect(body.data.id).toBe("t1");
    expect(body.data.attributes.region).toBe("GMS");
    expect(body.data.attributes.majorVersion).toBe(83);
    expect(body.data.attributes.usesPin).toBe(false);
    expect(body.data.attributes.characters).toBeDefined();
    expect(body.data.attributes.worlds).toBeDefined();
  });

  // The guard that stops a sparse/partial document reaching the write path
  // and erasing characters/worlds/cashShop: throwIfInvalid runs before any
  // transport call, so a partial payload never reaches api.patch either.
  it("refuses a partial attribute document", async () => {
    await expect(
      templatesService.update("t1", {
        socket: { handlers: [], writers: [] },
      } as unknown as TemplateAttributes),
    ).rejects.toThrow(/validation failed/i);
    expect(patch).not.toHaveBeenCalled();
    expect(put).not.toHaveBeenCalled();
  });
});

// handleUpdateConfigurationTemplate writes no response body on success, so
// api.patch resolves to undefined; reading response.data off it threw
// "Cannot read properties of undefined (reading 'data')" after the request had
// already succeeded server-side.
describe("templatesService.update bodiless PATCH response", () => {
  beforeEach(() => {
    patch.mockReset().mockResolvedValue(undefined);
    put.mockReset();
    getOne.mockReset();
  });

  it("returns the sent document when the server answers with no body", async () => {
    const attributes = fullAttributes();

    const result = await templatesService.update("t1", attributes);

    expect(result.id).toBe("t1");
    expect(result.attributes.region).toBe("GMS");
    expect(result.attributes.socket.handlers).toEqual([]);
  });

  it("does the same for a sparse patch()", async () => {
    const result = await templatesService.patch("t1", { usesPin: true });

    expect(result.id).toBe("t1");
    expect(result.attributes.usesPin).toBe(true);
  });
});

// templates.RestModel declares NPCs/Worlds without omitempty, so a template
// stored without an `npcs` key (template_jms_185_1.json and four others) is
// served as `"npcs": null`. Feeding that straight back into update() failed
// validation ("NPCs must be an array") and broke every Packet Matrix
// read-modify-write on those templates.
describe("templatesService.getById null-collection normalisation", () => {
  beforeEach(() => {
    patch
      .mockReset()
      .mockResolvedValue({ data: { id: "t1", attributes: fullAttributes() } });
    put.mockReset();
    getOne.mockReset();
  });

  it("reads a null npcs/worlds as an empty array", async () => {
    getOne.mockResolvedValue({
      id: "t1",
      attributes: { ...fullAttributes(), npcs: null, worlds: null },
    });

    const template = await templatesService.getById("t1");

    expect(template.attributes.npcs).toEqual([]);
    expect(template.attributes.worlds).toEqual([]);
  });

  it("round-trips such a template back through update", async () => {
    getOne.mockResolvedValue({
      id: "t1",
      attributes: { ...fullAttributes(), npcs: null, worlds: null },
    });

    const fresh = await templatesService.getById("t1");
    await expect(
      templatesService.update("t1", fresh.attributes),
    ).resolves.toBeDefined();
    expect(patch).toHaveBeenCalledTimes(1);
  });
});
