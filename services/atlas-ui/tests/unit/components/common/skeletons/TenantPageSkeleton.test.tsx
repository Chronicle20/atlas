import { render, screen } from '@testing-library/react';
import { TenantPageSkeleton } from '@/components/common/skeletons/TenantPageSkeleton';

describe('TenantPageSkeleton', () => {
  it('renders with default props', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toBeInTheDocument();
    expect(pageContainer).toHaveClass('flex', 'flex-col', 'flex-1', 'space-y-6', 'p-10', 'pb-16');
  });

  it('renders page header with title skeleton', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    const titleSkeleton = container.querySelector('.h-8.w-24');
    expect(titleSkeleton).toBeInTheDocument();
  });

  it('renders table skeleton with correct props', () => {
    render(<TenantPageSkeleton />);
    
    const tableSkeleton = screen.getByTestId('table-skeleton');
    expect(tableSkeleton).toBeInTheDocument();
  });

  it('passes animation prop to skeleton elements', () => {
    const { container } = render(<TenantPageSkeleton animation="wave" />);
    
    // Check that skeleton elements have the shimmer animation class
    const skeletonElements = container.querySelectorAll('.animate-shimmer');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses default pulse animation when no animation prop provided', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    // Check that skeleton elements have the pulse animation class
    const skeletonElements = container.querySelectorAll('.animate-pulse');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses no animation when animation is set to none', () => {
    const { container } = render(<TenantPageSkeleton animation="none" />);
    
    // Should not have animation classes
    const pulseElements = container.querySelectorAll('.animate-pulse');
    const shimmerElements = container.querySelectorAll('.animate-shimmer');
    expect(pulseElements.length).toBe(0);
    expect(shimmerElements.length).toBe(0);
  });

  it('has proper layout structure', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should have header section
    const headerSection = pageContainer.querySelector('.items-center.justify-between.space-y-2');
    expect(headerSection).toBeInTheDocument();
    
    // Should have table section with proper margin
    const tableSection = pageContainer.querySelector('.mt-4');
    expect(tableSection).toBeInTheDocument();
  });

  it('matches expected page padding and spacing', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    expect(pageContainer).toHaveClass('p-10', 'pb-16', 'space-y-6');
  });

  it('renders with proper semantic structure', () => {
    const { container } = render(<TenantPageSkeleton />);
    
    const pageContainer = container.firstChild as HTMLElement;
    
    // Should be a flex column layout
    expect(pageContainer).toHaveClass('flex-col');
    
    // Should fill available space
    expect(pageContainer).toHaveClass('flex-1');
  });
});