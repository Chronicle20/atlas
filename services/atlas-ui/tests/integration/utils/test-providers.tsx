import * as React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm, FieldValues } from 'react-hook-form';
import { ThemeProvider } from 'next-themes';

/**
 * Test providers for integration tests that need various context providers
 */

interface TestQueryProviderProps {
  children: React.ReactNode;
  queryClient?: QueryClient;
}

/**
 * Wraps components with React Query provider for testing
 */
export function TestQueryProvider({ children, queryClient }: TestQueryProviderProps) {
  const defaultQueryClient = React.useMemo(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: false,
            gcTime: 0,
          },
          mutations: {
            retry: false,
          },
        },
      }),
    []
  );

  return (
    <QueryClientProvider client={queryClient || defaultQueryClient}>
      {children}
    </QueryClientProvider>
  );
}

interface TestThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: string;
}

/**
 * Wraps components with Theme provider for testing
 */
export function TestThemeProvider({ children, defaultTheme = 'light' }: TestThemeProviderProps) {
  return (
    <ThemeProvider attribute="class" defaultTheme={defaultTheme} enableSystem={false}>
      {children}
    </ThemeProvider>
  );
}

interface TestFormProviderProps<T extends FieldValues> {
  children: React.ReactNode | ((form: ReturnType<typeof useForm<T>>) => React.ReactNode);
  defaultValues?: Partial<T>;
  mode?: 'onChange' | 'onBlur' | 'onSubmit' | 'onTouched' | 'all';
}

/**
 * Wraps components with React Hook Form provider for testing
 */
export function TestFormProvider<T extends FieldValues>({
  children,
  defaultValues,
  mode = 'onChange',
}: TestFormProviderProps<T>) {
  const form = useForm<T>({
    defaultValues: defaultValues as any,
    mode,
  });

  return (
    <FormProvider {...form}>
      {typeof children === 'function' ? children(form) : children}
    </FormProvider>
  );
}

interface AllProvidersProps {
  children: React.ReactNode;
  queryClient?: QueryClient;
  defaultTheme?: string;
}

/**
 * Combines all common providers for comprehensive testing
 */
export function AllProviders({ children, queryClient, defaultTheme }: AllProvidersProps) {
  return (
    <TestQueryProvider queryClient={queryClient}>
      <TestThemeProvider defaultTheme={defaultTheme}>
        {children}
      </TestThemeProvider>
    </TestQueryProvider>
  );
}

/**
 * Creates a wrapper function for React Testing Library that includes all providers
 */
export function createWrapper(options: {
  queryClient?: QueryClient;
  defaultTheme?: string;
} = {}) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <AllProviders
        queryClient={options.queryClient}
        defaultTheme={options.defaultTheme}
      >
        {children}
      </AllProviders>
    );
  };
}

/**
 * Custom hook for testing that provides a mock query client
 */
export function useMockQueryClient() {
  return React.useMemo(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: false,
            gcTime: 0,
          },
          mutations: {
            retry: false,
          },
        },
      }),
    []
  );
}