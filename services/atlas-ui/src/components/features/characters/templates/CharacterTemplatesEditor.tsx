import { useEffect, useMemo, useReducer } from "react";
import { useSearchParams } from "react-router-dom";
import type { CharacterTemplate } from "@/types/models/template";
import { EmptyState, ErrorDisplay, FormSkeleton } from "@/components/common";
import {
  editorReducer,
  initialEditorState,
  isDirty,
  picksFor,
  type PreviewPicks,
  type AppearancePoolKey,
} from "./editorState";
import { templateLabels } from "./jobNames";
import { TemplateSelector } from "./TemplateSelector";
import { TemplateActionsMenu } from "./TemplateActionsMenu";
import { IdentitySection } from "./IdentitySection";
import { AppearancePoolSection } from "./AppearancePoolSection";
import { AppearanceBrowserDialog } from "./AppearanceBrowserDialog";
import { EquipmentPoolSection } from "./EquipmentPoolSection";
import { StartingKitSection } from "./StartingKitSection";
import { PreviewCard } from "./PreviewCard";
import { SaveBar } from "./SaveBar";
import type { EquipmentPoolKey } from "./previewLoadout";

export interface TemplatesEditorAdapter {
  templates: CharacterTemplate[] | undefined;
  isLoading: boolean;
  error: Error | null;
  /** Fire the context's PATCH; call onSuccess only when it lands. */
  save: (templates: CharacterTemplate[], onSuccess: () => void) => void;
  isSaving: boolean;
}

interface CharacterTemplatesEditorProps {
  adapter: TemplatesEditorAdapter;
}

const APPEARANCE_SECTIONS: { dimension: AppearancePoolKey; title: string }[] = [
  { dimension: "faces", title: "Faces" },
  { dimension: "hairs", title: "Hairs" },
  { dimension: "hairColors", title: "Hair colors" },
  { dimension: "skinColors", title: "Skin tones" },
];

const EQUIPMENT_SECTIONS: { poolKey: EquipmentPoolKey; title: string }[] = [
  { poolKey: "tops", title: "Tops" },
  { poolKey: "bottoms", title: "Bottoms" },
  { poolKey: "shoes", title: "Shoes" },
  { poolKey: "weapons", title: "Weapons" },
];

