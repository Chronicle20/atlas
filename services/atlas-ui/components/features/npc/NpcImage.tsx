"use client"

import { useState } from "react";
import Image from "next/image";
import { User } from "lucide-react";
import { cn } from "@/lib/utils";

interface NpcImageProps {
  npcId: number;
  name?: string;
  iconUrl?: string;
  className?: string;
  size?: number;
}

export function NpcImage({ 
  npcId, 
  name, 
  iconUrl, 
  className,
  size = 48 
}: NpcImageProps) {
  const [imageError, setImageError] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // If no iconUrl provided or image failed to load, show placeholder
  const showPlaceholder = !iconUrl || imageError;

  const handleImageLoad = () => {
    setIsLoading(false);
  };

  const handleImageError = () => {
    setImageError(true);
    setIsLoading(false);
  };

  const altText = name ? `${name} (NPC ${npcId})` : `NPC ${npcId}`;

  return (
    <div className={cn(
      "relative overflow-hidden border border-border/50",
      className
    )}>
      {showPlaceholder ? (
        // Placeholder when no image or error
        <div className={cn(
          "w-full h-full flex items-center justify-center bg-muted",
          isLoading && "animate-pulse"
        )}>
          <User className="w-6 h-6 text-muted-foreground" />
        </div>
      ) : (
        <>
          {/* Loading skeleton */}
          {isLoading && (
            <div className="absolute inset-0 bg-muted animate-pulse flex items-center justify-center">
              <User className="w-6 h-6 text-muted-foreground" />
            </div>
          )}
          
          {/* Actual image */}
          <Image
            src={iconUrl}
            alt={altText}
            width={size}
            height={size}
            className={cn(
              "object-cover transition-opacity duration-200",
              isLoading ? "opacity-0" : "opacity-100"
            )}
            onLoad={handleImageLoad}
            onError={handleImageError}
            priority={false}
            unoptimized // MapleStory.io images are external
          />
        </>
      )}
    </div>
  );
}