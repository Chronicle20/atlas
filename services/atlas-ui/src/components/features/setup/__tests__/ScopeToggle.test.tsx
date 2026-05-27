import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ScopeToggle } from '@/components/features/setup/ScopeToggle';

describe('ScopeToggle', () => {
  it('renders both options and reflects the active value', () => {
    render(<ScopeToggle value="tenant" onChange={() => {}} region="GMS" version="83.1" />);
    expect(screen.getByRole('radio', { name: /this tenant/i })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    expect(screen.getByRole('radio', { name: /canonical/i })).toHaveAttribute(
      'aria-checked',
      'false',
    );
  });

  it('calls onChange when canonical is clicked', () => {
    const onChange = vi.fn();
    render(<ScopeToggle value="tenant" onChange={onChange} region="GMS" version="83.1" />);
    fireEvent.click(screen.getByRole('radio', { name: /canonical/i }));
    expect(onChange).toHaveBeenCalledWith('shared');
  });

  it('calls onChange when this-tenant is clicked', () => {
    const onChange = vi.fn();
    render(<ScopeToggle value="shared" onChange={onChange} region="GMS" version="83.1" />);
    fireEvent.click(screen.getByRole('radio', { name: /this tenant/i }));
    expect(onChange).toHaveBeenCalledWith('tenant');
  });

  it('shows the warning text only when shared is selected', () => {
    const { rerender } = render(
      <ScopeToggle value="tenant" onChange={() => {}} region="GMS" version="83.1" />,
    );
    expect(
      screen.queryByText(/replace the shared canonical baseline/i),
    ).not.toBeInTheDocument();
    rerender(<ScopeToggle value="shared" onChange={() => {}} region="GMS" version="83.1" />);
    expect(
      screen.getByText(/replace the shared canonical baseline.*GMS.*83\.1/i),
    ).toBeInTheDocument();
  });

  it('uses semantic destructive token (not raw amber scales) for the warning', () => {
    render(<ScopeToggle value="shared" onChange={() => {}} region="GMS" version="83.1" />);
    const warning = screen.getByText(/replace the shared canonical baseline/i);
    expect(warning.className).toContain('text-destructive');
    expect(warning.className).not.toMatch(/text-amber-/);
  });
});
