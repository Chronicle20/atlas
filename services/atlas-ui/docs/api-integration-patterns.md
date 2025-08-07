# API Integration Patterns

This document outlines the API integration patterns used in Atlas UI, with a focus on the MapleStory.io API integration for inventory item icons and data fetching.

## Overview

Atlas UI integrates with external APIs to enhance user experience with rich visual content. The primary external integration is with the MapleStory.io API for fetching item icons, names, and metadata.

## MapleStory.io API Integration

### API Endpoints

The MapleStory.io API provides several endpoints for fetching game-related data:

```
Base URL: https://maplestory.io/api/{region}/{version}/
```

#### Item Data Endpoints
- **Item Icon**: `GET /item/{itemId}/icon` - Returns item icon image
- **Item Data**: `GET /item/{itemId}` - Returns item metadata including name

#### Parameters
- **region**: Game region (e.g., "GMS", "JMS", "KMS", "MSEA")
- **version**: Major version number (e.g., "83", "251")
- **itemId**: Unique item template ID (e.g., "1002000")

### Service Implementation

The MapleStory API integration is implemented in `/services/api/maplestory.service.ts`:

```typescript
export class MapleStoryService extends BaseService {
  /**
   * Fetches item icon URL from MapleStory.io API
   */
  async getItemIcon(
    itemId: string, 
    region: string = 'GMS', 
    version: string = '83'
  ): Promise<string> {
    const response = await this.get<{ iconUrl: string }>(
      `/item/${itemId}/icon`,
      { region, version }
    );
    return response.iconUrl;
  }

  /**
   * Fetches item name and metadata from MapleStory.io API
   */
  async getItemName(
    itemId: string, 
    region: string = 'GMS', 
    version: string = '83'
  ): Promise<string> {
    const response = await this.get<{ name: string }>(
      `/item/${itemId}`,
      { region, version }
    );
    return response.name;
  }
}
```

### React Query Integration

Item data is fetched using React Query for optimal caching and error handling:

```typescript
// lib/hooks/useItemData.ts
export function useItemData(
  itemId: string,
  options: {
    enabled?: boolean;
    region?: string;
    version?: string;
    staleTime?: number;
    gcTime?: number;
    onError?: (error: Error) => void;
  } = {}
) {
  return useQuery({
    queryKey: ['item-data', itemId, options.region, options.version],
    queryFn: () => fetchItemData(itemId, options.region, options.version),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30 * 60 * 1000, // 30 minutes
    gcTime: options.gcTime ?? 24 * 60 * 60 * 1000,   // 24 hours
    onError: options.onError,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
}
```

## Integration Patterns

### 1. Lazy Loading Pattern

Components only fetch API data when needed, using intersection observers for viewport-based loading:

```typescript
const { shouldLoad, ref } = useLazyLoad<HTMLDivElement>({
  threshold: 0.1,
  rootMargin: '100px',
});

const { itemData, isLoading, hasError } = useItemData(itemId, {
  enabled: shouldLoad || shouldPreload,
  region,
  version: majorVersion?.toString(),
});
```

**Benefits:**
- Reduces unnecessary API calls
- Improves initial page load performance
- Provides smooth scrolling experience

### 2. Cache Warming Pattern

For frequently accessed data, use cache warming to preload multiple items:

```typescript
export function useItemDataCache() {
  const queryClient = useQueryClient();

  const warmCache = async (
    itemIds: string[],
    region?: string,
    version?: string
  ): Promise<PromiseSettledResult<ItemData>[]> => {
    const promises = itemIds.map(itemId =>
      queryClient.ensureQueryData({
        queryKey: ['item-data', itemId, region, version],
        queryFn: () => fetchItemData(itemId, region, version),
        staleTime: 30 * 60 * 1000,
      })
    );

    return Promise.allSettled(promises);
  };

  return { warmCache };
}
```

**Usage in Components:**
```typescript
const { warmCache } = useItemDataCache();

useEffect(() => {
  if (itemIds.length > 0) {
    warmCache(itemIds, region, majorVersion?.toString())
      .then((results) => {
        const successful = results.filter(r => r.status === 'fulfilled').length;
        console.log(`Preloaded metadata for ${successful} items`);
      });
  }
}, [itemIds, warmCache, region, majorVersion]);
```

### 3. Image Preloading Pattern

For better perceived performance, preload images when component mounts:

