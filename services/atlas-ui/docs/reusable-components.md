# Reusable Components Guide

This guide provides comprehensive examples for using the reusable components extracted from repeated patterns across the Atlas UI codebase. These components ensure consistency, reduce code duplication, and provide better user experience.

## Loading Components

### LoadingSpinner

A configurable spinner component for indicating loading states.

```tsx
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

// Small spinner
<LoadingSpinner size="sm" />

// Medium spinner (default)
<LoadingSpinner />
<LoadingSpinner size="md" />

// Large spinner
<LoadingSpinner size="lg" />

// Custom styling
<LoadingSpinner className="text-blue-500" />
```

**Use cases:**
- Button loading states
- Small content sections
- Inline loading indicators

### PageLoader

A full-page loading component for page-level loading states.

```tsx
import { PageLoader } from '@/components/common/PageLoader';

function DataPage() {
  const { data, isLoading } = useData();
  
  if (isLoading) return <PageLoader />;
  
  return <div>{/* page content */}</div>;
}
```

**Use cases:**
- Initial page loads
- Route transitions
- Full-page data fetching

### LoadingOverlay

An overlay component that shows loading state over existing content.

```tsx
import { LoadingOverlay } from '@/components/common/LoadingOverlay';

function FormSection() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  return (
    <LoadingOverlay loading={isSubmitting}>
      <form onSubmit={handleSubmit}>
        {/* form content */}
      </form>
    </LoadingOverlay>
  );
}
```

**Use cases:**
- Form submissions
- Partial page updates
- Async operations on existing content

## Error Handling

### ErrorDisplay

A consistent error display component with optional retry functionality.

```tsx
import { ErrorDisplay } from '@/components/common/ErrorDisplay';

// Basic error display
<ErrorDisplay error="Something went wrong" />

// With Error object
<ErrorDisplay error={new Error("Network failure")} />

// With retry button
<ErrorDisplay 
  error="Failed to load data" 
  retry={() => refetch()}
/>

// Custom styling and title
<ErrorDisplay
  error="Validation failed"
  title="Form Error"
  className="mb-4"
  showIcon={false}
/>
```

**Props:**
- `error`: Error object, error-like object, or string
- `retry?`: Optional retry function
- `title?`: Custom error title (default: "Error")
- `showIcon?`: Show/hide error icon (default: true)
- `className?`: Additional CSS classes

### ErrorBoundary

A React error boundary for catching JavaScript errors anywhere in the component tree.

```tsx
import { ErrorBoundary } from '@/components/common/ErrorBoundary';

function App() {
  return (
    <ErrorBoundary>
      <Header />
      <Main />
      <Footer />
    </ErrorBoundary>
  );
}

// With custom fallback
<ErrorBoundary 
  fallback={({ error, resetErrorBoundary }) => (
    <div>
      <h2>Something went wrong</h2>
      <details>{error.message}</details>
      <button onClick={resetErrorBoundary}>Try again</button>
    </div>
  )}
>
  <RiskyComponent />
</ErrorBoundary>
```

## Empty States

### EmptyState

A component for displaying when no data is available, with optional action buttons.

```tsx
import { EmptyState } from '@/components/common/EmptyState';
import { Users, Plus } from 'lucide-react';

// Basic empty state
<EmptyState 
  title="No users found"
  description="There are no users to display at this time."
/>

// With icon and action
<EmptyState
  icon={<Users className="h-12 w-12" />}
  title="No team members"
  description="Start building your team by inviting members."
  action={{
    label: "Invite Member",
    onClick: () => setShowInviteModal(true)
  }}
/>

// Custom styling
<EmptyState
  title="Empty Cart"
  description="Add some items to get started."
  className="py-16"
/>
```

**Props:**
- `title`: Main heading text
- `description?`: Optional description text
- `icon?`: Optional icon component
- `action?`: Optional action button with label and onClick
- `className?`: Additional CSS classes

## Data Tables

### DataTableWrapper

A wrapper component that handles all data table states (loading, error, empty, data).

