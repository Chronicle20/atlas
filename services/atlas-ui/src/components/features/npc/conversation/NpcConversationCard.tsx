import { useEffect, useState } from "react";
import { Save, Undo2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { useTenant } from "@/context/tenant-context";
import { conversationsService } from "@/services/api/conversations.service";
import { npcConversationKeys } from "@/lib/hooks/api/useNpcConversation";
import type { Conversation } from "@/types/models/conversation";
import { analyze } from "./graphAnalysis";
import { ConversationEditorPanel } from "./ConversationEditorPanel";

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

  const handleDraftChange = (next: Conversation) => {
    setDraft(next);
    setIsDirty(true);
  };

  const handleRevert = () => {
    setDraft(conversation);
    setIsDirty(false);
    toast.success("Reverted to saved version");
  };

  const handleSave = async () => {
    const analysis = analyze(draft);
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
      <div className="flex items-center justify-end gap-2 flex-wrap">
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
        <Button size="sm" disabled={!isDirty || saving} onClick={handleSave}>
          <Save className="h-3.5 w-3.5" />
          {saving ? "Saving…" : "Save"}
        </Button>
      </div>
      <ConversationEditorPanel
        draft={draft}
        onDraftChange={handleDraftChange}
        selectedStateId={selectedStateId}
        onSelectStateId={setSelectedStateId}
        maxHeight={maxHeight}
      />
    </div>
  );
}
