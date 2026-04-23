import { useMemo, useState } from "react";
import { AlertTriangle, Play, Square } from "lucide-react";
import { toast } from "sonner";

import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type {
  Conversation,
  ConversationState,
  ConversationStateType,
} from "@/types/models/conversation";
import {
  addChildState,
  clearTransition,
  deleteState,
  insertBefore,
  insertBetween,
  previewDelete,
  renameState,
  replaceState,
  switchStateType,
  type DeleteImpact,
} from "./editorOps";
import type { Transition } from "./transitions";
import { analyze, type SoftWarnings } from "./graphAnalysis";
import { ConversationCanvas } from "./ConversationCanvas";
import { ConversationInspector } from "./ConversationInspector";
import { STATE_TYPE_META } from "./stateMeta";

interface ConversationEditorPanelProps {
  draft: Conversation;
  onDraftChange: (next: Conversation) => void;
  selectedStateId: string | null;
  onSelectStateId: (id: string | null) => void;
  maxHeight?: number;
  readOnly?: boolean;
}

export function ConversationEditorPanel({
  draft,
  onDraftChange,
  selectedStateId,
  onSelectStateId,
  maxHeight = 640,
  readOnly = false,
}: ConversationEditorPanelProps) {
  const [showFullLoopEdges, setShowFullLoopEdges] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<DeleteImpact | null>(null);
  const [cascadeDelete, setCascadeDelete] = useState(false);

  const analysis = useMemo(() => analyze(draft), [draft]);

  const issueCount =
    analysis.unreachable.length +
    analysis.brokenRefs.length +
    analysis.duplicateIds.length;

  const handleUpdateState = (id: string, next: ConversationState) => {
    onDraftChange(replaceState(draft, id, next));
  };

  const handleRename = (oldId: string, newId: string) => {
    onDraftChange(renameState(draft, oldId, newId));
    onSelectStateId(newId);
  };

  const handleRequestDelete = (id: string) => {
    const impact = previewDelete(draft, id);
    setCascadeDelete(false);
    setPendingDelete(impact);
  };

  const handleSwitchType = (id: string, nextType: ConversationStateType) => {
    onDraftChange(switchStateType(draft, id, nextType));
  };

  const handleAddChild = (sourceId: string) => {
    const result = addChildState(draft, sourceId);
    if (!result) {
      toast.error("Cannot add a child to this state type.");
      return;
    }
    onDraftChange(result.conversation);
    onSelectStateId(result.newStateId);
  };

  const handleInsertBetween = (
    sourceId: string,
    kind: Transition["kind"],
    ordinal: number,
  ) => {
    const result = insertBetween(draft, sourceId, kind, ordinal);
    if (!result) {
      toast.error("Cannot insert between on this transition.");
      return;
    }
    onDraftChange(result.conversation);
    onSelectStateId(result.newStateId);
  };

  const handleInsertBefore = (targetId: string) => {
    const result = insertBefore(draft, targetId);
    if (!result) {
      toast.error("Cannot insert before this state.");
      return;
    }
    onDraftChange(result.conversation);
    onSelectStateId(result.newStateId);
  };

  const handleClearTransition = (
    sourceId: string,
    kind: Transition["kind"],
    ordinal: number,
  ) => {
    const result = clearTransition(draft, sourceId, kind, ordinal);
    if (!result) {
      toast.error("Cannot clear this transition.");
      return;
    }
    if (result.cascadedDeletedIds.length > 0) {
      toast.success(
        `Cleared; removed ${result.cascadedDeletedIds.length} newly unreachable state${result.cascadedDeletedIds.length === 1 ? "" : "s"}.`,
      );
    }
    onDraftChange(result.conversation);
  };

  const handleConfirmDelete = () => {
    if (!pendingDelete) return;
    if (pendingDelete.isStart) {
      toast.error("Cannot delete the start state.");
      return;
    }
    onDraftChange(
      deleteState(draft, pendingDelete.targetId, { cascade: cascadeDelete }),
    );
    if (selectedStateId === pendingDelete.targetId) {
      onSelectStateId(draft.attributes.startState);
    }
    setPendingDelete(null);
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3 flex-wrap">
        <StateSearch
          states={draft.attributes.states.map(s => ({ id: s.id, type: s.type }))}
          onSelect={onSelectStateId}
        />
        <LegendItem
          icon={<Play className="h-2.5 w-2.5 fill-emerald-500 text-emerald-500" />}
          label="start"
        />
        <LegendItem
          icon={<Square className="h-2.5 w-2.5 fill-orange-500 text-orange-500" />}
          label="end"
        />
        <div className="flex items-center gap-2">
          <Switch
            id={`show-loop-edges-${draft.id}`}
            checked={showFullLoopEdges}
            onCheckedChange={setShowFullLoopEdges}
          />
          <Label
            htmlFor={`show-loop-edges-${draft.id}`}
            className="text-xs font-medium"
          >
            Show full loop edges
          </Label>
        </div>
        {issueCount > 0 && (
          <span className="text-[11px] text-destructive flex items-center gap-1">
            <AlertTriangle className="h-3 w-3" />
            {issueCount} issue{issueCount === 1 ? "" : "s"}
          </span>
        )}
      </div>

      {issueCount > 0 && (
        <IssuesBanner
          unreachable={analysis.unreachable}
          broken={analysis.brokenRefs}
          duplicates={analysis.duplicateIds}
          onSelect={onSelectStateId}
        />
      )}
      {(analysis.softWarnings.deadEnds.length > 0 ||
        analysis.softWarnings.highFanOut.length > 0 ||
        analysis.softWarnings.duplicateChoiceLabels.length > 0) && (
        <WarningsBanner
          warnings={analysis.softWarnings}
          onSelect={onSelectStateId}
        />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-[minmax(260px,0.85fr)_2fr] gap-3">
        <div
          className="rounded-md border bg-background"
          style={{ height: maxHeight }}
        >
          <ConversationInspector
            conversation={draft}
            analysis={analysis}
            selectedStateId={selectedStateId}
            onSelect={id => onSelectStateId(id)}
            onUpdateState={handleUpdateState}
            onRename={handleRename}
            onDelete={handleRequestDelete}
            onSwitchType={handleSwitchType}
            onAddChild={handleAddChild}
            onInsertBetween={handleInsertBetween}
            onInsertBefore={handleInsertBefore}
            onClearTransition={handleClearTransition}
            readOnly={readOnly}
          />
        </div>
        <ConversationCanvas
          conversation={draft}
          selectedStateId={selectedStateId}
          onSelect={id => onSelectStateId(id)}
          showFullLoopEdges={showFullLoopEdges}
          height={maxHeight}
        />
      </div>

      <DeleteDialog
        impact={pendingDelete}
        cascade={cascadeDelete}
        onCascadeChange={setCascadeDelete}
        onConfirm={handleConfirmDelete}
        onClose={() => setPendingDelete(null)}
      />
    </div>
  );
}

function StateSearch({
  states,
  onSelect,
}: {
  states: Array<{ id: string; type: ConversationStateType }>;
  onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return [];
    return states.filter(s => s.id.toLowerCase().includes(q)).slice(0, 12);
  }, [query, states]);
  const pick = (id: string) => {
    onSelect(id);
    setQuery("");
    setOpen(false);
  };
  return (
    <div className="relative">
      <Input
        value={query}
        onChange={e => {
          setQuery(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          window.setTimeout(() => setOpen(false), 150);
        }}
        onKeyDown={e => {
          if (e.key === "Enter" && matches.length > 0) {
            pick(matches[0]!.id);
          } else if (e.key === "Escape") {
            setQuery("");
            setOpen(false);
          }
        }}
        placeholder="Search state id…"
        className="h-8 w-56 text-xs font-mono"
      />
      {open && matches.length > 0 && (
        <ul className="absolute z-20 top-full left-0 mt-1 w-full max-h-64 overflow-y-auto rounded-md border bg-popover shadow-md">
          {matches.map(m => (
            <li key={m.id}>
              <button
                type="button"
                onMouseDown={e => e.preventDefault()}
                onClick={() => pick(m.id)}
                className="flex items-center gap-2 w-full text-left px-2 py-1 hover:bg-accent text-xs"
              >
                <span className="text-[10px] text-muted-foreground shrink-0 w-16 truncate">
                  {STATE_TYPE_META[m.type].label}
                </span>
                <span className="font-mono truncate">{m.id}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function LegendItem({
  icon,
  label,
}: {
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <span className="flex items-center gap-1 text-muted-foreground text-xs">
      {icon}
      {label}
    </span>
  );
}

function IssuesBanner({
  unreachable,
  broken,
  duplicates,
  onSelect,
}: {
  unreachable: string[];
  broken: Array<{ source: string; target: string }>;
  duplicates: string[];
  onSelect: (id: string) => void;
}) {
  return (
    <div className="rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2 flex flex-col gap-1.5 text-xs">
      <div className="font-medium text-destructive flex items-center gap-1.5">
        <AlertTriangle className="h-3 w-3" />
        Validation issues
      </div>
      {unreachable.length > 0 && (
        <div className="flex flex-wrap gap-1 items-center">
          <span className="text-muted-foreground">Unreachable:</span>
          {unreachable.slice(0, 6).map(id => (
            <button
              key={id}
              type="button"
              onClick={() => onSelect(id)}
              className="font-mono text-[11px] text-primary hover:underline"
            >
              {id}
            </button>
          ))}
          {unreachable.length > 6 && (
            <span className="text-muted-foreground">
              +{unreachable.length - 6} more
            </span>
          )}
        </div>
      )}
      {broken.length > 0 && (
        <div className="flex flex-wrap gap-1 items-center">
          <span className="text-muted-foreground">Broken refs:</span>
          {broken.slice(0, 6).map((b, i) => (
            <button
              key={i}
              type="button"
              onClick={() => onSelect(b.source)}
              className="font-mono text-[11px] text-primary hover:underline"
              title={`${b.source} → ${b.target}`}
            >
              {b.source} → {b.target}
            </button>
          ))}
          {broken.length > 6 && (
            <span className="text-muted-foreground">
              +{broken.length - 6} more
            </span>
          )}
        </div>
      )}
      {duplicates.length > 0 && (
        <div>
          <span className="text-muted-foreground">Duplicate IDs: </span>
          <span className="font-mono text-[11px]">{duplicates.join(", ")}</span>
        </div>
      )}
    </div>
  );
}

function WarningsBanner({
  warnings,
  onSelect,
}: {
  warnings: SoftWarnings;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="rounded-md border border-amber-500/40 bg-amber-500/5 px-3 py-2 flex flex-col gap-1.5 text-xs">
      <div className="font-medium text-amber-700 dark:text-amber-400 flex items-center gap-1.5">
        <AlertTriangle className="h-3 w-3" />
        Warnings
      </div>
      {warnings.deadEnds.length > 0 && (
        <div className="flex flex-wrap gap-1 items-center">
          <span className="text-muted-foreground">
            Dead-end (can't reach a terminal):
          </span>
          {warnings.deadEnds.slice(0, 6).map(id => (
            <button
              key={id}
              type="button"
              onClick={() => onSelect(id)}
              className="font-mono text-[11px] text-primary hover:underline"
            >
              {id}
            </button>
          ))}
          {warnings.deadEnds.length > 6 && (
            <span className="text-muted-foreground">
              +{warnings.deadEnds.length - 6} more
            </span>
          )}
        </div>
      )}
      {warnings.highFanOut.length > 0 && (
        <div className="flex flex-wrap gap-1 items-center">
          <span className="text-muted-foreground">High fan-out (≥20):</span>
          {warnings.highFanOut.map(id => (
            <button
              key={id}
              type="button"
              onClick={() => onSelect(id)}
              className="font-mono text-[11px] text-primary hover:underline"
            >
              {id}
            </button>
          ))}
        </div>
      )}
      {warnings.duplicateChoiceLabels.length > 0 && (
        <div className="flex flex-wrap gap-1 items-center">
          <span className="text-muted-foreground">Duplicate choice labels:</span>
          {warnings.duplicateChoiceLabels.slice(0, 6).map((w, i) => (
            <button
              key={i}
              type="button"
              onClick={() => onSelect(w.source)}
              className="font-mono text-[11px] text-primary hover:underline"
              title={`${w.label} ×${w.count}`}
            >
              {w.source}
            </button>
          ))}
          {warnings.duplicateChoiceLabels.length > 6 && (
            <span className="text-muted-foreground">
              +{warnings.duplicateChoiceLabels.length - 6} more
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function DeleteDialog({
  impact,
  cascade,
  onCascadeChange,
  onConfirm,
  onClose,
}: {
  impact: DeleteImpact | null;
  cascade: boolean;
  onCascadeChange: (v: boolean) => void;
  onConfirm: () => void;
  onClose: () => void;
}) {
  if (impact?.isStart) {
    return (
      <Dialog open onOpenChange={v => !v && onClose()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cannot delete start state</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            <code className="font-mono">{impact.targetId}</code> is the
            conversation's start state. Rename or re-wire startState first, then
            retry the delete.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={onClose}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }
  return (
    <AlertDialog open={impact !== null} onOpenChange={v => !v && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete <code className="font-mono">{impact?.targetId}</code>?
          </AlertDialogTitle>
          <AlertDialogDescription>
            {impact && impact.incomingFromKept.length > 0 && (
              <span className="block mb-2">
                {impact.incomingFromKept.length} transition
                {impact.incomingFromKept.length === 1 ? "" : "s"} from other
                states will become null (explicit end).
              </span>
            )}
            {impact && impact.wouldBecomeUnreachable.length > 0 && (
              <span className="block mb-2 text-destructive">
                {impact.wouldBecomeUnreachable.length} state
                {impact.wouldBecomeUnreachable.length === 1 ? "" : "s"} will
                become unreachable:{" "}
                <code className="font-mono text-[11px]">
                  {impact.wouldBecomeUnreachable.slice(0, 6).join(", ")}
                  {impact.wouldBecomeUnreachable.length > 6
                    ? `, +${impact.wouldBecomeUnreachable.length - 6} more`
                    : ""}
                </code>
              </span>
            )}
            <span className="block">
              This takes effect in the draft. Save to persist.
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        {impact && impact.wouldBecomeUnreachable.length > 0 && (
          <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2">
            <Switch
              id="cascade-delete"
              checked={cascade}
              onCheckedChange={onCascadeChange}
            />
            <Label
              htmlFor="cascade-delete"
              className="text-xs font-medium leading-tight"
            >
              Also delete the {impact.wouldBecomeUnreachable.length} state
              {impact.wouldBecomeUnreachable.length === 1 ? "" : "s"} that would
              become unreachable
            </Label>
          </div>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
