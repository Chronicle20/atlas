/**
 * Base service tests
 * Validates the functionality of the BaseService class
 */

import { BaseService, type ServiceOptions, type BatchResult } from '../base.service';
import { api } from '@/lib/api/client';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  api: {
    getList: jest.fn(),
    getOne: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
    upload: jest.fn(),
    download: jest.fn(),
    clearCacheByPattern: jest.fn(),
    getCacheStats: jest.fn(),
  },
}));

interface TestModel {
  id: string;
  name: string;
  value: number;
}

interface TestCreateData {
  name: string;
  value: number;
}

// Concrete implementation for testing
class TestService extends BaseService {
  protected basePath = '/test';

  // Override validation for testing
  protected validate<T>(data: T): Array<{ field: string; message: string; value?: unknown }> {
    const errors: Array<{ field: string; message: string; value?: unknown }> = [];
    
    if (typeof data === 'object' && data !== null) {
      const obj = data as any;
      if (obj.name && obj.name.length < 2) {
        errors.push({ field: 'name', message: 'Name must be at least 2 characters', value: obj.name });
      }
      if (obj.value && obj.value < 0) {
        errors.push({ field: 'value', message: 'Value must be positive', value: obj.value });
      }
    }
    
    return errors;
  }

  // Override transformation for testing
  protected transformRequest<T>(data: T): T {
    if (typeof data === 'object' && data !== null) {
      const obj = data as any;
      return { ...obj, transformedRequest: true } as T;
    }
    return data;
  }

  protected transformResponse<T>(data: T): T {
    if (typeof data === 'object' && data !== null) {
      const obj = data as any;
      return { ...obj, transformedResponse: true } as T;
    }
    return data;
  }

  // Expose protected methods for testing
  public testGetAll<T>(options?: ServiceOptions) {
    return this.getAll<T>(options);
  }

  public testGetById<T>(id: string, options?: ServiceOptions) {
    return this.getById<T>(id, options);
  }

  public testCreate<T, D>(data: D, options?: ServiceOptions) {
    return this.create<T, D>(data, options);
  }

  public testUpdate<T, D>(id: string, data: D, options?: ServiceOptions) {
    return this.update<T, D>(id, data, options);
  }

  public testDelete(id: string, options?: ServiceOptions) {
    return this.delete(id, options);
  }

  public testCreateBatch<T, D>(items: D[], options?: ServiceOptions) {
    return this.createBatch<T, D>(items, options);
  }

  public testExists(id: string, options?: ServiceOptions) {
    return this.exists(id, options);
  }
}

