import { useEffect, useReducer, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import type { CharacterPreset } from "@/types/models/template";
import type { Tenant } from "@/types/models/tenant";
import { ErrorDisplay, FormSkeleton } from "@/components/common";
import { useRegisterDetailActionBar } from "@/components/DetailActionBarContext";
import { presetSchema } from "@/lib/schemas/character-presets.schema";
import {
  presetReducer,
  initialPresetEditorState,
  isDirty,
  presetDirty,
  projectForSave,
  selectedPreset,
} from "./presetEditorState";
import { PresetLibrary } from "./PresetLibrary";
import { PresetEditor } from "./PresetEditor";
import { AccountPickerDialog } from "./AccountPickerDialog";
import { ApplyPresetDialog } from "@/components/features/characters/ApplyPresetDialog";

export interface PresetsEditorAdapter {
  presets: CharacterPreset[] | undefined;
  isLoading: boolean;
  error: Error | null;
  /** Fire the context's PATCH; call onSuccess only when it lands. */
  save: (presets: CharacterPreset[], onSuccess: () => void) => void;
  isSaving: boolean;
  /** Present only in tenant context; absent on the template page hides Apply. */
  apply?: { tenant: Tenant };
}

interface CharacterPresetsEditorProps {
  adapter: PresetsEditorAdapter;
}

const PRESET_PARAM = "preset";

export function CharacterPresetsEditor({ adapter }: CharacterPresetsEditorProps) {
  const [state, dispatch] = useReducer(
    presetReducer,
    undefined,
    initialPresetEditorState,
  );
  const [searchParams, setSearchParams] = useSearchParams();

  // Apply orchestration (tenant context only).
  const [applyKey, setApplyKey] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [applyAccountId, setApplyAccountId] = useState<number | null>(null);
  const [applyDialogOpen, setApplyDialogOpen] = useState(false);

  // Seed exactly once, when the adapter first delivers data. The `loaded`
  // guard is what keeps a post-save invalidation refetch (adapter.presets
  // changing identity) from clobbering the in-progress working copy: after
  // the first seed the reducer is authoritative and re-seeding is skipped.
  useEffect(() => {
    if (adapter.presets && !state.loaded) {
      dispatch({ type: "load", presets: adapter.presets });
    }
  }, [adapter.presets, state.loaded]);

  // Deep-link: apply ?preset= to the selection ONCE, when the reducer first
  // loads. Resolves against a working preset by `id` then `key`; an
  // unresolvable value is left alone (library view, no error).
  //
  // Runs on load only (deps: [state.loaded]) and NOT on preset-count changes:
  // every internal mutation (open/add/duplicate/remove/discard) already calls
  // syncSelection() with the reducer's own post-mutation selection, so it owns
  // URL/selection agreement. Re-running this effect on every state change
  // would race the router the same way task-177's length-watcher did: a
  // reducer-first render could re-read a stale ?preset and clobber a
  // selection the handler already resolved.
  useEffect(() => {
    if (!state.loaded) return;
    const raw = searchParams.get(PRESET_PARAM);
    if (!raw) return;
    const match =
      state.presets.find((p) => p.id === raw) ??
      state.presets.find((p) => p.key === raw);
    if (match) {
      dispatch({ type: "select", key: match.key });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deep-link apply on load only; internal mutations own URL sync via syncSelection
  }, [state.loaded]);

  // Owns all URL writes for selection. Prefers the working preset's `id` in
  // the URL, falling back to its `key` (freshly added/duplicated presets have
  // no `id` yet). `null` returns to the library and clears the param.
  const syncSelection = (key: string | null) => {
    const urlValue =
      key === null ? null : (state.presets.find((p) => p.key === key)?.id ?? key);
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (urlValue === null) {
          next.delete(PRESET_PARAM);
        } else {
          next.set(PRESET_PARAM, urlValue);
        }
        return next;
      },
      { replace: true },
    );
  };

  const open = (key: string) => {
    dispatch({ type: "select", key });
    syncSelection(key);
  };
  const back = () => {
    dispatch({ type: "select", key: null });
    syncSelection(null);
  };
  const newPreset = () => {
    const key = `local-${state.localSeq}`;
    dispatch({ type: "addPreset" });
    syncSelection(key);
  };
  const duplicate = (key: string) => {
    const newKey = `local-${state.localSeq}`;
    dispatch({ type: "duplicatePreset", key });
    syncSelection(newKey);
  };
  const remove = (key: string) => {
    dispatch({ type: "removePreset", key });
    syncSelection(null);
  };
  const discard = () => {
    dispatch({ type: "discard" });
    syncSelection(null);
  };

  const dirty = isDirty(state);

  const onSave = () => {
    const projected = projectForSave(state);
    for (const p of projected) {
      const result = presetSchema.safeParse(p);
      if (!result.success) {
        const firstIssue = result.error.issues[0];
        const label = p.attributes.name || "Preset";
        toast.error(
          firstIssue
            ? `${label}: ${firstIssue.message}`
            : `${label}: validation failed`,
        );
        return;
      }
    }
    adapter.save(projected, () => dispatch({ type: "savedOk" }));
  };

  // Drive the shared detail-page action bar instead of a local Save/Discard
  // bar. Registers null while loading so the bar stays hidden until there is
  // a working copy to save.
  useRegisterDetailActionBar(
    state.loaded
      ? {
          dirty,
          isSaving: adapter.isSaving,
          onSave,
          onDiscard: discard,
        }
      : null,
  );

  const startApply = (key: string) => {
    if (!adapter.apply) return;
    const target = state.presets.find((p) => p.key === key);
    if (!target) return;
    if (!target.id) {
      toast.error("Save this preset before applying.");
      return;
    }
    if (presetDirty(state, key)) {
      toast.warning("Apply uses the last saved version of this preset.");
    }
    setApplyKey(key);
    setPickerOpen(true);
  };

  const handleAccountPicked = (accountId: number) => {
    setApplyAccountId(accountId);
    setApplyDialogOpen(true);
  };

  const handleApplyDialogOpenChange = (nextOpen: boolean) => {
    setApplyDialogOpen(nextOpen);
    if (!nextOpen) {
      setApplyAccountId(null);
      setApplyKey(null);
    }
  };

  // Seed-once gate: only the pre-load window shows skeleton/error, so a
  // transient refetch or save error never blanks an in-progress working copy.
  if (!state.loaded) {
    if (adapter.error) {
      return <ErrorDisplay error={adapter.error} />;
    }
    return <FormSkeleton fields={6} />;
  }

  const selected = selectedPreset(state);
  const dirtyKeys = new Set(
    state.presets.filter((p) => presetDirty(state, p.key)).map((p) => p.key),
  );
  const applyPresetId = applyKey
    ? state.presets.find((p) => p.key === applyKey)?.id
    : undefined;

  return (
    <>
      {selected ? (
        <PresetEditor
          preset={selected}
          onBack={back}
          onSetField={(path, value) =>
            dispatch({ type: "setField", key: selected.key, path, value })
          }
          onAddTag={(tag) => dispatch({ type: "addTag", key: selected.key, tag })}
          onRemoveTag={(tag) =>
            dispatch({ type: "removeTag", key: selected.key, tag })
          }
          onAddEquip={(templateId) =>
            dispatch({ type: "addEquip", key: selected.key, templateId })
          }
          onRemoveEquip={(index) =>
            dispatch({ type: "removeEquip", key: selected.key, index })
          }
          onSetEquipAvg={(index, value) =>
            dispatch({ type: "setEquipAvg", key: selected.key, index, value })
          }
          onAddInventory={(templateId) =>
            dispatch({ type: "addInventory", key: selected.key, templateId })
          }
          onRemoveInventory={(index) =>
            dispatch({ type: "removeInventory", key: selected.key, index })
          }
          onSetInventoryQty={(index, value) =>
            dispatch({
              type: "setInventoryQty",
              key: selected.key,
              index,
              value,
            })
          }
          onAddSkill={(skillId) =>
            dispatch({ type: "addSkill", key: selected.key, skillId })
          }
          onRemoveSkill={(index) =>
            dispatch({ type: "removeSkill", key: selected.key, index })
          }
          onSetSkillLevel={(index, value) =>
            dispatch({ type: "setSkillLevel", key: selected.key, index, value })
          }
          onDuplicate={() => duplicate(selected.key)}
          onRemove={() => remove(selected.key)}
          {...(adapter.apply
            ? { onApply: () => startApply(selected.key) }
            : {})}
        />
      ) : (
        <PresetLibrary
          presets={state.presets}
          dirtyKeys={dirtyKeys}
          canApply={!!adapter.apply}
          onOpen={open}
          onNew={newPreset}
          onDuplicate={duplicate}
          onApply={startApply}
        />
      )}
      {adapter.apply && (
        <>
          <AccountPickerDialog
            tenant={adapter.apply.tenant}
            open={pickerOpen}
            onOpenChange={setPickerOpen}
            onPick={handleAccountPicked}
          />
          {applyAccountId !== null && (
            <ApplyPresetDialog
              tenant={adapter.apply.tenant}
              accountId={applyAccountId}
              open={applyDialogOpen}
              onOpenChange={handleApplyDialogOpenChange}
              {...(applyPresetId ? { initialPresetId: applyPresetId } : {})}
            />
          )}
        </>
      )}
    </>
  );
}
