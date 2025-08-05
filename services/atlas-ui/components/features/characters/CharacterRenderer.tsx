"use client";

import { useState, useEffect, useRef } from 'react';
import Image from 'next/image';
import { Skeleton } from '@/components/ui/skeleton';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { Asset } from '@/services/api/inventory.service';
import type { CharacterRendererProps, MapleStoryCharacterData } from '@/types/models/maplestory';
import { cn } from '@/lib/utils';

interface CharacterRendererComponentProps extends Omit<CharacterRendererProps, 'equipment'> {
  inventory?: Asset[];
  size?: 'small' | 'medium' | 'large';
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
  fallbackAvatar = '/default-character-avatar.png',
  className,
  onImageLoad,
  onImageError,
}: CharacterRendererComponentProps) {
  const [imageUrl, setImageUrl] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);
  
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
              onImageLoad?.();
            }
          };
          img.onerror = () => {
            if (mountedRef.current) {
              const errorMsg = 'Failed to load character image';
              setError(errorMsg);
              setLoading(false);
              onImageError?.(new Error(errorMsg));
            }
          };
          img.src = result.url;
        }
      } catch (err) {
        if (mountedRef.current) {
          const errorMsg = err instanceof Error ? err.message : 'Failed to generate character image';
          setError(errorMsg);
          setLoading(false);
          onImageError?.(new Error(errorMsg));
          
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
  }, [character, inventory, scale, onImageLoad, onImageError]);
  
  // Loading state
  if (loading && showLoading) {
    return (
      <div className={cn(sizeClasses[size], 'flex items-center justify-center', className)}>
        <Skeleton 
          className={cn(sizeClasses[size], 'rounded-lg')}
          variant="rectangular"
          animation="pulse"
        />
      </div>
    );
  }
  
  // Error state - show fallback avatar
  if (error || !imageUrl) {
    return (
      <div className={cn(sizeClasses[size], 'flex items-center justify-center', className)}>
        <Image
          src={fallbackAvatar}
          alt={`${character.attributes.name} (fallback)`}
          width={sizeDimensions[size].width}
          height={sizeDimensions[size].height}
          className={cn('object-contain rounded-lg')}
          onError={() => {
            // If fallback also fails, handled by Next.js Image component
          }}
        />
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
            setError('Image failed to load');
            setLoading(false);
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
    <div className={cn(sizeClasses[size], 'flex items-center justify-center', className)}>
      <Skeleton 
        className={cn(sizeClasses[size], 'rounded-lg')}
        variant="rectangular"
        animation="pulse"
      />
    </div>
  );
}