import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { SaveBar } from "@/components/features/characters/templates/SaveBar";

/**
 * A detail page's dirty/save/discard state, surfaced in the shared detail-page
 * action bar (rendered once by TenantDetailLayout / TemplateDetailLayout, pinned
 * below the scrolling content). Any detail page can drive the common bar by
 * calling {@link useRegisterDetailActionBar}; pages that don't register show no
 * bar, so adoption is opt-in and per-page.
 */
export interface DetailActionBarConfig {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /** Count of blocking validation errors. > 0 disables Save and is reported
   *  in the bar. Omitted means "no validation gate". */
  blockingIssues?: number | undefined;
}

interface DetailActionBarContextValue {
  config: DetailActionBarConfig | null;
  register: (config: DetailActionBarConfig | null) => void;
}

const DetailActionBarContext =
  createContext<DetailActionBarContextValue | null>(null);

export function DetailActionBarProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<DetailActionBarConfig | null>(null);
  const register = useCallback(
    (next: DetailActionBarConfig | null) => setConfig(next),
    [],
  );
  return (
    <DetailActionBarContext.Provider value={{ config, register }}>
      {children}
    </DetailActionBarContext.Provider>
  );
}

/**
 * Register the calling page's save/discard state with the shared action bar.
 * Pass `null` to hide the bar. Callbacks are always invoked at their latest
 * identity (kept in a ref), so the bar never fires a stale closure; the bar is
 * re-pushed only when `dirty`/`isSaving`/presence change, avoiding render loops.
 */
export function useRegisterDetailActionBar(
  config: DetailActionBarConfig | null,
): void {
  const ctx = useContext(DetailActionBarContext);
  const register = ctx?.register;
  const latest = useRef(config);
  // Keep the latest callbacks/state in a ref (updated post-render) so the bar
  // always invokes the current onSave/onDiscard, never a stale closure.
  useEffect(() => {
    latest.current = config;
  });

  const present = config !== null;
  const dirty = config?.dirty ?? false;
  const isSaving = config?.isSaving ?? false;
  const blockingIssues = config?.blockingIssues;

  useEffect(() => {
    if (!register) return;
    if (!present) {
      register(null);
      return;
    }
    register({
      dirty,
      isSaving,
      blockingIssues,
      onSave: () => latest.current?.onSave(),
      onDiscard: () => latest.current?.onDiscard(),
    });
    return () => register(null);
  }, [register, present, dirty, isSaving, blockingIssues]);
}

/**
 * The shared action bar. Rendered once by the detail layout below the scrolling
 * content; renders nothing until a page registers via useRegisterDetailActionBar.
 */
export function DetailActionBar() {
  const ctx = useContext(DetailActionBarContext);
  const config = ctx?.config;
  if (!config) return null;
  return (
    <SaveBar
      dirty={config.dirty}
      isSaving={config.isSaving}
      blockingIssues={config.blockingIssues}
      onSave={config.onSave}
      onDiscard={config.onDiscard}
    />
  );
}

/**
 * Read the shared action bar's current state without registering anything.
 * Used by controls that render beside the bar's owner and need to know whether
 * the page has unsaved edits - e.g. the header Export button, which exports the
 * last SAVED document and says so while the page is dirty. Returns null outside
 * a DetailActionBarProvider, or when no page has registered.
 */
export function useDetailActionBarState(): DetailActionBarConfig | null {
  const ctx = useContext(DetailActionBarContext);
  return ctx?.config ?? null;
}
