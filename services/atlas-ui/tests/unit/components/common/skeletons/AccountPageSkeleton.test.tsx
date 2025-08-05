import { render, screen } from '@testing-library/react';
import { AccountPageSkeleton } from '@/components/common/skeletons/AccountPageSkeleton';

describe('AccountPageSkeleton', () => {
  it('renders with default props', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toBeInTheDocument();
    expect(pageContainer).toHaveClass('flex', 'flex-col', 'flex-1', 'space-y-6', 'p-10', 'pb-16');
  });

  it('renders page header with accounts title skeleton', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    // Accounts title is wider than Tenants (w-28 vs w-24)
    const titleSkeleton = container.querySelector('.h-8.w-28');
    expect(titleSkeleton).toBeInTheDocument();
  });

  it('renders table skeleton with accounts-specific configuration', () => {
    render(<AccountPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    expect(tableSkeleton).toBeInTheDocument();
    
    // Should have 10 rows and 6 columns (accounts have more columns than tenants)
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(60); // 10 rows * 6 columns
    
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(6);
  });

  it('passes animation prop to skeleton elements', () => {
    const { container } = render(<AccountPageSkeleton animation="wave" />);
    
    // Check that skeleton elements have the shimmer animation class
    const skeletonElements = container.querySelectorAll('.animate-shimmer');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses default pulse animation when no animation prop provided', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    // Check that skeleton elements have the pulse animation class
    const skeletonElements = container.querySelectorAll('.animate-pulse');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses no animation when animation is set to none', () => {
    const { container } = render(<AccountPageSkeleton animation="none" />);
    
    // Should not have animation classes
    const pulseElements = container.querySelectorAll('.animate-pulse');
    const shimmerElements = container.querySelectorAll('.animate-shimmer');
    expect(pulseElements.length).toBe(0);
    expect(shimmerElements.length).toBe(0);
  });

  it('has proper layout structure', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should have header section
    const headerSection = pageContainer.querySelector('.items-center.justify-between.space-y-2');
    expect(headerSection).toBeInTheDocument();
    
    // Should have table section with proper margin
    const tableSection = pageContainer.querySelector('.mt-4');
    expect(tableSection).toBeInTheDocument();
  });

  it('matches expected page padding and spacing', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toHaveClass('p-10', 'pb-16', 'space-y-6');
  });

  it('renders with proper semantic structure', () => {
    const { container } = render(<AccountPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should be a flex column layout
    expect(pageContainer).toHaveClass('flex-col');
    
    // Should fill available space
    expect(pageContainer).toHaveClass('flex-1');
  });

  it('has accounts-specific table configuration', () => {
    render(<AccountPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    
    // Verify it has more columns than TenantPageSkeleton (6 vs 5)
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(6);
    
    // Verify it has more rows than TenantPageSkeleton (10 vs 8)
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(60); // 10 rows * 6 columns
  });
});