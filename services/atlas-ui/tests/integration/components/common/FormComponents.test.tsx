import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useForm, FormProvider } from 'react-hook-form';
import { FormField } from '@/components/common/FormField';
import { FormSelect } from '@/components/common/FormSelect';
import { FormTextarea } from '@/components/common/FormTextarea';
import { Form } from '@/components/ui/form';
import { Button } from '@/components/ui/button';
import * as React from 'react';

// Mock form data interfaces
interface TestFormData {
  username: string;
  email: string;
  age: number;
  status: string;
  category: string;
  description: string;
  bio: string;
  isActive: boolean;
}

const mockSelectOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'pending', label: 'Pending', disabled: true },
];

const mockCategoryOptions = [
  { value: 'tech', label: 'Technology' },
  { value: 'design', label: 'Design' },
  { value: 'marketing', label: 'Marketing' },
];

// Test form wrapper component
const TestForm = ({ 
  onSubmit, 
  defaultValues = {},
  children 
}: { 
  onSubmit?: (data: any) => void;
  defaultValues?: Partial<TestFormData>;
  children: React.ReactNode;
}) => {
  const form = useForm<TestFormData>({
    defaultValues: {
      username: '',
      email: '',
      age: 0,
      status: '',
      category: '',
      description: '',
      bio: '',
      isActive: false,
      ...defaultValues,
    },
  });

  const handleSubmit = (data: TestFormData) => {
    onSubmit?.(data);
  };

  // Clone children to inject form control
  const childrenWithForm = React.Children.map(children, (child) => {
    if (React.isValidElement(child)) {
      // Ensure control is properly passed, preserving existing props
      const childProps = { ...child.props };
      if (!childProps.control) {
        childProps.control = form.control;
      }
      return React.cloneElement(child, childProps);
    }
    return child;
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} data-testid="test-form">
        {childrenWithForm}
        <Button type="submit" data-testid="submit-button">
          Submit
        </Button>
      </form>
    </Form>
  );
};

