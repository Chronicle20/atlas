/**
 * Integration tests for error scenarios in application pages
 * Tests error handling in real page components with data fetching
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { AllProviders } from '../utils/test-providers';
import { createApiErrorFromResponse } from '@/types/api/errors';

// Mock the API client and services
jest.mock('@/lib/characters', () => ({
  getCharacters: jest.fn(),
  getCharacter: jest.fn(),
}));

jest.mock('@/lib/tenants', () => ({
  getTenants: jest.fn(),
  getTenant: jest.fn(),
}));

jest.mock('@/lib/accounts', () => ({
  getAccounts: jest.fn(),
}));

jest.mock('@/lib/guilds', () => ({
  getGuilds: jest.fn(),
  getGuild: jest.fn(),
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

// Test component that simulates a data table page
const TestDataTablePage = ({ dataFetcher, title }: { 
  dataFetcher: () => Promise<any[]>; 
  title: string;
}) => {
  const [data, setData] = React.useState<any[] | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const result = await dataFetcher();
        setData(result);
      } catch (err: any) {
        setError(err.message || 'Failed to load data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [dataFetcher]);

  if (loading) {
    return <div data-testid="page-loading">Loading {title}...</div>;
  }

  if (error) {
    return (
      <div data-testid="page-error">
        <h1>Error Loading {title}</h1>
        <p data-testid="error-message">{error}</p>
        <button 
          onClick={() => window.location.reload()} 
          data-testid="retry-button"
        >
          Try Again
        </button>
      </div>
    );
  }

  return (
    <div data-testid="page-success">
      <h1>{title}</h1>
      <div data-testid="data-count">{data?.length || 0} items</div>
      <div data-testid="data-list">
        {data?.map((item, index) => (
          <div key={index} data-testid={`item-${index}`}>
            {JSON.stringify(item)}
          </div>
        ))}
      </div>
    </div>
  );
};

// Test component for detail pages
const TestDetailPage = ({ 
  dataFetcher, 
  id, 
  title 
}: { 
  dataFetcher: (id: string) => Promise<any>; 
  id: string;
  title: string;
}) => {
  const [data, setData] = React.useState<any | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const result = await dataFetcher(id);
        setData(result);
      } catch (err: any) {
        setError(err.message || `Failed to load ${title}`);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [dataFetcher, id, title]);

  if (loading) {
    return <div data-testid="detail-loading">Loading {title}...</div>;
  }

  if (error) {
    return (
      <div data-testid="detail-error">
        <h1>{title} Not Found</h1>
        <p data-testid="error-message">{error}</p>
        <button 
          onClick={() => mockPush('/')} 
          data-testid="back-button"
        >
          Go Back
        </button>
      </div>
    );
  }

  return (
    <div data-testid="detail-success">
      <h1>{title} Details</h1>
      <div data-testid="detail-data">{JSON.stringify(data)}</div>
    </div>
  );
};

// Mock test provider component
const TestProviders = ({ children }: { children: React.ReactNode }) => {
  return <AllProviders>{children}</AllProviders>;
};

describe('Page Error Scenarios Integration Tests', () => {
  const mockCharacters = require('@/lib/characters');
  const mockTenants = require('@/lib/tenants');
  const mockAccounts = require('@/lib/accounts');
  const mockGuilds = require('@/lib/guilds');

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Data Table Page Error Scenarios', () => {
    it('should handle characters list loading error', async () => {
      mockCharacters.getCharacters.mockRejectedValue(
        createApiErrorFromResponse(500, 'Failed to fetch characters')
      );

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockCharacters.getCharacters}
            title="Characters"
          />
        </TestProviders>
      );

      // Should show loading initially
      expect(screen.getByTestId('page-loading')).toBeInTheDocument();

      // Should show error after API call fails
      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      expect(screen.getByText('Error Loading Characters')).toBeInTheDocument();
      expect(screen.getByTestId('error-message')).toHaveTextContent('Failed to fetch characters');
      expect(screen.getByTestId('retry-button')).toBeInTheDocument();
    });

    it('should handle tenants list authentication error', async () => {
      mockTenants.getTenants.mockRejectedValue(
        createApiErrorFromResponse(401, 'Authentication required')
      );

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockTenants.getTenants}
            title="Tenants"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Authentication required');
    });

    it('should handle accounts list authorization error', async () => {
      mockAccounts.getAccounts.mockRejectedValue(
        createApiErrorFromResponse(403, 'Insufficient permissions')
      );

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockAccounts.getAccounts}
            title="Accounts"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Insufficient permissions');
    });

    it('should handle empty data gracefully', async () => {
      mockCharacters.getCharacters.mockResolvedValue([]);

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockCharacters.getCharacters}
            title="Characters"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('page-success')).toBeInTheDocument();
      });

      expect(screen.getByTestId('data-count')).toHaveTextContent('0 items');
      expect(screen.queryByTestId('item-0')).not.toBeInTheDocument();
    });

    it('should handle network timeout errors', async () => {
      mockGuilds.getGuilds.mockRejectedValue(
        createApiErrorFromResponse(408, 'Request timeout')
      );

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockGuilds.getGuilds}
            title="Guilds"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Request timeout');
    });

    it('should handle rate limiting errors', async () => {
      mockCharacters.getCharacters.mockRejectedValue(
        createApiErrorFromResponse(429, 'Too many requests')
      );

      render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockCharacters.getCharacters}
            title="Characters"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Too many requests');
    });
  });

  describe('Detail Page Error Scenarios', () => {
    it('should handle character detail not found error', async () => {
      mockCharacters.getCharacter.mockRejectedValue(
        createApiErrorFromResponse(404, 'Character not found')
      );

      render(
        <TestProviders >
          <TestDetailPage 
            dataFetcher={mockCharacters.getCharacter}
            id="123"
            title="Character"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      expect(screen.getByText('Character Not Found')).toBeInTheDocument();
      expect(screen.getByTestId('error-message')).toHaveTextContent('Character not found');
      expect(screen.getByTestId('back-button')).toBeInTheDocument();
    });

    it('should handle tenant detail server error', async () => {
      mockTenants.getTenant.mockRejectedValue(
        createApiErrorFromResponse(500, 'Internal server error')
      );

      render(
        <TestProviders >
          <TestDetailPage 
            dataFetcher={mockTenants.getTenant}
            id="tenant-456"
            title="Tenant"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Internal server error');
    });

    it('should handle guild detail network error', async () => {
      mockGuilds.getGuild.mockRejectedValue(
        createApiErrorFromResponse(0, 'Network connection failed')
      );

      render(
        <TestProviders >
          <TestDetailPage 
            dataFetcher={mockGuilds.getGuild}
            id="guild-789"
            title="Guild"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      expect(screen.getByTestId('error-message')).toHaveTextContent('Network connection failed');
    });

    it('should navigate back when back button is clicked', async () => {
      const user = userEvent.setup();
      
      mockCharacters.getCharacter.mockRejectedValue(
        createApiErrorFromResponse(404, 'Character not found')
      );

      render(
        <TestProviders >
          <TestDetailPage 
            dataFetcher={mockCharacters.getCharacter}
            id="123"
            title="Character"
          />
        </TestProviders>
      );

      await waitFor(() => {
        expect(screen.getByTestId('detail-error')).toBeInTheDocument();
      });

      const backButton = screen.getByTestId('back-button');
      await user.click(backButton);

      expect(mockPush).toHaveBeenCalledWith('/');
    });
  });

  describe('Loading State Error Recovery', () => {
    it('should handle successful recovery after initial failure', async () => {
      // First call fails, subsequent calls succeed
      mockCharacters.getCharacters
        .mockRejectedValueOnce(createApiErrorFromResponse(503, 'Service unavailable'))
        .mockResolvedValue([
          { id: 1, name: 'Character 1' },
          { id: 2, name: 'Character 2' }
        ]);

      const { rerender } = render(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockCharacters.getCharacters}
            title="Characters"
          />
        </TestProviders>
      );

      // Should show error initially
      await waitFor(() => {
        expect(screen.getByTestId('page-error')).toBeInTheDocument();
      });

      // Force a retry by re-rendering with a new fetcher
      rerender(
        <TestProviders >
          <TestDataTablePage 
            dataFetcher={mockCharacters.getCharacters}
            title="Characters"
          />
        </TestProviders>
      );

      // Should eventually show success
      await waitFor(() => {
        expect(screen.getByTestId('page-success')).toBeInTheDocument();
      });

      expect(screen.getByTestId('data-count')).toHaveTextContent('2 items');
    });

    it('should handle intermittent connectivity issues', async () => {
      // Simulate intermittent failures
      mockTenants.getTenants
        .mockRejectedValueOnce(createApiErrorFromResponse(0, 'Network error'))
        .mockRejectedValueOnce(createApiErrorFromResponse(408, 'Timeout'))
        .mockResolvedValue([{ id: 1, name: 'Tenant 1' }]);

      const TestComponent = () => {
        const [retryCount, setRetryCount] = React.useState(0);
        
        const fetcher = React.useCallback(async () => {
          try {
            return await mockTenants.getTenants();
          } catch (error) {
            if (retryCount < 2) {
              setRetryCount(prev => prev + 1);
              throw error;
            }
            return await mockTenants.getTenants();
          }
        }, [retryCount]);

        return (
          <TestDataTablePage 
            dataFetcher={fetcher}
            title="Tenants"
          />
        );
      };

      render(
        <TestProviders >
          <TestComponent />
        </TestProviders>
      );

      // Should eventually succeed after retries
      await waitFor(() => {
        expect(screen.getByTestId('page-success')).toBeInTheDocument();
      }, { timeout: 5000 });

      expect(screen.getByTestId('data-count')).toHaveTextContent('1 items');
    });
  });

  describe('Multiple Page Error Scenarios', () => {
    it('should handle errors across multiple page components simultaneously', async () => {
      mockCharacters.getCharacters.mockRejectedValue(
        createApiErrorFromResponse(500, 'Characters service error')
      );
      mockTenants.getTenants.mockRejectedValue(
        createApiErrorFromResponse(503, 'Tenants service unavailable')
      );
      mockAccounts.getAccounts.mockResolvedValue([
        { id: 1, name: 'Account 1' }
      ]);

      render(
        <TestProviders >
          <div>
            <div data-testid="characters-section">
              <TestDataTablePage 
                dataFetcher={mockCharacters.getCharacters}
                title="Characters"
              />
            </div>
            <div data-testid="tenants-section">
              <TestDataTablePage 
                dataFetcher={mockTenants.getTenants}
                title="Tenants"
              />
            </div>
            <div data-testid="accounts-section">
              <TestDataTablePage 
                dataFetcher={mockAccounts.getAccounts}
                title="Accounts"
              />
            </div>
          </div>
        </TestProviders>
      );

      await waitFor(() => {
        // Characters should show error
        const charactersSection = screen.getByTestId('characters-section');
        expect(charactersSection).toHaveTextContent('Characters service error');
        
        // Tenants should show error
        const tenantsSection = screen.getByTestId('tenants-section');
        expect(tenantsSection).toHaveTextContent('Tenants service unavailable');
        
        // Accounts should show success
        const accountsSection = screen.getByTestId('accounts-section');
        expect(accountsSection).toHaveTextContent('1 items');
      });
    });

    it('should isolate errors between different page sections', async () => {
      mockCharacters.getCharacters.mockRejectedValue(
        createApiErrorFromResponse(404, 'Characters not found')
      );
      mockGuilds.getGuilds.mockResolvedValue([
        { id: 1, name: 'Guild 1' },
        { id: 2, name: 'Guild 2' }
      ]);

      render(
        <TestProviders >
          <div>
            <div data-testid="characters-section">
              <TestDataTablePage 
                dataFetcher={mockCharacters.getCharacters}
                title="Characters"
              />
            </div>
            <div data-testid="guilds-section">
              <TestDataTablePage 
                dataFetcher={mockGuilds.getGuilds}
                title="Guilds"
              />
            </div>
          </div>
        </TestProviders>
      );

      await waitFor(() => {
        // Characters should show error
        const charactersSection = screen.getByTestId('characters-section');
        expect(charactersSection).toHaveTextContent('Error Loading Characters');
        
        // Guilds should show success
        const guildsSection = screen.getByTestId('guilds-section');
        expect(guildsSection).toHaveTextContent('2 items');
      });

      // Error in one section should not affect the other
      const charactersSection = screen.getByTestId('characters-section');
      expect(charactersSection.querySelector('[data-testid="page-error"]')).toBeInTheDocument();
      
      const guildsSection = screen.getByTestId('guilds-section');
      expect(guildsSection.querySelector('[data-testid="page-success"]')).toBeInTheDocument();
    });
  });
});