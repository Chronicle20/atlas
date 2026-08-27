import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MapPicker } from "../templates/MapPicker";
import { useSyncedNumberInput } from "../presets/useSyncedNumberInput";
import type { IdentityKey, MapleLifeClassDraft } from "./mapleLifeEditorState";

interface IdentitySectionProps {
  draft: MapleLifeClassDraft;
  jobs: {
    options: { id: number; name: string }[];
    isPending: boolean;
    isError: boolean;
  };
  onSetIdentity: (field: IdentityKey, value: number) => void;
  onSetLevel: (value: number) => void;
  /** Blocking messages keyed by field name ("jobId" | "level" | "mapId"). */
  errors?: Record<string, string[]>;
}

function FieldErrors({ messages }: { messages: string[] | undefined }) {
  if (!messages || messages.length === 0) return null;
  return (
    <div className="space-y-0.5">
      {messages.map((message) => (
        <p key={message} className="text-xs text-destructive">
          {message}
        </p>
      ))}
    </div>
  );
}

/**
 * Identity fields for the selected (ordinal, gender) class row. The job
 * field is fully editable for every ordinal.
 */
export function IdentitySection({
  draft,
  jobs,
  onSetIdentity,
  onSetLevel,
  errors,
}: IdentitySectionProps) {
  const [levelInput, setLevelInput] = useSyncedNumberInput(draft.level);

  return (
    <section className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label htmlFor="maple-life-job">Job</Label>
          {jobs.isPending ? (
            <Select value={String(draft.jobId)} disabled>
              <SelectTrigger id="maple-life-job" aria-label="Job">
                <SelectValue />
              </SelectTrigger>
              <SelectContent />
            </Select>
          ) : jobs.isError ? (
            <Input
              id="maple-life-job"
              aria-label="Job"
              type="number"
              value={draft.jobId}
              onChange={(e) => onSetIdentity("jobId", Number(e.target.value))}
            />
          ) : (
            <Select
              value={String(draft.jobId)}
              onValueChange={(v) => onSetIdentity("jobId", Number(v))}
            >
              <SelectTrigger id="maple-life-job" aria-label="Job">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {jobs.options.map((job) => (
                  <SelectItem key={job.id} value={String(job.id)}>
                    {job.name} ({job.id})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          {jobs.isPending && (
            <p className="text-xs text-muted-foreground">Loading job names…</p>
          )}
          {jobs.isError && (
            <p className="text-xs text-muted-foreground">
              Job names unavailable
            </p>
          )}
          <FieldErrors messages={errors?.["jobId"]} />
        </div>
        <div className="space-y-1">
          <Label htmlFor="maple-life-level">Level</Label>
          <Input
            id="maple-life-level"
            aria-label="Level"
            type="number"
            min={1}
            max={200}
            value={levelInput}
            onChange={(e) => {
              setLevelInput(e.target.value);
              onSetLevel(Number(e.target.value));
            }}
          />
          <FieldErrors messages={errors?.["level"]} />
        </div>
        <div className="space-y-1 sm:col-span-2">
          <Label>Starting map</Label>
          <MapPicker
            value={draft.mapId}
            onChange={(mapId) => onSetIdentity("mapId", mapId)}
          />
          <FieldErrors messages={errors?.["mapId"]} />
        </div>
      </div>
    </section>
  );
}
