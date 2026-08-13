import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { downloadJson } from "@/lib/utils/download-json";

describe("downloadJson", () => {
  let createObjectURL: ReturnType<typeof vi.fn>;
  let revokeObjectURL: ReturnType<typeof vi.fn>;
  let blobs: Blob[];
  let clickSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    // jsdom implements neither of these (src/test/setup.ts stubs only
    // matchMedia and ResizeObserver), so they are stubbed per-suite rather
    // than globally - a global stub would silently mask their absence in
    // unrelated suites.
    blobs = [];
    createObjectURL = vi.fn((blob: Blob) => {
      blobs.push(blob);
      return "blob:mock-url";
    });
    revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL,
      revokeObjectURL,
    });
    clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clickSpy.mockRestore();
  });

  it("writes a pretty-printed JSON blob with a trailing newline", async () => {
    downloadJson("template_gms_83_1.json", { region: "GMS", n: [1, 2] });

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(blobs).toHaveLength(1);
    expect(blobs[0]?.type).toBe("application/json");
    await expect(blobs[0]?.text()).resolves.toBe(
      `${JSON.stringify({ region: "GMS", n: [1, 2] }, null, 2)}\n`,
    );
  });

  it("clicks an anchor carrying the requested filename", () => {
    let downloadAttr: string | null = null;
    clickSpy.mockImplementation(function (this: HTMLAnchorElement) {
      downloadAttr = this.download;
    });

    downloadJson("tenant_gms_83_1.json", {});

    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(downloadAttr).toBe("tenant_gms_83_1.json");
  });

  it("revokes the object URL and leaves no anchor behind", () => {
    downloadJson("template_gms_83_1.json", {});

    expect(revokeObjectURL).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    expect(document.body.querySelector("a")).toBeNull();
  });

  it("propagates a serialisation failure without creating an object URL", () => {
    const cyclic: Record<string, unknown> = {};
    cyclic.self = cyclic;

    expect(() => downloadJson("x.json", cyclic)).toThrow();
    expect(createObjectURL).not.toHaveBeenCalled();
    expect(revokeObjectURL).not.toHaveBeenCalled();
    expect(document.body.querySelector("a")).toBeNull();
  });
});
