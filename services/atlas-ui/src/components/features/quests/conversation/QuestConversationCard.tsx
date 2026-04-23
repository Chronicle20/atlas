import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { Save, Undo2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { useTenant } from "@/context/tenant-context";
import { questConversationsService } from "@/services/api/quest-conversations.service";
import { questConversationKeys } from "@/lib/hooks/api/useQuestConversation";
import type {
  Conversation,
  QuestConversation,
  StateMachine,
} from "@/types/models/conversation";
import { ConversationEditorPanel } from "@/components/features/npc/conversation/ConversationEditorPanel";
import { analyze } from "@/components/features/npc/conversation/graphAnalysis";

type Machine = "start" | "end";

interface QuestConversationContextValue {
  draft: QuestConversation;
  isDirty: boolean;
  saving: boolean;
  hasEnd: boolean;
  startConversation: Conversation;
  endConversation: Conversation | null;
  selectedStart: string | null;
  selectedEnd: string | null;
  setSelectedStart: (id: string | null) => void;
  setSelectedEnd: (id: string | null) => void;
  applyStart: (next: Conversation) => void;
  applyEnd: (next: Conversation) => void;
  handleRevert: () => void;
  handleSave: () => Promise<void>;
}

const QuestConversationContext =
  createContext<QuestConversationContextValue | null>(null);

function useQuestConversationContext(): QuestConversationContextValue {
  const ctx = useContext(QuestConversationContext);
  if (!ctx) {
    throw new Error(
      "QuestConversation components must be used inside QuestConversationProvider",
    );
  }
  return ctx;
}

function machineToConversation(
  parentId: string,
  machine: StateMachine,
  label: Machine,
): Conversation {
  return {
    id: `${parentId}:${label}`,
    type: "conversations",
    attributes: {
      npcId: 0,
      startState: machine.startState,
      states: machine.states,
    },
  };
}

function conversationToMachine(conv: Conversation): StateMachine {
  return {
    startState: conv.attributes.startState,
    states: conv.attributes.states,
  };
}

interface QuestConversationProviderProps {
  conversation: QuestConversation;
  children: ReactNode;
}

export function QuestConversationProvider({
  conversation,
  children,
}: QuestConversationProviderProps) {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<QuestConversation>(conversation);
  const [isDirty, setIsDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isDirty) setDraft(conversation);
  }, [conversation, isDirty]);

  const hasEnd = !!draft.attributes.endStateMachine;

  const startConversation = useMemo(
    () =>
      machineToConversation(draft.id, draft.attributes.startStateMachine, "start"),
    [draft],
  );
  const endConversation = useMemo(() => {
    const end = draft.attributes.endStateMachine;
    return end ? machineToConversation(draft.id, end, "end") : null;
  }, [draft]);

  const [selectedStart, setSelectedStart] = useState<string | null>(
    () => draft.attributes.startStateMachine.startState,
  );
  const [selectedEnd, setSelectedEnd] = useState<string | null>(
    () => draft.attributes.endStateMachine?.startState ?? null,
  );

  const applyStart = (next: Conversation) => {
    setDraft(current => ({
      ...current,
      attributes: {
        ...current.attributes,
        startStateMachine: conversationToMachine(next),
      },
    }));
    setIsDirty(true);
  };

  const applyEnd = (next: Conversation) => {
    setDraft(current => ({
      ...current,
      attributes: {
        ...current.attributes,
        endStateMachine: conversationToMachine(next),
      },
    }));
    setIsDirty(true);
  };

  const handleRevert = () => {
    setDraft(conversation);
    setIsDirty(false);
    toast.success("Reverted to saved version");
  };

  const handleSave = async () => {
    const startAnalysis = analyze(startConversation);
    if (startAnalysis.duplicateIds.length > 0) {
      toast.error("Start machine has duplicate state IDs.");
      return;
    }
    if (startAnalysis.brokenRefs.length > 0) {
      toast.error("Start machine has transitions to missing states.");
      return;
    }
    if (endConversation) {
      const endAnalysis = analyze(endConversation);
      if (endAnalysis.duplicateIds.length > 0) {
        toast.error("End machine has duplicate state IDs.");
        return;
      }
      if (endAnalysis.brokenRefs.length > 0) {
        toast.error("End machine has transitions to missing states.");
        return;
      }
    }
    setSaving(true);
    try {
      await questConversationsService.update(draft.id, draft.attributes);
      queryClient.invalidateQueries({
        queryKey: questConversationKeys.byQuest(
          activeTenant?.id,
          draft.attributes.questId,
        ),
      });
      setIsDirty(false);
      toast.success("Quest conversation saved");
    } catch (err) {
      toast.error(
        "Failed to save: " + (err instanceof Error ? err.message : String(err)),
      );
    } finally {
      setSaving(false);
    }
  };

  const value: QuestConversationContextValue = {
    draft,
    isDirty,
    saving,
    hasEnd,
    startConversation,
    endConversation,
    selectedStart,
    selectedEnd,
    setSelectedStart,
    setSelectedEnd,
    applyStart,
    applyEnd,
    handleRevert,
    handleSave,
  };

  return (
    <QuestConversationContext.Provider value={value}>
      {children}
    </QuestConversationContext.Provider>
  );
}

export function QuestConversationToolbar() {
  const { isDirty, saving, handleRevert, handleSave } =
    useQuestConversationContext();
  return (
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
  );
}

interface MachineEditorProps {
  machine: Machine;
  maxHeight?: number;
}

export function QuestConversationMachineEditor({
  machine,
  maxHeight = 640,
}: MachineEditorProps) {
  const ctx = useQuestConversationContext();
  if (machine === "start") {
    return (
      <ConversationEditorPanel
        draft={ctx.startConversation}
        onDraftChange={ctx.applyStart}
        selectedStateId={ctx.selectedStart}
        onSelectStateId={ctx.setSelectedStart}
        maxHeight={maxHeight}
      />
    );
  }
  if (!ctx.endConversation) {
    return (
      <p className="text-sm text-muted-foreground py-6">
        No completion conversation defined.
      </p>
    );
  }
  return (
    <ConversationEditorPanel
      draft={ctx.endConversation}
      onDraftChange={ctx.applyEnd}
      selectedStateId={ctx.selectedEnd}
      onSelectStateId={ctx.setSelectedEnd}
      maxHeight={maxHeight}
    />
  );
}
