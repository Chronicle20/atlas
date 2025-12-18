"use client"

import { useState, useCallback, useEffect } from "react";
import Image from "next/image";
import { User, AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { errorLogger } from "@/services/errorLogger";
import { useLazyLoad } from "@/lib/hooks/useIntersectionObserver";
import { shouldUnoptimizeImageSrc, getImageLoadingStrategy } from "@/lib/utils/image";

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
  maintainLayout?: boolean; // Ensures consistent dimensions during load
  lazy?: boolean; // Enable lazy loading (default: true)
  lazyRootMargin?: string; // Custom root margin for lazy loading
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
  onRetry,
  maintainLayout = true,
  lazy = true,
  lazyRootMargin = "100px"
}: NpcImageProps) {
  const [imageError, setImageError] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [retryCount, setRetryCount] = useState(0);
  const [isRetrying, setIsRetrying] = useState(false);

  // Lazy loading hook
  const { shouldLoad, ref: lazyRef } = useLazyLoad<HTMLDivElement>({
    enabled: lazy,
    rootMargin: lazyRootMargin,
  });

  // Reset states when iconUrl changes
  useEffect(() => {
    if (iconUrl && typeof iconUrl === 'string') {
      setImageError(false);
      setIsLoading(true);
      setRetryCount(0);
      setIsRetrying(false);
    }
  }, [iconUrl]);

  // Determine if we should attempt to load the image
  const shouldAttemptLoad = !lazy || shouldLoad;
  
  // If no iconUrl provided, image failed to load, or lazy loading not triggered, show placeholder
  const showPlaceholder = !iconUrl || typeof iconUrl !== 'string' || imageError || !shouldAttemptLoad;
  const canRetry = retryCount < maxRetries && imageError && iconUrl && typeof iconUrl === 'string' && shouldAttemptLoad;

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
      
      if (onError) {
        onError(errorMsg);
      }
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
    if (onRetry) {
      onRetry();
    }
  }, [canRetry, npcId, retryCount, iconUrl, onRetry]);

  const altText = (typeof name === 'string' && name) ? `${name} (NPC ${npcId})` : `NPC ${npcId}`;

  return (
    <div 
      ref={lazyRef}
      className={cn(
        "relative overflow-hidden border border-border/50",
        maintainLayout && "flex-shrink-0", // Prevent flex shrinking
        className
      )}
      style={maintainLayout ? { width: size, height: size } : undefined}
    >
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
          ) : lazy && !shouldLoad ? (
            // Lazy loading placeholder
            <div className="flex flex-col items-center space-y-1">
              <User className="w-6 h-6 text-muted-foreground opacity-50" />
              <div className="w-2 h-2 bg-muted-foreground/30 rounded-full animate-pulse" />
            </div>
          ) : (
            <User className="w-6 h-6 text-muted-foreground" />
          )}
        </div>
      ) : (
        <>
          {/* Loading skeleton with progressive enhancement */}
          {(isLoading || isRetrying) && (
            <div className="absolute inset-0 bg-gradient-to-br from-muted to-muted/80 animate-pulse flex items-center justify-center">
              {isRetrying ? (
                <div className="flex flex-col items-center space-y-1">
                  <RefreshCw className="w-4 h-4 text-muted-foreground animate-spin" />
                  <span className="text-xs text-muted-foreground">Retrying...</span>
                </div>
              ) : (
                <div className="flex flex-col items-center space-y-1">
                  <User className="w-6 h-6 text-muted-foreground animate-pulse" />
                  <div className="w-8 h-1 bg-muted-foreground/20 rounded-full overflow-hidden">
                    <div className="w-full h-full bg-muted-foreground/40 animate-pulse" />
                  </div>
                </div>
              )}
            </div>
          )}
          
          {/* Actual image */}
          {iconUrl && typeof iconUrl === 'string' && (
            <Image
              key={`${npcId}-${iconUrl}-${retryCount}`} // Force re-render on retry
              src={iconUrl}
              alt={altText}
              width={size}
              height={size}
              className={cn(
                "w-full h-full object-contain transition-all duration-300 ease-in-out",
                (isLoading || isRetrying) ? "opacity-0 scale-95" : "opacity-100 scale-100"
              )}
              style={{ maxWidth: '100%', maxHeight: '100%' }}
              onLoad={handleImageLoad}
              onError={handleImageError}
              priority={false}
              loading={getImageLoadingStrategy()}
              unoptimized={shouldUnoptimizeImageSrc(iconUrl)}
            />
          )}
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