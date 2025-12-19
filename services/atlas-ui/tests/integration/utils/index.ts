/**
 * Integration test utilities index file
 * 
 * This file exports all test utilities for easy importing in integration tests
 */

// Test providers
export {
  TestQueryProvider,
  TestThemeProvider,
  TestFormProvider,
  AllProviders,
  createWrapper,
  useMockQueryClient,
} from './test-providers';

// Test helpers
export {
  waitForLoadingToComplete,
  waitForErrorToAppear,
  waitForEmptyStateToAppear,
  waitForDataTableToAppear,
  formHelpers,
  tableHelpers,
  overlayHelpers,
  errorHelpers,
  emptyStateHelpers,
  mockDataHelpers,
  stateSimulators,
  a11yHelpers,
} from './test-helpers';

// Common types for integration tests
export interface MockApiResponse<T> {
  data: T;
  status: number;
  message?: string;
}

export interface MockTableData {
  id: string;
  name: string;
  email: string;
  status: 'active' | 'inactive';
  createdAt: Date;
}

export interface MockFormData {
  username: string;
  email: string;
  age: number;
  status: string;
  description: string;
  isActive: boolean;
}

export interface MockSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

// Test constants
export const TEST_IDS = {
  LOADING_SPINNER: 'loading-spinner',
  PAGE_LOADER: 'page-loader',
  ERROR_DISPLAY: 'error-display',
  EMPTY_STATE: 'empty-state',
  FORM_FIELD: 'form-field',
  TEST_FORM: 'test-form',
  SUBMIT_BUTTON: 'submit-button',
} as const;

export const MOCK_DELAYS = {
  SHORT: 100,
  MEDIUM: 500,
  LONG: 1000,
  VERY_LONG: 2000,
} as const;

export const MOCK_ERRORS = {
  NETWORK_ERROR: 'Network request failed',
  VALIDATION_ERROR: 'Validation failed',
  SERVER_ERROR: 'Internal server error',
  NOT_FOUND: 'Resource not found',
  UNAUTHORIZED: 'Unauthorized access',
} as const;