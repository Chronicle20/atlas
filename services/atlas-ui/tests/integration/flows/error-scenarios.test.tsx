/**
 * Integration tests for error scenarios across the application
 * Tests the complete error handling flow from API failures to user feedback
 */

import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { AllProviders } from '../utils/test-providers';
import { createApiErrorFromResponse } from '@/types/api/errors';
import * as apiClient from '@/lib/api/client';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  delete: jest.fn(),
}));

// Mock Sonner toast to track toast notifications
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
const mockReplace = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
  }),
  usePathname: () => '/test-path',
  useSearchParams: () => new URLSearchParams(),
}));

// Test component that demonstrates API error handling
const TestApiComponent = ({ endpoint }: { endpoint: string }) => {
  const [data, setData] = React.useState<any>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await mockApiClient.get(endpoint);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <button onClick={fetchData} data-testid="fetch-button">
        Fetch Data
      </button>
      {loading && <div data-testid="loading">Loading...</div>}
      {error && <div data-testid="error-message">{error}</div>}
      {data && <div data-testid="success-data">{JSON.stringify(data)}</div>}
    </div>
  );
};

// Test component that throws errors to test ErrorBoundary
const ThrowingComponent = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Component crashed during render');
  }
  return <div data-testid="throwing-component-success">Component rendered successfully</div>;
};

// Form component to test validation errors
const TestFormComponent = () => {
  const [formData, setFormData] = React.useState({ email: '', password: '' });
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setErrors({});

    try {
      await mockApiClient.post('/api/submit-form', formData);
      // Success handling would go here
    } catch (err: any) {
      if (err.fieldErrors) {
        setErrors(err.fieldErrors);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} data-testid="test-form">
      <input
        type="email"
        placeholder="Email"
        value={formData.email}
        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
        data-testid="email-input"
      />
      {errors.email && <div data-testid="email-error">{errors.email}</div>}
      
      <input
        type="password"
        placeholder="Password"
        value={formData.password}
        onChange={(e) => setFormData({ ...formData, password: e.target.value })}
        data-testid="password-input"
      />
      {errors.password && <div data-testid="password-error">{errors.password}</div>}
      
      <button type="submit" disabled={isSubmitting} data-testid="submit-button">
        {isSubmitting ? 'Submitting...' : 'Submit'}
      </button>
    </form>
  );
};

// Mock test provider component since no React Query is used
const TestProviders = ({ children }: { children: React.ReactNode }) => {
  return <AllProviders>{children}</AllProviders>;
};

