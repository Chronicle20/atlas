import { render, screen } from '@testing-library/react';
import { useForm } from 'react-hook-form';
import { FormField } from '@/components/common/FormField';
import { Form } from '@/components/ui/form';

// Helper component that wraps FormField with react-hook-form
function TestFormField(props: any) {
  const form = useForm({
    defaultValues: {
      testField: '',
    },
  });

  return (
    <Form {...form}>
      <form>
        <FormField
          control={form.control}
          name="testField"
          {...props}
        />
      </form>
    </Form>
  );
}

describe('FormField', () => {
  it('renders with label', () => {
    const label = 'Test Field';
    render(<TestFormField label={label} />);
    
    expect(screen.getByTestId('form-field')).toBeInTheDocument();
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it('renders text input by default', () => {
    render(<TestFormField label="Test Field" />);
    
    const input = screen.getByRole('textbox');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'text');
  });

  it('renders number input when type is number', () => {
    render(<TestFormField label="Test Field" type="number" />);
    
    const input = screen.getByRole('spinbutton');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'number');
  });

  it('renders email input when type is email', () => {
    render(<TestFormField label="Test Field" type="email" />);
    
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('type', 'email');
  });

  it('renders password input when type is password', () => {
    render(<TestFormField label="Test Field" type="password" />);
    
    const input = screen.getByLabelText('Test Field');
    expect(input).toHaveAttribute('type', 'password');
  });

  it('renders with placeholder', () => {
    const placeholder = 'Enter value';
    render(<TestFormField label="Test Field" placeholder={placeholder} />);
    
    const input = screen.getByPlaceholderText(placeholder);
    expect(input).toBeInTheDocument();
  });

  it('renders with description', () => {
    const description = 'This is a helpful description';
    render(<TestFormField label="Test Field" description={description} />);
    
    expect(screen.getByText(description)).toBeInTheDocument();
  });

  it('renders custom content with render prop', () => {
    const customContent = 'Custom content';
    render(
      <TestFormField 
        label="Test Field" 
        render={() => <div data-testid="custom-content">{customContent}</div>}
      />
    );
    
    expect(screen.getByTestId('custom-content')).toBeInTheDocument();
    expect(screen.getByText(customContent)).toBeInTheDocument();
  });

  it('applies disabled attribute', () => {
    render(<TestFormField label="Test Field" disabled />);
    
    const input = screen.getByRole('textbox');
    expect(input).toBeDisabled();
  });

  it('applies min and max for number inputs', () => {
    render(<TestFormField label="Test Field" type="number" min={0} max={100} />);
    
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('min', '0');
    expect(input).toHaveAttribute('max', '100');
  });

  it('applies step for number inputs', () => {
    render(<TestFormField label="Test Field" type="number" step={0.5} />);
    
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('step', '0.5');
  });
});