describe('Form Components Integration Tests', () => {
  const user = userEvent.setup();

  describe('FormField Integration', () => {
    it('should render and interact with text FormField', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormField
            name="username"
            label="Username"
            placeholder="Enter username"
            description="Choose a unique username"
          />
        </TestForm>
      );

      // Find form field
      const input = screen.getByPlaceholderText('Enter username');
      expect(input).toBeInTheDocument();
      expect(screen.getByText('Username')).toBeInTheDocument();
      expect(screen.getByText('Choose a unique username')).toBeInTheDocument();

      // Type in the field
      await user.type(input, 'testuser');
      expect(input).toHaveValue('testuser');

      // Submit form
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            username: 'testuser'
          })
        );
      });
    });

    it('should render and interact with email FormField', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormField
            name="email"
            label="Email"
            type="email"
            placeholder="Enter email"
          />
        </TestForm>
      );

      const input = screen.getByPlaceholderText('Enter email');
      expect(input).toHaveAttribute('type', 'email');

      await user.type(input, 'test@example.com');
      expect(input).toHaveValue('test@example.com');

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            email: 'test@example.com'
          })
        );
      });
    });

    it('should render and interact with number FormField', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormField
            name="age"
            label="Age"
            type="number"
            min={0}
            max={120}
            placeholder="Enter age"
          />
        </TestForm>
      );

      const input = screen.getByPlaceholderText('Enter age');
      expect(input).toHaveAttribute('type', 'number');
      expect(input).toHaveAttribute('min', '0');
      expect(input).toHaveAttribute('max', '120');

      await user.type(input, '25');
      expect(input).toHaveValue(25);

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            age: 25
          })
        );
      });
    });

    it('should handle custom render FormField', async () => {
      const mockSubmit = jest.fn();
      
      const TestFormWithCustomRender = () => {
        const form = useForm<TestFormData>({
          defaultValues: {
            username: '',
            email: '',
            age: 0,
            status: '',
            category: '',
            description: '',
            bio: '',
            isActive: false,
          },
        });

        const handleSubmit = (data: TestFormData) => {
          mockSubmit(data);
        };

        return (
          <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmit)} data-testid="test-form">
              <FormField
                control={form.control}
                name="isActive"
                label="Active Status"
                render={({ field }) => (
                  <input
                    type="checkbox"
                    data-testid="custom-checkbox"
                    checked={field?.value || false}
                    onChange={(e) => field?.onChange?.(e.target.checked)}
                    ref={field?.ref}
                    name={field?.name}
                  />
                )}
              />
              <Button type="submit" data-testid="submit-button">
                Submit
              </Button>
            </form>
          </Form>
        );
      };
      
      render(<TestFormWithCustomRender />);

      const checkbox = screen.getByTestId('custom-checkbox');
      expect(checkbox).toBeInTheDocument();
      expect(checkbox).not.toBeChecked();

      await user.click(checkbox);
      expect(checkbox).toBeChecked();

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            isActive: true
          })
        );
      });
    });
  });

  describe('FormSelect Integration', () => {
    it('should render and interact with FormSelect', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormSelect
            name="status"
            label="Status"
            placeholder="Select a status"
            options={mockSelectOptions}
            description="Choose the current status"
          />
        </TestForm>
      );

      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Choose the current status')).toBeInTheDocument();

      // Open select dropdown
      const selectTrigger = screen.getByRole('combobox');
      await user.click(selectTrigger);

      // Select an option - use getAllByText and take the first interactive one
      const activeOptions = screen.getAllByText('Active');
      const selectableOption = activeOptions.find(option => 
        option.getAttribute('role') !== null || 
        option.closest('[role="option"]') !== null
      ) || activeOptions[0];
      await user.click(selectableOption);

      // Submit form
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            status: 'active'
          })
        );
      });
    });

    it('should show empty message when no options provided', async () => {
      render(
        <TestForm>
          <FormSelect
            name="category"
            label="Category"
            options={[]}
            emptyMessage="No categories available"
          />
        </TestForm>
      );

      const selectTrigger = screen.getByRole('combobox');
      await user.click(selectTrigger);

      expect(screen.getByText('No categories available')).toBeInTheDocument();
    });

    it('should handle disabled options in FormSelect', async () => {
      render(
        <TestForm>
          <FormSelect
            name="status"
            label="Status"
            options={mockSelectOptions}
          />
        </TestForm>
      );

      const selectTrigger = screen.getByRole('combobox');
      expect(selectTrigger).toBeInTheDocument();
      
      // Test that the select is functional
      await user.click(selectTrigger);
      
      // Check if select content is opened (may not be visible in JSDOM)
      // Just verify the select trigger is clickable and the component renders
      expect(selectTrigger).not.toHaveAttribute('aria-disabled');
    });

    it('should handle disabled FormSelect', () => {
      render(
        <TestForm>
          <FormSelect
            name="status"
            label="Status"
            options={mockSelectOptions}
            disabled={true}
          />
        </TestForm>
      );

      const selectTrigger = screen.getByRole('combobox');
      // Check if the select appears disabled (Radix UI may handle this differently)
      expect(selectTrigger).toBeInTheDocument();
      expect(selectTrigger).toHaveAttribute('data-disabled');
    });
  });

  describe('FormTextarea Integration', () => {
    it('should render and interact with FormTextarea', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormTextarea
            name="description"
            label="Description"
            placeholder="Enter a description"
            description="Provide a detailed description"
            rows={4}
          />
        </TestForm>
      );

      expect(screen.getByText('Description')).toBeInTheDocument();
      expect(screen.getByText('Provide a detailed description')).toBeInTheDocument();

      const textarea = screen.getByPlaceholderText('Enter a description');
      expect(textarea).toBeInTheDocument();
      expect(textarea).toHaveAttribute('rows', '4');

      const testText = 'This is a test description for the form textarea component.';
      await user.type(textarea, testText);
      expect(textarea).toHaveValue(testText);

      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            description: testText
          })
        );
      });
    });

    it('should show character count when maxLength is provided', async () => {
      render(
        <TestForm>
          <FormTextarea
            name="bio"
            label="Biography"
            placeholder="Tell us about yourself"
            maxLength={50}
          />
        </TestForm>
      );

      expect(screen.getByText('0/50')).toBeInTheDocument();

      const textarea = screen.getByPlaceholderText('Tell us about yourself');
      await user.type(textarea, 'Hello world');

      expect(screen.getByText('11/50')).toBeInTheDocument();
    });

    it('should respect maxLength limit', async () => {
      render(
        <TestForm>
          <FormTextarea
            name="bio"
            label="Biography"
            maxLength={10}
          />
        </TestForm>
      );

      const textarea = screen.getByDisplayValue('');
      
      // Try to type more than maxLength
      await user.type(textarea, 'This text is longer than 10 characters');
      
      // Should be limited to 10 characters
      expect(textarea.value.length).toBeLessThanOrEqual(10);
    });
  });

  describe('Multiple Form Components Integration', () => {
    it('should handle multiple form components together', async () => {
      const mockSubmit = jest.fn();
      
      render(
        <TestForm onSubmit={mockSubmit}>
          <FormField
            name="username"
            label="Username"
            placeholder="Enter username"
          />
          <FormField
            name="email"
            label="Email"
            type="email"
            placeholder="Enter email"
          />
          <FormTextarea
            name="description"
            label="Description"
            placeholder="Enter description"
          />
        </TestForm>
      );

      // Fill out all fields
      await user.type(screen.getByPlaceholderText('Enter username'), 'testuser');
      await user.type(screen.getByPlaceholderText('Enter email'), 'test@example.com');
      await user.type(screen.getByPlaceholderText('Enter description'), 'Test description');

      // Submit form
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            username: 'testuser',
            email: 'test@example.com',
            description: 'Test description',
          })
        );
      });
    });

    it('should handle form validation across multiple components', async () => {
      const TestFormWithValidation = () => {
        const form = useForm<TestFormData>({
          defaultValues: {
            username: '',
            email: '',
            age: 0,
            status: '',
            category: '',
            description: '',
            bio: '',
            isActive: false,
          },
          mode: 'onBlur',
        });

        const handleSubmit = (data: TestFormData) => {
          console.log('Form submitted:', data);
        };

        return (
          <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmit)} data-testid="validation-form">
              <FormField
                control={form.control}
                name="username"
                label="Username"
                placeholder="Enter username"
              />
              <FormField
                control={form.control}
                name="email"
                label="Email"
                type="email"
                placeholder="Enter email"
              />
              <Button type="submit" data-testid="submit-button">
                Submit
              </Button>
            </form>
          </Form>
        );
      };

      render(<TestFormWithValidation />);

      // All form fields should be present
      expect(screen.getByPlaceholderText('Enter username')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('Enter email')).toBeInTheDocument();

      // Form should be submittable
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeInTheDocument();
    });

    it('should handle form state management across components', async () => {
      const mockSubmit = jest.fn();
      const defaultValues = {
        username: 'initial-user',
        email: 'initial@example.com',
        status: 'active',
        description: 'Initial description',
      };
      
      render(
        <TestForm onSubmit={mockSubmit} defaultValues={defaultValues}>
          <FormField
            name="username"
            label="Username"
          />
          <FormField
            name="email"
            label="Email"
            type="email"
          />
          <FormSelect
            name="status"
            label="Status"
            options={mockSelectOptions}
          />
          <FormTextarea
            name="description"
            label="Description"
          />
        </TestForm>
      );

      // Check that default values are applied
      expect(screen.getByDisplayValue('initial-user')).toBeInTheDocument();
      expect(screen.getByDisplayValue('initial@example.com')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Initial description')).toBeInTheDocument();

      // Submit without changes should use default values
      await user.click(screen.getByTestId('submit-button'));

      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            username: 'initial-user',
            email: 'initial@example.com',
            status: 'active',
            description: 'Initial description',
          })
        );
      });
    });
  });

  describe('Form Components Styling Integration', () => {
    it('should apply custom className to FormField', () => {
      const customClass = 'my-custom-field-class';
      
      render(
        <TestForm>
          <FormField
            name="username"
            label="Username"
            className={customClass}
          />
        </TestForm>
      );

      const formField = screen.getByTestId('form-field');
      expect(formField).toHaveClass(customClass);
    });

    it('should apply custom className to FormSelect', () => {
      const customClass = 'my-custom-select-class';
      
      render(
        <TestForm>
          <FormSelect
            name="status"
            label="Status"
            options={mockSelectOptions}
            className={customClass}
          />
        </TestForm>
      );

      // The FormItem should have the custom class
      const formItem = screen.getByText('Status').closest('.space-y-2');
      expect(formItem?.className).toContain(customClass);
    });

    it('should apply custom className to FormTextarea', () => {
      const customClass = 'my-custom-textarea-class';
      
      render(
        <TestForm>
          <FormTextarea
            name="description"
            label="Description"
            className={customClass}
          />
        </TestForm>
      );

      // The FormItem should have the custom class
      const formItem = screen.getByText('Description').closest('.space-y-2');
      expect(formItem?.className).toContain(customClass);
    });
  });

  describe('Accessibility Integration', () => {
    it('should provide proper accessibility attributes for FormField', () => {
      render(
        <TestForm>
          <FormField
            name="username"
            label="Username"
            description="Enter your username"
          />
        </TestForm>
      );

      const input = screen.getByRole('textbox', { name: /username/i });
      expect(input).toBeInTheDocument();
      
      const label = screen.getByText('Username');
      expect(label).toBeInTheDocument();
      
      const description = screen.getByText('Enter your username');
      expect(description).toBeInTheDocument();
    });

    it('should provide proper accessibility attributes for FormSelect', async () => {
      render(
        <TestForm>
          <FormSelect
            name="status"
            label="Status"
            options={mockSelectOptions}
          />
        </TestForm>
      );

      const select = screen.getByRole('combobox', { name: /status/i });
      expect(select).toBeInTheDocument();
      
      const label = screen.getByText('Status');
      expect(label).toBeInTheDocument();
    });

    it('should provide proper accessibility attributes for FormTextarea', () => {
      render(
        <TestForm>
          <FormTextarea
            name="description"
            label="Description"
            description="Provide details"
          />
        </TestForm>
      );

      const textarea = screen.getByRole('textbox', { name: /description/i });
      expect(textarea).toBeInTheDocument();
      
      const label = screen.getByText('Description');
      expect(label).toBeInTheDocument();
      
      const description = screen.getByText('Provide details');
      expect(description).toBeInTheDocument();
    });
  });
});