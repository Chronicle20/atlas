import { Badge } from "@/components/ui/badge";
import {
  Star,
  Coins,
  Sword,
  Clock,
  Calendar,
  ScrollText,
  PawPrint,
  Sparkles,
} from "lucide-react";
import type { QuestRequirements } from "@/types/models/quest";
import { JobName } from "./EntityName";
import { EntityWidget } from "./EntityWidget";

interface RequirementGridProps {
  requirements: QuestRequirements;
  phase: "start" | "end";
}

export function RequirementGrid({ requirements, phase }: RequirementGridProps) {
  const chips: React.ReactNode[] = [];
  const blocks: React.ReactNode[] = [];

  if (requirements.levelMin || requirements.levelMax) {
    const levelText =
      requirements.levelMin && requirements.levelMax
        ? `${requirements.levelMin} - ${requirements.levelMax}`
        : requirements.levelMin
          ? `${requirements.levelMin}+`
          : `1 - ${requirements.levelMax}`;
    chips.push(
      <Chip
        key="level"
        icon={<Star className="h-3.5 w-3.5" />}
        label="Level"
        value={levelText}
      />,
    );
  }

  if (requirements.fameMin) {
    chips.push(
      <Chip
        key="fame"
        icon={<Star className="h-3.5 w-3.5" />}
        label="Fame"
        value={`${requirements.fameMin}+`}
      />,
    );
  }

  if (requirements.mesoMin || requirements.mesoMax) {
    const mesoText =
      requirements.mesoMin && requirements.mesoMax
        ? `${requirements.mesoMin.toLocaleString()} - ${requirements.mesoMax.toLocaleString()}`
        : requirements.mesoMin
          ? `${requirements.mesoMin.toLocaleString()}+`
          : `Max ${requirements.mesoMax?.toLocaleString()}`;
    chips.push(
      <Chip
        key="meso"
        icon={<Coins className="h-3.5 w-3.5" />}
        label="Meso"
        value={mesoText}
      />,
    );
  }

  if (requirements.jobs && requirements.jobs.length > 0) {
    blocks.push(
      <Block
        key="jobs"
        label={`Jobs (${requirements.jobs.length})`}
      >
        <div className="flex flex-wrap gap-1.5">
          {requirements.jobs.map((j) => (
            <Badge
              key={j}
              variant="secondary"
              className="gap-1 py-0.5 pl-1.5 pr-2 font-normal text-xs"
            >
              <Sword className="h-3 w-3 text-muted-foreground" />
              <JobName id={j} />
            </Badge>
          ))}
        </div>
      </Block>,
    );
  }

  if (requirements.petTamenessMin) {
    chips.push(
      <Chip
        key="petTameness"
        icon={<PawPrint className="h-3.5 w-3.5" />}
        label="Pet Tameness"
        value={`${requirements.petTamenessMin}+`}
      />,
    );
  }

  if (requirements.dayOfWeek) {
    chips.push(
      <Chip
        key="dayOfWeek"
        icon={<Calendar className="h-3.5 w-3.5" />}
        label="Day of Week"
        value={requirements.dayOfWeek}
      />,
    );
  }

  if (requirements.start || requirements.end) {
    const timeText =
      requirements.start && requirements.end
        ? `${requirements.start} - ${requirements.end}`
        : (requirements.start ?? requirements.end ?? "");
    chips.push(
      <Chip
        key="time"
        icon={<Clock className="h-3.5 w-3.5" />}
        label="Time"
        value={timeText}
      />,
    );
  }

  if (requirements.interval) {
    chips.push(
      <Chip
        key="interval"
        icon={<Clock className="h-3.5 w-3.5" />}
        label="Interval"
        value={formatInterval(requirements.interval)}
      />,
    );
  }

  if (requirements.completionCount) {
    chips.push(
      <Chip
        key="completionCount"
        icon={<ScrollText className="h-3.5 w-3.5" />}
        label="Completions"
        value={`${requirements.completionCount}`}
      />,
    );
  }

  if (requirements.infoNumber) {
    chips.push(
      <Chip
        key="infoNumber"
        icon={<ScrollText className="h-3.5 w-3.5" />}
        label="Info #"
        value={`${requirements.infoNumber}`}
      />,
    );
  }

  if (requirements.normalAutoStart) {
    chips.push(
      <Chip
        key="normalAutoStart"
        icon={<Sparkles className="h-3.5 w-3.5" />}
        label="Normal Auto Start"
        value="Yes"
      />,
    );
  }

  if (requirements.npcId) {
    blocks.push(
      <Block key="npc" label={phase === "start" ? "Start NPC" : "End NPC"}>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          <EntityWidget kind="npc" id={requirements.npcId} />
        </div>
      </Block>,
    );
  }

  if (requirements.quests && requirements.quests.length > 0) {
    blocks.push(
      <Block key="quests" label="Quest prerequisites">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {requirements.quests.map((q, i) => (
            <EntityWidget
              key={`${q.id}-${i}`}
              kind="quest"
              id={q.id}
              state={q.state as 0 | 1 | 2}
            />
          ))}
        </div>
      </Block>,
    );
  }

  if (requirements.items && requirements.items.length > 0) {
    blocks.push(
      <Block key="items" label="Items">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {requirements.items.map((it, i) => (
            <EntityWidget
              key={`${it.id}-${i}`}
              kind="item"
              id={it.id}
              count={it.count}
            />
          ))}
        </div>
      </Block>,
    );
  }

  if (requirements.mobs && requirements.mobs.length > 0) {
    blocks.push(
      <Block key="mobs" label="Mob kills">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {requirements.mobs.map((m, i) => (
            <EntityWidget
              key={`${m.id}-${i}`}
              kind="mob"
              id={m.id}
              count={m.count}
            />
          ))}
        </div>
      </Block>,
    );
  }

  if (requirements.fieldEnter && requirements.fieldEnter.length > 0) {
    blocks.push(
      <Block key="fieldEnter" label="Field enter">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {requirements.fieldEnter.map((mapId, i) => (
            <EntityWidget key={`${mapId}-${i}`} kind="map" id={mapId} />
          ))}
        </div>
      </Block>,
    );
  }

  if (requirements.pet && requirements.pet.length > 0) {
    blocks.push(
      <Block key="pets" label="Pets">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {requirements.pet.map((petId, i) => (
            <EntityWidget key={`${petId}-${i}`} kind="pet" id={petId} />
          ))}
        </div>
      </Block>,
    );
  }

  if (chips.length === 0 && blocks.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No {phase === "start" ? "start" : "completion"} requirements.
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
}

function Chip({ icon, label, value }: ChipProps) {
  return (
    <Badge
      variant="secondary"
      className="gap-1.5 py-1 pl-1.5 pr-2 font-normal"
    >
      <span className="text-muted-foreground">{icon}</span>
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

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}
