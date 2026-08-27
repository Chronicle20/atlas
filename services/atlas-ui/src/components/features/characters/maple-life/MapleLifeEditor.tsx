import { useEffect, useReducer, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { ErrorDisplay, FormSkeleton } from "@/components/common";
import { useRegisterDetailActionBar } from "@/components/DetailActionBarContext";
import { usePresetJobOptions } from "@/lib/hooks/usePresetJobOptions";
import { validateMapleLife } from "@/lib/schemas/maple-life.schema";
import type { MapleLifeConfig } from "@/types/models/template";
import {
  GENDER_COUNT,
  ORDINAL_COUNT,
  initialMapleLifeState,
  isDirty,
  isEmptyConfig,
  mapleLifeReducer,
  picksFor,
  projectForSave,
  selectedDraft,
  selectedLook,
} from "./mapleLifeEditorState";
import { warningMap } from "./mapleLifeWarnings";
import { ClassSelector } from "./ClassSelector";
import { IdentitySection } from "./IdentitySection";
import { AppearancePoolsSection } from "./AppearancePoolsSection";
import { ProgressionSection } from "./ProgressionSection";
import { SpSkillSection } from "./SpSkillSection";
import { StartingKitSection } from "./StartingKitSection";
import { MapleLifePreviewCard } from "./MapleLifePreviewCard";
import { MapleLifeEmptyState } from "./EmptyState";
import { SeedFromTemplateDialog } from "./SeedFromTemplateDialog";

export interface MapleLifeEditorAdapter {
  mapleLife: MapleLifeConfig | undefined;
  isLoading: boolean;
  error: Error | null;
  /** Fire the context's PATCH; call onSuccess only when it lands. */
  save: (config: MapleLifeConfig, onSuccess: () => void) => void;
  isSaving: boolean;
  /** Tenant context only; absence hides the seed-from-template action. */
  seedFrom?: { region: string; majorVersion: number; minorVersion: number };
}

interface MapleLifeEditorProps {
  adapter: MapleLifeEditorAdapter;
}

/** Pick only the present keys of `keys` out of `record`, in section-prop shape. */
function pickErrors(
  record: Record<string, string[]>,
  keys: readonly string[],
): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const key of keys) {
    const messages = record[key];
    if (messages) out[key] = messages;
  }
  return out;
}

const IDENTITY_KEYS = ["jobId", "level", "mapId"] as const;
const PROGRESSION_KEYS = [
  "sp",
  "ap",
  "meso",
  "str",
  "dex",
  "int",
  "luk",
  "hp",
  "mp",
] as const;
const SP_SKILL_KEYS = ["spSkillId"] as const;

