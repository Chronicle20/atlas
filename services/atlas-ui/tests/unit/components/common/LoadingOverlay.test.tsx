import { render, screen } from '@testing-library/react';
import { LoadingOverlay } from '@/components/common/LoadingOverlay';

describe('LoadingOverlay', () => {
  const TestChild = () => <div data-testid="test-child">Test Content</div>;

  it('renders children when not loading', () => {
    render(
      <LoadingOverlay loading={false}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const child = screen.getByTestId('test-child');
    expect(child).toBeInTheDocument();
    expect(child).toBeVisible();
    
    // Should not show loading spinner
    const spinner = screen.queryByTestId('loading-spinner');
    expect(spinner).not.toBeInTheDocument();
  });

  it('shows loading overlay when loading is true', () => {
    render(
      <LoadingOverlay loading={true}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const child = screen.getByTestId('test-child');
    expect(child).toBeInTheDocument();
    
    // Should show loading spinner
    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toBeInTheDocument();
  });

  it('disables pointer events on children when loading', () => {
    render(
      <LoadingOverlay loading={true}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const child = screen.getByTestId('test-child');
    const childContainer = child.parentElement;
    expect(childContainer).toHaveClass('pointer-events-none');
  });

  it('enables pointer events on children when not loading', () => {
    render(
      <LoadingOverlay loading={false}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const child = screen.getByTestId('test-child');
    const childContainer = child.parentElement;
    expect(childContainer).not.toHaveClass('pointer-events-none');
  });

  it('applies relative positioning to container', () => {
    const { container } = render(
      <LoadingOverlay loading={false}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const outerContainer = container.firstChild as HTMLElement;
    expect(outerContainer).toHaveClass('relative');
  });

  it('applies custom className to container', () => {
    const customClass = 'my-custom-overlay-class';
    const { container } = render(
      <LoadingOverlay loading={false} className={customClass}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const outerContainer = container.firstChild as HTMLElement;
    expect(outerContainer).toHaveClass(customClass);
    expect(outerContainer).toHaveClass('relative'); // Should still have relative
  });

  it('renders loading spinner with default medium size', () => {
    render(
      <LoadingOverlay loading={true}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-6', 'w-6'); // Medium size
  });

  it('renders loading spinner with small size', () => {
    render(
      <LoadingOverlay loading={true} spinnerSize="sm">
        <TestChild />
      </LoadingOverlay>
    );
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-4', 'w-4'); // Small size
  });

  it('renders loading spinner with large size', () => {
    render(
      <LoadingOverlay loading={true} spinnerSize="lg">
        <TestChild />
      </LoadingOverlay>
    );
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-8', 'w-8'); // Large size
  });

  it('positions overlay absolutely with correct styling', () => {
    const { container } = render(
      <LoadingOverlay loading={true}>
        <TestChild />
      </LoadingOverlay>
    );
    
    const overlay = container.querySelector('.absolute.inset-0');
    expect(overlay).toBeInTheDocument();
    expect(overlay).toHaveClass(
      'absolute',
      'inset-0',
      'bg-background/50',
      'flex',
      'items-center',
      'justify-center',
      'z-10'
    );
  });

  it('toggles loading state correctly', () => {
    const { rerender } = render(
      <LoadingOverlay loading={false}>
        <TestChild />
      </LoadingOverlay>
    );
    
    // Initially not loading
    expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    
    // Switch to loading
    rerender(
      <LoadingOverlay loading={true}>
        <TestChild />
      </LoadingOverlay>
    );
    
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    
    // Switch back to not loading
    rerender(
      <LoadingOverlay loading={false}>
        <TestChild />
      </LoadingOverlay>
    );
    
    expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
  });

  it('maintains children accessibility when loading', () => {
    render(
      <LoadingOverlay loading={true}>
        <button data-testid="test-button">Click me</button>
      </LoadingOverlay>
    );
    
    const button = screen.getByTestId('test-button');
    expect(button).toBeInTheDocument();
    
    // Button should be in DOM but not interactable due to pointer-events-none
    const buttonContainer = button.parentElement;
    expect(buttonContainer).toHaveClass('pointer-events-none');
  });

  it('renders multiple children correctly', () => {
    render(
      <LoadingOverlay loading={false}>
        <div data-testid="child-1">Child 1</div>
        <div data-testid="child-2">Child 2</div>
        <div data-testid="child-3">Child 3</div>
      </LoadingOverlay>
    );
    
    expect(screen.getByTestId('child-1')).toBeInTheDocument();
    expect(screen.getByTestId('child-2')).toBeInTheDocument();
    expect(screen.getByTestId('child-3')).toBeInTheDocument();
  });

  it('renders complex children correctly', () => {
    render(
      <LoadingOverlay loading={false}>
        <div>
          <h1>Title</h1>
          <form>
            <input type="text" placeholder="Test input" />
            <button type="submit">Submit</button>
          </form>
        </div>
      </LoadingOverlay>
    );
    
    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Test input')).toBeInTheDocument();
    expect(screen.getByText('Submit')).toBeInTheDocument();
  });
});