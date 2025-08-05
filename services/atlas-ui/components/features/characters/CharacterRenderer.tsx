"use client";

import { useState, useEffect, useRef, useCallback } from 'react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { CharacterRendererDetailSkeleton } from '@/components/common/skeletons/CharacterDetailSkeleton';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { Asset } from '@/services/api/inventory.service';
import type { CharacterRendererProps, MapleStoryCharacterData } from '@/types/models/maplestory';
import { cn } from '@/lib/utils';

interface CharacterRendererComponentProps extends Omit<CharacterRendererProps, 'equipment'> {
  inventory?: Asset[];
  size?: 'small' | 'medium' | 'large';
  maxRetries?: number;
  showRetryButton?: boolean;
}

type ErrorType = 'api_error' | 'image_load_error' | 'network_error' | 'fallback_error' | 'unknown_error';

interface ErrorState {
  type: ErrorType;
  message: string;
  isRetryable: boolean;
}

const sizeClasses = {
  small: 'w-32 h-32',
  medium: 'w-48 h-48', 
  large: 'w-64 h-64'
};

const sizeDimensions = {
  small: { width: 128, height: 128 },
  medium: { width: 192, height: 192 },
  large: { width: 256, height: 256 }
};

export function CharacterRenderer({
  character,
  inventory = [],
  scale = 2,
  size = 'medium',
  showLoading = true,
  fallbackAvatar = '/default-character-avatar.svg',
  className,
  onImageLoad,
  onImageError,
  maxRetries = 3,
  showRetryButton = true,
}: CharacterRendererComponentProps) {
  const [imageUrl, setImageUrl] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const [fallbackImageError, setFallbackImageError] = useState(false);
  const mountedRef = useRef(true);
  
  // Utility function to classify errors
  const classifyError = useCallback((err: unknown): ErrorState => {
    if (err instanceof Error) {
      const message = err.message.toLowerCase();
      
      if (message.includes('network') || message.includes('fetch')) {
        return {
          type: 'network_error',
          message: 'Network connection failed. Please check your internet connection.',
          isRetryable: true
        };
      }
      
      if (message.includes('api') || message.includes('service')) {
        return {
          type: 'api_error', 
          message: 'Character rendering service is temporarily unavailable.',
          isRetryable: true
        };
      }
      
      if (message.includes('load') || message.includes('image')) {
        return {
          type: 'image_load_error',
          message: 'Failed to load character image.',
          isRetryable: true
        };
      }
      
      return {
        type: 'unknown_error',
        message: err.message || 'An unexpected error occurred.',
        isRetryable: true
      };
    }
    
    return {
      type: 'unknown_error',
      message: 'An unexpected error occurred while rendering character.',
      isRetryable: true
    };
  }, []);

  // Retry mechanism
  const handleRetry = useCallback(() => {
    if (retryCount < maxRetries) {
      setRetryCount(prev => prev + 1);
      setError(null);
      setFallbackImageError(false);
      setLoading(true);
    }
  }, [retryCount, maxRetries]);
  
  useEffect(() => {
    mountedRef.current = true;
    
    const generateCharacterImage = async () => {
      try {
        setLoading(true);
        setError(null);
        
        // Convert Character model to MapleStoryCharacterData
        const mapleStoryData: MapleStoryCharacterData = mapleStoryService.characterToMapleStoryData(
          character,
          inventory
        );
        
        // Generate character image
        const result = await mapleStoryService.generateCharacterImage(mapleStoryData, {
          resize: scale,
        });
        
        if (mountedRef.current) {
          setImageUrl(result.url);
          
          // Preload image to handle loading state properly
          const img = new window.Image();
          img.onload = () => {
            if (mountedRef.current) {
              setLoading(false);
              setRetryCount(0); // Reset retry count on success
              onImageLoad?.();
            }
          };
          img.onerror = () => {
            if (mountedRef.current) {
              const errorState = classifyError(new Error('Failed to load character image'));
              setError(errorState);
              setLoading(false);
              onImageError?.(new Error(errorState.message));
            }
          };
          img.src = result.url;
        }
      } catch (err) {
        if (mountedRef.current) {
          const errorState = classifyError(err);
          setError(errorState);
          setLoading(false);
          onImageError?.(new Error(errorState.message));
          
          // Log error in development
          if (process.env.NODE_ENV === 'development') {
            console.error('Character rendering error:', err);
          }
        }
      }
    };
    
    generateCharacterImage();
    
    return () => {
      mountedRef.current = false;
    };
  }, [character, inventory, scale, onImageLoad, onImageError, retryCount]);
  
  // Loading state
  if (loading && showLoading) {
    return (
      <CharacterRendererDetailSkeleton 
        size={size} 
        className={className}
      />
    );
  }
  
  // Error state - show fallback avatar with error handling and retry option
  if (error || !imageUrl) {
    // Handle fallback image error - create ultimate fallback
    const handleFallbackError = () => {
      setFallbackImageError(true);
    };

    // If both main image and fallback failed, show inline SVG
    if (fallbackImageError) {
      return (
        <div className={cn(sizeClasses[size], 'flex flex-col items-center justify-center bg-gray-100 border-2 border-dashed border-gray-300 rounded-lg', className)}>
          {/* Inline SVG fallback */}
          <svg width="48" height="48" viewBox="0 0 48 48" className="text-gray-400 mb-2">
            <circle cx="24" cy="18" r="6" fill="currentColor" opacity="0.3"/>
            <path d="M12 36c0-6.627 5.373-12 12-12s12 5.373 12 12" stroke="currentColor" strokeWidth="2" fill="none" opacity="0.3"/>
          </svg>
          <div className="text-xs text-gray-500 text-center px-2 mb-2">
            {error?.message || 'Character image unavailable'}
          </div>
          {error?.isRetryable && showRetryButton && retryCount < maxRetries && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleRetry}
              className="text-xs px-2 py-1"
            >
              Retry ({retryCount + 1}/{maxRetries})
            </Button>
          )}
        </div>
      );
    }

    // Show fallback avatar with potential retry option
    return (
      <div className={cn(sizeClasses[size], 'flex flex-col items-center justify-center', className)}>
        <div className="relative">
          <Image
            src={fallbackAvatar}
            alt={`${character.attributes.name} (fallback)`}
            width={sizeDimensions[size].width}
            height={sizeDimensions[size].height}
            className={cn('object-contain rounded-lg')}
            onError={handleFallbackError}
          />
        </div>
        {error && (
          <div className="mt-2">
            <div className="text-xs text-gray-500 text-center px-2 mb-1">
              {error.message}
            </div>
            {error.isRetryable && showRetryButton && retryCount < maxRetries && (
              <Button
                size="sm"
                variant="outline"
                onClick={handleRetry}
                className="text-xs px-2 py-1"
              >
                Retry ({retryCount + 1}/{maxRetries})
              </Button>
            )}
          </div>
        )}
      </div>
    );
  }
  
  // Success state - show character image
  return (
    <div className={cn(sizeClasses[size], 'flex items-center justify-center', className)}>
      <Image
        src={imageUrl}
        alt={character.attributes.name}
        width={sizeDimensions[size].width}
        height={sizeDimensions[size].height}
        className={cn('object-contain rounded-lg')}
        onError={() => {
          // Handle image load error by falling back to error state
          if (mountedRef.current) {
            const errorState = classifyError(new Error('Character image failed to load'));
            setError(errorState);
            setLoading(false);
            onImageError?.(new Error(errorState.message));
          }
        }}
      />
    </div>
  );
}

// Export skeleton component for external use
export function CharacterRendererSkeleton({ 
  size = 'medium', 
  className 
}: { 
  size?: 'small' | 'medium' | 'large';
  className?: string;
}) {
  return (
    <CharacterRendererDetailSkeleton 
      size={size} 
      className={className}
    />
  );
}