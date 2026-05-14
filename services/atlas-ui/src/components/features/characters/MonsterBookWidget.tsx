import { useEffect } from "react";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import {
  useMonsterBookCardName,
  useMonsterBookCards,
  useMonsterBookCollection,
} from "@/lib/hooks/api/useMonsterBook";
import { createErrorFromUnknown } from "@/types/api/errors";
import type { MonsterBookCard, MonsterBookCollection } from "@/types/monster-book";
import type { Tenant } from "@/services/api/tenants.service";

const CARD_LEVEL_MAX = 5;

interface Props {
  characterId: number;
}

/**
 * Monster Book widget rendered on the character detail page. Surfaces the
 * collection summary plus an infinite-scroll list of owned cards. Card
 * names are resolved by chasing `consumable.monsterId → monster.name` via
 * the atlas-data REST endpoints; missing relations fall back to the raw
 * card id so an incomplete data dump doesn't blank the row.
 */
export function MonsterBookWidget({ characterId }: Props) {
  const { activeTenant } = useTenant();

  const collectionQuery = useMonsterBookCollection(activeTenant, characterId);
  const cardsQuery = useMonsterBookCards(activeTenant, characterId);

  // Surface load failures via toast in addition to the inline retry chip.
  // The error helper normalises unknown shapes (network/api/runtime) into
  // a single message string for sonner.
  useEffect(() => {
    if (collectionQuery.isError) {
      toast.error(createErrorFromUnknown(collectionQuery.error).message);
    }
  }, [collectionQuery.isError, collectionQuery.error]);

  if (collectionQuery.isLoading || (!activeTenant && !collectionQuery.isError)) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Monster Book</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton data-testid="monster-book-loading" className="h-32 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (collectionQuery.isError || !collectionQuery.data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Monster Book</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <p className="text-sm text-muted-foreground">
              Failed to load Monster Book.
            </p>
            <Button size="sm" variant="outline" onClick={() => collectionQuery.refetch()}>
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  const collection = collectionQuery.data;
  const cards: MonsterBookCard[] = (cardsQuery.data?.pages ?? []).flat();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            {collection.coverCardId > 0 && activeTenant ? (
              <CoverImage cardId={collection.coverCardId} tenant={activeTenant} />
            ) : (
              <div className="h-12 w-12 rounded border bg-muted" aria-hidden />
            )}
            <CardTitle>Monster Book</CardTitle>
          </div>
          <Badge variant="secondary">Lv. {collection.bookLevel}</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatsRow collection={collection} />
        {collection.totalUniqueCards === 0 ? (
          <p className="text-sm text-muted-foreground">No cards collected yet.</p>
        ) : (
          <CardList
            cards={cards}
            tenant={activeTenant}
            isLoadingMore={cardsQuery.isFetchingNextPage}
            hasNextPage={cardsQuery.hasNextPage ?? false}
            onLoadMore={() => cardsQuery.fetchNextPage()}
          />
        )}
      </CardContent>
    </Card>
  );
}

function StatsRow({ collection }: { collection: MonsterBookCollection }) {
  return (
    <div className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-5">
      <Stat label="Book Level" value={collection.bookLevel} />
      <Stat label="Unique" value={collection.totalUniqueCards} />
      <Stat label="Normal" value={collection.normalCount} />
      <Stat label="Special" value={collection.specialCount} />
      <Stat label="EXP Bonus" value={`${collection.expBonusPercent}%`} />
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex flex-col rounded border p-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}

function CardList({
  cards, tenant, isLoadingMore, hasNextPage, onLoadMore,
}: {
  cards: MonsterBookCard[];
  tenant: Tenant | null;
  isLoadingMore: boolean;
  hasNextPage: boolean;
  onLoadMore: () => void;
}) {
  return (
    <div className="space-y-2">
      <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {cards.map((card) => (
          <li key={card.cardId}>
            <CardRow card={card} tenant={tenant} />
          </li>
        ))}
      </ul>
      {hasNextPage && (
        <div className="flex justify-center">
          <Button size="sm" variant="outline" onClick={onLoadMore} disabled={isLoadingMore}>
            {isLoadingMore ? "Loading..." : "Load more"}
          </Button>
        </div>
      )}
    </div>
  );
}

function CardRow({ card, tenant }: { card: MonsterBookCard; tenant: Tenant | null }) {
  const { monsterId, name } = useMonsterBookCardName(tenant, card.cardId);

  const displayName = name ?? `Card ${card.cardId}`;
  const iconUrl = tenant && monsterId
    ? getAssetIconUrl(
        tenant.id,
        tenant.attributes.region,
        tenant.attributes.majorVersion,
        tenant.attributes.minorVersion,
        "mob",
        monsterId,
      )
    : null;

  return (
    <div className="flex items-center gap-3 rounded border p-2">
      {iconUrl ? (
        <img
          src={iconUrl}
          alt={displayName}
          width={32}
          height={32}
          loading="lazy"
          className="h-8 w-8 object-contain"
        />
      ) : (
        <div className="h-8 w-8 rounded bg-muted" aria-hidden />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">{displayName}</span>
          {card.isSpecial && (
            <Badge variant="default" className="shrink-0">Special</Badge>
          )}
        </div>
        <span className="text-xs text-muted-foreground">
          {card.level} / {CARD_LEVEL_MAX}
        </span>
      </div>
    </div>
  );
}

function CoverImage({ cardId, tenant }: { cardId: number; tenant: Tenant }) {
  const { monsterId } = useMonsterBookCardName(tenant, cardId);
  if (!monsterId) {
    return <div className="h-12 w-12 rounded border bg-muted" aria-hidden />;
  }
  const url = getAssetIconUrl(
    tenant.id,
    tenant.attributes.region,
    tenant.attributes.majorVersion,
    tenant.attributes.minorVersion,
    "mob",
    monsterId,
  );
  return (
    <img
      src={url}
      alt="Monster Book cover"
      width={48}
      height={48}
      loading="lazy"
      className="h-12 w-12 rounded border object-contain"
    />
  );
}
