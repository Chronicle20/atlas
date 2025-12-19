import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * Common test helpers for integration tests
 */

/**
 * Helper to wait for loading states to complete
 */
export async function waitForLoadingToComplete() {
  await waitFor(
    () => {
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      expect(screen.queryByTestId('page-loader')).not.toBeInTheDocument();
    },
    { timeout: 5000 }
  );
}

/**
 * Helper to wait for error states to appear
 */
export async function waitForErrorToAppear() {
  await waitFor(
    () => {
      expect(screen.getByTestId('error-display')).toBeInTheDocument();
    },
    { timeout: 5000 }
  );
}

/**
 * Helper to wait for empty states to appear
 */
export async function waitForEmptyStateToAppear() {
  await waitFor(
    () => {
      expect(screen.getByTestId('empty-state')).toBeInTheDocument();
    },
    { timeout: 5000 }
  );
}

/**
 * Helper to wait for data to load and table to appear
 */
export async function waitForDataTableToAppear() {
  await waitFor(
    () => {
      expect(screen.getByRole('table')).toBeInTheDocument();
    },
    { timeout: 5000 }
  );
}

/**
 * Helper to interact with form fields
 */
export const formHelpers = {
  /**
   * Fill a text input field
   */
  async fillTextInput(label: string | RegExp, value: string) {
    const user = userEvent.setup();
    const input = screen.getByRole('textbox', { name: label });
    await user.clear(input);
    await user.type(input, value);
    return input;
  },

  /**
   * Fill an email input field
   */
  async fillEmailInput(placeholder: string, value: string) {
    const user = userEvent.setup();
    const input = screen.getByPlaceholderText(placeholder);
    await user.clear(input);
    await user.type(input, value);
    return input;
  },

  /**
   * Fill a number input field
   */
  async fillNumberInput(label: string | RegExp, value: number) {
    const user = userEvent.setup();
    const input = screen.getByRole('spinbutton', { name: label });
    await user.clear(input);
    await user.type(input, value.toString());
    return input;
  },

  /**
   * Select an option from a select dropdown
   */
  async selectOption(selectLabel: string | RegExp, optionText: string) {
    const user = userEvent.setup();
    const select = screen.getByRole('combobox', { name: selectLabel });
    await user.click(select);
    
    const option = screen.getByRole('option', { name: optionText });
    await user.click(option);
    return select;
  },

  /**
   * Fill a textarea field
   */
  async fillTextarea(label: string | RegExp, value: string) {
    const user = userEvent.setup();
    const textarea = screen.getByRole('textbox', { name: label });
    await user.clear(textarea);
    await user.type(textarea, value);
    return textarea;
  },

  /**
   * Submit a form
   */
  async submitForm(formTestId = 'test-form') {
    const user = userEvent.setup();
    const form = screen.getByTestId(formTestId);
    const submitButton = within(form).getByRole('button', { name: /submit/i });
    await user.click(submitButton);
    return form;
  },
};

/**
 * Helper to interact with data tables
 */
export const tableHelpers = {
  /**
   * Get table rows (excluding header)
   */
  getTableRows() {
    const table = screen.getByRole('table');
    const rows = within(table).getAllByRole('row');
    return rows.slice(1); // Remove header row
  },

  /**
   * Get table headers
   */
  getTableHeaders() {
    const table = screen.getByRole('table');
    const headerRow = within(table).getAllByRole('row')[0];
    if (!headerRow) throw new Error('No header row found');
    return within(headerRow).getAllByRole('columnheader');
  },

  /**
   * Get cell value by row and column index
   */
  getCellValue(rowIndex: number, columnIndex: number) {
    const rows = this.getTableRows();
    const row = rows[rowIndex];
    if (!row) throw new Error(`No row found at index ${rowIndex}`);
    const cells = within(row).getAllByRole('cell');
    return cells[columnIndex];
  },

  /**
   * Check if table is empty
   */
  isTableEmpty() {
    try {
      this.getTableRows();
      return false;
    } catch {
      return true;
    }
  },
};

/**
 * Helper to interact with loading overlays
 */
export const overlayHelpers = {
  /**
   * Check if loading overlay is visible
   */
  isLoadingOverlayVisible() {
    return !!screen.queryByTestId('loading-spinner');
  },

  /**
   * Wait for loading overlay to appear
   */
  async waitForLoadingOverlay() {
    await waitFor(() => {
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });
  },

  /**
   * Wait for loading overlay to disappear
   */
  async waitForLoadingOverlayToDisappear() {
    await waitFor(() => {
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    });
  },
};

