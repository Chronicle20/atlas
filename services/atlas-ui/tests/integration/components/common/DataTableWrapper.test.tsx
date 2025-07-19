import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ColumnDef } from '@tanstack/react-table';
import { DataTableWrapper } from '@/components/common/DataTableWrapper';
import { Database, Plus } from 'lucide-react';

// Mock data for testing
interface TestData {
  id: string;
  name: string;
  email: string;
}

const mockData: TestData[] = [
  { id: '1', name: 'John Doe', email: 'john@example.com' },
  { id: '2', name: 'Jane Smith', email: 'jane@example.com' },
];

const mockColumns: ColumnDef<TestData>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
  },
  {
    accessorKey: 'email',
    header: 'Email',
  },
];

describe('DataTableWrapper Integration Tests', () => {
  describe('Loading State Integration', () => {
    it('should show PageLoader when loading is true', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
        />
      );

      // PageLoader should be rendered
      expect(screen.getByTestId('page-loader')).toBeInTheDocument();
      
      // Should not show data table, error, or empty state
      expect(screen.queryByRole('table')).not.toBeInTheDocument();
      expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
      expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument();
    });

    it('should hide PageLoader when loading becomes false', async () => {
      const { rerender } = render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
        />
      );

      expect(screen.getByTestId('page-loader')).toBeInTheDocument();

      // Change loading to false
      rerender(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          loading={false}
        />
      );

      await waitFor(() => {
        expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
        expect(screen.getByRole('table')).toBeInTheDocument();
      });
    });
  });

  describe('Error State Integration', () => {
    it('should show ErrorDisplay when error is provided', () => {
      const errorMessage = 'Failed to load data';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error={errorMessage}
        />
      );

      // ErrorDisplay should be rendered
      expect(screen.getByTestId('error-display')).toBeInTheDocument();
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
      
      // Should not show loader, data table, or empty state
      expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
      expect(screen.queryByRole('table')).not.toBeInTheDocument();
      expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument();
    });

    it('should show retry button in ErrorDisplay when onRefresh is provided', () => {
      const mockRefresh = jest.fn();
      const errorMessage = 'Network error';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error={errorMessage}
          onRefresh={mockRefresh}
        />
      );

      const retryButton = screen.getByText('Try Again');
      expect(retryButton).toBeInTheDocument();
      
      fireEvent.click(retryButton);
      expect(mockRefresh).toHaveBeenCalledTimes(1);
    });

    it('should handle Error object in ErrorDisplay', () => {
      const error = new Error('Database connection failed');
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error={error}
        />
      );

      expect(screen.getByTestId('error-display')).toBeInTheDocument();
      expect(screen.getByText('Database connection failed')).toBeInTheDocument();
    });

    it('should handle custom error object in ErrorDisplay', () => {
      const error = { message: 'Custom error message' };
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error={error}
        />
      );

      expect(screen.getByTestId('error-display')).toBeInTheDocument();
      expect(screen.getByText('Custom error message')).toBeInTheDocument();
    });
  });

  describe('Empty State Integration', () => {
    it('should show EmptyState when no data is provided', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
        />
      );

      // EmptyState should be rendered with default props
      expect(screen.getByTestId('empty-state')).toBeInTheDocument();
      expect(screen.getByText('No data available')).toBeInTheDocument();
      expect(screen.getByText('There are no items to display at this time.')).toBeInTheDocument();
      
      // Should show default Database icon
      const icon = screen.getByTestId('empty-state').querySelector('svg');
      expect(icon).toBeInTheDocument();
    });

    it('should show custom EmptyState when emptyState prop is provided', () => {
      const mockAction = jest.fn();
      const customEmptyState = {
        title: 'No users found',
        description: 'Start by adding your first user.',
        icon: <Plus data-testid="plus-icon" className="h-12 w-12" />,
        action: {
          label: 'Add User',
          onClick: mockAction,
        },
      };

      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          emptyState={customEmptyState}
        />
      );

      expect(screen.getByTestId('empty-state')).toBeInTheDocument();
      expect(screen.getByText('No users found')).toBeInTheDocument();
      expect(screen.getByText('Start by adding your first user.')).toBeInTheDocument();
      expect(screen.getByTestId('plus-icon')).toBeInTheDocument();
      
      const actionButton = screen.getByText('Add User');
      expect(actionButton).toBeInTheDocument();
      
      fireEvent.click(actionButton);
      expect(mockAction).toHaveBeenCalledTimes(1);
    });

    it('should merge custom emptyState with defaults', () => {
      const partialEmptyState = {
        title: 'Custom title only',
      };

      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          emptyState={partialEmptyState}
        />
      );

      expect(screen.getByText('Custom title only')).toBeInTheDocument();
      // Should still show default description
      expect(screen.getByText('There are no items to display at this time.')).toBeInTheDocument();
      // Should still show default Database icon
      const icon = screen.getByTestId('empty-state').querySelector('svg');
      expect(icon).toBeInTheDocument();
    });
  });

  describe('Success State Integration', () => {
    it('should render DataTable when data is provided', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
        />
      );

      // DataTable should be rendered
      expect(screen.getByRole('table')).toBeInTheDocument();
      
      // Should show data
      expect(screen.getByText('John Doe')).toBeInTheDocument();
      expect(screen.getByText('john@example.com')).toBeInTheDocument();
      expect(screen.getByText('Jane Smith')).toBeInTheDocument();
      expect(screen.getByText('jane@example.com')).toBeInTheDocument();
      
      // Should not show loading, error, or empty states
      expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
      expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
      expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument();
    });

    it('should pass through initialVisibilityState to DataTable', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          initialVisibilityState={['name']}
        />
      );

      expect(screen.getByRole('table')).toBeInTheDocument();
      // This would require inspecting DataTable implementation details
      // For now, just verify the table renders
    });

    it('should pass through onRefresh to DataTable when provided', () => {
      const mockRefresh = jest.fn();
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          onRefresh={mockRefresh}
        />
      );

      expect(screen.getByRole('table')).toBeInTheDocument();
      // The refresh functionality would be tested in DataTable unit tests
    });

    it('should pass through headerActions to DataTable when provided', () => {
      const mockAction = jest.fn();
      const headerActions = [
        {
          icon: <Plus data-testid="header-plus-icon" />,
          label: 'Add Item',
          onClick: mockAction,
        },
      ];
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          headerActions={headerActions}
        />
      );

      expect(screen.getByRole('table')).toBeInTheDocument();
      // The header actions would be tested in DataTable unit tests
    });
  });

  describe('State Priority Integration', () => {
    it('should prioritize loading state over error state', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
          error="Some error"
        />
      );

      expect(screen.getByTestId('page-loader')).toBeInTheDocument();
      expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
    });

    it('should prioritize error state over empty state', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error="Some error"
        />
      );

      expect(screen.getByTestId('error-display')).toBeInTheDocument();
      expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument();
    });

    it('should prioritize success state over empty state when data is provided', () => {
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
        />
      );

      expect(screen.getByRole('table')).toBeInTheDocument();
      expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument();
    });
  });

  describe('State Transitions Integration', () => {
    it('should transition from loading to success state', async () => {
      const { rerender } = render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
        />
      );

      expect(screen.getByTestId('page-loader')).toBeInTheDocument();

      rerender(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          loading={false}
        />
      );

      await waitFor(() => {
        expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
        expect(screen.getByRole('table')).toBeInTheDocument();
      });
    });

    it('should transition from loading to error state', async () => {
      const { rerender } = render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
        />
      );

      expect(screen.getByTestId('page-loader')).toBeInTheDocument();

      rerender(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={false}
          error="Load failed"
        />
      );

      await waitFor(() => {
        expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });
    });

    it('should transition from loading to empty state', async () => {
      const { rerender } = render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
        />
      );

      expect(screen.getByTestId('page-loader')).toBeInTheDocument();

      rerender(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={false}
        />
      );

      await waitFor(() => {
        expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
        expect(screen.getByTestId('empty-state')).toBeInTheDocument();
      });
    });

    it('should transition from error to success state after retry', async () => {
      let hasError = true;
      const mockRefresh = jest.fn(() => {
        hasError = false;
      });

      const { rerender } = render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error={hasError ? "Load failed" : null}
          onRefresh={mockRefresh}
        />
      );

      expect(screen.getByTestId('error-display')).toBeInTheDocument();

      const retryButton = screen.getByText('Try Again');
      fireEvent.click(retryButton);

      rerender(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          error={hasError ? "Load failed" : null}
        />
      );

      await waitFor(() => {
        expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
        expect(screen.getByRole('table')).toBeInTheDocument();
      });
    });
  });

  describe('Custom Styling Integration', () => {
    it('should apply custom className to wrapper', () => {
      const customClass = 'my-custom-wrapper-class';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={mockData}
          className={customClass}
        />
      );

      const wrapper = screen.getByRole('table').closest('div');
      expect(wrapper).toHaveClass(customClass);
      expect(wrapper).toHaveClass('w-full'); // Default class should also be present
    });

    it('should apply className to loading state wrapper', () => {
      const customClass = 'loading-wrapper-class';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          loading={true}
          className={customClass}
        />
      );

      const wrapper = screen.getByTestId('page-loader').closest('div');
      expect(wrapper).toHaveClass(customClass);
      expect(wrapper).toHaveClass('w-full');
    });

    it('should apply className to error state wrapper', () => {
      const customClass = 'error-wrapper-class';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          error="Some error"
          className={customClass}
        />
      );

      const wrapper = screen.getByTestId('error-display').closest('div');
      expect(wrapper).toHaveClass(customClass);
      expect(wrapper).toHaveClass('w-full');
    });

    it('should apply className to empty state wrapper', () => {
      const customClass = 'empty-wrapper-class';
      
      render(
        <DataTableWrapper
          columns={mockColumns}
          data={[]}
          className={customClass}
        />
      );

      const wrapper = screen.getByTestId('empty-state').closest('div');
      expect(wrapper).toHaveClass(customClass);
      expect(wrapper).toHaveClass('w-full');
    });
  });
});