describe('BaseService', () => {
  let testService: TestService;
  const mockApi = api as jest.Mocked<typeof api>;

  beforeEach(() => {
    testService = new TestService();
    jest.clearAllMocks();
  });

  describe('getAll', () => {
    it('should fetch and transform all resources', async () => {
      const mockData = [{ id: '1', name: 'Test', value: 100 }];
      mockApi.getList.mockResolvedValue(mockData);

      const result = await testService.testGetAll<TestModel>();

      expect(mockApi.getList).toHaveBeenCalledWith('/test', expect.any(Object));
      expect(result).toEqual([{ id: '1', name: 'Test', value: 100, transformedResponse: true }]);
    });

    it('should handle query options', async () => {
      const mockData = [{ id: '1', name: 'Test', value: 100 }];
      mockApi.getList.mockResolvedValue(mockData);

      await testService.testGetAll<TestModel>({
        search: 'test',
        sortBy: 'name',
        sortOrder: 'asc',
        limit: 10,
        offset: 0,
      });

      expect(mockApi.getList).toHaveBeenCalledWith(
        '/test?search=test&sortBy=name&sortOrder=asc&limit=10&offset=0',
        expect.any(Object)
      );
    });
  });

  describe('getById', () => {
    it('should fetch and transform a single resource', async () => {
      const mockData = { id: '1', name: 'Test', value: 100 };
      mockApi.getOne.mockResolvedValue(mockData);

      const result = await testService.testGetById<TestModel>('1');

      expect(mockApi.getOne).toHaveBeenCalledWith('/test/1', expect.any(Object));
      expect(result).toEqual({ id: '1', name: 'Test', value: 100, transformedResponse: true });
    });
  });

  describe('exists', () => {
    it('should return true when resource exists', async () => {
      const mockData = { id: '1', name: 'Test', value: 100 };
      mockApi.getOne.mockResolvedValue(mockData);

      const result = await testService.testExists('1');

      expect(result).toBe(true);
    });

    it('should return false when resource does not exist (404)', async () => {
      const error = new Error('Not Found');
      (error as any).status = 404;
      mockApi.getOne.mockRejectedValue(error);

      const result = await testService.testExists('1');

      expect(result).toBe(false);
    });

    it('should throw error for non-404 errors', async () => {
      const error = new Error('Server Error');
      (error as any).status = 500;
      mockApi.getOne.mockRejectedValue(error);

      await expect(testService.testExists('1')).rejects.toThrow('Server Error');
    });
  });

  describe('create', () => {
    it('should validate, transform, and create a resource', async () => {
      const inputData: TestCreateData = { name: 'Test', value: 100 };
      const mockResponse = { id: '1', name: 'Test', value: 100 };
      mockApi.post.mockResolvedValue(mockResponse);

      const result = await testService.testCreate<TestModel, TestCreateData>(inputData);

      expect(mockApi.post).toHaveBeenCalledWith(
        '/test',
        { name: 'Test', value: 100, transformedRequest: true },
        expect.any(Object)
      );
      expect(result).toEqual({ id: '1', name: 'Test', value: 100, transformedResponse: true });
    });

    it('should reject invalid data during validation', async () => {
      const inputData: TestCreateData = { name: 'X', value: -1 }; // Invalid: name too short, value negative

      await expect(testService.testCreate<TestModel, TestCreateData>(inputData)).rejects.toThrow(
        'Validation failed: Name must be at least 2 characters, Value must be positive'
      );

      expect(mockApi.post).not.toHaveBeenCalled();
    });

    it('should skip validation when disabled', async () => {
      const inputData: TestCreateData = { name: 'X', value: -1 };
      const mockResponse = { id: '1', name: 'X', value: -1 };
      mockApi.post.mockResolvedValue(mockResponse);

      const result = await testService.testCreate<TestModel, TestCreateData>(inputData, {
        validate: false,
      });

      expect(mockApi.post).toHaveBeenCalled();
      expect(result).toEqual({ id: '1', name: 'X', value: -1, transformedResponse: true });
    });
  });

  describe('update', () => {
    it('should validate, transform, and update a resource', async () => {
      const updateData = { name: 'Updated', value: 200 };
      const mockResponse = { id: '1', name: 'Updated', value: 200 };
      mockApi.put.mockResolvedValue(mockResponse);

      const result = await testService.testUpdate<TestModel, typeof updateData>('1', updateData);

      expect(mockApi.put).toHaveBeenCalledWith(
        '/test/1',
        { name: 'Updated', value: 200, transformedRequest: true },
        expect.any(Object)
      );
      expect(result).toEqual({ id: '1', name: 'Updated', value: 200, transformedResponse: true });
    });
  });

  describe('delete', () => {
    it('should delete a resource', async () => {
      mockApi.delete.mockResolvedValue(undefined);

      await testService.testDelete('1');

      expect(mockApi.delete).toHaveBeenCalledWith('/test/1', expect.any(Object));
    });
  });

  describe('createBatch', () => {
    it('should create multiple resources successfully', async () => {
      const items = [
        { name: 'Test1', value: 100 },
        { name: 'Test2', value: 200 },
      ];
      
      mockApi.post
        .mockResolvedValueOnce({ id: '1', name: 'Test1', value: 100 })
        .mockResolvedValueOnce({ id: '2', name: 'Test2', value: 200 });

      const result: BatchResult<TestModel> = await testService.testCreateBatch<TestModel, TestCreateData>(items);

      expect(result.successCount).toBe(2);
      expect(result.failureCount).toBe(0);
      expect(result.successes).toHaveLength(2);
      expect(result.failures).toHaveLength(0);
    });

    it('should handle partial failures in batch creation', async () => {
      const items = [
        { name: 'Test1', value: 100 },
        { name: 'X', value: -1 }, // This should fail validation
        { name: 'Test3', value: 300 },
      ];
      
      mockApi.post
        .mockResolvedValueOnce({ id: '1', name: 'Test1', value: 100 })
        .mockResolvedValueOnce({ id: '3', name: 'Test3', value: 300 });

      const result: BatchResult<TestModel> = await testService.testCreateBatch<TestModel, TestCreateData>(items);

      expect(result.successCount).toBe(2);
      expect(result.failureCount).toBe(1);
      expect(result.successes).toHaveLength(2);
      expect(result.failures).toHaveLength(1);
      expect(result.failures[0].error.message).toContain('Validation failed');
    });
  });

  describe('caching configuration', () => {
    it('should configure caching options', async () => {
      const mockData = [{ id: '1', name: 'Test', value: 100 }];
      mockApi.getList.mockResolvedValue(mockData);

      await testService.testGetAll<TestModel>({
        useCache: true,
        cacheConfig: {
          ttl: 10 * 60 * 1000, // 10 minutes
          staleWhileRevalidate: true,
        },
      });

      expect(mockApi.getList).toHaveBeenCalledWith('/test', expect.objectContaining({
        cacheConfig: expect.objectContaining({
          ttl: 10 * 60 * 1000,
          staleWhileRevalidate: true,
        }),
      }));
    });

    it('should disable caching when requested', async () => {
      const mockData = [{ id: '1', name: 'Test', value: 100 }];
      mockApi.getList.mockResolvedValue(mockData);

      await testService.testGetAll<TestModel>({ useCache: false });

      expect(mockApi.getList).toHaveBeenCalledWith('/test', expect.objectContaining({
        cacheConfig: false,
      }));
    });
  });
});