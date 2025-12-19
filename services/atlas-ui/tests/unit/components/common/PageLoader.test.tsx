import { render, screen } from '@testing-library/react';
import { PageLoader } from '@/components/common/PageLoader';

describe('PageLoader', () => {
  it('renders page loader', () => {
    render(<PageLoader />);
    
    const pageLoader = screen.getByTestId('page-loader');
    expect(pageLoader).toBeInTheDocument();
  });

  it('contains a loading spinner', () => {
    render(<PageLoader />);
    
    const loadingSpinner = screen.getByTestId('loading-spinner');
    expect(loadingSpinner).toBeInTheDocument();
  });

  it('loading spinner has large size', () => {
    render(<PageLoader />);
    
    const loadingSpinner = screen.getByTestId('loading-spinner');
    const icon = loadingSpinner.querySelector('svg');
    expect(icon).toHaveClass('h-8', 'w-8');
  });

  it('applies default className when none provided', () => {
    render(<PageLoader />);
    
    const pageLoader = screen.getByTestId('page-loader');
    expect(pageLoader).toHaveClass('flex', 'h-[50vh]', 'items-center', 'justify-center');
  });

  it('applies custom className when provided', () => {
    const customClass = 'my-custom-loader-class';
    render(<PageLoader className={customClass} />);
    
    const pageLoader = screen.getByTestId('page-loader');
    expect(pageLoader).toHaveClass(customClass);
    expect(pageLoader).not.toHaveClass('h-[50vh]');
  });
});