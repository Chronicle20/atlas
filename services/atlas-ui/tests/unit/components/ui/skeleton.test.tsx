import { render } from '@testing-library/react';
import { Skeleton } from '@/components/ui/skeleton';

describe('Skeleton', () => {
  it('renders with default props', () => {
    const { container } = render(<Skeleton className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveClass('bg-muted');
    expect(skeleton).toHaveClass('animate-pulse');
    expect(skeleton).toHaveClass('rounded-md');
  });

  it('renders with circular variant', () => {  
    const { container } = render(<Skeleton variant="circular" className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveClass('rounded-full');
    expect(skeleton).not.toHaveClass('rounded-md');
  });

  it('renders with rectangular variant', () => {
    const { container } = render(<Skeleton variant="rectangular" className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveClass('rounded-sm');
    expect(skeleton).not.toHaveClass('rounded-md');
  });

  it('renders with wave animation', () => {
    const { container } = render(<Skeleton animation="wave" className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveClass('animate-shimmer');
    expect(skeleton).not.toHaveClass('animate-pulse');
  });

  it('renders with no animation', () => {
    const { container } = render(<Skeleton animation="none" className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).not.toHaveClass('animate-pulse');
    expect(skeleton).not.toHaveClass('animate-shimmer');
  });

  it('applies custom className', () => {
    const { container } = render(<Skeleton className="custom-class h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveClass('custom-class');
  });

  it('forwards additional props', () => {
    const { container } = render(<Skeleton data-testid="test-skeleton" className="h-4 w-4" />);
    const skeleton = container.firstChild as HTMLElement;
    
    expect(skeleton).toHaveAttribute('data-testid', 'test-skeleton');
  });
});