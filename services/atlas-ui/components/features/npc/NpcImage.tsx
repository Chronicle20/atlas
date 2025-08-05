"use client"

import { useState, useCallback, useEffect } from "react";
import Image from "next/image";
import { User, AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { errorLogger } from "@/services/errorLogger";

interface NpcImageProps {
  npcId: number;
  name?: string;
  iconUrl?: string;
  className?: string;
  size?: number;
  showRetryButton?: boolean;
  maxRetries?: number;
  onError?: (error: string) => void;
  onRetry?: () => void;
}

export function NpcImage({ 
  npcId, 
  name, 
  iconUrl, 
  className,
  size = 48,
  showRetryButton = false,
  maxRetries = 2,
  onError,
  onRetry
}: NpcImageProps) {
  const [imageError, setImageError] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [retryCount, setRetryCount] = useState(0);
  const [isRetrying, setIsRetrying] = useState(false);

  // Reset states when iconUrl changes
  useEffect(() => {
    if (iconUrl) {
      setImageError(false);
      setIsLoading(true);
      setRetryCount(0);
      setIsRetrying(false);
    }
  }, [iconUrl]);

  // If no iconUrl provided or image failed to load, show placeholder
  const showPlaceholder = !iconUrl || imageError;
  const canRetry = retryCount < maxRetries && imageError && iconUrl;

  const handleImageLoad = useCallback(() => {
    setIsLoading(false);
    setImageError(false);
    setIsRetrying(false);
  }, []);

  const handleImageError = useCallback(() => {
    setIsLoading(false);
    setIsRetrying(false);
    
    // Only set error if we haven't exceeded retry limit
    if (retryCount < maxRetries) {
      // Auto-retry with exponential backoff
      const retryDelay = Math.min(1000 * Math.pow(2, retryCount), 5000);
      setTimeout(() => {
        setRetryCount(prev => prev + 1);
        setIsRetrying(true);
        setIsLoading(true);
        setImageError(false);
      }, retryDelay);
    } else {
      setImageError(true);
      
      // Log error for monitoring
      const errorMsg = `Failed to load NPC image after ${maxRetries} retries`;
      errorLogger.logError(new Error(errorMsg), undefined, {
        userId: npcId.toString(), // Use userId field for NPC ID
        url: iconUrl || 'unknown',
      });
      
      onError?.(errorMsg);
    }
  }, [retryCount, maxRetries, npcId, iconUrl, onError]);

  const handleManualRetry = useCallback(() => {
    if (!canRetry) return;
    
    errorLogger.logUserAction('npc_image_manual_retry', { 
      npcId, 
      retryCount,
      iconUrl 
    });
    
    setRetryCount(prev => prev + 1);
    setIsRetrying(true);
    setIsLoading(true);
    setImageError(false);
    onRetry?.();
  }, [canRetry, npcId, retryCount, iconUrl, onRetry]);

  const altText = name ? `${name} (NPC ${npcId})` : `NPC ${npcId}`;

  return (
    <div className={cn(
      "relative overflow-hidden border border-border/50",
      className
    )}>
      {showPlaceholder ? (
        // Placeholder when no image or error
        <div className={cn(
          "w-full h-full flex flex-col items-center justify-center bg-muted",
          isLoading && "animate-pulse"
        )}>
          {isRetrying ? (
            <RefreshCw className="w-4 h-4 text-muted-foreground animate-spin" />
          ) : imageError ? (
            <>
              <AlertTriangle className="w-4 h-4 text-muted-foreground mb-1" />
              {showRetryButton && canRetry && (
                <Button
                  onClick={handleManualRetry}
                  size="sm"
                  variant="ghost"
                  className="h-6 px-2 text-xs"
                >
                  <RefreshCw className="w-3 h-3 mr-1" />
                  Retry
                </Button>
              )}
            </>
          ) : (
            <User className="w-6 h-6 text-muted-foreground" />
          )}
        </div>
      ) : (
        <>
          {/* Loading skeleton */}
          {(isLoading || isRetrying) && (
            <div className="absolute inset-0 bg-muted animate-pulse flex items-center justify-center">
              {isRetrying ? (
                <RefreshCw className="w-4 h-4 text-muted-foreground animate-spin" />
              ) : (
                <User className="w-6 h-6 text-muted-foreground" />
              )}
            </div>
          )}
          
          {/* Actual image */}
          <Image
            key={`${iconUrl}-${retryCount}`} // Force re-render on retry
            src={iconUrl}
            alt={altText}
            width={size}
            height={size}
            className={cn(
              "object-cover transition-opacity duration-200",
              (isLoading || isRetrying) ? "opacity-0" : "opacity-100"
            )}
            onLoad={handleImageLoad}
            onError={handleImageError}
            priority={false}
            unoptimized // MapleStory.io images are external
          />
        </>
      )}
      
      {/* Error indicator for development */}
      {process.env.NODE_ENV === 'development' && imageError && (
        <div className="absolute bottom-0 right-0 bg-destructive text-destructive-foreground text-xs px-1 rounded-tl">
          {retryCount}/{maxRetries}
        </div>
      )}
    </div>
  );
}