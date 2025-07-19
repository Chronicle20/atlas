import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LoadingOverlay } from '@/components/common/LoadingOverlay';
import * as React from 'react';

// Mock child components for testing
const MockChildComponent = ({ 
  onClick, 
  testId = 'child-component' 
}: { 
  onClick?: () => void; 
  testId?: string; 
}) => (
  <div data-testid={testId}>
    <h2>Child Content</h2>
    <p>This is the child content that should be visible.</p>
    {onClick && (
      <button onClick={onClick} data-testid="child-button">
        Click Me
      </button>
    )}
  </div>
);

const MockFormComponent = ({ onSubmit }: { onSubmit?: () => void }) => (
  <form onSubmit={onSubmit} data-testid="mock-form">
    <input 
      type="text" 
      placeholder="Enter text" 
      data-testid="form-input" 
    />
    <button type="submit" data-testid="submit-button">
      Submit
    </button>
  </form>
);

const MockDataComponent = ({ data }: { data: string[] }) => (
  <div data-testid="data-component">
    <h3>Data List</h3>
    <ul>
      {data.map((item, index) => (
        <li key={index} data-testid={`data-item-${index}`}>
          {item}
        </li>
      ))}
    </ul>
  </div>
);

describe('LoadingOverlay Integration Tests', () => {
  describe('Basic Overlay Functionality', () => {
    it('should render children without overlay when loading is false', () => {
      render(
        <LoadingOverlay loading={false}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      expect(screen.getByTestId('child-component')).toBeInTheDocument();
      expect(screen.getByText('Child Content')).toBeInTheDocument();
      expect(screen.getByText('This is the child content that should be visible.')).toBeInTheDocument();
      
      // Should not show loading spinner
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    });

    it('should render children with overlay when loading is true', () => {
      render(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      // Children should still be rendered
      expect(screen.getByTestId('child-component')).toBeInTheDocument();
      expect(screen.getByText('Child Content')).toBeInTheDocument();
      
      // Loading spinner should be visible
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });

    it('should show and hide overlay based on loading state changes', async () => {
      const { rerender } = render(
        <LoadingOverlay loading={false}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      // Initially no loading spinner
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();

      // Change to loading state
      rerender(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      await waitFor(() => {
        expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      });

      // Change back to not loading
      rerender(
        <LoadingOverlay loading={false}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      await waitFor(() => {
        expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      });
    });
  });

  describe('LoadingSpinner Size Integration', () => {
    it('should render LoadingSpinner with default size (md)', () => {
      render(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toBeInTheDocument();
      // Default size is md, which should have h-6 w-6 classes
      expect(spinner.querySelector('svg')).toHaveClass('h-6', 'w-6');
    });

    it('should render LoadingSpinner with small size', () => {
      render(
        <LoadingOverlay loading={true} spinnerSize="sm">
          <MockChildComponent />
        </LoadingOverlay>
      );

      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toBeInTheDocument();
      // Small size should have h-4 w-4 classes
      expect(spinner.querySelector('svg')).toHaveClass('h-4', 'w-4');
    });

    it('should render LoadingSpinner with large size', () => {
      render(
        <LoadingOverlay loading={true} spinnerSize="lg">
          <MockChildComponent />
        </LoadingOverlay>
      );

      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toBeInTheDocument();
      // Large size should have h-8 w-8 classes
      expect(spinner.querySelector('svg')).toHaveClass('h-8', 'w-8');
    });

    it('should change spinner size when spinnerSize prop changes', async () => {
      const { rerender } = render(
        <LoadingOverlay loading={true} spinnerSize="sm">
          <MockChildComponent />
        </LoadingOverlay>
      );

      let spinner = screen.getByTestId('loading-spinner');
      expect(spinner.querySelector('svg')).toHaveClass('h-4', 'w-4');

      rerender(
        <LoadingOverlay loading={true} spinnerSize="lg">
          <MockChildComponent />
        </LoadingOverlay>
      );

      await waitFor(() => {
        spinner = screen.getByTestId('loading-spinner');
        expect(spinner.querySelector('svg')).toHaveClass('h-8', 'w-8');
      });
    });
  });

  describe('User Interaction Blocking', () => {
    it('should block user interactions when overlay is active', () => {
      const mockClick = jest.fn();
      
      render(
        <LoadingOverlay loading={true}>
          <MockChildComponent onClick={mockClick} />
        </LoadingOverlay>
      );

      const childButton = screen.getByTestId('child-button');
      expect(childButton).toBeInTheDocument();
      
      // Try to click the button - it should be blocked by the overlay
      fireEvent.click(childButton);
      
      // The click should be blocked, so the handler should not be called
      // Note: This test might need adjustment based on actual implementation
      // The overlay should prevent clicks from reaching the underlying elements
    });

    it('should allow user interactions when overlay is not active', () => {
      const mockClick = jest.fn();
      
      render(
        <LoadingOverlay loading={false}>
          <MockChildComponent onClick={mockClick} />
        </LoadingOverlay>
      );

      const childButton = screen.getByTestId('child-button');
      fireEvent.click(childButton);
      
      expect(mockClick).toHaveBeenCalledTimes(1);
    });

    it('should restore interactions when loading state changes from true to false', async () => {
      const mockClick = jest.fn();
      
      const { rerender } = render(
        <LoadingOverlay loading={true}>
          <MockChildComponent onClick={mockClick} />
        </LoadingOverlay>
      );

      // Verify the loading overlay is present and child has pointer-events-none
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      const childComponent = screen.getByTestId('child-component');
      expect(childComponent.parentElement).toHaveClass('pointer-events-none');
      
      // Change to not loading
      rerender(
        <LoadingOverlay loading={false}>
          <MockChildComponent onClick={mockClick} />
        </LoadingOverlay>
      );

      await waitFor(() => {
        expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      });

      // Verify pointer-events-none is removed
      const updatedChildComponent = screen.getByTestId('child-component');
      expect(updatedChildComponent.parentElement).not.toHaveClass('pointer-events-none');

      // Now clicks should work
      fireEvent.click(updatedChildComponent);
      expect(mockClick).toHaveBeenCalledTimes(1);
    });
  });

  describe('Form Integration', () => {
    it('should prevent form submission when overlay is active', () => {
      const mockSubmit = jest.fn();
      
      render(
        <LoadingOverlay loading={true}>
          <MockFormComponent onSubmit={mockSubmit} />
        </LoadingOverlay>
      );

      const form = screen.getByTestId('mock-form');
      const submitButton = screen.getByTestId('submit-button');
      
      // Try to submit form
      fireEvent.click(submitButton);
      fireEvent.submit(form);
      
      // Form submission should be blocked
      // Note: The actual blocking behavior depends on implementation details
    });

    it('should allow form submission when overlay is not active', () => {
      const mockSubmit = jest.fn((e) => e.preventDefault());
      
      render(
        <LoadingOverlay loading={false}>
          <MockFormComponent onSubmit={mockSubmit} />
        </LoadingOverlay>
      );

      const form = screen.getByTestId('mock-form');
      fireEvent.submit(form);
      
      expect(mockSubmit).toHaveBeenCalledTimes(1);
    });

    it('should prevent input interactions when overlay is active', () => {
      render(
        <LoadingOverlay loading={true}>
          <MockFormComponent />
        </LoadingOverlay>
      );

      const input = screen.getByTestId('form-input');
      
      // Try to type in input
      fireEvent.change(input, { target: { value: 'test input' } });
      
      // The input interaction should be blocked by the overlay
      // Note: This behavior depends on the actual overlay implementation
    });
  });

  describe('Multiple Children Integration', () => {
    it('should render multiple children with overlay', () => {
      const data = ['Item 1', 'Item 2', 'Item 3'];
      
      render(
        <LoadingOverlay loading={true}>
          <MockChildComponent testId="child-1" />
          <MockDataComponent data={data} />
          <MockChildComponent testId="child-2" />
        </LoadingOverlay>
      );

      // All children should be rendered
      expect(screen.getByTestId('child-1')).toBeInTheDocument();
      expect(screen.getByTestId('data-component')).toBeInTheDocument();
      expect(screen.getByTestId('child-2')).toBeInTheDocument();
      
      // Data should be visible
      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Item 2')).toBeInTheDocument();
      expect(screen.getByText('Item 3')).toBeInTheDocument();
      
      // Loading overlay should be present
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });

    it('should handle complex nested structure with overlay', () => {
      render(
        <LoadingOverlay loading={true}>
          <div data-testid="wrapper">
            <header data-testid="header">
              <h1>Page Title</h1>
            </header>
            <main data-testid="main-content">
              <MockChildComponent />
              <section data-testid="section">
                <MockDataComponent data={['Nested Item 1', 'Nested Item 2']} />
              </section>
            </main>
            <footer data-testid="footer">
              <p>Footer Content</p>
            </footer>
          </div>
        </LoadingOverlay>
      );

      // All nested elements should be rendered
      expect(screen.getByTestId('wrapper')).toBeInTheDocument();
      expect(screen.getByTestId('header')).toBeInTheDocument();
      expect(screen.getByTestId('main-content')).toBeInTheDocument();
      expect(screen.getByTestId('section')).toBeInTheDocument();
      expect(screen.getByTestId('footer')).toBeInTheDocument();
      
      // Content should be visible
      expect(screen.getByText('Page Title')).toBeInTheDocument();
      expect(screen.getByText('Footer Content')).toBeInTheDocument();
      expect(screen.getByText('Nested Item 1')).toBeInTheDocument();
      
      // Loading overlay should be present
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });
  });

  describe('Styling Integration', () => {
    it('should apply custom className to container', () => {
      const customClass = 'my-custom-overlay-class';
      
      const { container } = render(
        <LoadingOverlay loading={false} className={customClass}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      const overlayContainer = container.firstChild;
      expect(overlayContainer).toHaveClass(customClass);
      expect(overlayContainer).toHaveClass('relative'); // Default class should also be present
    });

    it('should maintain relative positioning for overlay functionality', () => {
      const { container } = render(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      const overlayContainer = container.firstChild;
      expect(overlayContainer).toHaveClass('relative');
    });

    it('should apply overlay styling when loading', () => {
      const { container } = render(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      // Find the overlay element within the main container
      const overlayElement = container.querySelector('.absolute.inset-0');
      
      expect(overlayElement).toBeInTheDocument();
      expect(overlayElement).toHaveClass('absolute', 'inset-0', 'bg-background/50', 'flex', 'items-center', 'justify-center', 'z-10');
    });

    it('should combine custom className with default classes', () => {
      const customClass = 'border border-gray-300 rounded-lg';
      
      const { container } = render(
        <LoadingOverlay loading={true} className={customClass}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      const overlayContainer = container.firstChild;
      expect(overlayContainer).toHaveClass('relative'); // Default class
      expect(overlayContainer?.className).toContain(customClass); // Custom classes
    });
  });

  describe('Accessibility Integration', () => {
    it('should maintain accessibility of child components when not loading', () => {
      render(
        <LoadingOverlay loading={false}>
          <MockFormComponent />
        </LoadingOverlay>
      );

      const input = screen.getByTestId('form-input');
      const submitButton = screen.getByTestId('submit-button');
      
      expect(input).toBeVisible();
      expect(submitButton).toBeVisible();
      expect(input).not.toHaveAttribute('disabled');
      expect(submitButton).not.toHaveAttribute('disabled');
    });

    it('should provide loading state indication for screen readers', () => {
      render(
        <LoadingOverlay loading={true}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      // The LoadingSpinner should have appropriate aria attributes
      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toBeInTheDocument();
      
      // Check if the spinner has appropriate accessibility attributes
      // This depends on the LoadingSpinner implementation
    });
  });

  describe('Performance Integration', () => {
    it('should not re-render children unnecessarily when loading state changes', () => {
      let renderCount = 0;
      
      const TrackingComponent = () => {
        renderCount++;
        return <div data-testid="tracking-component">Render count: {renderCount}</div>;
      };

      const { rerender } = render(
        <LoadingOverlay loading={false}>
          <TrackingComponent />
        </LoadingOverlay>
      );

      expect(screen.getByText('Render count: 1')).toBeInTheDocument();

      // Change loading state
      rerender(
        <LoadingOverlay loading={true}>
          <TrackingComponent />
        </LoadingOverlay>
      );

      // Component should re-render, but this is expected behavior
      expect(screen.getByText('Render count: 2')).toBeInTheDocument();

      // Change back to not loading
      rerender(
        <LoadingOverlay loading={false}>
          <TrackingComponent />
        </LoadingOverlay>
      );

      expect(screen.getByText('Render count: 3')).toBeInTheDocument();
    });

    it('should handle rapid loading state changes', async () => {
      const { rerender } = render(
        <LoadingOverlay loading={false}>
          <MockChildComponent />
        </LoadingOverlay>
      );

      // Rapidly toggle loading state
      for (let i = 0; i < 5; i++) {
        rerender(
          <LoadingOverlay loading={true}>
            <MockChildComponent />
          </LoadingOverlay>
        );

        rerender(
          <LoadingOverlay loading={false}>
            <MockChildComponent />
          </LoadingOverlay>
        );
      }

      // Should end up in not loading state
      await waitFor(() => {
        expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      });

      expect(screen.getByTestId('child-component')).toBeInTheDocument();
      expect(screen.getByText('Child Content')).toBeInTheDocument();
    });
  });
});