```tsx
import { DataTableWrapper } from '@/components/common/DataTableWrapper';
import { ColumnDef } from '@tanstack/react-table';

interface User {
  id: string;
  name: string;
  email: string;
}

const columns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: "Name",
  },
  {
    accessorKey: "email", 
    header: "Email",
  },
];

function UsersTable() {
  const { data, isLoading, error, refetch } = useUsers();
  
  return (
    <DataTableWrapper
      columns={columns}
      data={data || []}
      loading={isLoading}
      error={error}
      onRefresh={refetch}
      emptyState={{
        title: "No users found",
        description: "Create your first user to get started.",
        action: {
          label: "Add User",
          onClick: () => setShowCreateModal(true)
        }
      }}
    />
  );
}
```

**Props:**
- `columns`: TanStack table column definitions
- `data`: Array of data objects
- `loading?`: Loading state boolean
- `error?`: Error object or string
- `onRefresh?`: Optional refresh function
- `emptyState?`: Custom empty state configuration
- `headerActions?`: Optional header action buttons
- `className?`: Additional CSS classes

## Form Components

### FormField

A reusable form field component that works with react-hook-form and shadcn/ui.

```tsx
import { FormField } from '@/components/common/FormField';
import { useForm } from 'react-hook-form';

function UserForm() {
  const form = useForm<{
    username: string;
    email: string;
    age: number;
    bio: string;
  }>();
  
  return (
    <form>
      {/* Text field */}
      <FormField
        control={form.control}
        name="username"
        label="Username"
        placeholder="Enter username"
        description="Choose a unique username"
      />
      
      {/* Email field */}
      <FormField
        control={form.control}
        name="email"
        label="Email"
        type="email"
        placeholder="user@example.com"
      />
      
      {/* Number field */}
      <FormField
        control={form.control}
        name="age"
        label="Age"
        type="number"
        min={0}
        max={120}
        placeholder="Enter age"
      />
      
      {/* Custom field with render prop */}
      <FormField
        control={form.control}
        name="bio"
        label="Biography"
        render={({ field }) => (
          <Textarea {...field} placeholder="Tell us about yourself" />
        )}
      />
    </form>
  );
}
```

### FormSelect

A reusable select field component for dropdowns.

```tsx
import { FormSelect } from '@/components/common/FormSelect';

<FormSelect
  control={form.control}
  name="country"
  label="Country"
  placeholder="Select a country"
  options={[
    { value: "us", label: "United States" },
    { value: "ca", label: "Canada" },
    { value: "uk", label: "United Kingdom" },
  ]}
  description="Choose your country of residence"
/>
```

### FormTextarea

A reusable textarea field component.

```tsx
import { FormTextarea } from '@/components/common/FormTextarea';

<FormTextarea
  control={form.control}
  name="description"
  label="Description"
  placeholder="Enter description"
  rows={4}
  description="Provide a detailed description"
/>
```

## Skeleton Components

### TableSkeleton

Loading skeleton for table structures.

```tsx
import { TableSkeleton } from '@/components/common/TableSkeleton';

function DataTable() {
  const { data, isLoading } = useData();
  
  if (isLoading) return <TableSkeleton rows={5} columns={4} />;
  
  return <DataTable data={data} />;
}

// Custom configuration
<TableSkeleton 
  rows={10} 
  columns={6}
  className="mb-4"
/>
```

### CardSkeleton

Loading skeleton for card layouts.

```tsx
import { CardSkeleton } from '@/components/common/CardSkeleton';

function UserCard() {
  const { user, isLoading } = useUser();
  
  if (isLoading) return <CardSkeleton />;
  
  return <UserCard user={user} />;
}

// Multiple cards
<div className="grid grid-cols-3 gap-4">
  {Array.from({ length: 6 }).map((_, i) => (
    <CardSkeleton key={i} />
  ))}
</div>
```

## Complete Examples

### Data Management Page

Here's a complete example showing how these components work together:

