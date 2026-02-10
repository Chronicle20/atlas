import { Skeleton } from "@/components/ui/skeleton";

export default function BansLoading() {
    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center justify-between">
                <Skeleton className="h-8 w-32" />
                <div className="flex items-center gap-4">
                    <Skeleton className="h-9 w-40" />
                    <Skeleton className="h-9 w-32" />
                </div>
            </div>
            <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                {Array.from({ length: 10 }).map((_, i) => (
                    <Skeleton key={i} className="h-12 w-full" />
                ))}
            </div>
        </div>
    );
}
