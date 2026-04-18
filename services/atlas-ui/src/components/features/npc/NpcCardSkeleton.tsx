import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

/**
 * NpcCardSkeleton provides a loading state for individual NPC cards.
 * Matches the structure of NpcCard with image, name, and action buttons.
 */
export function NpcCardSkeleton() {
  return (
    <Card className="overflow-hidden">
      <CardHeader className="p-2 pb-1 flex justify-between items-start">
        <div className="flex items-center space-x-2 min-w-0 flex-1">
          {/* NPC Image skeleton */}
          <Skeleton className="w-9 h-9 rounded-md flex-shrink-0" />

          {/* NPC Name and ID skeleton */}
          <div className="min-w-0 flex-1">
            <Skeleton className="h-4 w-20 mb-1" /> {/* NPC name */}
            <Skeleton className="h-3 w-12" /> {/* NPC ID */}
          </div>
        </div>

        {/* Dropdown menu skeleton */}
        <Skeleton className="h-6 w-6 flex-shrink-0" />
      </CardHeader>

      <CardContent className="p-2 pt-0">
        <div className="flex space-x-1">
          {/* Action buttons skeleton */}
          <Skeleton className="h-6 w-6" /> {/* Shop button */}
          <Skeleton className="h-6 w-6" /> {/* Conversation button */}
        </div>
      </CardContent>
    </Card>
  );
}