```tsx
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DataTableWrapper } from '@/components/common/DataTableWrapper';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { LoadingOverlay } from '@/components/common/LoadingOverlay';

function UsersManagementPage() {
  const [isDeleting, setIsDeleting] = useState(false);
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['users'],
    queryFn: fetchUsers,
  });
  
  const handleDelete = async (userId: string) => {
    setIsDeleting(true);
    try {
      await deleteUser(userId);
      refetch();
    } finally {
      setIsDeleting(false);
    }
  };
  
  return (
    <ErrorBoundary>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold">Users</h1>
          <Button onClick={() => setShowCreateModal(true)}>
            Add User
          </Button>
        </div>
        
        <LoadingOverlay loading={isDeleting}>
          <DataTableWrapper
            columns={userColumns}
            data={data || []}
            loading={isLoading}
            error={error}
            onRefresh={refetch}
            emptyState={{
              title: "No users found",
              description: "Create your first user to get started.",
              action: {
                label: "Add User", 
                onClick: () => setShowCreateModal(true)
              }
            }}
            headerActions={[
              {
                label: "Export",
                onClick: handleExport,
                icon: <Download className="h-4 w-4" />
              }
            ]}
          />
        </LoadingOverlay>
      </div>
    </ErrorBoundary>
  );
}
```

### Form with Validation

```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormField, FormSelect, FormTextarea } from '@/components/common';
import { ErrorDisplay } from '@/components/common/ErrorDisplay';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

const userSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  email: z.string().email("Invalid email address"),
  role: z.string().min(1, "Please select a role"),
  bio: z.string().optional(),
});

function CreateUserForm() {
  const form = useForm({
    resolver: zodResolver(userSchema),
  });
  
  const { mutate, isLoading, error } = useCreateUser();
  
  const onSubmit = (data: UserFormData) => {
    mutate(data);
  };
  
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      {error && <ErrorDisplay error={error} />}
      
      <FormField
        control={form.control}
        name="name"
        label="Full Name"
        placeholder="Enter full name"
      />
      
      <FormField
        control={form.control}
        name="email"
        label="Email"
        type="email"
        placeholder="user@example.com"
      />
      
      <FormSelect
        control={form.control}
        name="role"
        label="Role"
        placeholder="Select a role"
        options={roleOptions}
      />
      
      <FormTextarea
        control={form.control}
        name="bio"
        label="Bio"
        placeholder="Tell us about this user"
        rows={3}
      />
      
      <Button type="submit" disabled={isLoading}>
        {isLoading && <LoadingSpinner size="sm" />}
        Create User
      </Button>
    </form>
  );
}
```

## Migration Notes

When migrating existing code to use these components:

1. **Replace Loading Patterns**: Change `<div>Loading...</div>` to appropriate loading components
2. **Replace Error Patterns**: Change `<div>Error: {error}</div>` to `<ErrorDisplay error={error} />`
3. **Use DataTableWrapper**: Replace direct DataTable usage with DataTableWrapper for consistent states
4. **Update Forms**: Use FormField components instead of manual form field patterns
5. **Add Error Boundaries**: Wrap major sections with ErrorBoundary for better error handling

## Best Practices

1. **Choose the Right Loading Component**:
   - `LoadingSpinner` for small, inline loading states
   - `PageLoader` for full-page loading
   - `LoadingOverlay` for overlaying existing content

2. **Provide Meaningful Error Messages**: Always include helpful error messages and retry options when possible

3. **Use Empty States Effectively**: Provide clear next steps for users when no data is available

4. **Test Error Boundaries**: Ensure error boundaries are placed at appropriate levels in your component tree

5. **Consistent Styling**: Use className props to maintain design system consistency

## TypeScript Support

All components are fully typed with TypeScript:

- Props are strictly typed with helpful IntelliSense
- Generic components (like DataTableWrapper) preserve type safety
- Error objects are properly typed
- Form components integrate seamlessly with react-hook-form types

This ensures compile-time safety and excellent developer experience.