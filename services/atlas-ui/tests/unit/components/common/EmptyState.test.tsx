import { render, screen, fireEvent } from '@testing-library/react';
import { EmptyState } from '@/components/common/EmptyState';

describe('EmptyState', () => {
  it('renders with required title', () => {
    const title = 'No items found';
    render(<EmptyState title={title} />);
    
    const emptyState = screen.getByTestId('empty-state');
    expect(emptyState).toBeInTheDocument();
    expect(screen.getByText(title)).toBeInTheDocument();
  });

  it('renders with description', () => {
    const title = 'No items found';
    const description = 'There are no items to display at this time.';
    render(<EmptyState title={title} description={description} />);
    
    expect(screen.getByText(description)).toBeInTheDocument();
  });

  it('renders without description when not provided', () => {
    const title = 'No items found';
    render(<EmptyState title={title} />);
    
    const emptyState = screen.getByTestId('empty-state');
    const description = emptyState.querySelector('p');
    expect(description).not.toBeInTheDocument();
  });

  it('renders with icon', () => {
    const title = 'No items found';
    const icon = <div data-testid="custom-icon">📦</div>;
    render(<EmptyState title={title} icon={icon} />);
    
    expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
  });

  it('renders without icon when not provided', () => {
    const title = 'No items found';
    render(<EmptyState title={title} />);
    
    const emptyState = screen.getByTestId('empty-state');
    const iconContainer = emptyState.querySelector('.mb-4.text-muted-foreground');
    expect(iconContainer).not.toBeInTheDocument();
  });

  it('renders with action button', () => {
    const title = 'No items found';
    const action = {
      label: 'Add Item',
      onClick: jest.fn(),
    };
    render(<EmptyState title={title} action={action} />);
    
    const actionButton = screen.getByText('Add Item');
    expect(actionButton).toBeInTheDocument();
  });

  it('calls action onClick when button is clicked', () => {
    const title = 'No items found';
    const mockOnClick = jest.fn();
    const action = {
      label: 'Add Item',
      onClick: mockOnClick,
    };
    render(<EmptyState title={title} action={action} />);
    
    const actionButton = screen.getByText('Add Item');
    fireEvent.click(actionButton);
    
    expect(mockOnClick).toHaveBeenCalledTimes(1);
  });

  it('renders without action button when not provided', () => {
    const title = 'No items found';
    render(<EmptyState title={title} />);
    
    const emptyState = screen.getByTestId('empty-state');
    const button = emptyState.querySelector('button');
    expect(button).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    const title = 'No items found';
    const customClass = 'my-custom-empty-state';
    render(<EmptyState title={title} className={customClass} />);
    
    const emptyState = screen.getByTestId('empty-state');
    expect(emptyState).toHaveClass(customClass);
  });

  it('renders all props together', () => {
    const title = 'No items found';
    const description = 'Try adding some items.';
    const icon = <div data-testid="custom-icon">📦</div>;
    const action = {
      label: 'Add Item',
      onClick: jest.fn(),
    };
    
    render(
      <EmptyState 
        title={title} 
        description={description} 
        icon={icon} 
        action={action} 
      />
    );
    
    expect(screen.getByText(title)).toBeInTheDocument();
    expect(screen.getByText(description)).toBeInTheDocument();
    expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    expect(screen.getByText('Add Item')).toBeInTheDocument();
  });
});