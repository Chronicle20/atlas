import { createQueryClient } from '../query-client';

describe('Query Client Configuration', () => {
  let queryClient: ReturnType<typeof createQueryClient>;

  beforeEach(() => {
    queryClient = createQueryClient();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('should create a query client with default configuration', () => {
    expect(queryClient).toBeDefined();
    expect(queryClient.getDefaultOptions()).toBeDefined();
  });

  it('should have correct query default options', () => {
    const defaultOptions = queryClient.getDefaultOptions();
    
    expect(defaultOptions.queries?.staleTime).toBe(5 * 60 * 1000); // 5 minutes
    expect(defaultOptions.queries?.gcTime).toBe(10 * 60 * 1000); // 10 minutes
    expect(defaultOptions.queries?.retry).toBe(3);
    expect(defaultOptions.queries?.refetchOnWindowFocus).toBe(false);
    expect(defaultOptions.queries?.refetchOnReconnect).toBe(true);
    expect(defaultOptions.queries?.refetchOnMount).toBe(true);
  });

  it('should have correct mutation default options', () => {
    const defaultOptions = queryClient.getDefaultOptions();
    
    expect(defaultOptions.mutations?.retry).toBe(1);
    expect(defaultOptions.mutations?.retryDelay).toBe(1000);
  });

  it('should have exponential backoff retry delay for queries', () => {
    const defaultOptions = queryClient.getDefaultOptions();
    const retryDelay = defaultOptions.queries?.retryDelay as (attemptIndex: number) => number;
    
    if (typeof retryDelay === 'function') {
      expect(retryDelay(0)).toBe(1000); // First retry: 1 second
      expect(retryDelay(1)).toBe(2000); // Second retry: 2 seconds
      expect(retryDelay(2)).toBe(4000); // Third retry: 4 seconds
      expect(retryDelay(10)).toBe(30000); // Max retry: 30 seconds
    } else {
      fail('retryDelay should be a function');
    }
  });

  it('should support basic query operations', () => {
    const queryKey = ['test'];
    const queryData = { id: 1, name: 'test' };

    // Set query data
    queryClient.setQueryData(queryKey, queryData);

    // Get query data
    const retrievedData = queryClient.getQueryData(queryKey);
    expect(retrievedData).toEqual(queryData);

    // Remove query
    queryClient.removeQueries({ queryKey });
    expect(queryClient.getQueryData(queryKey)).toBeUndefined();
  });
});