# Error handling

Atlas UI's error handling is built on four primitives:

1. **API client** (`src/lib/api/client.ts`) — parses JSON:API error bodies, retries 5xx / 429, throws `ApiError` subtypes.
2. **React Query** — owns in-flight state and retry semantics for all data fetches. Hooks under `src/lib/hooks/api/` expose `.error` / `.isError` / `.refetch` and retry transient failures by default.
3. **Error boundary** (`src/components/common/error-boundary.tsx` — `RouteErrorBoundary`) wraps `<Routes>` and catches render-time exceptions.
4. **Error pages** (`src/components/common/ErrorPage.tsx`, `not-found-page.tsx`) — the 404 and 500 fallback UIs.

Toasts (via `sonner`) are the user-facing channel for recoverable errors (failed mutations, validation errors, etc). The `errorLogger` in `src/services/errorLogger.ts` forwards logged errors to the configured remote sink.

## API errors

```ts
import { api } from "@/lib/api/client";
import { createErrorFromUnknown } from "@/types/api/errors";

try {
  const account = await accountsService.getAccountById(activeTenant, id);
} catch (err) {
  const errorInfo = createErrorFromUnknown(err, "Failed to fetch account");
  toast.error(errorInfo.message);
}
```

The client calls `createApiErrorFromResponse(status, message)` on non-OK responses. Status → default message mapping lives in `src/types/api/errors.ts`. When the response body is JSON:API-shaped (`{ errors: [{ detail, title }] }` or `{ error: { detail } }` or `{ message }`), the client extracts the first useful string via `sanitizeErrorData` (`src/lib/api/errors.ts`) so downstream toasts don't leak sensitive fields.

## React Query hooks

List and detail hooks expose `error` / `isError` / `isLoading` directly:

```tsx
const accountsQuery = useAccounts(activeTenant);

if (accountsQuery.isLoading) return <Skeleton />;
if (accountsQuery.isError) {
  return <ErrorDisplay error={accountsQuery.error?.message ?? "Failed to load accounts"} />;
}
```

Mutation hooks take an `onSuccess` / `onError` callback; pages typically toast on error:

```tsx
const updateAccount = useUpdateAccount();

const handleSubmit = (data: Account) => {
  updateAccount.mutate(data, {
    onSuccess: () => toast.success("Saved"),
    onError: (err) => toast.error(err.message),
  });
};
```

Retries are handled by React Query's defaults (3 attempts on query failures, once on mutations). Tune per-query via `retry: n` if needed.

## Route-level error boundary

`<RouteErrorBoundary>` wraps the entire `<Routes>` tree in `App.tsx`. Any render exception bubbles to `<ErrorFallback>`, which logs to `errorLogger`, shows a "Something went wrong" card, and provides "Try Again" + "Go Home" buttons. Stack traces and error details appear only under `import.meta.env.DEV`.

## Not-found

React Router routes unmatched paths to `<NotFoundPage />` (`src/components/common/not-found-page.tsx`). Services can programmatically trigger the same page by throwing a `NotFoundError` from a route loader.

## Logging

`errorLogger` (`src/services/errorLogger.ts`) is a singleton that the error boundary, query hooks, and service catch blocks call into. It batches to a remote endpoint configured by:

- `VITE_ERROR_ENDPOINT` — remote sink URL (optional; logging is disabled if unset).
- `VITE_ERROR_API_KEY` — auth header.
- `VITE_BUILD_VERSION` — attached to every report.

In development (`import.meta.env.DEV`), the logger also writes to `console.error`.

## Conventions

- **Always toast on mutation failure.** Users should know whether their save succeeded; silent errors are worse than verbose ones.
- **Never swallow the error object.** If a catch block can't act, re-throw.
- **Use `createErrorFromUnknown` before touching the message.** Raw errors from deep in the stack may leak stack traces or sensitive payload fields.
- **Prefer a React Query hook's error state over a try/catch.** The hook already owns retry + staleness; duplicating the logic in a component creates race conditions.
- **Don't wrap `<RouteErrorBoundary>` inside individual pages.** One boundary at the root is enough and lets the whole app recover via a single "Try Again" click.
