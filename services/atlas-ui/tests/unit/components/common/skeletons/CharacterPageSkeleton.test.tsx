import { render, screen } from '@testing-library/react';
import { CharacterPageSkeleton } from '@/components/common/skeletons/CharacterPageSkeleton';

describe('CharacterPageSkeleton', () => {
  it('renders with default structure', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toBeInTheDocument();
    expect(pageContainer).toHaveClass('flex', 'flex-col', 'flex-1', 'space-y-6', 'p-10', 'pb-16');
  });

  it('renders page header with characters title skeleton', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    // Characters title is wider than Accounts and Tenants (w-32)
    const titleSkeleton = container.querySelector('.h-8.w-32');
    expect(titleSkeleton).toBeInTheDocument();
  });

  it('renders table skeleton with characters-specific configuration', () => {
    render(<CharacterPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    expect(tableSkeleton).toBeInTheDocument();
    
    // Should have 12 rows and 7 columns (characters have the most columns)
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(84); // 12 rows * 7 columns
    
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(7);
  });

  it('uses default pulse animation', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    // Check that skeleton elements have the pulse animation class
    const pulseElements = container.querySelectorAll('.animate-pulse');
    expect(pulseElements.length).toBeGreaterThan(0);
  });

  it('has proper layout structure', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should have header section
    const headerSection = pageContainer.querySelector('.items-center.justify-between.space-y-2');
    expect(headerSection).toBeInTheDocument();
    
    // Should have table section with proper margin
    const tableSection = pageContainer.querySelector('.mt-4');
    expect(tableSection).toBeInTheDocument();
  });

  it('matches expected page padding and spacing', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toHaveClass('p-10', 'pb-16', 'space-y-6');
  });

  it('renders with proper semantic structure', () => {
    const { container } = render(<CharacterPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should be a flex column layout
    expect(pageContainer).toHaveClass('flex-col');
    
    // Should fill available space
    expect(pageContainer).toHaveClass('flex-1');
  });

  it('has characters-specific table configuration', () => {
    render(<CharacterPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    
    // Verify it has the most columns (7)
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders).toHaveLength(7);
    
    // Verify it has the most rows (12)
    const tableCells = screen.getAllByRole('cell');
    expect(tableCells).toHaveLength(84); // 12 rows * 7 columns
  });

  it('shows header actions in table', () => {
    render(<CharacterPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    
    // Should have header actions (search, filters, etc.)
    const actionsContainer = tableSkeleton.querySelector('.flex.items-center.justify-between');
    expect(actionsContainer).toBeInTheDocument();
  });

  it('shows table header', () => {
    render(<CharacterPageSkeleton />);
    
    const tableHeaders = screen.getAllByRole('columnheader');
    expect(tableHeaders.length).toBeGreaterThan(0);
  });
});