export function CharacterTemplatesEditor({
  adapter,
}: CharacterTemplatesEditorProps) {
  const [state, dispatch] = useReducer(
    editorReducer,
    undefined,
    initialEditorState,
  );
  const [searchParams, setSearchParams] = useSearchParams();

  // Seed exactly once, when the adapter first delivers data. The `loaded`
  // guard is what keeps a post-save invalidation refetch (adapter.templates
  // changing identity) from clobbering the in-progress working copy: after
  // the first seed the reducer is authoritative and re-seeding is skipped.
  useEffect(() => {
    if (adapter.templates && !state.loaded) {
      dispatch({ type: "load", templates: adapter.templates });
    }
  }, [adapter.templates, state.loaded]);

  // URL -> selection on load and whenever the template count changes. Reads
  // ?tpl=, clamps invalid / out-of-range values to 0, and writes the clamped
  // value back with { replace: true }. Both mutations are guarded so the
  // effect settles after one pass (no render loop): the dispatch only fires
  // when the clamped index actually differs from the selection, and the URL
  // write only fires when the serialized value actually differs.
  useEffect(() => {
    if (!state.loaded) return;
    const raw = searchParams.get("tpl") ?? "0";
    const parsed = Number.parseInt(raw, 10);
    const inRange =
      Number.isFinite(parsed) &&
      parsed >= 0 &&
      parsed <= state.templates.length - 1;
    const clamped = inRange ? parsed : 0;
    if (clamped !== state.selectedIndex) {
      dispatch({ type: "select", index: clamped });
    }
    if (String(clamped) !== raw) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set("tpl", String(clamped));
          return next;
        },
        { replace: true },
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- run on load + template-count changes only
  }, [state.loaded, state.templates.length]);

  const syncSelection = (index: number) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set("tpl", String(index));
        return next;
      },
      { replace: true },
    );
  };

  const select = (index: number) => {
    dispatch({ type: "select", index });
    syncSelection(index);
  };
  const addTemplate = () => {
    dispatch({ type: "addTemplate" });
    syncSelection(state.templates.length); // new template's index
  };
  const duplicateTemplate = () => {
    dispatch({ type: "duplicateTemplate" });
    syncSelection(state.templates.length);
  };
  const removeTemplate = () => {
    dispatch({ type: "removeTemplate" });
    syncSelection(
      Math.min(state.selectedIndex, Math.max(state.templates.length - 2, 0)),
    );
  };

  const dirty = useMemo(() => isDirty(state), [state]);

  // Seed-once gate: only the pre-load window shows skeleton/error, so a
  // transient refetch or save error never blanks an in-progress working copy.
  if (!state.loaded) {
    if (adapter.error) {
      return <ErrorDisplay error={adapter.error} />;
    }
    return <FormSkeleton fields={6} />;
  }

  const template = state.templates[state.selectedIndex];
  const picks = picksFor(state, state.selectedIndex);
  const labels = templateLabels(state.templates);

  if (state.templates.length === 0) {
    return (
      <EmptyState
        title="No character templates"
        description="Templates define which classes, looks, and starting gear players can pick at character creation. Add one to get started."
        action={{ label: "Add template", onClick: addTemplate }}
      />
    );
  }

  return (
    <div className="space-y-4">
      <TemplateSelector
        templates={state.templates}
        selectedIndex={state.selectedIndex}
        onSelect={select}
        onAdd={addTemplate}
      />
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_252px]">
        <div className="order-2 space-y-6 lg:order-1">
          {template && (
            <>
              <IdentitySection
                template={template}
                onSetIdentity={(field, value) =>
                  dispatch({ type: "setIdentity", field, value })
                }
                actions={
                  <TemplateActionsMenu
                    label={labels[state.selectedIndex] ?? ""}
                    onDuplicate={duplicateTemplate}
                    onRemove={removeTemplate}
                  />
                }
              />
              {APPEARANCE_SECTIONS.map(({ dimension, title }) => (
                <AppearancePoolSection
                  key={dimension}
                  dimension={dimension}
                  title={title}
                  template={template}
                  picks={picks}
                  onPick={(pick: keyof PreviewPicks, idx: number) =>
                    dispatch({ type: "setPreviewPick", pick, value: idx })
                  }
                  onRemoveEntry={(entryIndex) =>
                    dispatch({
                      type: "removePoolEntry",
                      pool: dimension,
                      entryIndex,
                    })
                  }
                  renderAddDialog={(open, onOpenChange) => (
                    <AppearanceBrowserDialog
                      dimension={dimension}
                      template={template}
                      picks={picks}
                      open={open}
                      onOpenChange={onOpenChange}
                      onAdd={(id) =>
                        dispatch({ type: "addPoolEntry", pool: dimension, id })
                      }
                    />
                  )}
                />
              ))}
              {EQUIPMENT_SECTIONS.map(({ poolKey, title }) => (
                <EquipmentPoolSection
                  key={poolKey}
                  poolKey={poolKey}
                  title={title}
                  ids={template[poolKey]}
                  onAdd={(id) =>
                    dispatch({ type: "addPoolEntry", pool: poolKey, id })
                  }
                  onRemove={(entryIndex) =>
                    dispatch({
                      type: "removePoolEntry",
                      pool: poolKey,
                      entryIndex,
                    })
                  }
                />
              ))}
              <StartingKitSection
                items={template.items}
                skills={template.skills}
                onAddItem={(id) =>
                  dispatch({ type: "addPoolEntry", pool: "items", id })
                }
                onRemoveItem={(entryIndex) =>
                  dispatch({
                    type: "removePoolEntry",
                    pool: "items",
                    entryIndex,
                  })
                }
                onAddSkill={(id) =>
                  dispatch({ type: "addPoolEntry", pool: "skills", id })
                }
                onRemoveSkill={(entryIndex) =>
                  dispatch({
                    type: "removePoolEntry",
                    pool: "skills",
                    entryIndex,
                  })
                }
              />
            </>
          )}
        </div>
        <div className="order-1 lg:order-2">
          {template && <PreviewCard template={template} picks={picks} />}
        </div>
      </div>
      <SaveBar
        dirty={dirty}
        isSaving={adapter.isSaving}
        onSave={() =>
          adapter.save(state.templates, () => dispatch({ type: "savedOk" }))
        }
        onDiscard={() => dispatch({ type: "discard" })}
      />
    </div>
  );
}
