/**
 * Integration tests for form error scenarios
 * Tests error handling in form components with validation and submission
 */

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { AllProviders } from '../utils/test-providers';
import { createApiErrorFromResponse } from '@/types/api/errors';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  post: jest.fn(),
  put: jest.fn(),
  delete: jest.fn(),
}));

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

// Test form component with comprehensive validation
const TestUserForm = ({ 
  onSubmit, 
  initialData = {} 
}: { 
  onSubmit: (data: any) => Promise<any>;
  initialData?: any;
}) => {
  const [formData, setFormData] = React.useState({
    name: initialData.name || '',
    email: initialData.email || '',
    password: initialData.password || '',
    confirmPassword: initialData.confirmPassword || '',
    role: initialData.role || 'user',
  });
  
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
        setSubmitError(err.message || 'An error occurred while submitting the form');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    
    // Clear field error when user starts typing
    if (errors[field]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  return (
    <form onSubmit={handleSubmit} data-testid="user-form">
      <div>
        <label htmlFor="name">Name</label>
        <input
          id="name"
          type="text"
          value={formData.name}
          onChange={(e) => handleInputChange('name', e.target.value)}
          data-testid="name-input"
          aria-invalid={!!errors.name}
        />
        {errors.name && (
          <div data-testid="name-error" role="alert">
            {errors.name.join(', ')}
          </div>
        )}
      </div>

      <div>
        <label htmlFor="email">Email</label>
        <input
          id="email"
          type="email"
          value={formData.email}
          onChange={(e) => handleInputChange('email', e.target.value)}
          data-testid="email-input"
          aria-invalid={!!errors.email}
        />
        {errors.email && (
          <div data-testid="email-error" role="alert">
            {errors.email.join(', ')}
          </div>
        )}
      </div>

      <div>
        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          value={formData.password}
          onChange={(e) => handleInputChange('password', e.target.value)}
          data-testid="password-input"
          aria-invalid={!!errors.password}
        />
        {errors.password && (
          <div data-testid="password-error" role="alert">
            {errors.password.join(', ')}
          </div>
        )}
      </div>

      <div>
        <label htmlFor="confirmPassword">Confirm Password</label>
        <input
          id="confirmPassword"
          type="password"
          value={formData.confirmPassword}
          onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
          data-testid="confirm-password-input"
          aria-invalid={!!errors.confirmPassword}
        />
        {errors.confirmPassword && (
          <div data-testid="confirm-password-error" role="alert">
            {errors.confirmPassword.join(', ')}
          </div>
        )}
      </div>

      <div>
        <label htmlFor="role">Role</label>
        <select
          id="role"
          value={formData.role}
          onChange={(e) => handleInputChange('role', e.target.value)}
          data-testid="role-select"
          aria-invalid={!!errors.role}
        >
          <option value="user">User</option>
          <option value="admin">Admin</option>
          <option value="moderator">Moderator</option>
        </select>
        {errors.role && (
          <div data-testid="role-error" role="alert">
            {errors.role.join(', ')}
          </div>
        )}
      </div>

      {submitError && (
        <div data-testid="submit-error" role="alert">
          {submitError}
        </div>
      )}

      <button 
        type="submit" 
        disabled={isSubmitting}
        data-testid="submit-button"
      >
        {isSubmitting ? 'Submitting...' : 'Submit'}
      </button>
    </form>
  );
};

// Test component for file upload scenarios
const TestFileUploadForm = ({ onUpload }: { onUpload: (file: File) => Promise<any> }) => {
  const [file, setFile] = React.useState<File | null>(null);
  const [uploading, setUploading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [progress, setProgress] = React.useState(0);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      setFile(selectedFile);
      setError(null);
    }
  };

  const handleUpload = async () => {
    if (!file) return;

    setUploading(true);
    setError(null);
    setProgress(0);

    try {
      // Simulate progress
      const progressInterval = setInterval(() => {
        setProgress(prev => Math.min(prev + 10, 90));
      }, 100);

      await onUpload(file);
      
      clearInterval(progressInterval);
      setProgress(100);
    } catch (err: any) {
      setError(err.message || 'Upload failed');
      setProgress(0);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div data-testid="upload-form">
      <div>
        <input
          type="file"
          onChange={handleFileChange}
          data-testid="file-input"
          accept=".jpg,.jpeg,.png,.pdf"
        />
      </div>

      {file && (
        <div data-testid="file-info">
          <p>Selected: {file.name}</p>
          <p>Size: {(file.size / 1024).toFixed(2)} KB</p>
        </div>
      )}

      {uploading && (
        <div data-testid="upload-progress">
          <div>Uploading... {progress}%</div>
          <div 
            style={{ width: `${progress}%`, height: '10px', backgroundColor: 'blue' }}
            data-testid="progress-bar"
          />
        </div>
      )}

      {error && (
        <div data-testid="upload-error" role="alert">
          {error}
        </div>
      )}

      <button
        onClick={handleUpload}
        disabled={!file || uploading}
        data-testid="upload-button"
      >
        {uploading ? 'Uploading...' : 'Upload'}
      </button>
    </div>
  );
};

// Mock test provider component
const TestProviders = ({ children }: { children: React.ReactNode }) => {
  return <AllProviders>{children}</AllProviders>;
};

describe('Form Error Scenarios Integration Tests', () => {
  const mockApiClient = require('@/lib/api/client');

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Validation Error Scenarios', () => {
    it('should display field-specific validation errors', async () => {
      const user = userEvent.setup();
      
      const validationError = {
        message: 'Validation failed',
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        fieldErrors: {
          name: ['Name is required', 'Name must be at least 2 characters'],
          email: ['Invalid email format'],
          password: ['Password must be at least 8 characters', 'Password must contain uppercase letter'],
          confirmPassword: ['Passwords do not match'],
        },
      };

      const mockSubmit = jest.fn().mockRejectedValue(validationError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      // Submit form with invalid data
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
        expect(screen.getByTestId('password-error')).toBeInTheDocument();
        expect(screen.getByTestId('confirm-password-error')).toBeInTheDocument();
      });

      // Check specific error messages
      expect(screen.getByTestId('name-error')).toHaveTextContent('Name is required, Name must be at least 2 characters');
      expect(screen.getByTestId('email-error')).toHaveTextContent('Invalid email format');
      expect(screen.getByTestId('password-error')).toHaveTextContent('Password must be at least 8 characters, Password must contain uppercase letter');
      expect(screen.getByTestId('confirm-password-error')).toHaveTextContent('Passwords do not match');
    });

    it('should clear field errors when user starts typing', async () => {
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
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      // Submit form to show validation errors
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
      });

      // Start typing in name field
      await user.type(screen.getByTestId('name-input'), 'John');

      // Name error should be cleared
      expect(screen.queryByTestId('name-error')).not.toBeInTheDocument();
      
      // Email error should still be present
      expect(screen.getByTestId('email-error')).toBeInTheDocument();
    });

    it('should handle mixed validation and server errors', async () => {
      const user = userEvent.setup();
      
      const mixedError = {
        message: 'Multiple issues found',
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        fieldErrors: {
          email: ['Email already exists'],
        },
      };

      const mockSubmit = jest.fn().mockRejectedValue(mixedError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      // Fill form with valid data
      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.type(screen.getByTestId('email-input'), 'existing@example.com');
      await user.type(screen.getByTestId('password-input'), 'Password123');
      await user.type(screen.getByTestId('confirm-password-input'), 'Password123');

      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('email-error')).toHaveTextContent('Email already exists');
      
      // Other fields should not have errors
      expect(screen.queryByTestId('name-error')).not.toBeInTheDocument();
      expect(screen.queryByTestId('password-error')).not.toBeInTheDocument();
    });
  });

  describe('Network Error Scenarios', () => {
    it('should handle network connection errors', async () => {
      const user = userEvent.setup();
      
      const networkError = createApiErrorFromResponse(0, 'Network request failed');
      const mockSubmit = jest.fn().mockRejectedValue(networkError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      // Fill and submit form
      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.type(screen.getByTestId('email-input'), 'john@example.com');
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/network request failed/i);
    });

    it('should handle timeout errors', async () => {
      const user = userEvent.setup();
      
      const timeoutError = createApiErrorFromResponse(408, 'Request timeout');
      const mockSubmit = jest.fn().mockRejectedValue(timeoutError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/request timeout/i);
    });

    it('should handle server errors', async () => {
      const user = userEvent.setup();
      
      const serverError = createApiErrorFromResponse(500, 'Internal server error');
      const mockSubmit = jest.fn().mockRejectedValue(serverError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/internal server error/i);
    });
  });

  describe('Authentication and Authorization Errors', () => {
    it('should handle authentication errors', async () => {
      const user = userEvent.setup();
      
      const authError = createApiErrorFromResponse(401, 'Authentication required');
      const mockSubmit = jest.fn().mockRejectedValue(authError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/authentication required/i);
    });

    it('should handle authorization errors', async () => {
      const user = userEvent.setup();
      
      const authzError = createApiErrorFromResponse(403, 'Insufficient permissions');
      const mockSubmit = jest.fn().mockRejectedValue(authzError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('submit-error')).toHaveTextContent(/insufficient permissions/i);
    });
  });

  describe('File Upload Error Scenarios', () => {
    it('should handle file upload size limit errors', async () => {
      const user = userEvent.setup();
      
      const sizeError = createApiErrorFromResponse(413, 'File too large');
      const mockUpload = jest.fn().mockRejectedValue(sizeError);

      render(
        <TestProviders >
          <TestFileUploadForm onUpload={mockUpload} />
        </TestProviders>
      );

      // Create a mock file
      const file = new File(['content'], 'large-file.jpg', { type: 'image/jpeg' });
      
      const fileInput = screen.getByTestId('file-input');
      await user.upload(fileInput, file);

      expect(screen.getByTestId('file-info')).toBeInTheDocument();
      expect(screen.getByText('Selected: large-file.jpg')).toBeInTheDocument();

      const uploadButton = screen.getByTestId('upload-button');
      await user.click(uploadButton);

      await waitFor(() => {
        expect(screen.getByTestId('upload-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('upload-error')).toHaveTextContent(/file too large/i);
    });

    it('should handle unsupported file type errors', async () => {
      const user = userEvent.setup();
      
      const typeError = createApiErrorFromResponse(415, 'Unsupported file type');
      const mockUpload = jest.fn().mockRejectedValue(typeError);

      render(
        <TestProviders >
          <TestFileUploadForm onUpload={mockUpload} />
        </TestProviders>
      );

      const file = new File(['content'], 'document.exe', { type: 'application/octet-stream' });
      
      const fileInput = screen.getByTestId('file-input');
      await user.upload(fileInput, file);

      const uploadButton = screen.getByTestId('upload-button');
      await user.click(uploadButton);

      await waitFor(() => {
        expect(screen.getByTestId('upload-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('upload-error')).toHaveTextContent(/unsupported file type/i);
    });

    it('should handle upload network interruption', async () => {
      const user = userEvent.setup();
      
      const networkError = createApiErrorFromResponse(0, 'Network connection lost');
      const mockUpload = jest.fn().mockRejectedValue(networkError);

      render(
        <TestProviders >
          <TestFileUploadForm onUpload={mockUpload} />
        </TestProviders>
      );

      const file = new File(['content'], 'photo.jpg', { type: 'image/jpeg' });
      
      const fileInput = screen.getByTestId('file-input');
      await user.upload(fileInput, file);

      const uploadButton = screen.getByTestId('upload-button');
      await user.click(uploadButton);

      // Should show progress initially
      expect(screen.getByTestId('upload-progress')).toBeInTheDocument();

      await waitFor(() => {
        expect(screen.getByTestId('upload-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('upload-error')).toHaveTextContent(/network connection lost/i);
      
      // Progress should be reset
      expect(screen.queryByTestId('upload-progress')).not.toBeInTheDocument();
    });
  });

  describe('Form State Management During Errors', () => {
    it('should maintain form state during validation errors', async () => {
      const user = userEvent.setup();
      
      const validationError = {
        message: 'Validation failed',
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        fieldErrors: {
          email: ['Invalid email format'],
        },
      };

      const mockSubmit = jest.fn().mockRejectedValue(validationError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      // Fill form
      await user.type(screen.getByTestId('name-input'), 'John Doe');
      await user.type(screen.getByTestId('email-input'), 'invalid-email');
      await user.type(screen.getByTestId('password-input'), 'Password123');

      // Submit form
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
      });

      // Form values should be preserved
      expect(screen.getByTestId('name-input')).toHaveValue('John Doe');
      expect(screen.getByTestId('email-input')).toHaveValue('invalid-email');
      expect(screen.getByTestId('password-input')).toHaveValue('Password123');
    });

    it('should disable form during submission', async () => {
      const user = userEvent.setup();
      
      // Mock a slow submission
      const mockSubmit = jest.fn().mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 1000))
      );

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      // Button should be disabled and show loading state
      expect(submitButton).toBeDisabled();
      expect(submitButton).toHaveTextContent('Submitting...');
    });

    it('should re-enable form after error', async () => {
      const user = userEvent.setup();
      
      const serverError = createApiErrorFromResponse(500, 'Server error');
      const mockSubmit = jest.fn().mockRejectedValue(serverError);

      render(
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.type(screen.getByTestId('name-input'), 'John Doe');
      
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByTestId('submit-error')).toBeInTheDocument();
      });

      // Form should be re-enabled after error
      expect(submitButton).not.toBeDisabled();
      expect(submitButton).toHaveTextContent('Submit');
    });
  });

  describe('Accessibility During Error States', () => {
    it('should properly announce errors to screen readers', async () => {
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
        <TestProviders >
          <TestUserForm onSubmit={mockSubmit} />
        </TestProviders>
      );

      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
        expect(screen.getByTestId('email-error')).toBeInTheDocument();
      });

      // Error messages should have role="alert" for screen readers
      expect(screen.getByTestId('name-error')).toHaveAttribute('role', 'alert');
      expect(screen.getByTestId('email-error')).toHaveAttribute('role', 'alert');

      // Input fields should have aria-invalid attribute
      expect(screen.getByTestId('name-input')).toHaveAttribute('aria-invalid', 'true');
      expect(screen.getByTestId('email-input')).toHaveAttribute('aria-invalid', 'true');
      
      // Fields without errors should not have aria-invalid
      expect(screen.getByTestId('password-input')).toHaveAttribute('aria-invalid', 'false');
    });
  });
});