export function MapleLifeEditor({ adapter }: MapleLifeEditorProps) {
  const [state, dispatch] = useReducer(
    mapleLifeReducer,
    undefined,
    initialMapleLifeState,
  );
  const [searchParams, setSearchParams] = useSearchParams();
  const [seedOpen, setSeedOpen] = useState(false);

  const jobs = usePresetJobOptions();
  const jobNameById =
    jobs.isPending || jobs.isError
      ? null
      : new Map(jobs.options.map((o) => [o.id, o.name]));

  // Seed exactly once. The `loaded` guard is what keeps a post-save
  // invalidation refetch (adapter.mapleLife changing identity) from
  // clobbering the in-progress working copy: after the first seed the
  // reducer is authoritative and re-seeding is skipped. `!adapter.isLoading
  // && !adapter.error` also fires the load when the query SETTLES with
  // `undefined` (no mapleLife block for this tenant/template), so an
  // unconfigured tenant reaches the empty state instead of sitting on a
  // permanent skeleton -- but a query that settled into an ERROR still
  // blocks the seed, so the pre-load gate below shows ErrorDisplay rather
  // than silently treating "failed to load" as "nothing configured".
  useEffect(() => {
    if (state.loaded) return;
    if (
      adapter.mapleLife !== undefined ||
      (!adapter.isLoading && !adapter.error)
    ) {
      dispatch({ type: "load", config: adapter.mapleLife });
    }
  }, [adapter.mapleLife, adapter.isLoading, adapter.error, state.loaded]);

  // Deep-link: apply ?ordinal=/?gender= to the selection ONCE, when the
  // adapter first seeds the reducer. Parses both, clamps out-of-range or
  // unparseable values to 0, and writes both clamped values back together
  // with { replace: true }. Runs on load only (deps: [state.loaded]) and NOT
  // on selection changes: every click goes through select() -> syncSelection,
  // which owns URL/selection agreement directly. The grid is a fixed 5x2, so
  // unlike the templates editor there is no length to re-watch on mutation.
  useEffect(() => {
    if (!state.loaded) return;
    const rawOrdinal = searchParams.get("ordinal") ?? "0";
    const rawGender = searchParams.get("gender") ?? "0";
    const parsedOrdinal = Number.parseInt(rawOrdinal, 10);
    const parsedGender = Number.parseInt(rawGender, 10);
    const clampedOrdinal =
      Number.isFinite(parsedOrdinal) &&
      parsedOrdinal >= 0 &&
      parsedOrdinal <= ORDINAL_COUNT - 1
        ? parsedOrdinal
        : 0;
    const clampedGender =
      Number.isFinite(parsedGender) &&
      parsedGender >= 0 &&
      parsedGender <= GENDER_COUNT - 1
        ? parsedGender
        : 0;
    if (clampedOrdinal !== state.ordinal || clampedGender !== state.gender) {
      dispatch({
        type: "select",
        ordinal: clampedOrdinal,
        gender: clampedGender,
      });
    }
    if (
      String(clampedOrdinal) !== rawOrdinal ||
      String(clampedGender) !== rawGender
    ) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set("ordinal", String(clampedOrdinal));
          next.set("gender", String(clampedGender));
          return next;
        },
        { replace: true },
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deep-link apply on load only; selection changes own URL sync via syncSelection
  }, [state.loaded]);

  const syncSelection = (ordinal: number, gender: number) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set("ordinal", String(ordinal));
        next.set("gender", String(gender));
        return next;
      },
      { replace: true },
    );
  };

  const select = (ordinal: number, gender: number) => {
    dispatch({ type: "select", ordinal, gender });
    syncSelection(ordinal, gender);
  };

  const discardChanges = () => {
    dispatch({ type: "discard" });
  };

  const projected = projectForSave(state);
  const issues = validateMapleLife(projected);
  const warnings = warningMap(state);
  let blockingIssues = 0;
  for (const messages of issues.values()) blockingIssues += messages.length;

  const dirty = isDirty(state);
  const isEmpty = isEmptyConfig(projected);

  // Drive the shared detail-page action bar instead of a local Save/Discard
  // bar. Registers null while loading/empty so the bar stays hidden until
  // there is a config to save. No post-save refetch: nothing in mapleLife is
  // server-assigned (no ids), so the request echo is already the truth.
  useRegisterDetailActionBar(
    state.loaded && !isEmpty
      ? {
          dirty,
          isSaving: adapter.isSaving,
          blockingIssues,
          onSave: () =>
            adapter.save(projected, () => dispatch({ type: "savedOk" })),
          onDiscard: discardChanges,
        }
      : null,
  );

  // Seed-once gate: only the pre-load window shows skeleton/error, so a
  // transient refetch or save error never blanks an in-progress working copy.
  if (!state.loaded) {
    if (adapter.error) {
      return <ErrorDisplay error={adapter.error} />;
    }
    return <FormSkeleton fields={8} />;
  }

  if (isEmpty) {
    return (
      <>
        <MapleLifeEmptyState
          {...(adapter.seedFrom ? { onSeed: () => setSeedOpen(true) } : {})}
          onAddRows={() => dispatch({ type: "materialiseAll" })}
        />
        {adapter.seedFrom && (
          <SeedFromTemplateDialog
            open={seedOpen}
            onOpenChange={setSeedOpen}
            seedFrom={adapter.seedFrom}
            onSeed={(config) => dispatch({ type: "seedFromTemplate", config })}
          />
        )}
      </>
    );
  }

  // Re-key the schema's issues (which address the PROJECTED array) onto the
  // FIXED 5x2 grid (which the warnings already address), using each
  // projected entry's own ordinal/gender fields as the lookup -- never
  // merged with the warning map.
  const classGridErrors = new Map<string, Record<string, string[]>>();
  const lookGridErrors = new Map<number, Record<string, string[]>>();
  for (const [path, messages] of issues) {
    const classMatch = /^classes\.(\d+)\.(.+)$/.exec(path);
    if (classMatch) {
      const entry = projected.classes[Number(classMatch[1])];
      const field = classMatch[2]!;
      if (entry) {
        const key = `classes.${entry.ordinal}.${entry.gender}`;
        const bucket = classGridErrors.get(key) ?? {};
        const bareField = field.startsWith("stats.")
          ? field.slice("stats.".length)
          : field;
        bucket[bareField] = [...(bucket[bareField] ?? []), ...messages];
        classGridErrors.set(key, bucket);
      }
      continue;
    }
    const lookMatch = /^looks\.(\d+)\.(.+)$/.exec(path);
    if (lookMatch) {
      const entry = projected.looks[Number(lookMatch[1])];
      const dimension = lookMatch[2]!;
      if (entry) {
        const bucket = lookGridErrors.get(entry.gender) ?? {};
        bucket[dimension] = [...(bucket[dimension] ?? []), ...messages];
        lookGridErrors.set(entry.gender, bucket);
      }
    }
    // Untargeted paths (e.g. "looks" for a missing looks row) still count
    // toward blockingIssues above but have no single grid cell to render in.
  }

  const rowErrors =
    classGridErrors.get(`classes.${state.ordinal}.${state.gender}`) ?? {};
  const identityErrors = pickErrors(rowErrors, IDENTITY_KEYS);
  const progressionErrors = pickErrors(rowErrors, PROGRESSION_KEYS);
  const spSkillErrors = pickErrors(rowErrors, SP_SKILL_KEYS);
  const looksErrors = lookGridErrors.get(state.gender) ?? {};
  const spSkillWarnings =
    warnings.get(`classes.${state.ordinal}.${state.gender}.spSkillId`) ?? [];

  const draft = selectedDraft(state);
  const look = selectedLook(state);
  const picks = picksFor(state, state.gender);

  return (
    <div className="space-y-4">
      <ClassSelector
        ordinal={state.ordinal}
        gender={state.gender}
        drafts={state.drafts}
        jobNameById={jobNameById}
        onSelect={select}
      />
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_252px]">
        <div className="order-2 space-y-6 lg:order-1">
          <IdentitySection
            draft={draft}
            jobs={jobs}
            onSetIdentity={(field, value) =>
              dispatch({ type: "setIdentity", field, value })
            }
            onSetLevel={(value) =>
              dispatch({ type: "setScalar", field: "level", value })
            }
            errors={identityErrors}
          />
          <AppearancePoolsSection
            look={look}
            draft={draft}
            picks={picks}
            onPick={(pick, index) =>
              dispatch({ type: "setPreviewPick", pick, value: index })
            }
            onAddEntry={(dimension, id) =>
              dispatch({ type: "addLookEntry", dimension, id })
            }
            onRemoveEntry={(dimension, entryIndex) =>
              dispatch({ type: "removeLookEntry", dimension, entryIndex })
            }
            errors={looksErrors}
          />
          <ProgressionSection
            draft={draft}
            onSetStat={(stat, value) =>
              dispatch({ type: "setStat", stat, value })
            }
            onSetScalar={(field, value) =>
              dispatch({ type: "setScalar", field, value })
            }
            onSetSpBook={(index, value) =>
              dispatch({ type: "setSpBook", index, value })
            }
            errors={progressionErrors}
          />
          <SpSkillSection
            draft={draft}
            onSetSpSkillId={(value) =>
              dispatch({ type: "setSpSkillId", value })
            }
            errors={spSkillErrors}
            warnings={spSkillWarnings}
          />
          <StartingKitSection
            equipment={draft.equipment}
            inventory={draft.inventory}
            onAddEquipment={(templateId) =>
              dispatch({ type: "addEquipment", templateId })
            }
            onRemoveEquipment={(entryIndex) =>
              dispatch({ type: "removeEquipment", entryIndex })
            }
            onSetEquipmentAvg={(entryIndex, value) =>
              dispatch({ type: "setEquipmentAvg", entryIndex, value })
            }
            onAddInventory={(templateId) =>
              dispatch({ type: "addInventory", templateId })
            }
            onRemoveInventory={(entryIndex) =>
              dispatch({ type: "removeInventory", entryIndex })
            }
            onSetInventoryQty={(entryIndex, value) =>
              dispatch({ type: "setInventoryQty", entryIndex, value })
            }
          />
        </div>
        <div className="order-1 lg:order-2">
          <MapleLifePreviewCard draft={draft} look={look} picks={picks} />
        </div>
      </div>
    </div>
  );
}
