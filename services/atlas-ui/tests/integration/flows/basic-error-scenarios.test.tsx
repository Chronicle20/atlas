/**
 * Basic error scenarios integration tests
 * Tests error handling in application components and flows
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { AllProviders } from '../utils/test-providers';
import { createApiErrorFromResponse } from '@/types/api/errors';

// Mock Sonner toast
jest.mock('sonner', () => ({
  toast: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
    loading: jest.fn(),
    promise: jest.fn(),
    dismiss: jest.fn(),
  },
  Toaster: ({ children }: { children?: React.ReactNode }) => <div data-testid="toaster">{children}</div>,
}));

// Mock Next.js router
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
  }),
  usePathname: () => '/test-path',
  useSearchParams: () => new URLSearchParams(),
}));

// Test component that simulates API data fetching with error handling
const TestDataComponent = ({ 
  onFetch,
  showErrorBoundary = false 
}: { 
  onFetch: () => Promise<any>;
  showErrorBoundary?: boolean;
}) => {
  const [data, setData] = React.useState<any>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  if (showErrorBoundary) {
    throw new Error('Component crashed during render');
  }

  const handleFetch = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await onFetch();
      setData(result);
    } catch (err: any) {
      setError(err.message || 'An unexpected error occurred');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div data-testid="test-data-component">
      <button onClick={handleFetch} data-testid="fetch-button" disabled={loading}>
        {loading ? 'Loading...' : 'Fetch Data'}
      </button>
      
      {error && (
        <div data-testid="error-display" role="alert">
          <span>Error: {error}</span>
          <button onClick={handleFetch} data-testid="retry-button">
            Retry
          </button>
        </div>
      )}
      
      {data && (
        <div data-testid="success-data">
          Data: {JSON.stringify(data)}
        </div>
      )}
    </div>
  );
};

// Test form component with validation
const TestFormComponent = ({ onSubmit }: { onSubmit: (data: any) => Promise<any> }) => {
  const [formData, setFormData] = React.useState({ name: '', email: '' });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setErrors({});
    setSubmitError(null);

    try {
      await onSubmit(formData);
    } catch (err: any) {
      if (err.code === 'VALIDATION_ERROR' && err.fieldErrors) {
        setErrors(err.fieldErrors);
      } else {
        setSubmitError(err.message || 'Submit failed');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} data-testid="test-form">
      <div>
        <input
          type="text"
          placeholder="Name"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          data-testid="name-input"
        />
        {errors.name && (
          <div data-testid="name-error" role="alert">
            {errors.name.join(', ')}
          </div>
        )}
      </div>

      <div>
        <input
          type="email"
          placeholder="Email"
          value={formData.email}
          onChange={(e) => setFormData({ ...formData, email: e.target.value })}
          data-testid="email-input"
        />
        {errors.email && (
          <div data-testid="email-error" role="alert">
            {errors.email.join(', ')}
          </div>
        )}
      </div>

      {submitError && (
        <div data-testid="submit-error" role="alert">
          {submitError}
        </div>
      )}

      <button type="submit" disabled={isSubmitting} data-testid="submit-button">
        {isSubmitting ? 'Submitting...' : 'Submit'}
      </button>
    </form>
  );
};

// Test provider component
const TestProviders = ({ 
  children, 
  withErrorBoundary = false 
}: { 
  children: React.ReactNode;
  withErrorBoundary?: boolean;
}) => {
  return <AllProviders withErrorBoundary={withErrorBoundary}>{children}</AllProviders>;
};

describe('Basic Error Scenarios Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('API Error Handling', () => {
    it('should handle network connection errors', async () => {
      const user = userEvent.setup();
      
      const mockFetch = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(0, 'Network connection failed')
      );

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-display')).toHaveTextContent(/network connection failed/i);
      expect(screen.getByTestId('retry-button')).toBeInTheDocument();
    });

    it('should handle authentication errors (401)', async () => {
      const user = userEvent.setup();
      
      const mockFetch = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(401, 'Authentication required')
      );

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-display')).toHaveTextContent(/authentication required/i);
    });

    it('should handle server errors (500)', async () => {
      const user = userEvent.setup();
      
      const mockFetch = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(500, 'Internal server error')
      );

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-display')).toHaveTextContent(/internal server error/i);
    });

    it('should handle successful data fetching', async () => {
      const user = userEvent.setup();
      
      const mockData = { id: 1, name: 'Test Data' };
      const mockFetch = jest.fn().mockResolvedValue(mockData);

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      expect(screen.getByTestId('success-data')).toHaveTextContent(JSON.stringify(mockData));
    });

    it('should allow retry after error', async () => {
      const user = userEvent.setup();
      
      const mockData = { id: 1, name: 'Success Data' };
      const mockFetch = jest.fn()
        .mockRejectedValueOnce(createApiErrorFromResponse(500, 'Server error'))
        .mockResolvedValueOnce(mockData);

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      // First attempt fails
      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      // Retry succeeds
      const retryButton = screen.getByTestId('retry-button');
      await user.click(retryButton);

      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
      expect(screen.getByTestId('success-data')).toHaveTextContent(JSON.stringify(mockData));
    });
  });

  describe('Form Validation Errors', () => {
    it('should handle validation errors with field-specific messages', async () => {
      const user = userEvent.setup();
      
      const validationError = {
        message: 'Validation failed',
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        fieldErrors: {
          name: ['Name is required'],
          email: ['Invalid email format'],
        },
      };

      const mockSubmit = jest.fn().mockRejectedValue(validationError);

      render(
        <TestProviders>
          <TestFormComponent onSubmit={mockSubmit} />
        </TestProviders>
      );

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('name-error')).toHaveTextContent('Name is required');
      expect(screen.getByTestId('email-error')).toHaveTextContent('Invalid email format');
    });

    it('should handle general submit errors', async () => {
      const user = userEvent.setup();
      
      const submitError = createApiErrorFromResponse(500, 'Server error during submit');
      const mockSubmit = jest.fn().mockRejectedValue(submitError);

      render(
        <TestProviders>
          <TestFormComponent onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.type(screen.getByTestId('email-input'), 'john@example.com');

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/server error during submit/i);
    });

    it('should preserve form data during errors', async () => {
      const user = userEvent.setup();
      
      const submitError = createApiErrorFromResponse(400, 'Bad request');
      const mockSubmit = jest.fn().mockRejectedValue(submitError);

      render(
        <TestProviders>
          <TestFormComponent onSubmit={mockSubmit} />
        </TestProviders>
      );

      const nameInput = screen.getByTestId('name-input');
      const emailInput = screen.getByTestId('email-input');

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      // Form data should be preserved
      expect(nameInput).toHaveValue('John Doe');
      expect(emailInput).toHaveValue('john@example.com');
    });
  });

  describe('React Error Boundary Integration', () => {
    // Suppress console.error for expected errors
    const originalConsoleError = console.error;
    beforeAll(() => {
      console.error = jest.fn();
    });

    afterAll(() => {
      console.error = originalConsoleError;
    });

    it('should catch component render errors', () => {
      const mockFetch = jest.fn();

      render(
        <TestProviders withErrorBoundary={true}>
          <TestDataComponent onFetch={mockFetch} showErrorBoundary={true} />
        </TestProviders>
      );

      // Should not render the component successfully
      expect(screen.queryByTestId('test-data-component')).not.toBeInTheDocument();
      
      // Should render error boundary fallback
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText(/Component crashed during render/)).toBeInTheDocument();
    });

    it('should show error boundary recovery options', () => {
      const mockFetch = jest.fn();

      render(
        <TestProviders withErrorBoundary={true}>
          <TestDataComponent onFetch={mockFetch} showErrorBoundary={true} />
        </TestProviders>
      );

      // Should show recovery buttons
      expect(screen.getByText('Try Again')).toBeInTheDocument();
      expect(screen.getByText('Go Home')).toBeInTheDocument();
    });
  });

  describe('Loading States', () => {
    it('should show loading state during async operations', async () => {
      const user = userEvent.setup();
      
      let resolvePromise: (value: any) => void;
      const mockFetch = jest.fn().mockImplementation(() => 
        new Promise(resolve => { resolvePromise = resolve; })
      );

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      // Should show loading state
      expect(fetchButton).toBeDisabled();
      expect(fetchButton).toHaveTextContent('Loading...');

      // Resolve the promise
      resolvePromise!({ data: 'test' });

      await waitFor(() => {
        expect(fetchButton).not.toBeDisabled();
        expect(fetchButton).toHaveTextContent('Fetch Data');
      });
    });

    it('should disable form during submission', async () => {
      const user = userEvent.setup();
      
      let resolvePromise: (value: any) => void;
      const mockSubmit = jest.fn().mockImplementation(() => 
        new Promise(resolve => { resolvePromise = resolve; })
      );

      render(
        <TestProviders>
          <TestFormComponent onSubmit={mockSubmit} />
        </TestProviders>
      );

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      // Should show submitting state
      expect(submitButton).toBeDisabled();
      expect(submitButton).toHaveTextContent('Submitting...');

      // Resolve the promise
      resolvePromise!({ success: true });

      await waitFor(() => {
        expect(submitButton).not.toBeDisabled();
        expect(submitButton).toHaveTextContent('Submit');
      });
    });
  });

  describe('Multiple Error Scenarios', () => {
    it('should handle independent component errors', async () => {
      const user = userEvent.setup();
      
      const mockFetch1 = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(404, 'Not found')
      );
      const mockFetch2 = jest.fn().mockResolvedValue({ data: 'success' });

      render(
        <TestProviders>
          <div>
            <div data-testid="component-1">
              <TestDataComponent onFetch={mockFetch1} />
            </div>
            <div data-testid="component-2">
              <TestDataComponent onFetch={mockFetch2} />
            </div>
          </div>
        </TestProviders>
      );

      const fetchButton1 = screen.getAllByTestId('fetch-button')[0];
      const fetchButton2 = screen.getAllByTestId('fetch-button')[1];

      await user.click(fetchButton1);
      await user.click(fetchButton2);

      await waitFor(() => {
        const component1 = screen.getByTestId('component-1');
        const component2 = screen.getByTestId('component-2');

        // Component 1 should show error
        expect(component1.querySelector('[data-testid="error-display"]')).toBeInTheDocument();
        
        // Component 2 should show success
        expect(component2.querySelector('[data-testid="success-data"]')).toBeInTheDocument();
      });
    });
  });

  describe('Error Recovery and State Management', () => {
    it('should clear errors when operation succeeds', async () => {
      const user = userEvent.setup();
      
      const mockFetch = jest.fn()
        .mockRejectedValueOnce(createApiErrorFromResponse(500, 'Server error'))
        .mockResolvedValueOnce({ data: 'success' });

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      // First attempt fails
      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      // Second attempt succeeds
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      // Error should be cleared
      expect(screen.queryByTestId('error-display')).not.toBeInTheDocument();
    });

    it('should maintain component interactivity after errors', async () => {
      const user = userEvent.setup();
      
      const mockFetch = jest.fn().mockRejectedValue(
        createApiErrorFromResponse(503, 'Service unavailable')
      );

      render(
        <TestProviders>
          <TestDataComponent onFetch={mockFetch} />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-display')).toBeInTheDocument();
      });

      // Component should still be interactive
      expect(fetchButton).not.toBeDisabled();
      expect(screen.getByTestId('retry-button')).toBeInTheDocument();
      
      // Should be able to trigger another request
      await user.click(screen.getByTestId('retry-button'));
      
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });
});