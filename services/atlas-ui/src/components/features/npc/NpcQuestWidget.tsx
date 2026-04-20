import { Link } from "react-router-dom";
import { Scroll } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { NpcQuestRole } from "@/types/models/npc";
import type { QuestDefinition } from "@/types/models/quest";

interface NpcQuestWidgetProps {
  quest: QuestDefinition;
  role: NpcQuestRole;
}

const ROLE_LABEL: Record<NpcQuestRole, string> = {
  initiator: "Initiator",
  completer: "Completer",
  both: "Initiator & Completer",
};

export function NpcQuestWidget({ quest, role }: NpcQuestWidgetProps) {
  const { name, parent } = quest.attributes;
  return (
    <Link
      to={`/quests/${quest.id}`}
      className="flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
    >
      <div className="flex items-center gap-2 flex-wrap">
        <Scroll className="h-4 w-4 text-muted-foreground shrink-0" />
        <span className="text-sm font-medium truncate">
          {name || `Quest #${quest.id}`}
        </span>
      </div>
      <div className="flex items-center gap-2 flex-wrap">
        {parent && <Badge variant="secondary">{parent}</Badge>}
        <Badge variant="secondary">{ROLE_LABEL[role]}</Badge>
      </div>
    </Link>
  );
}