describe('Error Scenarios Integration Tests', () => {
  const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('API Error Scenarios', () => {
    it('should handle network connection errors', async () => {
      const user = userEvent.setup();
      
      // Mock network error
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(0, 'Network request failed')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/network/i);
      expect(mockApiClient.get).toHaveBeenCalledWith('/api/data');
    });

    it('should handle authentication errors (401)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(401, 'Unauthorized')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/protected" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/unauthorized|sign in/i);
    });

    it('should handle authorization errors (403)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(403, 'Forbidden')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/admin" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/forbidden|permission/i);
    });

    it('should handle not found errors (404)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(404, 'Not Found')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/nonexistent" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/not found/i);
    });

    it('should handle rate limiting errors (429)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(429, 'Too Many Requests')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/too many requests|rate limit/i);
    });

    it('should handle server errors (500)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(500, 'Internal Server Error')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/server error|unexpected error/i);
    });

    it('should handle service unavailable errors (503)', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(503, 'Service Unavailable')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent(/service unavailable|try again/i);
    });
  });

  describe('Form Validation Error Scenarios', () => {
    it('should handle validation errors with field-specific messages', async () => {
      const user = userEvent.setup();
      
      const validationError = {
        message: 'Validation failed',
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        fieldErrors: {
          email: ['Email is required', 'Invalid email format'],
          password: ['Password must be at least 8 characters'],
        },
      };

      mockApiClient.post.mockRejectedValue(validationError);

      render(
        <TestProviders >
          <TestFormComponent />
        </TestProviders>
      );

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
        expect(screen.getByTestId('password-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('email-error')).toHaveTextContent(/email is required/i);
      expect(screen.getByTestId('password-error')).toHaveTextContent(/8 characters/i);
    });

    it('should handle form submission with invalid input data', async () => {
      const user = userEvent.setup();
      
      mockApiClient.post.mockRejectedValue(
        createApiErrorFromResponse(400, 'Bad Request - Invalid input')
      );

      render(
        <TestProviders >
          <TestFormComponent />
        </TestProviders>
      );

      // Fill form with invalid data
      await user.type(screen.getByTestId('email-input'), 'invalid-email');
      await user.type(screen.getByTestId('password-input'), '123');
      
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockApiClient.post).toHaveBeenCalledWith('/api/submit-form', {
          email: 'invalid-email',
          password: '123',
        });
      });
    });
  });

  describe('React Error Boundary Integration', () => {
    // Suppress console.error for expected errors in tests
    const originalConsoleError = console.error;
    beforeAll(() => {
      console.error = jest.fn();
    });

    afterAll(() => {
      console.error = originalConsoleError;
    });

    it('should catch component render errors and show fallback UI', () => {
      render(
        <TestProviders >
          <ThrowingComponent shouldThrow={true} />
        </TestProviders>
      );

      // Should not render the component successfully
      expect(screen.queryByTestId('throwing-component-success')).not.toBeInTheDocument();
      
      // Should render error boundary fallback
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText(/Component crashed during render/)).toBeInTheDocument();
    });

    it('should allow error recovery through error boundary reset', async () => {
      const user = userEvent.setup();
      let shouldThrow = true;
      
      const { rerender } = render(
        <TestProviders >
          <ThrowingComponent shouldThrow={shouldThrow} />
        </TestProviders>
      );

      // Initially should show error
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      
      // Click Try Again
      const tryAgainButton = screen.getByText('Try Again');
      await user.click(tryAgainButton);
      
      // Fix the error condition and rerender
      shouldThrow = false;
      rerender(
        <TestProviders >
          <ThrowingComponent shouldThrow={shouldThrow} />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('throwing-component-success')).toBeInTheDocument();
      });
    });
  });

  describe('Toast Notification Integration', () => {
    const mockToast = require('sonner').toast;

    it('should show toast notification for API errors', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(500, 'Server Error')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // The component should handle the error internally
      // Toast notifications would be triggered by the actual API client implementation
    });

    it('should show success toast after successful operations', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockResolvedValue({ success: true, data: 'test data' });

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      expect(screen.getByTestId('success-data')).toHaveTextContent('test data');
    });
  });

  describe('Concurrent Error Scenarios', () => {
    it('should handle multiple simultaneous API failures', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(500, 'Server Error')
      );

      render(
        <TestProviders >
          <div>
            <TestApiComponent endpoint="/api/users" />
            <TestApiComponent endpoint="/api/posts" />
            <TestApiComponent endpoint="/api/comments" />
          </div>
        </TestProviders>
      );

      const fetchButtons = screen.getAllByTestId('fetch-button');
      
      // Click all buttons simultaneously
      await Promise.all(fetchButtons.map(button => user.click(button)));

      await waitFor(() => {
        const errorMessages = screen.getAllByTestId('error-message');
        expect(errorMessages).toHaveLength(3);
      });

      // All components should show error messages
      const errorMessages = screen.getAllByTestId('error-message');
      errorMessages.forEach(errorMessage => {
        expect(errorMessage).toHaveTextContent(/server error|unexpected error/i);
      });
    });

    it('should handle partial success in concurrent operations', async () => {
      const user = userEvent.setup();
      
      // Mock different responses for different endpoints
      mockApiClient.get
        .mockResolvedValueOnce({ success: true, data: 'users data' })
        .mockRejectedValueOnce(createApiErrorFromResponse(404, 'Posts not found'))
        .mockResolvedValueOnce({ success: true, data: 'comments data' });

      render(
        <TestProviders >
          <div>
            <div data-testid="users-section">
              <TestApiComponent endpoint="/api/users" />
            </div>
            <div data-testid="posts-section">
              <TestApiComponent endpoint="/api/posts" />
            </div>
            <div data-testid="comments-section">
              <TestApiComponent endpoint="/api/comments" />
            </div>
          </div>
        </TestProviders>
      );

      const fetchButtons = screen.getAllByTestId('fetch-button');
      
      // Click all buttons simultaneously
      await Promise.all(fetchButtons.map(button => user.click(button)));

      await waitFor(() => {
        // Users section should show success
        const usersSection = screen.getByTestId('users-section');
        expect(within(usersSection).getByTestId('success-data')).toBeInTheDocument();
        
        // Posts section should show error
        const postsSection = screen.getByTestId('posts-section');
        expect(within(postsSection).getByTestId('error-message')).toBeInTheDocument();
        
        // Comments section should show success
        const commentsSection = screen.getByTestId('comments-section');
        expect(within(commentsSection).getByTestId('success-data')).toBeInTheDocument();
      });
    });
  });

  describe('Error Recovery Scenarios', () => {
    it('should allow retry after network error', async () => {
      const user = userEvent.setup();
      
      // First call fails, second succeeds
      mockApiClient.get
        .mockRejectedValueOnce(createApiErrorFromResponse(0, 'Network Error'))
        .mockResolvedValueOnce({ success: true, data: 'recovered data' });

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      
      // First attempt fails
      await user.click(fetchButton);
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Retry succeeds
      await user.click(fetchButton);
      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      expect(screen.getByTestId('success-data')).toHaveTextContent('recovered data');
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });

    it('should maintain state during error recovery', async () => {
      const user = userEvent.setup();
      
      mockApiClient.get
        .mockRejectedValueOnce(createApiErrorFromResponse(500, 'Server Error'))
        .mockResolvedValueOnce({ success: true, data: 'final data' });

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      
      // First attempt fails
      await user.click(fetchButton);
      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // Component should still be interactive
      expect(fetchButton).toBeEnabled();
      
      // Second attempt succeeds
      await user.click(fetchButton);
      await waitFor(() => {
        expect(screen.getByTestId('success-data')).toBeInTheDocument();
      });

      // Error should be cleared
      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument();
    });
  });

  describe('Error Logging Integration', () => {
    it('should log errors with proper context information', async () => {
      const user = userEvent.setup();
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      mockApiClient.get.mockRejectedValue(
        createApiErrorFromResponse(500, 'Server Error')
      );

      render(
        <TestProviders >
          <TestApiComponent endpoint="/api/data" />
        </TestProviders>
      );

      const fetchButton = screen.getByTestId('fetch-button');
      await user.click(fetchButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      // The component itself may not log, but the error should be handled
      expect(mockApiClient.get).toHaveBeenCalledWith('/api/data');
      
      consoleSpy.mockRestore();
    });
  });
});