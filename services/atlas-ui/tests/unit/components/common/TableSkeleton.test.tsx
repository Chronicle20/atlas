import { render, screen } from '@testing-library/react';
import { TableSkeleton } from '@/components/common/TableSkeleton';

describe('TableSkeleton', () => {
  it('renders table skeleton with default props', () => {
    render(<TableSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    expect(tableSkeleton).toBeInTheDocument();
  });

  it('renders header actions by default', () => {
    render(<TableSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    const actionsContainer = tableSkeleton.querySelector('.flex.items-center.justify-between');
    expect(actionsContainer).toBeInTheDocument();
  });

  it('hides header actions when showActions is false', () => {
    render(<TableSkeleton showActions={false} />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    const actionsContainer = tableSkeleton.querySelector('.flex.items-center.justify-between');
    expect(actionsContainer).not.toBeInTheDocument();
  });

  it('renders table header by default', () => {
    render(<TableSkeleton />);
    
    // Look for table header structure
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(4); // default columns
  });

  it('hides table header when showHeader is false', () => {
    render(<TableSkeleton showHeader={false} />);
    
    // Should not have table headers
    const tableHeaders = screen.queryAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(0);
  });

  it('renders default number of rows and columns', () => {
    render(<TableSkeleton />);
    
    // Check for table cells (rows * columns)
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(20); // 5 rows * 4 columns
  });

  it('renders custom number of rows', () => {
    const customRows = 3;
    render(<TableSkeleton rows={customRows} />);
    
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(12); // 3 rows * 4 columns
  });

  it('renders custom number of columns', () => {
    const customColumns = 6;
    render(<TableSkeleton columns={customColumns} />);
    
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(6);
    
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(30); // 5 rows * 6 columns
  });

  it('renders custom rows and columns', () => {
    const customRows = 3;
    const customColumns = 2;
    render(<TableSkeleton rows={customRows} columns={customColumns} />);
    
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(2);
    
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(6); // 3 rows * 2 columns
  });

  it('applies custom className', () => {
    const customClass = 'my-custom-table-skeleton';
    render(<TableSkeleton className={customClass} />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    expect(tableSkeleton).toHaveClass(customClass);
  });

  it('renders skeleton elements in header actions', () => {
    render(<TableSkeleton showActions={true} />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    const skeletons = tableSkeleton.querySelectorAll('[data-testid="table-skeleton"] > div:first-child .h-10, [data-testid="table-skeleton"] > div:first-child .h-8');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('passes animation prop to skeleton elements', () => {
    const { container } = render(<TableSkeleton animation="wave" />);
    
    // Check that skeleton elements have the shimmer animation class
    const skeletonElements = container.querySelectorAll('.animate-shimmer');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses default pulse animation when no animation prop provided', () => {
    const { container } = render(<TableSkeleton />);
    
    // Check that skeleton elements have the pulse animation class
    const skeletonElements = container.querySelectorAll('.animate-pulse');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });
});