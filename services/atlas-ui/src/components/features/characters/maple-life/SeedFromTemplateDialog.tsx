import { useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useTemplates } from "@/lib/hooks/api/useTemplates";
import type { MapleLifeConfig } from "@/types/models/template";
import { isEmptyConfig } from "./mapleLifeEditorState";

export interface SeedCandidate {
  id: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
  lookCount: number;
  classCount: number;
  eligible: boolean;
  mapleLife: MapleLifeConfig | undefined;
}

interface SeedFromTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  seedFrom: { region: string; majorVersion: number; minorVersion: number };
  onSeed: (config: MapleLifeConfig) => void;
}

/**
 * FR-12.3: lets a tenant with no `mapleLife` block copy one from a template
 * matching its region/major/minor. Candidate resolution uses `useTemplates()`
 * (full attributes) filtered client-side on all three fields, never
 * `useTemplatesByRegionAndVersion` — that hook calls `getOne` and always
 * returns a one-element array, so the multi-match picker could never render.
 *
 * Ineligible candidates (no `mapleLife`, or an empty one) are listed with
 * their reason rather than hidden — "the template you expect has no block"
 * has to be visible, not indistinguishable from a zero-match.
 */
export function SeedFromTemplateDialog({
  open,
  onOpenChange,
  seedFrom,
  onSeed,
}: SeedFromTemplateDialogProps) {
  const { data: templates } = useTemplates();

  const candidates = useMemo<SeedCandidate[]>(() => {
    return (templates ?? [])
      .filter(
        (t) =>
          t.attributes.region === seedFrom.region &&
          t.attributes.majorVersion === seedFrom.majorVersion &&
          t.attributes.minorVersion === seedFrom.minorVersion,
      )
      .map((t) => {
        const mapleLife = t.attributes.mapleLife;
        return {
          id: t.id,
          region: t.attributes.region,
          majorVersion: t.attributes.majorVersion,
          minorVersion: t.attributes.minorVersion,
          lookCount: mapleLife?.looks.length ?? 0,
          classCount: mapleLife?.classes.length ?? 0,
          eligible: !isEmptyConfig(mapleLife),
          mapleLife,
        };
      });
  }, [
    templates,
    seedFrom.region,
    seedFrom.majorVersion,
    seedFrom.minorVersion,
  ]);

  const eligible = candidates.filter((c) => c.eligible);
  const ineligible = candidates.filter((c) => !c.eligible);
  const singleEligible = eligible.length === 1 ? eligible[0] : undefined;

  const handleSeed = (candidate: SeedCandidate) => {
    if (!candidate.mapleLife) {
      return;
    }
    onSeed(structuredClone(candidate.mapleLife));
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Seed from template</DialogTitle>
          <DialogDescription>
            Copy the Maple Life block from a template matching {seedFrom.region}{" "}
            v{seedFrom.majorVersion}.{seedFrom.minorVersion}.
          </DialogDescription>
        </DialogHeader>

        {eligible.length === 0 && (
          <p className="text-sm text-muted-foreground">
            No template of this region and version carries a Maple Life block.
          </p>
        )}

        {singleEligible && (
          <Button onClick={() => handleSeed(singleEligible)}>
            Seed from {singleEligible.id}
          </Button>
        )}

        {eligible.length > 1 && (
          <ul className="flex flex-col gap-2">
            {eligible.map((candidate) => (
              <li key={candidate.id}>
                <Button
                  variant="outline"
                  className="w-full justify-start"
                  onClick={() => handleSeed(candidate)}
                >
                  {candidate.id} — {candidate.region} v{candidate.majorVersion}.
                  {candidate.minorVersion} — {candidate.classCount} classes ·{" "}
                  {candidate.lookCount} looks
                </Button>
              </li>
            ))}
          </ul>
        )}

        {ineligible.length > 0 && (
          <ul className="flex flex-col gap-1">
            {ineligible.map((candidate) => (
              <li key={candidate.id} className="text-sm text-muted-foreground">
                {candidate.id} — no Maple Life block on this template
              </li>
            ))}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  );
}
