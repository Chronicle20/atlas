import { Link } from "react-router-dom";
import {
  Package,
  UserCircle2,
  Skull,
  MapPin,
  ScrollText,
  Wand2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useItemData } from "@/lib/hooks/useItemData";
import { useMobData } from "@/lib/hooks/useMobData";
import { useSkillData } from "@/lib/hooks/useSkillData";
import { useMap } from "@/lib/hooks/api/useMaps";
import { useQuest } from "@/lib/hooks/api/useQuests";
import { useTenant } from "@/context/tenant-context";
import { JobName } from "./EntityName";

export type EntityKind =
  | "npc"
  | "item"
  | "mob"
  | "map"
  | "skill"
  | "quest"
  | "pet";

export interface EntityWidgetProps {
  kind: EntityKind;
  id: number;
  count?: number | undefined;
  state?: 0 | 1 | 2 | undefined;
  prop?: number | undefined;
  period?: number | undefined;
  job?: number | undefined;
  gender?: number | undefined;
  jobs?: number[] | undefined;
}

export function EntityWidget(props: EntityWidgetProps) {
  const { kind, id } = props;
  switch (kind) {
    case "npc":
      return <NpcWidget {...props} />;
    case "item":
      return <ItemWidget {...props} kind="item" />;
    case "pet":
      return <ItemWidget {...props} kind="pet" />;
    case "mob":
      return <MobWidget {...props} />;
    case "skill":
      return <SkillWidget {...props} />;
    case "map":
      return <MapWidget {...props} />;
    case "quest":
      return <QuestWidget {...props} />;
    default:
      return <Shell {...props} kind={kind} id={id} />;
  }
}

function NpcWidget(props: EntityWidgetProps) {
  const q = useNpcData(props.id);
  return (
    <Shell
      {...props}
      name={q.name}
      iconUrl={q.iconUrl}
      isLoading={q.isLoading}
    />
  );
}

function ItemWidget(props: EntityWidgetProps) {
  const q = useItemData(props.id);
  return (
    <Shell
      {...props}
      name={q.name}
      iconUrl={q.iconUrl}
      isLoading={q.isLoading}
    />
  );
}

function MobWidget(props: EntityWidgetProps) {
  const q = useMobData(props.id);
  return (
    <Shell
      {...props}
      name={q.name}
      iconUrl={q.iconUrl}
      isLoading={q.isLoading}
    />
  );
}

function SkillWidget(props: EntityWidgetProps) {
  const q = useSkillData(props.id);
  return (
    <Shell
      {...props}
      name={q.name}
      iconUrl={q.iconUrl}
      isLoading={q.isLoading}
    />
  );
}

function MapWidget(props: EntityWidgetProps) {
  const q = useMap(String(props.id));
  return (
    <Shell
      {...props}
      name={q.data?.attributes.name}
      isLoading={q.isLoading}
    />
  );
}

function QuestWidget(props: EntityWidgetProps) {
  const { activeTenant } = useTenant();
  const q = useQuest(activeTenant, String(props.id));
  return (
    <Shell
      {...props}
      name={q.data?.attributes.name}
      isLoading={q.isLoading}
    />
  );
}

interface ShellProps extends EntityWidgetProps {
  name?: string | undefined;
  iconUrl?: string | undefined;
  isLoading?: boolean | undefined;
}

function fallbackIcon(kind: EntityKind) {
  const cls = "h-5 w-5 text-muted-foreground";
  switch (kind) {
    case "npc":
      return <UserCircle2 className={cls} />;
    case "item":
    case "pet":
      return <Package className={cls} />;
    case "mob":
      return <Skull className={cls} />;
    case "map":
      return <MapPin className={cls} />;
    case "skill":
      return <Wand2 className={cls} />;
    case "quest":
      return <ScrollText className={cls} />;
  }
}

function defaultName(kind: EntityKind, id: number): string {
  switch (kind) {
    case "npc":
      return `NPC #${id}`;
    case "item":
      return `Item #${id}`;
    case "pet":
      return `Pet #${id}`;
    case "mob":
      return `Monster #${id}`;
    case "map":
      return `Map #${id}`;
    case "skill":
      return `Skill #${id}`;
    case "quest":
      return `Quest #${id}`;
  }
}

function routeFor(kind: EntityKind, id: number): string | null {
  switch (kind) {
    case "npc":
      return `/npcs/${id}`;
    case "item":
    case "pet":
      return `/items/${id}`;
    case "mob":
      return `/monsters/${id}`;
    case "map":
      return `/maps/${id}`;
    case "quest":
      return `/quests/${id}`;
    case "skill":
      return null;
  }
}

function stateLabel(state: 0 | 1 | 2): string {
  return state === 0
    ? "not started"
    : state === 1
      ? "in progress"
      : "completed";
}

function formatPeriod(minutes: number): string {
  if (minutes < 60) return `${minutes}m`;
  if (minutes < 1440) return `${Math.floor(minutes / 60)}h`;
  return `${Math.floor(minutes / 1440)}d`;
}

function renderAffix(props: ShellProps): string | null {
  const parts: string[] = [];
  const { kind, count, prop, period, job, gender, jobs } = props;

  if (count !== undefined) {
    if (kind === "mob") {
      parts.push(`${count} kill${count === 1 ? "" : "s"}`);
    } else if (count < 0) {
      parts.push(`× ${count} (consumed)`);
    } else {
      parts.push(`× ${count}`);
    }
  }

  if (prop !== undefined) {
    if (prop === -1) parts.push("Guaranteed");
    else if (prop === 0) parts.push("Selection");
    else parts.push(`${prop}% chance`);
  }

  if (period !== undefined && period > 0) {
    parts.push(formatPeriod(period));
  }

  if (gender !== undefined && gender !== -1) {
    parts.push(gender === 0 ? "♂" : "♀");
  }

  if (job !== undefined && job !== 0 && (!jobs || jobs.length === 0)) {
    parts.push(`for job ${job}`);
  }

  return parts.length > 0 ? parts.join(" · ") : null;
}

function Shell(props: ShellProps) {
  const { kind, id, state, jobs, name, iconUrl, isLoading } = props;
  const displayName = name ?? defaultName(kind, id);
  const route = routeFor(kind, id);
  const affix = renderAffix(props);
  const hasSubline =
    affix !== null || state !== undefined || (jobs && jobs.length > 0);

  const body = (
    <div className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors min-w-0">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center">
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={displayName}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          fallbackIcon(kind)
        )}
      </div>
      <div className="flex-1 min-w-0 flex flex-col">
        {isLoading && !name ? (
          <Skeleton className="h-4 w-24" />
        ) : (
          <p className="text-sm font-medium truncate">{displayName}</p>
        )}
        {hasSubline && (
          <div className="text-xs text-muted-foreground flex items-center gap-1.5 flex-wrap">
            {affix && <span>{affix}</span>}
            {state !== undefined && (
              <Badge variant="outline" className="h-4 px-1 text-[10px]">
                {stateLabel(state)}
              </Badge>
            )}
            {jobs && jobs.length > 0 && (
              <span>
                for{" "}
                {jobs.map((j, i) => (
                  <span key={j}>
                    {i > 0 && ", "}
                    <JobName id={j} />
                  </span>
                ))}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          {route ? (
            <Link to={route} className="block min-w-0">
              {body}
            </Link>
          ) : (
            <div className="block min-w-0">{body}</div>
          )}
        </TooltipTrigger>
        <TooltipContent copyable>
          <p>{String(id)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
