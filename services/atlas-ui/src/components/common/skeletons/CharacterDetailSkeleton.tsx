import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader } from '@/components/ui/card';

/**
 * CharacterDetailSkeleton provides a detailed loading state for the Character Detail page.
 * Matches the exact layout structure with character rendering, attributes, and inventory sections.
 */
export function CharacterDetailSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen overflow-auto">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton className="h-8 w-48" /> {/* Character name */}
        </div>
      </div>

      {/* Main content area - Character rendering and attributes */}
      <div className="flex flex-row gap-6">
        {/* Character Rendering Card */}
        <Card className="w-auto flex-shrink-0">
          <CardHeader>
            <Skeleton className="h-6 w-20" /> {/* "Character" title */}
          </CardHeader>
          <CardContent className="flex justify-center">
            <div className="w-64 h-64 flex items-center justify-center">
              {/* Enhanced character rendering skeleton with character-like shape */}
              <div className="relative w-48 h-48">
                {/* Character silhouette skeleton */}
                <Skeleton className="absolute inset-0 rounded-lg" />
                {/* Character features overlay for more realistic loading */}
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="relative">
                    {/* Head area */}
                    <Skeleton className="w-16 h-20 rounded-full mx-auto mb-2 opacity-60" />
                    {/* Body area */}
                    <Skeleton className="w-12 h-24 rounded-sm mx-auto mb-2 opacity-60" />
                    {/* Legs area */}
                    <div className="flex justify-center gap-1">
                      <Skeleton className="w-4 h-16 rounded-sm opacity-60" />
                      <Skeleton className="w-4 h-16 rounded-sm opacity-60" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Character Attributes Card */}
        <Card className="flex-1">
          <CardHeader>
            <div className="flex items-center justify-between">
              <Skeleton className="h-6 w-20" /> {/* "Attributes" title */}
              <Skeleton className="h-8 w-24" /> {/* "Change Map" button */}
            </div>
          </CardHeader>
          <CardContent className="grid grid-cols-4 gap-2 text-sm">
            {/* Attribute skeleton items - 9 total attributes */}
            {Array.from({ length: 9 }).map((_, index) => (
              <div key={index} className="space-y-1">
                <Skeleton className="h-3 w-16" /> {/* Attribute label */}
                <Skeleton className="h-4 w-12" /> {/* Attribute value */}
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      {/* Inventory Section */}
      <div className="space-y-4">
        <Skeleton className="h-6 w-20" /> {/* "Inventory" title */}
        
        <div className="grid grid-cols-1 gap-4">
          {/* Skeleton for compartments - typically 3-5 compartments */}
          {Array.from({ length: 4 }).map((_, compartmentIndex) => (
            <Card key={compartmentIndex} className="border rounded-md">
              {/* Compartment header */}
              <div className="flex justify-between items-center w-full p-4">
                <div className="flex items-center gap-2">
                  <Skeleton className="h-5 w-24" /> {/* Compartment name */}
                </div>
                <Skeleton className="h-4 w-12" /> {/* Capacity count */}
              </div>
              
              {/* Compartment content */}
              <div className="p-4 pt-0">
                <div className="flex flex-wrap gap-3 pt-4">
                  {/* Asset skeleton items - vary the count per compartment */}
                  {Array.from({ length: compartmentIndex === 0 ? 8 : 3 + compartmentIndex }).map((_, assetIndex) => (
                    <Card key={assetIndex} className="overflow-hidden relative py-0 w-[100px]">
                      <div className="absolute top-0 right-0 p-1">
                        <Skeleton className="h-4 w-4 rounded-sm" /> {/* Delete button */}
                      </div>
                      <div className="p-1 pl-3 pb-1 text-left">
                        <Skeleton className="h-4 w-6" /> {/* Slot number */}
                      </div>
                      <div className="p-2 pt-0 text-center">
                        <Skeleton className="h-5 w-16" /> {/* Template ID */}
                      </div>
                    </Card>
                  ))}
                </div>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}

/**
 * Enhanced Character Renderer Skeleton specifically for the character detail page.
 * Provides a more detailed loading state with character-like shape.
 */
export function CharacterRendererDetailSkeleton({ 
  size = 'large', 
  className 
}: { 
  size?: 'small' | 'medium' | 'large';
  className?: string | undefined;
}) {
  const sizeClasses = {
    small: 'w-32 h-32',
    medium: 'w-48 h-48', 
    large: 'w-64 h-64'
  };

  const skeletonSizes = {
    small: { head: 'w-8 h-10', body: 'w-6 h-12', leg: 'w-2 h-8' },
    medium: { head: 'w-12 h-15', body: 'w-9 h-18', leg: 'w-3 h-12' },
    large: { head: 'w-16 h-20', body: 'w-12 h-24', leg: 'w-4 h-16' }
  };

  const currentSizes = skeletonSizes[size];

  return (
    <div className={`${sizeClasses[size]} flex items-center justify-center ${className}`}>
      <div className="relative">
        {/* Base skeleton */}
        <Skeleton className={`${sizeClasses[size]} rounded-lg absolute inset-0`} />
        
        {/* Character-like overlay for better UX */}
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="relative">
            {/* Head */}
            <Skeleton className={`${currentSizes.head} rounded-full mx-auto mb-2 opacity-60`} />
            {/* Body */}
            <Skeleton className={`${currentSizes.body} rounded-sm mx-auto mb-2 opacity-60`} />
            {/* Legs */}
            <div className="flex justify-center gap-1">
              <Skeleton className={`${currentSizes.leg} rounded-sm opacity-60`} />
              <Skeleton className={`${currentSizes.leg} rounded-sm opacity-60`} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}