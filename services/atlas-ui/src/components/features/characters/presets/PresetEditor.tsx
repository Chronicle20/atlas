import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { WorkingPreset, PresetFieldPath } from "./presetEditorState";
import { jobLabel } from "./presetJobs";
import { PresetActionsMenu } from "./PresetActionsMenu";
import { IdentitySection } from "./IdentitySection";
import { ClassAppearanceSection } from "./ClassAppearanceSection";
import { SpawnProgressionSection } from "./SpawnProgressionSection";
import { BaseStatsSection } from "./BaseStatsSection";
import { EquipmentSection } from "./EquipmentSection";
import { InventorySection } from "./InventorySection";
import { SkillsSection } from "./SkillsSection";
import { PresetPreviewCard } from "./PresetPreviewCard";

export interface PresetEditorProps {
  preset: WorkingPreset;
  onBack: () => void;
  onSetField: (path: PresetFieldPath, value: number | string) => void;
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  onAddEquip: (templateId: number) => void;
  onRemoveEquip: (index: number) => void;
  onSetEquipAvg: (index: number, value: boolean) => void;
  onAddInventory: (templateId: number) => void;
  onRemoveInventory: (index: number) => void;
  onSetInventoryQty: (index: number, value: number) => void;
  onAddSkill: (skillId: number) => void;
  onRemoveSkill: (index: number) => void;
  onSetSkillLevel: (index: number, value: number) => void;
  onDuplicate: () => void;
  onRemove: () => void;
  /** Present only in tenant context. */
  onApply?: () => void;
}

export function PresetEditor({
  preset,
  onBack,
  onSetField,
  onAddTag,
  onRemoveTag,
  onAddEquip,
  onRemoveEquip,
  onSetEquipAvg,
  onAddInventory,
  onRemoveInventory,
  onSetInventoryQty,
  onAddSkill,
  onRemoveSkill,
  onSetSkillLevel,
  onDuplicate,
  onRemove,
  onApply,
}: PresetEditorProps) {
  const attrs = preset.attributes;

  return (
    <div className="space-y-4">
      <Button
        type="button"
        variant="ghost"
        aria-label="Preset library"
        onClick={onBack}
      >
        <ArrowLeft className="size-4" /> Preset library
      </Button>

      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-lg font-semibold">{attrs.name}</h2>
        <Badge variant="secondary">{jobLabel(attrs.jobId)}</Badge>
        <span className="text-sm text-muted-foreground">
          Lv {attrs.level}
          {attrs.gm > 0 ? ` · GM ${attrs.gm}` : ""}
        </span>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_252px]">
        <div className="order-2 space-y-6 lg:order-1">
          <IdentitySection
            attrs={attrs}
            onSetField={onSetField}
            onAddTag={onAddTag}
            onRemoveTag={onRemoveTag}
            actions={
              <PresetActionsMenu
                onDuplicate={onDuplicate}
                onRemove={onRemove}
                {...(onApply ? { onApply } : {})}
                canApply={preset.id !== undefined}
                applyDisabledReason="Save this preset before applying"
              />
            }
          />
          <ClassAppearanceSection attrs={attrs} onSetField={onSetField} />
          <SpawnProgressionSection attrs={attrs} onSetField={onSetField} />
          <BaseStatsSection
            attrs={attrs}
            onSetStat={(stat, value) => onSetField(`stats.${stat}`, value)}
          />
          <EquipmentSection
            equipment={attrs.equipment}
            onAdd={onAddEquip}
            onRemove={onRemoveEquip}
            onSetAvg={onSetEquipAvg}
          />
          <InventorySection
            inventory={attrs.inventory}
            onAdd={onAddInventory}
            onRemove={onRemoveInventory}
            onSetQty={onSetInventoryQty}
          />
          <SkillsSection
            skills={attrs.skills}
            onAdd={onAddSkill}
            onRemove={onRemoveSkill}
            onSetLevel={onSetSkillLevel}
          />
        </div>
        <div className="order-1 lg:order-2">
          <PresetPreviewCard attrs={attrs} />
        </div>
      </div>
    </div>
  );
}
