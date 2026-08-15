import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useTenant } from "@/context/tenant-context";
import { usePendingChanges } from "@/lib/hooks/api/usePendingChanges";
import { createErrorFromUnknown } from "@/types/api/errors";
import type { PendingChange } from "@/services/api/pending-changes.service";
import { CancelPendingChangeDialog } from "@/components/features/characters/CancelPendingChangeDialog";

const TYPE_LABELS: Record<PendingChange["type"], string> = {
  NAME_CHANGE: "Name Change",
  WORLD_TRANSFER: "World Transfer",
};

const STATUS_VARIANTS: Record<
  PendingChange["status"],
  "default" | "secondary" | "destructive" | "outline"
> = {
  PENDING: "secondary",
  APPLIED: "default",
  CANCELLED: "outline",
  REJECTED: "destructive",
  EXPIRED: "outline",
};

function requestedValue(change: PendingChange): string {
  return change.type === "NAME_CHANGE"
    ? (change.requestedName ?? "")
    : `World ${change.destinationWorldId}`;
}

interface PendingChangesPanelProps {
  characterId: string;
  characterName?: string | undefined;
}

/**
 * Lists a character's pending cash-shop name-change and world-transfer
 * requests, with the operator able to cancel a still-`PENDING` record.
 * Read + cancel only — no create or edit affordance (FR-2.10).
 */
export function PendingChangesPanel({
  characterId,
  characterName,
}: PendingChangesPanelProps) {
  const { activeTenant } = useTenant();
  const {
    data: changes,
    isPending,
    error,
  } = usePendingChanges(activeTenant, characterId);
  const [cancelling, setCancelling] = useState<PendingChange | null>(null);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pending Changes</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {error && (
          <p className="text-sm text-destructive">
            {createErrorFromUnknown(error).message}
          </p>
        )}
        {!error && isPending && (
          <p className="text-sm text-muted-foreground">Loading...</p>
        )}
        {!error && !isPending && (!changes || changes.length === 0) && (
          <p className="text-sm text-muted-foreground">No pending changes.</p>
        )}
        {!error &&
          changes?.map((change) => (
            <div
              key={change.id}
              className="flex flex-col gap-1 rounded-md border p-3 text-sm"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium">{TYPE_LABELS[change.type]}</span>
                <Badge variant={STATUS_VARIANTS[change.status]}>
                  {change.status}
                </Badge>
              </div>
              <div>{requestedValue(change)}</div>
              {change.status === "PENDING" ? (
                <div className="text-xs text-muted-foreground">
                  Requested {change.createdAt} &middot; Expires{" "}
                  {change.expiresAt}
                </div>
              ) : (
                <div className="text-xs text-muted-foreground">
                  Resolved {change.resolvedAt ?? "unknown"}
                  {change.reason ? ` — ${change.reason}` : ""}
                </div>
              )}
              {change.status === "PENDING" && (
                <div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCancelling(change)}
                  >
                    Cancel
                  </Button>
                </div>
              )}
            </div>
          ))}
      </CardContent>
      {cancelling && (
        <CancelPendingChangeDialog
          change={cancelling}
          characterName={characterName ?? `Character ${characterId}`}
          characterId={characterId}
          open={!!cancelling}
          onOpenChange={(open) => {
            if (!open) setCancelling(null);
          }}
        />
      )}
    </Card>
  );
}
