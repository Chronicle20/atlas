/**
 * Stale-chunk recovery for code-split route pages.
 *
 * Route pages in App.tsx are `React.lazy`, so each one ships as its own
 * content-hashed Vite chunk (`assets/ReactorsPage-<hash>.js`). A redeploy
 * replaces `/usr/share/nginx/html/assets/` wholesale, so a tab that loaded the
 * previous build still holds the old module graph and asks for a filename the
 * new image doesn't contain. The import rejects and the route error boundary
 * renders "Failed to fetch dynamically imported module".
 *
 * The recovery is to reload once — the fresh `index.html` (served `no-cache`,
 * see services/atlas-ui/nginx.conf) points at the current chunk hashes.
 *
 * The reload is rate-limited via sessionStorage so a build that is genuinely
 * broken degrades to the error boundary instead of reload-looping, while a
 * deploy that happens later in the same session still self-heals.
 */
import { lazy, type ComponentType, type LazyExoticComponent } from "react";

const RELOAD_STAMP_KEY = "atlas-ui:chunk-reload-at";

/** A second reload inside this window is suppressed — the build is broken. */
const RELOAD_COOLDOWN_MS = 10_000;

/**
 * Browsers word this failure differently; none of them expose a stable error
 * code, so message matching is the only option.
 *
 * - Chrome/Edge: "Failed to fetch dynamically imported module: <url>"
 * - Firefox:     "error loading dynamically imported module"
 * - Safari:      "Importing a module script failed."
 *
 * The MIME-type variant is what our own nginx produced before the `/assets/`
 * location existed: `try_files` fell back to index.html, so a missing chunk
 * came back as `text/html` rather than a 404.
 */
const CHUNK_ERROR_PATTERNS = [
  /failed to fetch dynamically imported module/i,
  /error loading dynamically imported module/i,
  /importing a module script failed/i,
  /expected a javascript(-or-wasm)? module script/i,
];

export function isChunkLoadError(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === "string"
        ? error
        : "";
  return CHUNK_ERROR_PATTERNS.some((pattern) => pattern.test(message));
}

function reloadedRecently(now: number): boolean {
  try {
    const stamp = window.sessionStorage.getItem(RELOAD_STAMP_KEY);
    if (!stamp) return false;
    const at = Number(stamp);
    return Number.isFinite(at) && now - at < RELOAD_COOLDOWN_MS;
  } catch {
    // Private-mode / disabled storage: treat as "already reloaded" so we can
    // never loop. The error boundary still tells the user to refresh.
    return true;
  }
}

function markReloaded(now: number): void {
  try {
    window.sessionStorage.setItem(RELOAD_STAMP_KEY, String(now));
  } catch {
    // Best effort — reloadedRecently() already fails closed.
  }
}

/**
 * Wraps an import factory with the reload-once recovery. Exported for tests —
 * `React.lazy` defers the factory until render, so this is the only seam where
 * the recovery decision can be driven directly.
 *
 * Any rejection that isn't a chunk load failure (a genuine runtime error
 * inside the module, say) is rethrown untouched so it reaches the route error
 * boundary.
 */
export function withChunkReload<T>(
  factory: () => Promise<T>,
): () => Promise<T> {
  return () =>
    factory().catch((error: unknown) => {
      if (!isChunkLoadError(error)) throw error;

      const now = Date.now();
      if (reloadedRecently(now)) throw error;

      markReloaded(now);
      window.location.reload();

      // reload() doesn't halt execution synchronously. Returning a promise
      // that never settles keeps Suspense showing PageLoader until the
      // navigation tears the document down, rather than flashing the error
      // boundary on the way out.
      return new Promise<T>(() => {});
    });
}

/**
 * Drop-in replacement for `React.lazy` that reloads the page once when the
 * chunk fetch fails because the deployed build moved out from under this tab.
 */
export function lazyWithReload<
  T extends ComponentType<Record<string, unknown>>,
>(factory: () => Promise<{ default: T }>): LazyExoticComponent<T> {
  return lazy(withChunkReload(factory));
}
