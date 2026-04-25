import { Skeleton } from "@/components/ui/skeleton"

export function QuestStatusSkeleton() {
    return (
        <div>
            <Skeleton className="h-10 w-full mb-4" />
            <div className="space-y-3">
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
            </div>
        </div>
    )
}
