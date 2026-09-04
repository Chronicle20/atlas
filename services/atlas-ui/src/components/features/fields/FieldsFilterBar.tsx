import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { WorldData, ChannelData } from "@/services/api/worlds.service";

const ANY_CHANNEL = "__any__";

export interface FieldsFilterBarProps {
  worlds: WorldData[];
  channels: ChannelData[];
  worldId: number;
  channelId: number | null;
  mapQuery: string;
  resultCount: number;
  onWorldChange: (worldId: number) => void;
  onChannelChange: (channelId: number | null) => void;
  onMapQueryChange: (query: string) => void;
  onClear: () => void;
}

/**
 * World / channel / map filters for the Fields locator (FR-11/FR-12/FR-13),
 * styled as the app's search card (`pages/ItemsPage.tsx`'s Search Items
 * card is the reference, bug-fields-ui item 19). World and channel are
 * shadcn `Select`s sourced entirely from the API (`useWorlds`/`useChannels`
 * results passed down as props) — there is no hard-coded option list. The
 * map filter is a free-text `Input`, not a select: FR-13 requires a
 * *search* over name and id, not a pick from every map. `mapQuery`/
 * `onMapQueryChange` make it a fully controlled input — the raw text and
 * its debouncing are owned by the parent (`FieldsPage`), so there is a
 * single source of truth and no remount is needed to reset it.
 *
 * The channel `Select`'s option labels display 1-indexed (bug-fields-ui
 * item 15) — the option *values*, the selected state, and the API filter
 * all stay 0-based.
 */
export function FieldsFilterBar({
  worlds,
  channels,
  worldId,
  channelId,
  mapQuery,
  resultCount,
  onWorldChange,
  onChannelChange,
  onMapQueryChange,
  onClear,
}: FieldsFilterBarProps) {
  const sortedWorlds = [...worlds].sort((a, b) => Number(a.id) - Number(b.id));

  return (
    <Card>
      <CardHeader>
        <CardTitle>Search Fields</CardTitle>
        <CardDescription>
          Filter by world, channel, and map name or id. {resultCount}{" "}
          {resultCount === 1 ? "field" : "fields"} match.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-4 items-end">
          <div className="flex-1 relative">
            <Input
              placeholder="Filter by map name or id..."
              aria-label="Map filter"
              value={mapQuery}
              onChange={(event) => onMapQueryChange(event.target.value)}
            />
          </div>
          <Button variant="outline" onClick={onClear}>
            Clear
          </Button>
        </div>

        <div className="flex flex-wrap gap-3 items-center">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">World</span>
            <Select
              value={String(worldId)}
              onValueChange={(value) => onWorldChange(Number(value))}
            >
              <SelectTrigger className="w-[160px]" aria-label="World">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {sortedWorlds.map((world) => (
                  <SelectItem key={world.id} value={world.id}>
                    {world.attributes.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Channel</span>
            <Select
              value={channelId === null ? ANY_CHANNEL : String(channelId)}
              onValueChange={(value) =>
                onChannelChange(value === ANY_CHANNEL ? null : Number(value))
              }
            >
              <SelectTrigger className="w-[160px]" aria-label="Channel">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ANY_CHANNEL}>Any channel</SelectItem>
                {channels.map((channel) => (
                  <SelectItem
                    key={channel.id}
                    value={String(channel.attributes.channelId)}
                  >
                    {channel.attributes.channelId + 1}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
