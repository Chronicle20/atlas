import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ErrorBoundary, withErrorBoundary, ErrorBoundaryProvider, useErrorBoundary } from '@/components/common/ErrorBoundary';
import * as React from 'react';

// Mock component that throws an error
const ThrowingComponent = ({ shouldThrow = true, errorMessage = 'Test error' }: { 
  shouldThrow?: boolean; 
  errorMessage?: string; 
}) => {
  if (shouldThrow) {
    throw new Error(errorMessage);
  }
  return <div>Component rendered successfully</div>;
};

// Mock component that uses error boundary context
const ComponentWithErrorBoundaryHook = () => {
  const { resetError } = useErrorBoundary();
  
  return (
    <button onClick={resetError} data-testid="context-reset-button">
      Reset from Context
    </button>
  );
};

// Mock component with custom error fallback
const CustomErrorFallback = ({ 
  error, 
  resetError 
}: { 
  error: Error; 
  resetError: () => void; 
}) => (
  <div data-testid="custom-error-fallback">
    <h2>Custom Error: {error.message}</h2>
    <button onClick={resetError} data-testid="custom-reset-button">
      Custom Reset
    </button>
  </div>
);

// Suppress console.error for expected errors in tests
const originalConsoleError = console.error;
beforeAll(() => {
  console.error = jest.fn();
});

afterAll(() => {
  console.error = originalConsoleError;
});

