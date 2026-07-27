import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { isChunkLoadError, withChunkReload } from "../lazy-with-reload";

const CHUNK_ERROR = new Error(
  "Failed to fetch dynamically imported module: http://dev.atlas.home/assets/ReactorsPage-ljr49AsY.js",
);
const RELOAD_STAMP_KEY = "atlas-ui:chunk-reload-at";

describe("isChunkLoadError", () => {
  it.each([
    "Failed to fetch dynamically imported module: http://dev.atlas.home/assets/ReactorsPage-ljr49AsY.js",
    "error loading dynamically imported module",
    "Importing a module script failed.",
    'Failed to load module script: Expected a JavaScript module script but the server responded with a MIME type of "text/html".',
  ])("recognises %s", (message) => {
    expect(isChunkLoadError(new Error(message))).toBe(true);
  });

  it("recognises a bare string rejection", () => {
    expect(
      isChunkLoadError("Failed to fetch dynamically imported module"),
    ).toBe(true);
  });

  it.each([
    new Error("Cannot read properties of undefined (reading 'map')"),
    new Error("Network request failed"),
    new TypeError("x is not a function"),
  ])("does not claim unrelated error: %s", (error) => {
    expect(isChunkLoadError(error)).toBe(false);
  });

  it("does not claim non-error values", () => {
    expect(isChunkLoadError(undefined)).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError({ message: "whatever" })).toBe(false);
  });
});

describe("withChunkReload", () => {
  let reload: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    window.sessionStorage.clear();
    reload = vi.fn();
    originalLocation = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: { ...originalLocation, reload },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: originalLocation,
    });
    window.sessionStorage.clear();
  });

  /** Resolves to "pending" if the promise hasn't settled by the next tick. */
  async function settlement(promise: Promise<unknown>) {
    return Promise.race([
      promise.then(
        () => "resolved",
        () => "rejected",
      ),
      new Promise((resolve) => setTimeout(() => resolve("pending"), 0)),
    ]);
  }

  it("passes a successful import straight through", async () => {
    const mod = { default: () => null };
    const wrapped = withChunkReload(() => Promise.resolve(mod));

    await expect(wrapped()).resolves.toBe(mod);
    expect(reload).not.toHaveBeenCalled();
  });

  it("reloads once on a chunk load error and never settles", async () => {
    const wrapped = withChunkReload(() => Promise.reject(CHUNK_ERROR));

    const result = await settlement(wrapped());

    expect(result).toBe("pending");
    expect(reload).toHaveBeenCalledTimes(1);
    expect(window.sessionStorage.getItem(RELOAD_STAMP_KEY)).not.toBeNull();
  });

  it("suppresses a second reload inside the cooldown and rethrows", async () => {
    window.sessionStorage.setItem(RELOAD_STAMP_KEY, String(Date.now()));
    const wrapped = withChunkReload(() => Promise.reject(CHUNK_ERROR));

    await expect(wrapped()).rejects.toThrow(CHUNK_ERROR);
    expect(reload).not.toHaveBeenCalled();
  });

  it("reloads again once the cooldown has elapsed", async () => {
    window.sessionStorage.setItem(
      RELOAD_STAMP_KEY,
      String(Date.now() - 60_000),
    );
    const wrapped = withChunkReload(() => Promise.reject(CHUNK_ERROR));

    await settlement(wrapped());

    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("rethrows a non-chunk error without reloading", async () => {
    const error = new Error("boom inside the module");
    const wrapped = withChunkReload(() => Promise.reject(error));

    await expect(wrapped()).rejects.toThrow(error);
    expect(reload).not.toHaveBeenCalled();
  });

  it("fails closed when sessionStorage is unavailable", async () => {
    const getItem = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new Error("SecurityError");
      });
    const wrapped = withChunkReload(() => Promise.reject(CHUNK_ERROR));

    await expect(wrapped()).rejects.toThrow(CHUNK_ERROR);
    expect(reload).not.toHaveBeenCalled();

    getItem.mockRestore();
  });
});
