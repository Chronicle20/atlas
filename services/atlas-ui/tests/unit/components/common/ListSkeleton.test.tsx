import { render, screen } from '@testing-library/react';
import { ListSkeleton } from '@/components/common/ListSkeleton';

describe('ListSkeleton', () => {
  it('renders with default props', () => {
    render(<ListSkeleton />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    expect(skeleton).toBeInTheDocument();
    
    // Should render 5 items by default
    const items = skeleton.querySelectorAll('[data-testid*="skeleton-item"], div > div');
    expect(items.length).toBeGreaterThanOrEqual(5);
  });

  it('renders correct number of items', () => {
    render(<ListSkeleton items={3} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const items = skeleton.children;
    expect(items).toHaveLength(3);
  });

  it('shows avatar when showAvatar is true', () => {
    render(<ListSkeleton items={1} showAvatar={true} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const avatarElement = skeleton.querySelector('.rounded-full');
    expect(avatarElement).toBeInTheDocument();
  });

  it('hides avatar when showAvatar is false', () => {
    render(<ListSkeleton items={1} showAvatar={false} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const avatarElement = skeleton.querySelector('.rounded-full');
    expect(avatarElement).not.toBeInTheDocument();
  });

  it('shows actions when showActions is true', () => {
    render(<ListSkeleton items={1} showActions={true} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const item = skeleton.firstChild as HTMLElement;
    const actionElements = item.querySelectorAll('div:last-child > div');
    expect(actionElements.length).toBeGreaterThan(0);
  });

  it('applies custom className', () => {
    render(<ListSkeleton className="custom-class" />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    expect(skeleton).toHaveClass('custom-class');
  });

  it('renders compact variant correctly', () => {
    render(<ListSkeleton variant="compact" items={1} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    expect(skeleton).toHaveClass('space-y-1');
    
    const item = skeleton.firstChild as HTMLElement;
    expect(item).toHaveClass('py-2', 'px-3');
  });

  it('renders card variant correctly', () => {
    render(<ListSkeleton variant="card" items={1} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    expect(skeleton).toHaveClass('space-y-3');
    
    const item = skeleton.firstChild as HTMLElement;
    expect(item).toHaveClass('border', 'rounded-lg', 'shadow-sm');
  });

  it('renders default variant correctly', () => {
    render(<ListSkeleton variant="default" items={1} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    expect(skeleton).toHaveClass('border', 'rounded-lg', 'overflow-hidden');
    
    const item = skeleton.firstChild as HTMLElement;
    expect(item).toHaveClass('border-b');
  });

  it('hides subtext when showSubtext is false', () => {
    render(<ListSkeleton items={1} showSubtext={false} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const item = skeleton.firstChild as HTMLElement;
    const contentArea = item.querySelector('.flex-1');
    const skeletonElements = contentArea?.children;
    
    // Should only have one skeleton element (primary text) when subtext is hidden
    expect(skeletonElements).toHaveLength(1);
  });

  it('shows subtext when showSubtext is true', () => {
    render(<ListSkeleton items={1} showSubtext={true} />);
    
    const skeleton = screen.getByTestId('list-skeleton');
    const item = skeleton.firstChild as HTMLElement;
    const contentArea = item.querySelector('.flex-1');
    const skeletonElements = contentArea?.children;
    
    // Should have two skeleton elements (primary text + subtext) when subtext is shown
    expect(skeletonElements).toHaveLength(2);
  });
});