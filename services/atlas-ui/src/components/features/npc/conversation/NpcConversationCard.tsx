import { useEffect, useMemo, useState } from "react";
import {
  AlertTriangle,
  CornerUpLeft,
  Play,
  Save,
  Square,
  Undo2,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
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
import { useTenant } from "@/context/tenant-context";
import { conversationsService } from "@/services/api/conversations.service";
import { npcConversationKeys } from "@/lib/hooks/api/useNpcConversation";
import type {
  Conversation,
  ConversationState,
  ConversationStateType,
} from "@/types/models/conversation";
import { analyze } from "./graphAnalysis";
import { ConversationCanvas } from "./ConversationCanvas";
import { ConversationInspector } from "./ConversationInspector";
import { STATE_TYPE_META } from "./stateMeta";
import {
  deleteState,
  previewDelete,
  renameState,
  replaceState,
  type DeleteImpact,
} from "./editorOps";

interface NpcConversationCardProps {
  conversation: Conversation;
  maxHeight?: number;
}

export function NpcConversationCard({
  conversation,
  maxHeight = 640,
}: NpcConversationCardProps) {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<Conversation>(conversation);
  const [isDirty, setIsDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isDirty) setDraft(conversation);
  }, [conversation, isDirty]);

  const [selectedStateId, setSelectedStateId] = useState<string | null>(() => {
    const start = draft.attributes.startState;
    return draft.attributes.states.some(s => s.id === start) ? start : null;
  });
  const [showFullLoopEdges, setShowFullLoopEdges] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<DeleteImpact | null>(null);
  const [cascadeDelete, setCascadeDelete] = useState(false);

  const analysis = useMemo(() => analyze(draft), [draft]);

  const typeBreakdown = useMemo(() => {
    const counts = new Map<ConversationStateType, number>();
    for (const s of draft.attributes.states) {
      counts.set(s.type, (counts.get(s.type) ?? 0) + 1);
    }
    return Array.from(counts.entries()).sort(([, a], [, b]) => b - a);
  }, [draft]);

  const issueCount =
    analysis.unreachable.length +
    analysis.brokenRefs.length +
    analysis.duplicateIds.length;

  const handleUpdateState = (id: string, next: ConversationState) => {
    setDraft(current => replaceState(current, id, next));
    setIsDirty(true);
  };

  const handleRename = (oldId: string, newId: string) => {
    setDraft(current => renameState(current, oldId, newId));
    setIsDirty(true);
    setSelectedStateId(newId);
  };

  const handleRequestDelete = (id: string) => {
    const impact = previewDelete(draft, id);
    setCascadeDelete(false);
    setPendingDelete(impact);
  };

  const handleConfirmDelete = () => {
    if (!pendingDelete) return;
    if (pendingDelete.isStart) {
      toast.error("Cannot delete the start state.");
      return;
    }
    setDraft(current =>
      deleteState(current, pendingDelete.targetId, { cascade: cascadeDelete }),
    );
    setIsDirty(true);
    if (selectedStateId === pendingDelete.targetId) {
      setSelectedStateId(draft.attributes.startState);
    }
    setPendingDelete(null);
  };

  const handleRevert = () => {
    setDraft(conversation);
    setIsDirty(false);
    toast.success("Reverted to saved version");
  };

  const handleSave = async () => {
    if (analysis.duplicateIds.length > 0) {
      toast.error("Cannot save: duplicate state IDs.");
      return;
    }
    if (analysis.brokenRefs.length > 0) {
      toast.error("Cannot save: transitions point at missing states.");
      return;
    }
    setSaving(true);
    try {
      await conversationsService.update(draft.id, draft.attributes);
      queryClient.invalidateQueries({
        queryKey: npcConversationKeys.byNpc(
          activeTenant?.id,
          draft.attributes.npcId,
        ),
      });
      setIsDirty(false);
      toast.success("Conversation saved");
    } catch (err) {
      toast.error(
        "Failed to save: " + (err instanceof Error ? err.message : String(err)),
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-3 flex-wrap text-xs">
        <div className="flex items-center gap-3 flex-wrap">
          <span className="text-muted-foreground">
            {draft.attributes.states.length} states
          </span>
          {typeBreakdown.map(([type, n]) => (
            <span key={type} className="text-muted-foreground">
              {n} {STATE_TYPE_META[type].label.toLowerCase()}
            </span>
          ))}
          {analysis.terminals.size > 0 && (
            <span className="text-muted-foreground">
              {analysis.terminals.size} terminal
            </span>
          )}
          {analysis.backEdges.size > 0 && (
            <span className="text-muted-foreground flex items-center gap-1">
              <CornerUpLeft className="h-2.5 w-2.5" />
              {analysis.backEdges.size} back-edge
              {analysis.backEdges.size === 1 ? "" : "s"}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <LegendItem
            icon={
              <Play className="h-2.5 w-2.5 fill-emerald-500 text-emerald-500" />
            }
            label="start"
          />
          <LegendItem
            icon={
              <Square className="h-2.5 w-2.5 fill-orange-500 text-orange-500" />
            }
            label="end"
          />
          {issueCount > 0 && (
            <Badge
              variant="destructive"
              className="text-[10px] px-1.5 py-0 gap-1"
            >
              <AlertTriangle className="h-2.5 w-2.5" />
              {issueCount} issue{issueCount === 1 ? "" : "s"}
            </Badge>
          )}
        </div>
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <Switch
            id="show-loop-edges"
            checked={showFullLoopEdges}
            onCheckedChange={setShowFullLoopEdges}
          />
          <Label htmlFor="show-loop-edges" className="text-xs font-medium">
            Show full loop edges
          </Label>
        </div>
        <div className="ml-auto flex items-center gap-2">
          {isDirty && (
            <span className="text-[11px] text-amber-600 dark:text-amber-400">
              Unsaved changes
            </span>
          )}
          <Button
            size="sm"
            variant="outline"
            disabled={!isDirty || saving}
            onClick={handleRevert}
          >
            <Undo2 className="h-3.5 w-3.5" />
            Revert
          </Button>
          <Button
            size="sm"
            disabled={!isDirty || saving}
            onClick={handleSave}
          >
            <Save className="h-3.5 w-3.5" />
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
      </div>

      {issueCount > 0 && (
        <IssuesBanner
          unreachable={analysis.unreachable}
          broken={analysis.brokenRefs}
          duplicates={analysis.duplicateIds}
          onSelect={setSelectedStateId}
        />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-[2fr_minmax(260px,0.85fr)] gap-3">
        <ConversationCanvas
          conversation={draft}
          selectedStateId={selectedStateId}
          onSelect={setSelectedStateId}
          showFullLoopEdges={showFullLoopEdges}
          height={maxHeight}
        />
        <div
          className="rounded-md border bg-background"
          style={{ height: maxHeight }}
        >
          <ConversationInspector
            conversation={draft}
            analysis={analysis}
            selectedStateId={selectedStateId}
            onSelect={setSelectedStateId}
            onUpdateState={handleUpdateState}
            onRename={handleRename}
            onDelete={handleRequestDelete}
            readOnly={false}
          />
        </div>
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

function LegendItem({
  icon,
  label,
}: {
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <span className="flex items-center gap-1 text-muted-foreground">
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
    <AlertDialog
      open={impact !== null}
      onOpenChange={v => !v && onClose()}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete{" "}
            <code className="font-mono">{impact?.targetId}</code>?
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
            <span className="block">This takes effect in the draft. Save to persist.</span>
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
