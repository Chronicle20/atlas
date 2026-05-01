/**
 * Optimized wrapper for CharacterRenderer with advanced performance features
 */

import { memo, useMemo, Suspense, lazy, Component, type ErrorInfo, type ReactNode } from 'react';
import { CharacterRenderer, CharacterRendererSkeleton } from './CharacterRenderer';
import { useCharacterImagePreloader } from '@/lib/hooks/useCharacterImage';
import { characterToLoadout } from '@/services/api/characterRender.service';
import { useTenant } from '@/context/tenant-context';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

// Simple Error Boundary implementation
interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  fallbackRender: (props: { error: Error; resetErrorBoundary: () => void }) => ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.props.onError?.(error, errorInfo);
  }

  resetErrorBoundary = () => {
    this.setState({ hasError: false });
  };

  override render() {
    if (this.state.hasError && this.state.error) {
      return this.props.fallbackRender({
        error: this.state.error,
        resetErrorBoundary: this.resetErrorBoundary,
      });
    }

    return this.props.children;
  }
}

interface OptimizedCharacterRendererProps {
  character: Character;
  inventory?: Asset[];
  scale?: number;
  size?: 'small' | 'medium' | 'large';
  priority?: boolean;
  lazy?: boolean;
  showLoading?: boolean;
  fallbackAvatar?: string;
  className?: string;
  onImageLoad?: () => void;
  onImageError?: (error: Error) => void;
  region?: string;
  majorVersion?: number;
  
  // Performance optimization options
  enablePreload?: boolean;
  prefetchVariants?: boolean;
  enableErrorBoundary?: boolean;
  suspenseFallback?: React.ReactNode;
  
  // Batch preloading options
  preloadSiblings?: Array<{
    character: Character;
    inventory?: Asset[];
    scale?: number;
  }>;
}

// Error fallback component
function CharacterRenderErrorFallback({ 
  error, 
  resetErrorBoundary,
  character,
  className 
}: {
  error: Error;
  resetErrorBoundary: () => void;
  character: Character;
  className?: string;
}) {
  return (
    <div className={`flex flex-col items-center justify-center p-4 border-2 border-dashed border-red-300 rounded-lg bg-red-50 ${className}`}>
      <div className="text-red-600 text-sm mb-2">
        Failed to render {character.attributes.name}
      </div>
      <button
        onClick={resetErrorBoundary}
        className="text-xs bg-red-100 hover:bg-red-200 px-2 py-1 rounded border border-red-300"
      >
        Retry
      </button>
      {import.meta.env.DEV && (
        <details className="mt-2 text-xs text-red-500">
          <summary>Error details</summary>
          <pre className="mt-1 text-xs">{error.message}</pre>
        </details>
      )}
    </div>
  );
}

// Dynamically import character renderer for code splitting
const DynamicCharacterRenderer = lazy(
  () => import('./CharacterRenderer').then(mod => ({ default: mod.CharacterRenderer }))
);

/**
 * Memoized, optimized character renderer with error boundaries and performance enhancements
 */
const OptimizedCharacterRendererComponent = memo<OptimizedCharacterRendererProps>(({
  character,
  inventory = [],
  scale = 2,
  size = 'medium',
  priority = false,
  lazy = true,
  showLoading = true,
  fallbackAvatar = '/default-character-avatar.svg',
  className,
  onImageLoad,
  onImageError,
  region,
  majorVersion,
  enablePreload = true,
  prefetchVariants = false,
  enableErrorBoundary = true,
  suspenseFallback,
  preloadSiblings = [],
}) => {
  const { preloadImages } = useCharacterImagePreloader();
  const { activeTenant } = useTenant();

  // Memoize character data to prevent unnecessary re-renders
  const characterData = useMemo(() => ({
    character,
    inventory,
    scale,
  }), [character, inventory, scale]);

  // Preload sibling characters for better UX
  useMemo(() => {
    if (preloadSiblings.length > 0 && activeTenant) {
      // Convert Character to MapleStoryCharacterData first, then preload
      const siblingData = preloadSiblings.map(sibling => {
        const loadout = characterToLoadout(sibling.character, sibling.inventory || []);
        return {
          character: {
            id: sibling.character.id,
            name: sibling.character.attributes.name,
            level: sibling.character.attributes.level,
            jobId: sibling.character.attributes.jobId,
            hair: loadout.hair,
            face: loadout.face,
            skinColor: loadout.skin,
            gender: sibling.character.attributes.gender,
            equipment: loadout.equipment,
            tenant: activeTenant.id,
            region: activeTenant.attributes.region,
            majorVersion: activeTenant.attributes.majorVersion,
            minorVersion: activeTenant.attributes.minorVersion,
          },
          options: { resize: sibling.scale || scale },
        };
      });

      // Fire and forget - don't await
      preloadImages(siblingData);
    }
  }, [preloadSiblings, preloadImages, scale, activeTenant]);
  
  const rendererComponent = (
    <CharacterRenderer
      character={characterData.character}
      inventory={characterData.inventory}
      scale={characterData.scale}
      size={size}
      priority={priority}
      lazy={lazy}
      showLoading={showLoading}
      fallbackAvatar={fallbackAvatar}
      className={className || ''}
      {...(region && { region })}
      {...(majorVersion && { majorVersion })}
      {...(onImageLoad && { onImageLoad })}
      {...(onImageError && { onImageError })}
      enablePreload={enablePreload}
      prefetchVariants={prefetchVariants}
    />
  );

  // Wrap with error boundary if enabled
  const errorBoundaryComponent = enableErrorBoundary ? (
    <ErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }) => (
        <CharacterRenderErrorFallback
          error={error}
          resetErrorBoundary={resetErrorBoundary}
          character={character}
          className={className || ''}
        />
      )}
      onError={(error, errorInfo) => {
        if (import.meta.env.DEV) {
          console.error('Character renderer error:', error, errorInfo);
        }
        onImageError?.(error);
      }}
    >
      {rendererComponent}
    </ErrorBoundary>
  ) : rendererComponent;

  // Wrap with suspense if fallback provided
  if (suspenseFallback) {
    return (
      <Suspense fallback={suspenseFallback}>
        {errorBoundaryComponent}
      </Suspense>
    );
  }

  return errorBoundaryComponent;
});

