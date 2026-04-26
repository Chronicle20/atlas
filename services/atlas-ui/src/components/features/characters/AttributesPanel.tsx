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
import { useCharacterEffectiveStats } from "@/lib/hooks/api/useCharacterEffectiveStats";
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

  // atlas-effective-stats returns post-equip primary stats and HP/MP caps.
  // While loading or on error we fall through to the raw character record so
  // the panel stays populated. `error` is exposed (but not surfaced visually
  // by default) so a missing/unhealthy backend service is observable in the
  // browser console rather than silently rendering base values.
  const {
    data: effective,
    isError: effectiveErrored,
    error: effectiveError,
  } = useCharacterEffectiveStats(tenant, a.worldId, character.id);

  if (effectiveErrored) {
    // eslint-disable-next-line no-console
    console.warn(
      "[AttributesPanel] atlas-effective-stats fetch failed; HP/MP and primary-stat bonuses are using the raw character record as a fallback.",
      effectiveError,
    );
  }

  const hpBonus = effective ? Math.max(0, effective.maxHP - a.maxHp) : 0;
  const mpBonus = effective ? Math.max(0, effective.maxMP - a.maxMp) : 0;

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

        {/* Group C — core stats (base from character + bonus from effective) */}
        <div className="grid grid-cols-4 gap-2 text-sm">
          <StatCell label="STR" base={a.strength} effective={effective?.strength} />
          <StatCell label="DEX" base={a.dexterity} effective={effective?.dexterity} />
          <StatCell label="INT" base={a.intelligence} effective={effective?.intelligence} />
          <StatCell label="LUK" base={a.luck} effective={effective?.luck} />
        </div>

        {/* Group D — vitals & resources (HP/MP cap from effective when available) */}
        <div className="grid grid-cols-4 gap-2 text-sm">
          <HpMpBar
            label="HP"
            cur={a.hp}
            max={effective?.maxHP ?? a.maxHp}
            bonus={hpBonus}
            colorClass="bg-red-500/70"
          />
          <HpMpBar
            label="MP"
            cur={a.mp}
            max={effective?.maxMP ?? a.maxMp}
            bonus={mpBonus}
            colorClass="bg-blue-500/70"
          />
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

// Renders `base +bonus` when atlas-effective-stats has resolved a higher
// value than the character's raw stat (i.e. the diff is positive), or the
// bare base while loading / when there's no diff.
function StatCell({
  label,
  base,
  effective,
}: {
  label: string;
  base: number;
  effective: number | undefined;
}) {
  const bonus = effective != null ? effective - base : 0;
  return (
    <div>
      <strong>{label}:</strong> {base}
      {bonus > 0 && (
        <span className="text-emerald-600 dark:text-emerald-400"> +{bonus}</span>
      )}
    </div>
  );
}
