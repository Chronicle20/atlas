import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorDisplay } from '@/components/common/ErrorDisplay';

describe('ErrorDisplay', () => {
  it('renders with string error', () => {
    const errorMessage = 'Something went wrong';
    render(<ErrorDisplay error={errorMessage} />);
    
    const errorDisplay = screen.getByTestId('error-display');
    expect(errorDisplay).toBeInTheDocument();
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('renders with Error object', () => {
    const error = new Error('Network error');
    render(<ErrorDisplay error={error} />);
    
    expect(screen.getByText('Network error')).toBeInTheDocument();
  });

  it('renders with custom error object', () => {
    const error = { message: 'Custom error message' };
    render(<ErrorDisplay error={error} />);
    
    expect(screen.getByText('Custom error message')).toBeInTheDocument();
  });

  it('renders with custom title', () => {
    const customTitle = 'Custom Error Title';
    render(<ErrorDisplay error="Error message" title={customTitle} />);
    
    expect(screen.getByText(customTitle)).toBeInTheDocument();
  });

  it('shows icon by default', () => {
    render(<ErrorDisplay error="Error message" />);
    
    const errorDisplay = screen.getByTestId('error-display');
    const icon = errorDisplay.querySelector('svg');
    expect(icon).toBeInTheDocument();
  });

  it('hides icon when showIcon is false', () => {
    render(<ErrorDisplay error="Error message" showIcon={false} />);
    
    const errorDisplay = screen.getByTestId('error-display');
    const icon = errorDisplay.querySelector('svg[data-testid]');
    expect(icon).not.toBeInTheDocument();
  });

  it('renders retry button when retry function is provided', () => {
    const mockRetry = jest.fn();
    render(<ErrorDisplay error="Error message" retry={mockRetry} />);
    
    const retryButton = screen.getByText('Try Again');
    expect(retryButton).toBeInTheDocument();
  });

  it('calls retry function when retry button is clicked', () => {
    const mockRetry = jest.fn();
    render(<ErrorDisplay error="Error message" retry={mockRetry} />);
    
    const retryButton = screen.getByText('Try Again');
    fireEvent.click(retryButton);
    
    expect(mockRetry).toHaveBeenCalledTimes(1);
  });

  it('does not render retry button when retry function is not provided', () => {
    render(<ErrorDisplay error="Error message" />);
    
    const retryButton = screen.queryByText('Try Again');
    expect(retryButton).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    const customClass = 'my-custom-error-class';
    render(<ErrorDisplay error="Error message" className={customClass} />);
    
    const errorDisplay = screen.getByTestId('error-display');
    expect(errorDisplay).toHaveClass(customClass);
  });
});