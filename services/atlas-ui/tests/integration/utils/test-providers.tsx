import * as React from 'react';
import { FormProvider, useForm, FieldValues } from 'react-hook-form';
import { ThemeProvider } from 'next-themes';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';

/**
 * Test providers for integration tests that need various context providers
 */

/**
 * Simple wrapper for children when no query provider is needed
 */
export function TestQueryProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

interface TestThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: string | undefined;
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
  defaultTheme?: string | undefined;
  withErrorBoundary?: boolean;
}

/**
 * Combines all common providers for comprehensive testing
 */
export function AllProviders({ children, defaultTheme, withErrorBoundary = false }: AllProvidersProps) {
  const content = (
    <TestQueryProvider>
      <TestThemeProvider defaultTheme={defaultTheme}>
        {children}
      </TestThemeProvider>
    </TestQueryProvider>
  );

  if (withErrorBoundary) {
    return (
      <ErrorBoundary enableErrorReporting={false}>
        {content}
      </ErrorBoundary>
    );
  }

  return content;
}

/**
 * Creates a wrapper function for React Testing Library that includes all providers
 */
export function createWrapper(options: {
  defaultTheme?: string | undefined;
  withErrorBoundary?: boolean;
} = {}) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    const props: any = {
      defaultTheme: options.defaultTheme,
    };
    
    if (options.withErrorBoundary !== undefined) {
      props.withErrorBoundary = options.withErrorBoundary;
    }
    
    return (
      <AllProviders {...props}>
        {children}
      </AllProviders>
    );
  };
}

/**
 * Simple mock for query client functionality when not using React Query
 */
export function useMockQueryClient() {
  return React.useMemo(() => ({}), []);
}