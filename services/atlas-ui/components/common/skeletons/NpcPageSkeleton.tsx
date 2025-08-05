import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader } from '@/components/ui/card';

/**
 * NpcPageSkeleton provides a loading state specifically for the NPCs page.
 * Matches the card grid layout with header actions and individual NPC cards.
 */
export function NpcPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-4 h-[calc(100vh-4rem)] overflow-hidden">
      <div className="flex flex-col space-y-4">
        {/* Page title */}
        <Skeleton className="h-8 w-16" /> {/* "NPCs" title */}
        
        {/* Header actions */}
        <div className="flex items-center justify-between">
          <div className="flex gap-2 items-center">
            <Skeleton className="h-10 w-10" /> {/* Refresh button */}
          </div>
          <div className="flex items-center gap-2">
            <Skeleton className="h-9 w-20" /> {/* Actions dropdown */}
          </div>
        </div>
      </div>

      {/* Card grid content */}
      <div className="overflow-auto h-[calc(100vh-10rem)] pr-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 12 }).map((_, index) => (
            <Card key={`npc-skeleton-${index}`} className="overflow-hidden">
              <CardHeader className="pb-2 flex justify-between items-start">
                <Skeleton className="h-6 w-20" /> {/* NPC title */}
                <Skeleton className="h-8 w-8" /> {/* Menu button */}
              </CardHeader>
              <CardContent className="pb-2">
                <div className="text-sm flex space-x-2">
                  <Skeleton className="h-8 w-8" /> {/* Shop button */}
                  <Skeleton className="h-8 w-8" /> {/* Conversation button */}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}