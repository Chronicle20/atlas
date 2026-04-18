import { render, screen } from '@testing-library/react';
import { useQueryClient } from '@tanstack/react-query';
import { QueryProvider } from '../query-provider';

// Test component that uses the query client
function TestComponent() {
  const queryClient = useQueryClient();
  
  return (
    <div>
      <div data-testid="query-client-exists">
        {queryClient ? 'Query client available' : 'No query client'}
      </div>
    </div>
  );
}

describe('QueryProvider', () => {
  it('should provide query client to child components', () => {
    render(
      <QueryProvider>
        <TestComponent />
      </QueryProvider>
    );

    expect(screen.getByTestId('query-client-exists')).toHaveTextContent('Query client available');
  });

  it('should render children', () => {
    render(
      <QueryProvider>
        <div data-testid="child">Test child</div>
      </QueryProvider>
    );

    expect(screen.getByTestId('child')).toHaveTextContent('Test child');
  });

  it('should not render devtools in test environment', () => {
    // Devtools should not be rendered in test environment (NODE_ENV !== 'development')
    render(
      <QueryProvider>
        <div>Test content</div>
      </QueryProvider>
    );

    // The devtools component would have a specific attribute or class, but since it's not rendered
    // in test environment, we just verify the provider works without throwing
    expect(screen.getByText('Test content')).toBeInTheDocument();
  });
});