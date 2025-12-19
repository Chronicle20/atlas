import { render, screen } from '@testing-library/react';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

describe('LoadingSpinner', () => {
  it('renders with default props', () => {
    render(<LoadingSpinner />);
    
    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toBeInTheDocument();
    expect(spinner).toHaveClass('flex', 'items-center', 'justify-center', 'p-4');
  });

  it('renders with small size', () => {
    render(<LoadingSpinner size="sm" />);
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-4', 'w-4');
  });

  it('renders with medium size (default)', () => {
    render(<LoadingSpinner size="md" />);
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-6', 'w-6');
  });

  it('renders with large size', () => {
    render(<LoadingSpinner size="lg" />);
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('h-8', 'w-8');
  });

  it('applies custom className', () => {
    const customClass = 'my-custom-class';
    render(<LoadingSpinner className={customClass} />);
    
    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toHaveClass(customClass);
  });

  it('has spinning animation', () => {
    render(<LoadingSpinner />);
    
    const spinner = screen.getByTestId('loading-spinner');
    const icon = spinner.querySelector('svg');
    expect(icon).toHaveClass('animate-spin');
  });
});