OptimizedCharacterRendererComponent.displayName = 'OptimizedCharacterRenderer';

/**
 * Performance-optimized character renderer with intelligent caching and loading strategies
 */
export function OptimizedCharacterRenderer(props: OptimizedCharacterRendererProps) {
  // Use dynamic import for non-critical renders
  if (!props.priority && import.meta.env.PROD) {
    return (
      <Suspense fallback={<CharacterRendererSkeleton size="medium" />}>
        <DynamicCharacterRenderer {...props} />
      </Suspense>
    );
  }

  return <OptimizedCharacterRendererComponent {...props} />;
}

/**
 * Character renderer gallery component with batch optimization
 */
interface CharacterGalleryProps {
  characters: Array<{
    character: Character;
    inventory?: Asset[];
    scale?: number;
  }>;
  itemSize?: 'small' | 'medium' | 'large';
  priority?: boolean;
  className?: string;
  region?: string;
  majorVersion?: number;
  onCharacterClick?: (character: Character) => void;
}

export function CharacterGallery({
  characters,
  itemSize = 'medium',
  priority = false,
  className = '',
  region,
  majorVersion,
  onCharacterClick,
}: CharacterGalleryProps) {
  // Preload all character images in batch
  const { preloadImages } = useCharacterImagePreloader();
  const { activeTenant } = useTenant();

  useMemo(() => {
    if (priority && characters.length > 0 && activeTenant) {
      const characterData = characters.map(({ character, inventory = [], scale = 2 }) => {
        const loadout = characterToLoadout(character, inventory);
        return {
          character: {
            id: character.id,
            name: character.attributes.name,
            level: character.attributes.level,
            jobId: character.attributes.jobId,
            hair: loadout.hair,
            face: loadout.face,
            skinColor: loadout.skin,
            gender: character.attributes.gender,
            equipment: loadout.equipment,
            tenant: activeTenant.id,
            region: activeTenant.attributes.region,
            majorVersion: activeTenant.attributes.majorVersion,
            minorVersion: activeTenant.attributes.minorVersion,
          },
          options: { resize: scale },
        };
      });
      preloadImages(characterData);
    }
  }, [characters, priority, preloadImages, activeTenant]);

  return (
    <div className={`grid gap-4 ${className}`}>
      {characters.map(({ character, inventory, scale }, index) => (
        <div
          key={character.id}
          className={onCharacterClick ? 'cursor-pointer hover:opacity-80' : undefined}
          onClick={() => onCharacterClick?.(character)}
        >
          <OptimizedCharacterRenderer
            character={character}
            inventory={inventory || []}
            scale={scale || 2}
            size={itemSize}
            priority={priority && index < 3} // Prioritize first 3 images
            lazy={!priority}
            {...(region && { region })}
            {...(majorVersion && { majorVersion })}
            enablePreload={priority}
            prefetchVariants={false} // Disable for gallery to avoid too many requests
            preloadSiblings={
              priority 
                ? characters.slice(index + 1, index + 3) // Preload next 2 characters
                : []
            }
          />
          <div className="mt-2 text-center">
            <div className="font-medium">{character.attributes.name}</div>
            <div className="text-sm text-gray-600">Lv. {character.attributes.level}</div>
          </div>
        </div>
      ))}
    </div>
  );
}

export default OptimizedCharacterRenderer;