```typescript
useEffect(() => {
  if (typeof window !== 'undefined' && itemData?.iconUrl && shouldPreload) {
    const img = new window.Image();
    img.src = itemData.iconUrl;
    
    img.onload = () => {
      console.log(`Preloaded image for item ${itemId}`);
    };
    
    img.onerror = () => {
      console.warn(`Failed to preload image for item ${itemId}`);
    };
    
    return () => {
      img.onload = null;
      img.onerror = null;
    };
  }
}, [itemData?.iconUrl, shouldPreload, itemId]);
```

### 4. Error Handling and Fallbacks

Implement graceful degradation when API calls fail:

```typescript
const displayError = hasError || errorMessage || (!isLoading && !hasIcon && !hasName);

return (
  <div>
    {displayError ? (
      // Fallback to text display
      <div className="flex items-center">
        <Package className="h-6 w-6 text-muted-foreground" />
        <span>{itemId}</span>
      </div>
    ) : (
      // Rich content with icon and name
      <div className="flex items-center">
        <Image src={itemData.iconUrl} alt={itemData.name} width={32} height={32} />
        <span>{itemData.name}</span>
      </div>
    )}
  </div>
);
```

## Performance Considerations

### Caching Strategy

- **Short-term cache**: 30 minutes stale time for frequently changing data
- **Long-term cache**: 24 hours garbage collection time
- **Persistent cache**: React Query persister for cross-session caching

### Request Optimization

- **Deduplication**: React Query automatically deduplicates identical requests
- **Batching**: Use `Promise.allSettled()` for batch operations
- **Retry Logic**: Exponential backoff with maximum retry limits

### Memory Management

- **Cache Size Limits**: Configure maximum cache entries
- **Automatic Cleanup**: Unused cache entries are garbage collected
- **Selective Preloading**: Only preload first 12 items to balance performance

## Error Monitoring

All API errors are logged for monitoring and debugging:

```typescript
const onError = (error: Error) => {
  const context = {
    userId: 'character_inventory_user',
    tenantId: 'atlas_ui',
    url: window.location.href,
  };
  
  errorLogger.logError(error, undefined, context).catch((loggingError) => {
    console.warn('Failed to log API error:', loggingError);
  });
};
```

## Best Practices

### 1. **Always Provide Fallbacks**
Never assume API calls will succeed. Always provide fallback content or error states.

### 2. **Use Proper Loading States** 
Show appropriate loading indicators while data is being fetched.

### 3. **Implement Progressive Enhancement**
Start with basic functionality and enhance with API-driven features.

### 4. **Cache Appropriately**
Balance between data freshness and performance. Use longer cache times for static data.

### 5. **Handle Network Failures**
Implement retry logic and offline-friendly behavior where possible.

### 6. **Log for Monitoring**
Log API errors and performance metrics for production monitoring.

### 7. **Test Error Scenarios**
Include tests for API failure scenarios and fallback behavior.

## Component Integration Examples

### Basic InventoryCard Usage

```tsx
import { InventoryCard } from '@/components/features/characters/InventoryCard';

function InventoryDisplay({ assets }: { assets: Asset[] }) {
  return (
    <div className="grid grid-cols-4 gap-3">
      {assets.map(asset => (
        <InventoryCard
          key={asset.id}
          asset={asset}
          region="GMS"
          majorVersion={83}
          onDelete={(assetId) => handleDelete(assetId)}
        />
      ))}
    </div>
  );
}
```

### Advanced InventoryGrid Usage

```tsx
import { InventoryGrid } from '@/components/features/characters/InventoryGrid';

function CompartmentView({ compartment, assets }: ComponentViewProps) {
  return (
    <InventoryGrid
      compartment={compartment}
      assets={assets}
      onDeleteAsset={handleDeleteAsset}
      deletingAssetId={currentlyDeleting}
      region="GMS"
      majorVersion={83}
      isLoading={isLoadingAssets}
    />
  );
}
```

## Future Enhancements

### Planned Improvements

1. **WebSocket Support**: Real-time updates for inventory changes
2. **Background Sync**: Sync data when connectivity is restored
3. **Advanced Caching**: LRU cache with size limits and priority-based eviction
4. **Image Optimization**: WebP format support with fallbacks
5. **CDN Integration**: Cache images on CDN for better global performance

### Monitoring and Analytics

- Track API response times and error rates
- Monitor cache hit ratios and performance impact
- Collect user interaction metrics for optimization

This documentation provides a comprehensive overview of the API integration patterns used for inventory item icons and can serve as a reference for implementing similar patterns in other parts of the application.