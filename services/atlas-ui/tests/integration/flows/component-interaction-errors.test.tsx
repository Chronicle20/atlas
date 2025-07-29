/**
 * Integration tests for component interaction error scenarios
 * Tests error handling when components interact with each other and services
 */

import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { AllProviders } from '../utils/test-providers';
import { createApiErrorFromResponse } from '@/types/api/errors';

// Mock various services and modules
jest.mock('@/lib/api/client');
jest.mock('@/context/tenant-context');

// Mock Sonner toast
jest.mock('sonner', () => ({
  toast: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
    loading: jest.fn(),
    promise: jest.fn(),
    dismiss: jest.fn(),
  },
  Toaster: ({ children }: { children?: React.ReactNode }) => <div data-testid="toaster">{children}</div>,
}));

// Mock Next.js router
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
  }),
  usePathname: () => '/test-path',
  useSearchParams: () => new URLSearchParams(),
}));

// Test component with data table and detail view interaction
const TestDataTableWithDetail = ({ 
  fetchItems, 
  fetchItemDetail,
  deleteItem 
}: {
  fetchItems: () => Promise<any[]>;
  fetchItemDetail: (id: string) => Promise<any>;
  deleteItem: (id: string) => Promise<void>;
}) => {
  const [items, setItems] = React.useState<any[]>([]);
  const [selectedItem, setSelectedItem] = React.useState<any | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [deleteLoading, setDeleteLoading] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [detailError, setDetailError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const loadItems = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchItems();
        setItems(data);
      } catch (err: any) {
        setError(err.message || 'Failed to load items');
      } finally {
        setLoading(false);
      }
    };

    loadItems();
  }, [fetchItems]);

  const handleItemClick = async (item: any) => {
    try {
      setDetailLoading(true);
      setDetailError(null);
      const details = await fetchItemDetail(item.id);
      setSelectedItem({ ...item, ...details });
    } catch (err: any) {
      setDetailError(err.message || 'Failed to load item details');
    } finally {
      setDetailLoading(false);
    }
  };

  const handleDelete = async (itemId: string) => {
    try {
      setDeleteLoading(itemId);
      await deleteItem(itemId);
      setItems(prev => prev.filter(item => item.id !== itemId));
      if (selectedItem?.id === itemId) {
        setSelectedItem(null);
      }
    } catch (err: any) {
      // Error handling could show a toast here
      console.error('Delete failed:', err.message);
    } finally {
      setDeleteLoading(null);
    }
  };

  if (loading) {
    return <div data-testid="table-loading">Loading items...</div>;
  }

  if (error) {
    return (
      <div data-testid="table-error">
        <p>Error: {error}</p>
        <button onClick={() => window.location.reload()} data-testid="retry-button">
          Retry
        </button>
      </div>
    );
  }

  return (
    <div data-testid="data-table-with-detail">
      <div data-testid="items-list">
        <h2>Items</h2>
        {items.length === 0 ? (
          <div data-testid="empty-state">No items found</div>
        ) : (
          <ul>
            {items.map(item => (
              <li key={item.id} data-testid={`item-${item.id}`}>
                <button
                  onClick={() => handleItemClick(item)}
                  data-testid={`item-button-${item.id}`}
                >
                  {item.name}
                </button>
                <button
                  onClick={() => handleDelete(item.id)}
                  disabled={deleteLoading === item.id}
                  data-testid={`delete-button-${item.id}`}
                >
                  {deleteLoading === item.id ? 'Deleting...' : 'Delete'}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div data-testid="detail-panel">
        <h2>Details</h2>
        {detailLoading && <div data-testid="detail-loading">Loading details...</div>}
        {detailError && (
          <div data-testid="detail-error">
            <p>Error: {detailError}</p>
            <button 
              onClick={() => selectedItem && handleItemClick(selectedItem)}
              data-testid="retry-detail-button"
            >
              Retry
            </button>
          </div>
        )}
        {selectedItem && !detailLoading && !detailError && (
          <div data-testid="detail-content">
            <p>Name: {selectedItem.name}</p>
            <p>Details: {selectedItem.details || 'No additional details'}</p>
          </div>
        )}
        {!selectedItem && !detailLoading && !detailError && (
          <div data-testid="no-selection">Select an item to view details</div>
        )}
      </div>
    </div>
  );
};

// Test component with form and related data updates
const TestFormWithRelatedData = ({
  fetchRelatedData,
  submitForm,
  validateField,
}: {
  fetchRelatedData: () => Promise<any[]>;
  submitForm: (data: any) => Promise<any>;
  validateField: (field: string, value: string) => Promise<boolean>;
}) => {
  const [formData, setFormData] = React.useState({ name: '', category: '' });
  const [relatedData, setRelatedData] = React.useState<any[]>([]);
  const [fieldValidation, setFieldValidation] = React.useState<Record<string, string>>({});
  const [relatedDataError, setRelatedDataError] = React.useState<string | null>(null);
  const [submitError, setSubmitError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  React.useEffect(() => {
    const loadRelatedData = async () => {
      try {
        setRelatedDataError(null);
        const data = await fetchRelatedData();
        setRelatedData(data);
      } catch (err: any) {
        setRelatedDataError(err.message || 'Failed to load related data');
      }
    };

    loadRelatedData();
  }, [fetchRelatedData]);

  const handleFieldChange = async (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    
    // Clear previous validation error
    setFieldValidation(prev => {
      const newValidation = { ...prev };
      delete newValidation[field];
      return newValidation;
    });

    // Validate field asynchronously
    if (value.trim()) {
      try {
        const isValid = await validateField(field, value);
        if (!isValid) {
          setFieldValidation(prev => ({
            ...prev,
            [field]: `Invalid ${field}`,
          }));
        }
      } catch (err: any) {
        setFieldValidation(prev => ({
          ...prev,
          [field]: `Validation failed: ${err.message}`,
        }));
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setIsSubmitting(true);
      setSubmitError(null);
      await submitForm(formData);
      // Success handling would go here
    } catch (err: any) {
      setSubmitError(err.message || 'Failed to submit form');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div data-testid="form-with-related-data">
      <div data-testid="related-data-section">
        <h3>Related Data</h3>
        {relatedDataError ? (
          <div data-testid="related-data-error">
            Error: {relatedDataError}
          </div>
        ) : (
          <div data-testid="related-data-list">
            {relatedData.map(item => (
              <div key={item.id} data-testid={`related-item-${item.id}`}>
                {item.name}
              </div>
            ))}
          </div>
        )}
      </div>

      <form onSubmit={handleSubmit} data-testid="main-form">
        <div>
          <input
            type="text"
            placeholder="Name"
            value={formData.name}
            onChange={(e) => handleFieldChange('name', e.target.value)}
            data-testid="name-input"
          />
          {fieldValidation.name && (
            <div data-testid="name-validation-error">
              {fieldValidation.name}
            </div>
          )}
        </div>

        <div>
          <input
            type="text"
            placeholder="Category"
            value={formData.category}
            onChange={(e) => handleFieldChange('category', e.target.value)}
            data-testid="category-input"
          />
          {fieldValidation.category && (
            <div data-testid="category-validation-error">
              {fieldValidation.category}
            </div>
          )}
        </div>

        {submitError && (
          <div data-testid="submit-error">
            {submitError}
          </div>
        )}

        <button 
          type="submit" 
          disabled={isSubmitting || Object.keys(fieldValidation).length > 0}
          data-testid="submit-button"
        >
          {isSubmitting ? 'Submitting...' : 'Submit'}
        </button>
      </form>
    </div>
  );
};

// Mock test provider component
const TestProviders = ({ children }: { children: React.ReactNode }) => {
  return <AllProviders>{children}</AllProviders>;
};

describe('Component Interaction Error Scenarios', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Data Table and Detail View Interactions', () => {
    it('should handle list loading error while maintaining interaction capability', async () => {
      const mockFetchItems = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(500, 'Failed to fetch items')
      );
      const mockFetchDetail = jest.fn();
      const mockDeleteItem = jest.fn();

      render(
        <TestProviders >
          <TestDataTableWithDetail
            fetchItems={mockFetchItems}
            fetchItemDetail={mockFetchDetail}
            deleteItem={mockDeleteItem}
          />
        </TestProviders>
      );

      // Should show loading initially
      expect(screen.getByTestId('table-loading')).toBeInTheDocument();

      // Should show error after API call fails
      await waitFor(() => {
        expect(screen.getByTestId('table-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Failed to fetch items/)).toBeInTheDocument();
      expect(screen.getByTestId('retry-button')).toBeInTheDocument();
    });

    it('should handle detail loading error while preserving list functionality', async () => {
      const user = userEvent.setup();
      
      const mockItems = [
        { id: '1', name: 'Item 1' },
        { id: '2', name: 'Item 2' },
      ];

      const mockFetchItems = jest.fn().mockResolvedValue(mockItems);
      const mockFetchDetail = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(404, 'Item details not found')
      );
      const mockDeleteItem = jest.fn();

      render(
        <TestProviders >
          <TestDataTableWithDetail
            fetchItems={mockFetchItems}
            fetchItemDetail={mockFetchDetail}
            deleteItem={mockDeleteItem}
          />
        </TestProviders>
      );

      // Wait for items to load
      await waitFor(() => {
        expect(screen.getByTestId('items-list')).toBeInTheDocument();
      });

      // Click on first item
      const itemButton = screen.getByTestId('item-button-1');
      await user.click(itemButton);

      // Should show detail loading initially
      expect(screen.getByTestId('detail-loading')).toBeInTheDocument();

      // Should show detail error
      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Item details not found/)).toBeInTheDocument();
      
      // List should still be functional
      expect(screen.getByTestId('items-list')).toBeInTheDocument();
      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Item 2')).toBeInTheDocument();
    });

    it('should handle delete operation error while preserving data integrity', async () => {
      const user = userEvent.setup();
      
      const mockItems = [
        { id: '1', name: 'Item 1' },
        { id: '2', name: 'Item 2' },
      ];

      const mockFetchItems = jest.fn().mockResolvedValue(mockItems);
      const mockFetchDetail = jest.fn().mockResolvedValue({ details: 'Some details' });
      const mockDeleteItem = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(403, 'Not authorized to delete')
      );

      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

      render(
        <TestProviders >
          <TestDataTableWithDetail
            fetchItems={mockFetchItems}
            fetchItemDetail={mockFetchDetail}
            deleteItem={mockDeleteItem}
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('items-list')).toBeInTheDocument();
      });

      // Try to delete first item
      const deleteButton = screen.getByTestId('delete-button-1');
      await user.click(deleteButton);

      // Should show deleting state
      expect(screen.getByText('Deleting...')).toBeInTheDocument();

      await waitFor(() => {
        expect(mockDeleteItem).toHaveBeenCalledWith('1');
      });

      // Button should be re-enabled after error
      await waitFor(() => {
        expect(screen.getByTestId('delete-button-1')).not.toBeDisabled();
      });

      // Item should still be in the list (not deleted due to error)
      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Item 2')).toBeInTheDocument();

      // Error should be logged
      expect(consoleSpy).toHaveBeenCalledWith('Delete failed:', 'Not authorized to delete');
      
      consoleSpy.mockRestore();
    });

    it('should handle multiple simultaneous operations with mixed results', async () => {
      const user = userEvent.setup();
      
      const mockItems = [
        { id: '1', name: 'Item 1' },
        { id: '2', name: 'Item 2' },
        { id: '3', name: 'Item 3' },
      ];

      const mockFetchItems = jest.fn().mockResolvedValue(mockItems);
      const mockFetchDetail = jest.fn()
        .mockResolvedValueOnce({ details: 'Details for Item 1' })
        .mockRejectedValueOnce(createApiErrorFromResponse(500, 'Server error for Item 2'));
      const mockDeleteItem = jest.fn()
        .mockResolvedValueOnce(undefined) // Success for Item 3
        .mockRejectedValueOnce(createApiErrorFromResponse(403, 'Forbidden')); // Error for any other

      render(
        <TestProviders >
          <TestDataTableWithDetail
            fetchItems={mockFetchItems}
            fetchItemDetail={mockFetchDetail}
            deleteItem={mockDeleteItem}
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('items-list')).toBeInTheDocument();
      });

      // Click on Item 1 for details (should succeed)
      await user.click(screen.getByTestId('item-button-1'));

      await waitFor(() => {
        expect(screen.getByTestId('detail-content')).toBeInTheDocument();
      });

      expect(screen.getByText('Details: Details for Item 1')).toBeInTheDocument();

      // Try to delete Item 3 (should succeed)
      await user.click(screen.getByTestId('delete-button-3'));

      await waitFor(() => {
        expect(screen.queryByTestId('item-3')).not.toBeInTheDocument();
      });

      // Click on Item 2 for details (should fail)
      await user.click(screen.getByTestId('item-button-2'));

      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Server error for Item 2/)).toBeInTheDocument();
    });
  });

  describe('Form and Related Data Interactions', () => {
    it('should handle related data loading error while preserving form functionality', async () => {
      const user = userEvent.setup();
      
      const mockFetchRelatedData = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(500, 'Failed to load categories')
      );
      const mockSubmitForm = jest.fn().mockResolvedValue({ success: true });
      const mockValidateField = jest.fn().mockResolvedValue(true);

      render(
        <TestProviders >
          <TestFormWithRelatedData
            fetchRelatedData={mockFetchRelatedData}
            submitForm={mockSubmitForm}
            validateField={mockValidateField}
          />
        </TestProviders>
      );

      // Should show related data error
      await waitFor(() => {
        expect(screen.getByTestId('related-data-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Failed to load categories/)).toBeInTheDocument();

      // Form should still be functional
      const nameInput = screen.getByTestId('name-input');
      await user.type(nameInput, 'Test Name');

      expect(nameInput).toHaveValue('Test Name');
      expect(mockValidateField).toHaveBeenCalledWith('name', 'Test Name');
    });

    it('should handle field validation errors during form interaction', async () => {
      const user = userEvent.setup();
      
      const mockFetchRelatedData = jest.fn().mockResolvedValue([
        { id: '1', name: 'Category 1' },
      ]);
      const mockSubmitForm = jest.fn();
      const mockValidateField = jest.fn()
        .mockResolvedValueOnce(false) // First validation fails
        .mockResolvedValueOnce(true);  // Second validation succeeds

      render(
        <TestProviders >
          <TestFormWithRelatedData
            fetchRelatedData={mockFetchRelatedData}
            submitForm={mockSubmitForm}
            validateField={mockValidateField}
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('related-data-list')).toBeInTheDocument();
      });

      // Type invalid name
      const nameInput = screen.getByTestId('name-input');
      await user.type(nameInput, 'invalid');

      // Should show validation error
      await waitFor(() => {
        expect(screen.getByTestId('name-validation-error')).toBeInTheDocument();
      });

      expect(screen.getByText('Invalid name')).toBeInTheDocument();

      // Submit button should be disabled
      expect(screen.getByTestId('submit-button')).toBeDisabled();

      // Clear and type valid name
      await user.clear(nameInput);
      await user.type(nameInput, 'valid-name');

      // Validation error should be cleared
      await waitFor(() => {
        expect(screen.queryByTestId('name-validation-error')).not.toBeInTheDocument();
      });

      // Submit button should be enabled
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });

    it('should handle validation service error gracefully', async () => {
      const user = userEvent.setup();
      
      const mockFetchRelatedData = jest.fn().mockResolvedValue([]);
      const mockSubmitForm = jest.fn();
      const mockValidateField = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(503, 'Validation service unavailable')
      );

      render(
        <TestProviders >
          <TestFormWithRelatedData
            fetchRelatedData={mockFetchRelatedData}
            submitForm={mockSubmitForm}
            validateField={mockValidateField}
          />
        </TestProviders>
      );

      const nameInput = screen.getByTestId('name-input');
      await user.type(nameInput, 'test-name');

      // Should show validation service error
      await waitFor(() => {
        expect(screen.getByTestId('name-validation-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Validation failed: Validation service unavailable/)).toBeInTheDocument();
    });

    it('should handle form submission error while preserving form state', async () => {
      const user = userEvent.setup();
      
      const mockFetchRelatedData = jest.fn().mockResolvedValue([
        { id: '1', name: 'Category 1' },
      ]);
      const mockSubmitForm = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(400, 'Invalid form data')
      );
      const mockValidateField = jest.fn().mockResolvedValue(true);

      render(
        <TestProviders >
          <TestFormWithRelatedData
            fetchRelatedData={mockFetchRelatedData}
            submitForm={mockSubmitForm}
            validateField={mockValidateField}
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('related-data-list')).toBeInTheDocument();
      });

      // Fill form
      await user.type(screen.getByTestId('name-input'), 'Test Name');
      await user.type(screen.getByTestId('category-input'), 'Test Category');

      // Submit form
      await user.click(screen.getByTestId('submit-button'));

      // Should show submit error
      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByText('Invalid form data')).toBeInTheDocument();

      // Form data should be preserved
      expect(screen.getByTestId('name-input')).toHaveValue('Test Name');
      expect(screen.getByTestId('category-input')).toHaveValue('Test Category');

      // Form should be re-enabled
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
      expect(screen.getByTestId('submit-button')).toHaveTextContent('Submit');
    });
  });

  describe('Cross-Component Error Propagation', () => {
    it('should isolate errors between independent components', async () => {
      const user = userEvent.setup();
      
      // Mock functions for first component (success)
      const mockFetchItems1 = jest.fn().mockResolvedValue([
        { id: '1', name: 'Item 1' },
      ]);
      const mockFetchDetail1 = jest.fn().mockResolvedValue({ details: 'Details 1' });
      const mockDeleteItem1 = jest.fn().mockResolvedValue(undefined);

      // Mock functions for second component (failure)
      const mockFetchItems2 = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(500, 'Service unavailable')
      );
      const mockFetchDetail2 = jest.fn();
      const mockDeleteItem2 = jest.fn();

      render(
        <TestProviders >
          <div>
            <div data-testid="component-1">
              <TestDataTableWithDetail
                fetchItems={mockFetchItems1}
                fetchItemDetail={mockFetchDetail1}
                deleteItem={mockDeleteItem1}
              />
            </div>
            <div data-testid="component-2">
              <TestDataTableWithDetail
                fetchItems={mockFetchItems2}
                fetchItemDetail={mockFetchDetail2}
                deleteItem={mockDeleteItem2}
              />
            </div>
          </div>
        </TestProviders>
      );

      await waitFor(() => {
        // Component 1 should show success
        const component1 = screen.getByTestId('component-1');
        expect(within(component1).getByTestId('items-list')).toBeInTheDocument();
        expect(within(component1).getByText('Item 1')).toBeInTheDocument();

        // Component 2 should show error
        const component2 = screen.getByTestId('component-2');
        expect(within(component2).getByTestId('table-error')).toBeInTheDocument();
        expect(within(component2).getByText(/Service unavailable/)).toBeInTheDocument();
      });

      // Component 1 should still be interactive
      const component1 = screen.getByTestId('component-1');
      const itemButton = within(component1).getByTestId('item-button-1');
      await user.click(itemButton);

      await waitFor(() => {
        expect(within(component1).getByTestId('detail-content')).toBeInTheDocument();
      });
    });

    it('should handle cascading errors in dependent components', async () => {
      // This test would simulate a scenario where one component's error
      // affects another component that depends on its data
      const user = userEvent.setup();
      
      const mockFetchRelatedData = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(401, 'Authentication required')
      );
      const mockSubmitForm = jest.fn();
      const mockValidateField = jest.fn().mockResolvedValue(true);

      render(
        <TestProviders >
          <TestFormWithRelatedData
            fetchRelatedData={mockFetchRelatedData}
            submitForm={mockSubmitForm}
            validateField={mockValidateField}
          />
        </TestProviders>
      );

      // Related data should show auth error
      await waitFor(() => {
        expect(screen.getByTestId('related-data-error')).toBeInTheDocument();
      });

      expect(screen.getByText(/Authentication required/)).toBeInTheDocument();

      // Form should still be functional but without related data context
      await user.type(screen.getByTestId('name-input'), 'Test Name');
      
      // Validation should still work
      expect(mockValidateField).toHaveBeenCalledWith('name', 'Test Name');
    });
  });
});