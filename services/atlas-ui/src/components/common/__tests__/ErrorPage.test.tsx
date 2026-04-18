import { vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ErrorPage, Error404Page, Error500Page } from '../ErrorPage';

function renderWithRouter(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('ErrorPage', () => {
  it('renders 404 error page correctly', () => {
    renderWithRouter(<ErrorPage statusCode={404} />);

    expect(screen.getByText('Page Not Found')).toBeInTheDocument();
    expect(screen.getByText(/The page you.*re looking for doesn.*t exist or has been moved/)).toBeInTheDocument();
    expect(screen.getByText('Error Code: 404')).toBeInTheDocument();
  });

  it('renders 500 error page correctly', () => {
    renderWithRouter(<ErrorPage statusCode={500} />);

    expect(screen.getByText('Internal Server Error')).toBeInTheDocument();
    expect(screen.getByText(/unexpected error occurred on the server/)).toBeInTheDocument();
    expect(screen.getByText('Error Code: 500')).toBeInTheDocument();
  });

  it('renders custom title and message', () => {
    renderWithRouter(
      <ErrorPage
        statusCode={404}
        title="Custom Title"
        message="Custom message"
      />
    );

    expect(screen.getByText('Custom Title')).toBeInTheDocument();
    expect(screen.getByText('Custom message')).toBeInTheDocument();
  });

  it('shows retry button when enabled', () => {
    const mockRetry = vi.fn();
    renderWithRouter(
      <ErrorPage
        statusCode={500}
        showRetryButton={true}
        onRetry={mockRetry}
      />
    );

    const retryButton = screen.getByRole('button', { name: /try again/i });
    expect(retryButton).toBeInTheDocument();

    fireEvent.click(retryButton);
    expect(mockRetry).toHaveBeenCalledTimes(1);
  });

  it('shows go back button when enabled', () => {
    const mockBack = vi.fn();
    Object.defineProperty(window, 'history', {
      value: { back: mockBack },
      writable: true,
    });

    renderWithRouter(<ErrorPage statusCode={404} showBackButton={true} />);

    const backButton = screen.getByRole('button', { name: /go back/i });
    expect(backButton).toBeInTheDocument();

    fireEvent.click(backButton);
    expect(mockBack).toHaveBeenCalledTimes(1);
  });

  it('shows home button by default', () => {
    renderWithRouter(<ErrorPage statusCode={404} />);

    const homeLink = screen.getByRole('link', { name: /go home/i });
    expect(homeLink).toBeInTheDocument();
    expect(homeLink).toHaveAttribute('href', '/');
  });
});

describe('Pre-configured error pages', () => {
  it('renders Error404Page with correct defaults', () => {
    renderWithRouter(<Error404Page />);

    expect(screen.getByText('Page Not Found')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /go home/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument();
  });

  it('renders Error500Page with correct defaults', () => {
    renderWithRouter(<Error500Page />);

    expect(screen.getByText('Internal Server Error')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /go home/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
  });
});
