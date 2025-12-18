import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

/**
 * NpcCardSkeleton provides a loading state for individual NPC cards.
 * Matches the structure of NpcCard with image, name, and action buttons.
 */
export function NpcCardSkeleton() {
  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-2 flex justify-between items-start">
        <div className="flex items-center space-x-3 min-w-0 flex-1">
          {/* NPC Image skeleton */}
          <Skeleton className="w-12 h-12 rounded-md flex-shrink-0" />
          
          {/* NPC Name and ID skeleton */}
          <div className="min-w-0 flex-1">
            <Skeleton className="h-6 w-24 mb-1" /> {/* NPC name */}
            <Skeleton className="h-4 w-16" /> {/* NPC ID */}
          </div>
        </div>
        
        {/* Dropdown menu skeleton */}
        <Skeleton className="h-8 w-8 flex-shrink-0" />
      </CardHeader>
      
      <CardContent className="pb-2">
        <div className="flex space-x-2">
          {/* Action buttons skeleton */}
          <Skeleton className="h-8 w-8" /> {/* Shop button */}
          <Skeleton className="h-8 w-8" /> {/* Conversation button */}
        </div>
      </CardContent>
    </Card>
  );
}