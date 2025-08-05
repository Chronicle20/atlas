import { render, screen } from '@testing-library/react';
import { FormSkeleton } from '@/components/common/FormSkeleton';

describe('FormSkeleton', () => {
  it('renders with default props', () => {
    render(<FormSkeleton />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    expect(skeleton).toBeInTheDocument();
    
    // Should render 4 fields by default + 1 button section = 5 total children
    const fieldContainers = skeleton.children;
    expect(fieldContainers).toHaveLength(5); // 4 fields + 1 button section
  });

  it('renders correct number of fields', () => {
    render(<FormSkeleton fields={6} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const fieldElements = skeleton.querySelectorAll('[key*="skeleton-field"]');
    // Check that we have field containers (since keys are not visible in DOM, count field spaces)
    const fieldSpaces = skeleton.querySelectorAll('.space-y-2');
    expect(fieldSpaces.length).toBeGreaterThanOrEqual(6);
  });

  it('shows labels when showLabels is true', () => {
    render(<FormSkeleton fields={2} showLabels={true} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const labelElements = skeleton.querySelectorAll('.w-24'); // Default label width class
    expect(labelElements.length).toBeGreaterThanOrEqual(2);
  });

  it('hides labels when showLabels is false', () => {
    render(<FormSkeleton fields={2} showLabels={false} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const fieldContainers = skeleton.querySelectorAll('.space-y-2');
    
    // Each field container should only have one child (the input skeleton) when labels are hidden
    fieldContainers.forEach(container => {
      const children = container.children;
      expect(children).toHaveLength(1); // Only input skeleton, no label
    });
  });

  it('shows help text when showHelpText is true', () => {
    render(<FormSkeleton fields={2} showHelpText={true} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const helpTextElements = skeleton.querySelectorAll('.h-3.w-3\\/4');
    expect(helpTextElements.length).toBeGreaterThanOrEqual(2);
  });

  it('shows submit button when showSubmitButton is true', () => {
    render(<FormSkeleton showSubmitButton={true} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const buttonSection = skeleton.querySelector('.flex.items-center.justify-end');
    expect(buttonSection).toBeInTheDocument();
    
    const submitButton = buttonSection?.querySelector('.w-24');
    expect(submitButton).toBeInTheDocument();
  });

  it('hides submit button when showSubmitButton is false', () => {
    render(<FormSkeleton showSubmitButton={false} showActionButtons={false} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const buttonSection = skeleton.querySelector('.flex.items-center.justify-end');
    expect(buttonSection).not.toBeInTheDocument();
  });

  it('shows action buttons when showActionButtons is true', () => {
    render(<FormSkeleton showActionButtons={true} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const buttonSection = skeleton.querySelector('.flex.items-center.justify-end');
    expect(buttonSection).toBeInTheDocument();
    
    const actionButton = buttonSection?.querySelector('.w-20');
    expect(actionButton).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<FormSkeleton className="custom-form-class" />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    expect(skeleton).toHaveClass('custom-form-class');
  });

  it('renders compact variant correctly', () => {
    render(<FormSkeleton variant="compact" fields={1} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    expect(skeleton).toHaveClass('space-y-3');
    
    // Check for compact field height
    const inputElement = skeleton.querySelector('.h-8');
    expect(inputElement).toBeInTheDocument();
    
    // Check for compact label width
    const labelElement = skeleton.querySelector('.w-16');
    expect(labelElement).toBeInTheDocument();
  });

  it('renders wide variant correctly', () => {
    render(<FormSkeleton variant="wide" fields={1} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    expect(skeleton).toHaveClass('space-y-8');
    
    // Check for wide field height
    const inputElement = skeleton.querySelector('.h-12');
    expect(inputElement).toBeInTheDocument();
    
    // Check for wide label width
    const labelElement = skeleton.querySelector('.w-32');
    expect(labelElement).toBeInTheDocument();
  });

  it('renders default variant correctly', () => {
    render(<FormSkeleton variant="default" fields={1} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    expect(skeleton).toHaveClass('space-y-6');
    
    // Check for default field height
    const inputElement = skeleton.querySelector('.h-10');
    expect(inputElement).toBeInTheDocument();
    
    // Check for default label width
    const labelElement = skeleton.querySelector('.w-24');
    expect(labelElement).toBeInTheDocument();
  });

  it('renders form fields with rectangular variant', () => {
    render(<FormSkeleton fields={2} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const rectangularElements = skeleton.querySelectorAll('.rounded-sm');
    expect(rectangularElements.length).toBeGreaterThanOrEqual(2); // At least 2 input fields
  });

  it('renders button section with proper spacing', () => {
    render(<FormSkeleton showSubmitButton={true} showActionButtons={true} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const buttonSection = skeleton.querySelector('.flex.items-center.justify-end.space-x-3.pt-4');
    expect(buttonSection).toBeInTheDocument();
    
    // Should have both action button and submit button
    const buttons = buttonSection?.children;
    expect(buttons).toHaveLength(2);
  });

  it('does not render button section when all button options are false', () => {
    render(<FormSkeleton showSubmitButton={false} showActionButtons={false} />);
    
    const skeleton = screen.getByTestId('form-skeleton');
    const buttonSection = skeleton.querySelector('.flex.items-center.justify-end');
    expect(buttonSection).not.toBeInTheDocument();
  });

  it('passes animation prop to skeleton elements', () => {
    const { container } = render(<FormSkeleton animation="wave" fields={2} />);
    
    // Check that skeleton elements have the shimmer animation class
    const skeletonElements = container.querySelectorAll('.animate-shimmer');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('uses default pulse animation when no animation prop provided', () => {
    const { container } = render(<FormSkeleton fields={2} />);
    
    // Check that skeleton elements have the pulse animation class
    const skeletonElements = container.querySelectorAll('.animate-pulse');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });
});