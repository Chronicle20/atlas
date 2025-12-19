import { render } from '@testing-library/react';
import { CardSkeleton } from '@/components/common/CardSkeleton';

describe('CardSkeleton', () => {
  it('renders with default props', () => {
    const { container } = render(<CardSkeleton />);
    
    // Should have card structure
    expect(container.firstChild).toBeInTheDocument();
    
    // Should render skeleton elements
    const skeletons = container.querySelectorAll('.bg-muted');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders default variant correctly', () => {
    const { container } = render(<CardSkeleton variant="default" />);
    
    // Should have skeleton elements
    const skeletons = container.querySelectorAll('.bg-muted');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders grid variant correctly', () => {
    const { container } = render(<CardSkeleton variant="grid" />);
    
    // Should render skeleton elements
    const skeletons = container.querySelectorAll('.bg-muted');
    expect(skeletons.length).toBeGreaterThan(0);
    
    // Grid variant should have compact structure
    const gridCard = container.querySelector('[data-slot="card"]');
    expect(gridCard).toHaveClass('overflow-hidden', 'relative', 'py-0');
  });

  it('renders detailed variant correctly', () => {
    const { container } = render(<CardSkeleton variant="detailed" />);
    
    // Should have skeleton elements
    const skeletons = container.querySelectorAll('.bg-muted');
    expect(skeletons.length).toBeGreaterThan(0);
    
    // Should show footer by default for detailed variant
    const footer = container.querySelector('.border-t');
    expect(footer).toBeInTheDocument();
  });

  it('shows header when showHeader is true', () => {
    const { container } = render(<CardSkeleton showHeader={true} />);
    
    const header = container.querySelector('[data-slot="card-header"]');
    expect(header).toBeInTheDocument();
  });

  it('hides header when showHeader is false', () => {
    const { container } = render(<CardSkeleton showHeader={false} />);
    
    const header = container.querySelector('[data-slot="card-header"]');
    expect(header).not.toBeInTheDocument();
  });

  it('shows footer when showFooter is true', () => {
    const { container } = render(<CardSkeleton showFooter={true} />);
    
    const footer = container.querySelector('.border-t');
    expect(footer).toBeInTheDocument();
  });

  it('hides footer when showFooter is false', () => {
    const { container } = render(<CardSkeleton showFooter={false} />);
    
    const footer = container.querySelector('.border-t');
    expect(footer).not.toBeInTheDocument();
  });

  it('renders custom number of lines', () => {
    const customLines = 5;
    const { container } = render(<CardSkeleton lines={customLines} />);
    
    // Should have skeleton elements
    const skeletons = container.querySelectorAll('.bg-muted');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('applies custom className', () => {
    const customClass = 'my-custom-card-class';
    const { container } = render(<CardSkeleton className={customClass} />);
    
    const card = container.querySelector('[data-slot="card"]');
    expect(card).toHaveClass(customClass);
  });

  it('applies custom width style', () => {
    const customWidth = '300px';
    const { container } = render(<CardSkeleton width={customWidth} />);
    
    const card = container.querySelector('[data-slot="card"]');
    expect(card).toHaveStyle(`width: ${customWidth}`);
  });

  it('renders with auto width by default', () => {
    const { container } = render(<CardSkeleton />);
    
    const card = container.querySelector('[data-slot="card"]');
    expect(card).toHaveStyle('width: auto');
  });

  it('detailed variant shows footer by default', () => {
    const { container } = render(<CardSkeleton variant="detailed" />);
    
    const footer = container.querySelector('.border-t');
    expect(footer).toBeInTheDocument();
  });

  it('default and grid variants hide footer by default', () => {
    const { container: defaultContainer } = render(<CardSkeleton variant="default" />);
    const defaultFooter = defaultContainer.querySelector('.border-t');
    expect(defaultFooter).not.toBeInTheDocument();

    const { container: gridContainer } = render(<CardSkeleton variant="grid" />);
    const gridFooter = gridContainer.querySelector('.border-t');
    expect(gridFooter).not.toBeInTheDocument();
  });

  it('grid variant has proper compact styling', () => {
    const { container } = render(<CardSkeleton variant="grid" />);
    
    const card = container.querySelector('[data-slot="card"]');
    expect(card).toHaveClass('overflow-hidden', 'relative', 'py-0');
  });

  it('shows header action when showHeaderAction is true', () => {
    const { container } = render(<CardSkeleton showHeaderAction={true} />);
    
    // Should have justify-between layout when action is shown
    const headerContainer = container.querySelector('.justify-between');
    expect(headerContainer).toBeInTheDocument();
  });

  it('shows header action for grid variant in absolute position', () => {
    const { container } = render(<CardSkeleton variant="grid" showHeaderAction={true} />);
    
    const actionContainer = container.querySelector('.absolute.top-0.right-0');
    expect(actionContainer).toBeInTheDocument();
  });
});