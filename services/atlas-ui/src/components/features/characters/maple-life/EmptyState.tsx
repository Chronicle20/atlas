import { EmptyState } from "@/components/common/EmptyState";
import { Button } from "@/components/ui/button";

const TENANT_EMPTY_TITLE = "Maple Life is disabled for this tenant";
const TENANT_EMPTY_BODY =
  "There is no mapleLife block on this configuration, so a player using a Cash/0543 item will fail with ErrMapleLifeNotConfigured.";
const TEMPLATE_EMPTY_TITLE = "This template has no Maple Life block";
const TEMPLATE_EMPTY_BODY =
  "Add the ten class rows to author one. Nothing is persisted until you save.";

interface MapleLifeEmptyStateProps {
  /** Tenant context shows the seed action; template context shows "add rows". */
  onSeed?: () => void;
  onAddRows: () => void;
}

/**
 * Wraps the shared `EmptyState` with Maple-Life-specific copy (FR-12).
 * Tenant callers pass `onSeed` and get an additional "Seed from template"
 * action; template callers omit it and only see "Add the ten class rows".
 */
export function MapleLifeEmptyState({
  onSeed,
  onAddRows,
}: MapleLifeEmptyStateProps) {
  const title = onSeed ? TENANT_EMPTY_TITLE : TEMPLATE_EMPTY_TITLE;
  const description = onSeed ? TENANT_EMPTY_BODY : TEMPLATE_EMPTY_BODY;

  return (
    <div className="flex flex-col items-center gap-4">
      <EmptyState title={title} description={description} />
      <div className="flex items-center justify-center gap-2">
        {onSeed && <Button onClick={onSeed}>Seed from template</Button>}
        <Button variant="outline" onClick={onAddRows}>
          Add the ten class rows
        </Button>
      </div>
    </div>
  );
}