/**
 * Helper to interact with error boundaries and error displays
 */
export const errorHelpers = {
  /**
   * Check if error display is visible
   */
  isErrorDisplayVisible() {
    return !!screen.queryByTestId('error-display');
  },

  /**
   * Get error message text
   */
  getErrorMessage() {
    const errorDisplay = screen.getByTestId('error-display');
    return within(errorDisplay).getByRole('alert');
  },

  /**
   * Click retry button in error display
   */
  async clickRetryButton() {
    const user = userEvent.setup();
    const retryButton = screen.getByText('Try Again');
    await user.click(retryButton);
  },

  /**
   * Trigger an error in a component for testing error boundaries
   */
  triggerError(component: any) {
    // This would be implemented based on specific error scenarios
    throw new Error('Test error for error boundary');
  },
};

/**
 * Helper to interact with empty states
 */
export const emptyStateHelpers = {
  /**
   * Check if empty state is visible
   */
  isEmptyStateVisible() {
    return !!screen.queryByTestId('empty-state');
  },

  /**
   * Get empty state title
   */
  getEmptyStateTitle() {
    const emptyState = screen.getByTestId('empty-state');
    return within(emptyState).getByRole('heading');
  },

  /**
   * Click action button in empty state
   */
  async clickEmptyStateAction() {
    const user = userEvent.setup();
    const emptyState = screen.getByTestId('empty-state');
    const actionButton = within(emptyState).getByRole('button');
    await user.click(actionButton);
  },
};

/**
 * Utility to create mock data for testing
 */
export const mockDataHelpers = {
  /**
   * Create mock table data
   */
  createMockTableData(count: number = 5) {
    return Array.from({ length: count }, (_, index) => ({
      id: (index + 1).toString(),
      name: `Item ${index + 1}`,
      email: `item${index + 1}@example.com`,
      status: index % 2 === 0 ? 'active' : 'inactive',
      createdAt: new Date(Date.now() - index * 24 * 60 * 60 * 1000),
    }));
  },

  /**
   * Create mock select options
   */
  createMockSelectOptions(count: number = 3) {
    return Array.from({ length: count }, (_, index) => ({
      value: `option-${index + 1}`,
      label: `Option ${index + 1}`,
      disabled: index === count - 1, // Make last option disabled
    }));
  },

  /**
   * Create mock form data
   */
  createMockFormData() {
    return {
      username: 'testuser',
      email: 'test@example.com',
      age: 25,
      status: 'active',
      description: 'Test description',
      isActive: true,
    };
  },
};

/**
 * Utility to simulate different states for component testing
 */
export const stateSimulators = {
  /**
   * Simulate loading state
   */
  async simulateLoading(duration: number = 1000) {
    await new Promise(resolve => setTimeout(resolve, duration));
  },

  /**
   * Simulate error state
   */
  simulateError(message: string = 'Test error') {
    throw new Error(message);
  },

  /**
   * Simulate network delay
   */
  async simulateNetworkDelay(min: number = 500, max: number = 2000) {
    const delay = Math.random() * (max - min) + min;
    await new Promise(resolve => setTimeout(resolve, delay));
  },
};

/**
 * Accessibility testing helpers
 */
export const a11yHelpers = {
  /**
   * Check if element has proper ARIA labels
   */
  hasAriaLabel(element: HTMLElement) {
    return element.hasAttribute('aria-label') || element.hasAttribute('aria-labelledby');
  },

  /**
   * Check if form field has proper associations
   */
  hasProperFormAssociation(inputElement: HTMLElement) {
    const id = inputElement.getAttribute('id');
    if (!id) return false;

    const label = document.querySelector(`label[for="${id}"]`);
    const ariaLabelledBy = inputElement.getAttribute('aria-labelledby');
    
    return !!label || !!ariaLabelledBy;
  },

  /**
   * Check if interactive elements are keyboard accessible
   */
  isKeyboardAccessible(element: HTMLElement) {
    const tabIndex = element.getAttribute('tabindex');
    const isInteractive = ['button', 'input', 'select', 'textarea', 'a'].includes(
      element.tagName.toLowerCase()
    );
    
    return isInteractive || (tabIndex !== null && parseInt(tabIndex) >= 0);
  },
};