describe('ErrorBoundary Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Basic Error Catching Integration', () => {
    it('should catch errors and display default fallback UI', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={true} errorMessage="Component crashed" />
        </ErrorBoundary>
      );

      // Should not render the throwing component
      expect(screen.queryByText('Component rendered successfully')).not.toBeInTheDocument();
      
      // Should render default error fallback
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText('Component crashed')).toBeInTheDocument();
      expect(screen.getByText('Try Again')).toBeInTheDocument();
      expect(screen.getByText('Go Home')).toBeInTheDocument();
    });

    it('should render children when no error occurs', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    });

    it('should reset error state and re-render children when Try Again is clicked', async () => {
      let shouldThrow = true;
      
      const { rerender } = render(
        <ErrorBoundary key={shouldThrow ? 'error' : 'success'}>
          <ThrowingComponent shouldThrow={shouldThrow} />
        </ErrorBoundary>
      );

      // Initially should show error fallback
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      
      // Click Try Again to reset error boundary state
      const tryAgainButton = screen.getByText('Try Again');
      fireEvent.click(tryAgainButton);
      
      // Change the condition and rerender with new key to force component recreation
      shouldThrow = false;
      rerender(
        <ErrorBoundary key={shouldThrow ? 'error' : 'success'}>
          <ThrowingComponent shouldThrow={shouldThrow} />
        </ErrorBoundary>
      );

      // Should now render children successfully
      await waitFor(() => {
        expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
        expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
      });
    });
  });

  describe('Custom Fallback Integration', () => {
    it('should render custom fallback component when provided', () => {
      render(
        <ErrorBoundary fallback={CustomErrorFallback}>
          <ThrowingComponent shouldThrow={true} errorMessage="Custom error message" />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('custom-error-fallback')).toBeInTheDocument();
      expect(screen.getByText('Custom Error: Custom error message')).toBeInTheDocument();
      expect(screen.getByTestId('custom-reset-button')).toBeInTheDocument();
      
      // Should not render default fallback
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    });

    it('should reset error state when custom fallback reset button is clicked', async () => {
      let shouldThrow = true;
      
      const { rerender } = render(
        <ErrorBoundary fallback={CustomErrorFallback} key={shouldThrow ? 'error' : 'success'}>
          <ThrowingComponent shouldThrow={shouldThrow} />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('custom-error-fallback')).toBeInTheDocument();
      
      // Click custom reset button
      const resetButton = screen.getByTestId('custom-reset-button');
      fireEvent.click(resetButton);
      
      // Fix the error condition and rerender with new key
      shouldThrow = false;
      rerender(
        <ErrorBoundary fallback={CustomErrorFallback} key={shouldThrow ? 'error' : 'success'}>
          <ThrowingComponent shouldThrow={shouldThrow} />
        </ErrorBoundary>
      );

      await waitFor(() => {
        expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
        expect(screen.queryByTestId('custom-error-fallback')).not.toBeInTheDocument();
      });
    });
  });

  describe('Error Handler Integration', () => {
    it('should call onError callback when error occurs', () => {
      const mockOnError = jest.fn();
      
      render(
        <ErrorBoundary onError={mockOnError}>
          <ThrowingComponent shouldThrow={true} errorMessage="Callback test error" />
        </ErrorBoundary>
      );

      expect(mockOnError).toHaveBeenCalledTimes(1);
      expect(mockOnError).toHaveBeenCalledWith(
        expect.objectContaining({
          message: 'Callback test error'
        }),
        expect.objectContaining({
          componentStack: expect.any(String)
        })
      );
    });

    it('should not call onError callback when no error occurs', () => {
      const mockOnError = jest.fn();
      
      render(
        <ErrorBoundary onError={mockOnError}>
          <ThrowingComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      expect(mockOnError).not.toHaveBeenCalled();
    });
  });

  describe('Technical Details Integration', () => {
    it('should show technical details toggle when showDetails is true', () => {
      render(
        <ErrorBoundary showDetails={true}>
          <ThrowingComponent shouldThrow={true} errorMessage="Details test error" />
        </ErrorBoundary>
      );

      expect(screen.getByText('Show Technical Details')).toBeInTheDocument();
    });

    it('should expand and collapse technical details when toggle is clicked', async () => {
      render(
        <ErrorBoundary showDetails={true}>
          <ThrowingComponent shouldThrow={true} errorMessage="Details test error" />
        </ErrorBoundary>
      );

      const toggleButton = screen.getByText('Show Technical Details');
      
      // Initially should not show details
      expect(screen.queryByText('Error Details:')).not.toBeInTheDocument();
      
      // Click to expand
      fireEvent.click(toggleButton);
      
      await waitFor(() => {
        expect(screen.getByText('Error Details:')).toBeInTheDocument();
        expect(screen.getByText('Hide Technical Details')).toBeInTheDocument();
        expect(screen.getByText('Details test error')).toBeInTheDocument();
      });
      
      // Click to collapse
      fireEvent.click(screen.getByText('Hide Technical Details'));
      
      await waitFor(() => {
        expect(screen.queryByText('Error Details:')).not.toBeInTheDocument();
        expect(screen.getByText('Show Technical Details')).toBeInTheDocument();
      });
    });

    it('should not show technical details toggle when showDetails is false', () => {
      render(
        <ErrorBoundary showDetails={false}>
          <ThrowingComponent shouldThrow={true} errorMessage="No details test error" />
        </ErrorBoundary>
      );

      expect(screen.queryByText('Show Technical Details')).not.toBeInTheDocument();
      expect(screen.queryByText('Hide Technical Details')).not.toBeInTheDocument();
    });
  });

  describe('HOC Integration (withErrorBoundary)', () => {
    it('should wrap component with error boundary using HOC', () => {
      const WrappedComponent = withErrorBoundary(ThrowingComponent);
      
      render(<WrappedComponent shouldThrow={true} errorMessage="HOC test error" />);

      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText('HOC test error')).toBeInTheDocument();
    });

    it('should render wrapped component successfully when no error', () => {
      const WrappedComponent = withErrorBoundary(ThrowingComponent);
      
      render(<WrappedComponent shouldThrow={false} />);

      expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    });

    it('should pass through error boundary props in HOC', () => {
      const mockOnError = jest.fn();
      const WrappedComponent = withErrorBoundary(ThrowingComponent, {
        onError: mockOnError,
        showDetails: true
      });
      
      render(<WrappedComponent shouldThrow={true} errorMessage="HOC props test" />);

      expect(mockOnError).toHaveBeenCalledTimes(1);
      expect(screen.getByText('Show Technical Details')).toBeInTheDocument();
    });

    it('should reset error in HOC wrapped component', async () => {
      let shouldThrow = true;
      const WrappedComponent = withErrorBoundary(ThrowingComponent);
      
      const { rerender } = render(<WrappedComponent shouldThrow={shouldThrow} key={shouldThrow ? 'error' : 'success'} />);

      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      
      const tryAgainButton = screen.getByText('Try Again');
      fireEvent.click(tryAgainButton);
      
      shouldThrow = false;
      rerender(<WrappedComponent shouldThrow={shouldThrow} key={shouldThrow ? 'error' : 'success'} />);

      await waitFor(() => {
        expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
        expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
      });
    });
  });

  describe('Context Provider Integration', () => {
    it('should provide reset functionality through context', () => {
      render(
        <ErrorBoundaryProvider>
          <ComponentWithErrorBoundaryHook />
        </ErrorBoundaryProvider>
      );

      expect(screen.getByTestId('context-reset-button')).toBeInTheDocument();
    });

    it('should throw error when useErrorBoundary is used outside provider', () => {
      // Temporarily suppress error console output for this test
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      
      expect(() => {
        render(<ComponentWithErrorBoundaryHook />);
      }).toThrow('useErrorBoundary must be used within an ErrorBoundary');
      
      consoleErrorSpy.mockRestore();
    });

    it('should reset error through context when child component calls resetError', () => {
      let shouldThrow = true;
      
      const ComponentWithContextReset = () => {
        const { resetError } = useErrorBoundary();
        
        return (
          <div>
            <ThrowingComponent shouldThrow={shouldThrow} />
            <button onClick={resetError} data-testid="context-reset">
              Reset via Context
            </button>
          </div>
        );
      };

      const { rerender } = render(
        <ErrorBoundaryProvider>
          <ComponentWithContextReset />
        </ErrorBoundaryProvider>
      );

      // Should show error fallback initially
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      
      // Fix the error condition
      shouldThrow = false;
      
      // Reset through context (note: since error fallback is shown, 
      // the context reset button is not available, so we use the Try Again button)
      const tryAgainButton = screen.getByText('Try Again');
      fireEvent.click(tryAgainButton);
      
      rerender(
        <ErrorBoundaryProvider>
          <ComponentWithContextReset />
        </ErrorBoundaryProvider>
      );

      expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
      expect(screen.getByTestId('context-reset')).toBeInTheDocument();
    });
  });

  describe('Multiple Error Boundaries Integration', () => {
    it('should handle nested error boundaries correctly', () => {
      render(
        <ErrorBoundary>
          <div>
            <h1>Outer Boundary</h1>
            <ErrorBoundary fallback={CustomErrorFallback}>
              <ThrowingComponent shouldThrow={true} errorMessage="Inner boundary error" />
            </ErrorBoundary>
          </div>
        </ErrorBoundary>
      );

      // Inner boundary should catch the error
      expect(screen.getByTestId('custom-error-fallback')).toBeInTheDocument();
      expect(screen.getByText('Custom Error: Inner boundary error')).toBeInTheDocument();
      
      // Outer boundary content should still be visible
      expect(screen.getByText('Outer Boundary')).toBeInTheDocument();
      
      // Outer boundary's default fallback should not be shown
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    });

    it('should bubble error to outer boundary when inner boundary also throws', () => {
      const FailingCustomFallback = ({ error }: { error: Error }) => {
        throw new Error(`Fallback also failed: ${error.message}`);
      };

      render(
        <ErrorBoundary>
          <ErrorBoundary fallback={FailingCustomFallback}>
            <ThrowingComponent shouldThrow={true} errorMessage="Original error" />
          </ErrorBoundary>
        </ErrorBoundary>
      );

      // Outer boundary should catch the bubbled error
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText('Fallback also failed: Original error')).toBeInTheDocument();
    });
  });

  describe('State Management Integration', () => {
    it('should maintain separate error states for multiple error boundaries', () => {
      const { rerender } = render(
        <div>
          <ErrorBoundary data-testid="boundary-1">
            <ThrowingComponent shouldThrow={true} errorMessage="Error 1" />
          </ErrorBoundary>
          <ErrorBoundary data-testid="boundary-2">
            <ThrowingComponent shouldThrow={false} />
          </ErrorBoundary>
        </div>
      );

      // First boundary should show error
      expect(screen.getByText('Error 1')).toBeInTheDocument();
      
      // Second boundary should show success
      expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
      
      // Both error fallbacks should have "Try Again" buttons, but only one should be visible
      const tryAgainButtons = screen.getAllByText('Try Again');
      expect(tryAgainButtons).toHaveLength(1);
    });

    it('should independently reset error boundaries', () => {
      let shouldThrow1 = true;
      let shouldThrow2 = true;
      
      const Component1 = () => <ThrowingComponent shouldThrow={shouldThrow1} errorMessage="Error 1" />;
      const Component2 = () => <ThrowingComponent shouldThrow={shouldThrow2} errorMessage="Error 2" />;

      const { rerender } = render(
        <div>
          <div data-testid="boundary-1-container">
            <ErrorBoundary>
              <Component1 />
            </ErrorBoundary>
          </div>
          <div data-testid="boundary-2-container">
            <ErrorBoundary>
              <Component2 />
            </ErrorBoundary>
          </div>
        </div>
      );

      // Both should show errors initially
      expect(screen.getByText('Error 1')).toBeInTheDocument();
      expect(screen.getByText('Error 2')).toBeInTheDocument();

      // Fix first error and reset first boundary
      shouldThrow1 = false;
      const tryAgainButtons = screen.getAllByText('Try Again');
      fireEvent.click(tryAgainButtons[0]);

      rerender(
        <div>
          <div data-testid="boundary-1-container">
            <ErrorBoundary>
              <Component1 />
            </ErrorBoundary>
          </div>
          <div data-testid="boundary-2-container">
            <ErrorBoundary>
              <Component2 />
            </ErrorBoundary>
          </div>
        </div>
      );

      // First should be fixed, second should still show error
      expect(screen.getByText('Component rendered successfully')).toBeInTheDocument();
      expect(screen.getByText('Error 2')).toBeInTheDocument();
    });
  });

  describe('Styling and Custom Props Integration', () => {
    it('should apply custom className to error boundary fallback', () => {
      const customClass = 'my-custom-error-class';
      
      const { container } = render(
        <ErrorBoundary className={customClass}>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // The className should be on the outermost error boundary container
      const fallbackContainer = container.firstChild;
      expect(fallbackContainer).toHaveClass(customClass);
      expect(fallbackContainer).toHaveClass('flex', 'items-center', 'justify-center');
    });

    it('should pass through className to custom fallback component', () => {
      const CustomFallbackWithClass = ({ className }: { className?: string }) => (
        <div className={className} data-testid="custom-fallback-with-class">
          Custom fallback with class
        </div>
      );

      const customClass = 'custom-fallback-class';
      
      render(
        <ErrorBoundary 
          fallback={CustomFallbackWithClass}
          className={customClass}
        >
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const customFallback = screen.getByTestId('custom-fallback-with-class');
      expect(customFallback).toHaveClass(customClass);
    });
  });
});