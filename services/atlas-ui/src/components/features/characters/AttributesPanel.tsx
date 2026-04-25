import type React from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { Link } from "react-router-dom";
import { MapCell } from "@/components/map-cell";
import { HpMpBar } from "./HpMpBar";
import { useCharacterGuild } from "@/lib/hooks/api/useCharacterGuild";
import type { Character } from "@/types/models/character";
import type { Tenant, TenantConfig } from "@/services/api/tenants.service";

interface Props {
  character: Character;
  tenantConfig: TenantConfig;
  tenant: Tenant;
}

const GENDER_LABEL = (g: number) => (g === 0 ? "Male" : "Female");
const mesoFmt = (n: number) => new Intl.NumberFormat().format(n);

export function AttributesPanel({ character, tenantConfig, tenant }: Props) {
  const a = character.attributes;
  const worldName = tenantConfig.attributes.worlds[a.worldId]?.name ?? "Unknown";

  return (
    <Card className="flex-1">
      <CardContent className="space-y-3 pt-4">
        {/* Group A — identity badges */}
        <div className="flex items-center gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="secondary" tabIndex={0} className="cursor-help">
                  {worldName}
                </Badge>
              </TooltipTrigger>
              <TooltipContent copyable>
                <p>{String(a.worldId)}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="secondary" tabIndex={0} className="cursor-help">
                  {GENDER_LABEL(a.gender)}
                </Badge>
              </TooltipTrigger>
              <TooltipContent copyable>
                <p>{String(a.gender)}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>

        {/* Group B — progression */}
        <div className="grid grid-cols-3 gap-2 text-sm">
          <div>
            <strong>Level:</strong> {a.level}
          </div>
          <div>
            <strong>Experience:</strong> {a.experience}
          </div>
          <div className="flex items-center gap-1">
            <strong>Map:</strong>{" "}
            <Link to={`/maps/${a.mapId}`}>
              <MapCell mapId={String(a.mapId)} tenant={tenant} />
            </Link>
          </div>
        </div>

        {/* Group C — core stats */}
        <div className="grid grid-cols-4 gap-2 text-sm">
          <div>
            <strong>STR:</strong> {a.strength}
          </div>
          <div>
            <strong>DEX:</strong> {a.dexterity}
          </div>
          <div>
            <strong>INT:</strong> {a.intelligence}
          </div>
          <div>
            <strong>LUK:</strong> {a.luck}
          </div>
        </div>

        {/* Group D — vitals & resources */}
        <div className="grid grid-cols-4 gap-2 text-sm">
          <HpMpBar label="HP" cur={a.hp} max={a.maxHp} colorClass="bg-red-500/70" />
          <HpMpBar label="MP" cur={a.mp} max={a.maxMp} colorClass="bg-blue-500/70" />
          <div>
            <strong>Mesos:</strong> {mesoFmt(a.meso)}
          </div>
          <div>
            <strong>Fame:</strong> {a.fame}
          </div>
        </div>

        {/* Group E — affiliations */}
        <div className="grid grid-cols-2 gap-2 text-sm">
          <GuildRow tenant={tenant} characterId={character.id} />
          <div className="opacity-70">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span tabIndex={0}>
                    <strong>Alliance:</strong> Not available
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  Alliance data is not yet exposed by the backend.
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function GuildRow({
  tenant,
  characterId,
}: {
  tenant: Tenant;
  characterId: string;
}) {
  const { guild, isLoading, error } = useCharacterGuild(tenant, characterId);
  let body: React.ReactNode = "None";
  if (isLoading) body = <Skeleton className="h-4 w-20 inline-block" />;
  else if (error) body = "Unknown";
  else if (guild) body = guild.attributes.name;
  return (
    <div>
      <strong>Guild:</strong> {body}
    </div>
  );
}
