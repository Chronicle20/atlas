/**
 * @jest-environment jsdom
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateTenantDialog } from '../CreateTenantDialog';
import { templatesService } from '@/services/api';
import type { TemplateOption } from '@/services/api';

// Mock the services
jest.mock('@/services/api', () => ({
  templatesService: {
    getTemplateOptions: jest.fn(),
  },
  onboardingService: {
    onboardTenantByVersion: jest.fn(),
  },
  ConfigurationCreationError: class ConfigurationCreationError extends Error {
    tenantId: string;
    constructor(message: string, tenantId: string) {
      super(message);
      this.name = 'ConfigurationCreationError';
      this.tenantId = tenantId;
    }
  },
  TenantCreationError: class TenantCreationError extends Error {
    constructor(message: string) {
      super(message);
      this.name = 'TenantCreationError';
    }
  },
  TemplateNotFoundError: class TemplateNotFoundError extends Error {
    constructor() {
      super('Template not found');
      this.name = 'TemplateNotFoundError';
    }
  },
}));

// Mock sonner toast
jest.mock('sonner', () => ({
  toast: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

const mockTemplatesService = templatesService as jest.Mocked<typeof templatesService>;
// Note: onboardingService is mocked via jest.mock but we use integration tests for submission flow

describe('CreateTenantDialog', () => {
  const mockTemplateOptions: TemplateOption[] = [
    {
      id: 'template-1',
      attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 },
    },
    {
      id: 'template-2',
      attributes: { region: 'GMS', majorVersion: 95, minorVersion: 1 },
    },
    {
      id: 'template-3',
      attributes: { region: 'JMS', majorVersion: 185, minorVersion: 1 },
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    mockTemplatesService.getTemplateOptions.mockResolvedValue(mockTemplateOptions);
  });

  const defaultProps = {
    open: true,
    onOpenChange: jest.fn(),
    onSuccess: jest.fn(),
  };

  it('renders dialog with form fields when open', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Create New Tenant')).toBeInTheDocument();
    });

    expect(screen.getByPlaceholderText(/enter tenant name/i)).toBeInTheDocument();
    expect(screen.getByText(/region/i)).toBeInTheDocument();
  });

  it('fetches template options when dialog opens', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(mockTemplatesService.getTemplateOptions).toHaveBeenCalled();
    });
  });

  it('shows loading skeletons while fetching template options', () => {
    // Make the promise hang to show loading state
    mockTemplatesService.getTemplateOptions.mockImplementation(
      () => new Promise(() => {})
    );

    render(<CreateTenantDialog {...defaultProps} />);

    // Dialog should be rendering in loading state
    expect(screen.getByText('Create New Tenant')).toBeInTheDocument();
  });

  it('shows error message when template options fail to load', async () => {
    mockTemplatesService.getTemplateOptions.mockRejectedValue(new Error('Network error'));

    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load template options/i)).toBeInTheDocument();
    });
  });

  it('shows message when no templates available', async () => {
    mockTemplatesService.getTemplateOptions.mockResolvedValue([]);

    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText(/no templates available/i)).toBeInTheDocument();
    });
  });

  it('populates region dropdown with unique regions', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(mockTemplatesService.getTemplateOptions).toHaveBeenCalled();
    });

    // The hidden native select should contain the options
    // GMS and JMS should be available as options from the mock data
    await waitFor(() => {
      expect(screen.getByText('GMS')).toBeInTheDocument();
      expect(screen.getByText('JMS')).toBeInTheDocument();
    });
  });

  it('shows Major Version label when template options are loaded', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    // Wait for loading to complete
    await waitFor(() => {
      expect(screen.getByText('Major Version')).toBeInTheDocument();
    });
  });

  it('shows Minor Version label when template options are loaded', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    // Wait for loading to complete
    await waitFor(() => {
      expect(screen.getByText('Minor Version')).toBeInTheDocument();
    });
  });

  it('calls onOpenChange when cancel button is clicked', async () => {
    const onOpenChange = jest.fn();
    render(<CreateTenantDialog {...defaultProps} onOpenChange={onOpenChange} />);

    await waitFor(() => {
      expect(mockTemplatesService.getTemplateOptions).toHaveBeenCalled();
    });

    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    fireEvent.click(cancelButton);

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('allows typing in tenant name field', async () => {
    render(<CreateTenantDialog {...defaultProps} />);

    await waitFor(() => {
      expect(mockTemplatesService.getTemplateOptions).toHaveBeenCalled();
    });

    // Fill in name
    const nameInput = screen.getByPlaceholderText(/enter tenant name/i);
    await userEvent.type(nameInput, 'Test Tenant');
    expect((nameInput as HTMLInputElement).value).toBe('Test Tenant');
  });

  it('does not render when closed', () => {
    render(<CreateTenantDialog {...defaultProps} open={false} />);

    expect(screen.queryByText('Create New Tenant')).not.toBeInTheDocument();
  });
});
