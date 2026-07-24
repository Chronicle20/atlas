import { useState, type ReactNode } from "react";
import type { CharacterTemplate } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { KNOWN_CLASSES, worldNameFromJobIndex } from "./jobNames";
import type { IdentityField } from "./editorState";
import { MapPicker } from "./MapPicker";

interface IdentitySectionProps {
  template: CharacterTemplate;
  onSetIdentity: (field: IdentityField, value: number) => void;
  /** Rendered top-right of the section header (kebab anchor — FR-3.1). */
  actions?: ReactNode;
}

export function IdentitySection({
  template,
  onSetIdentity,
  actions,
}: IdentitySectionProps) {
  const [advanced, setAdvanced] = useState(false);

  const classValue = `${template.jobIndex}.${template.subJobIndex}`;
  const known = KNOWN_CLASSES.find(
    (c) =>
      c.jobIndex === template.jobIndex &&
      c.subJobIndex === template.subJobIndex,
  );
  const classLabel =
    known?.label ??
    `${worldNameFromJobIndex(template.jobIndex)} (${classValue})`;

  const parseNumeric = (raw: string): number | undefined => {
    const n = Number(raw);
    return raw.trim() !== "" && Number.isFinite(n) ? n : undefined;
  };

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Identity</h3>
        {actions}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label htmlFor="tpl-class">Class</Label>
          <Select
            value={known ? classValue : ""}
            onValueChange={(v) => {
              const [jobRaw, subRaw] = v.split(".");
              const job = Number(jobRaw ?? 0);
              const sub = Number(subRaw ?? 0);
              onSetIdentity("jobIndex", job);
              onSetIdentity("subJobIndex", sub);
            }}
          >
            <SelectTrigger id="tpl-class" aria-label="Class">
              <SelectValue placeholder={classLabel}>{classLabel}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {KNOWN_CLASSES.map((c) => (
                <SelectItem
                  key={c.label}
                  value={`${c.jobIndex}.${c.subJobIndex}`}
                >
                  {c.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            variant="link"
            size="sm"
            className="h-auto p-0 text-xs"
            onClick={() => setAdvanced((a) => !a)}
          >
            Advanced
          </Button>
          {advanced && (
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1">
                <Label htmlFor="tpl-job-index" className="text-xs">
                  Job index
                </Label>
                <Input
                  id="tpl-job-index"
                  inputMode="numeric"
                  defaultValue={template.jobIndex}
                  onChange={(e) => {
                    const n = parseNumeric(e.target.value);
                    if (n !== undefined) onSetIdentity("jobIndex", n);
                  }}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="tpl-subjob-index" className="text-xs">
                  Sub job index
                </Label>
                <Input
                  id="tpl-subjob-index"
                  inputMode="numeric"
                  defaultValue={template.subJobIndex}
                  onChange={(e) => {
                    const n = parseNumeric(e.target.value);
                    if (n !== undefined) onSetIdentity("subJobIndex", n);
                  }}
                />
              </div>
            </div>
          )}
        </div>
        <div className="space-y-1">
          <Label htmlFor="tpl-gender">Gender</Label>
          <Select
            value={String(template.gender)}
            onValueChange={(v) => onSetIdentity("gender", Number(v))}
          >
            <SelectTrigger id="tpl-gender" aria-label="Gender">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">Male</SelectItem>
              <SelectItem value="1">Female</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1 sm:col-span-2">
          <Label>Starting map</Label>
          <MapPicker
            value={template.mapId}
            onChange={(mapId) => onSetIdentity("mapId", mapId)}
          />
        </div>
      </div>
    </section>
  );
}
