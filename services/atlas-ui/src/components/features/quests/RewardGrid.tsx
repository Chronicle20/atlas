import { Badge } from "@/components/ui/badge";
import { Star, Coins, Clock } from "lucide-react";
import type { QuestActions } from "@/types/models/quest";
import { EntityWidget } from "./EntityWidget";

interface RewardGridProps {
  actions: QuestActions;
  phase: "start" | "end";
  omitNextQuest?: boolean;
}

export function RewardGrid({
  actions,
  phase,
  omitNextQuest = false,
}: RewardGridProps) {
  const chips: React.ReactNode[] = [];
  const blocks: React.ReactNode[] = [];

  if (actions.exp && actions.exp !== 0) {
    chips.push(
      <Chip
        key="exp"
        icon={<Star className="h-3.5 w-3.5" />}
        label="EXP"
        value={`${actions.exp > 0 ? "+" : ""}${actions.exp.toLocaleString()}`}
        tone={actions.exp > 0 ? "positive" : "negative"}
      />,
    );
  }

  if (actions.money && actions.money !== 0) {
    chips.push(
      <Chip
        key="money"
        icon={<Coins className="h-3.5 w-3.5" />}
        label="Meso"
        value={`${actions.money > 0 ? "+" : ""}${actions.money.toLocaleString()}`}
        tone={actions.money > 0 ? "positive" : "negative"}
      />,
    );
  }

  if (actions.fame && actions.fame !== 0) {
    chips.push(
      <Chip
        key="fame"
        icon={<Star className="h-3.5 w-3.5" />}
        label="Fame"
        value={`${actions.fame > 0 ? "+" : ""}${actions.fame}`}
        tone={actions.fame > 0 ? "positive" : "negative"}
      />,
    );
  }

  if (actions.levelMin) {
    chips.push(
      <Chip
        key="levelMin"
        icon={<Star className="h-3.5 w-3.5" />}
        label="Min Level"
        value={`${actions.levelMin}+`}
      />,
    );
  }

  if (actions.interval) {
    chips.push(
      <Chip
        key="interval"
        icon={<Clock className="h-3.5 w-3.5" />}
        label="Interval"
        value={`${actions.interval}s`}
      />,
    );
  }

  if (actions.npcId) {
    blocks.push(
      <Block key="npc" label="NPC">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          <EntityWidget kind="npc" id={actions.npcId} />
        </div>
      </Block>,
    );
  }

  if (actions.items && actions.items.length > 0) {
    blocks.push(
      <Block key="items" label="Items">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {actions.items.map((item, i) => (
            <EntityWidget
              key={`${item.id}-${i}`}
              kind="item"
              id={item.id}
              count={item.count}
              prop={item.prop}
              period={item.period}
              gender={item.gender}
              job={item.job}
            />
          ))}
        </div>
      </Block>,
    );
  }

  if (actions.skills && actions.skills.length > 0) {
    blocks.push(
      <Block key="skills" label="Skills">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {actions.skills.map((skill, i) => (
            <EntityWidget
              key={`${skill.id}-${i}`}
              kind="skill"
              id={skill.id}
              count={skill.level}
              jobs={skill.jobs}
            />
          ))}
        </div>
      </Block>,
    );
  }

  if (actions.buffItemId) {
    blocks.push(
      <Block key="buff" label="Buff item">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          <EntityWidget kind="item" id={actions.buffItemId} />
        </div>
      </Block>,
    );
  }

  if (!omitNextQuest && actions.nextQuest) {
    blocks.push(
      <Block key="nextQuest" label="Next quest">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          <EntityWidget kind="quest" id={actions.nextQuest} />
        </div>
      </Block>,
    );
  }

  if (chips.length === 0 && blocks.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No {phase === "start" ? "start" : "completion"} actions.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {chips.length > 0 && (
        <div className="flex flex-wrap gap-2">{chips}</div>
      )}
      {blocks}
    </div>
  );
}

interface ChipProps {
  icon: React.ReactNode;
  label: React.ReactNode;
  value: React.ReactNode;
  tone?: "positive" | "negative" | "neutral";
}

function Chip({ icon, label, value, tone = "neutral" }: ChipProps) {
  const variant =
    tone === "negative" ? "destructive" : tone === "positive" ? "default" : "secondary";
  return (
    <Badge variant={variant} className="gap-1.5 py-1 pl-1.5 pr-2 font-normal">
      <span className="opacity-80">{icon}</span>
      <span className="font-medium">{label}:</span>
      <span>{value}</span>
    </Badge>
  );
}

interface BlockProps {
  label: string;
  children: React.ReactNode;
}

function Block({ label, children }: BlockProps) {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        {label}
      </p>
      {children}
    </div>
  );
}
