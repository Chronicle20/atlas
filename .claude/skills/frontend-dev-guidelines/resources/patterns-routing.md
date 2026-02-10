# Routing & Pages Patterns

## Overview

Atlas UI uses the Next.js App Router. All pages use `"use client"` — this is a client-heavy admin application that relies on React hooks and browser APIs rather than server-side data fetching.

## Route Structure

```
app/
├── layout.tsx              # Root layout (server component with providers)
├── page.tsx                # Dashboard home
├── error.tsx               # Root error boundary
├── global-error.tsx        # Global error handler
├── not-found.tsx           # 404 handler
├── accounts/
│   └── page.tsx            # Account list
├── bans/
│   ├── page.tsx            # Ban list
│   └── [banId]/
│       └── page.tsx        # Ban detail
├── characters/
│   ├── page.tsx            # Character list
│   └── [id]/
│       └── page.tsx        # Character detail
├── npcs/
│   ├── page.tsx            # NPC list
│   └── [id]/
│       └── page.tsx        # NPC detail
└── tenants/
    ├── page.tsx            # Tenant list
    └── [id]/
        └── .../page.tsx    # Tenant detail views
```

## List Page Pattern

```tsx
"use client";

import { useState, useEffect, useCallback } from "react";
import { useTenant } from "@/context/tenant-context";
import { DataTable } from "@/components/data-table";

export default function BansPage() {
  const { activeTenant } = useTenant();
  const [bans, setBans] = useState<Ban[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedBan, setSelectedBan] = useState<Ban | null>(null);

  // Memoized fetch function
  const fetchBans = useCallback(async () => {
    if (!activeTenant) return;
    setLoading(true);
    try {
      const data = await bansService.getAllBans(activeTenant);
      setBans(data);
      setError(null);
    } catch (err) {
      const errorInfo = createErrorFromUnknown(err, "Failed to fetch bans");
      setError(errorInfo.message);
    } finally {
      setLoading(false);
    }
  }, [activeTenant]);

  useEffect(() => { fetchBans(); }, [fetchBans]);

  if (loading) return <BansPageSkeleton />;

  return (
    <div className="flex flex-col gap-4 p-4">
      <DataTable
        columns={columns}
        data={bans}
        onRefresh={fetchBans}
        headerActions={[
          { icon: <Plus />, label: "Create Ban", onClick: () => setCreateDialogOpen(true) },
        ]}
      />

      <CreateBanDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        tenant={activeTenant}
        onSuccess={fetchBans}
      />
      <DeleteBanDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        ban={selectedBan}
        tenant={activeTenant}
        onSuccess={fetchBans}
      />
    </div>
  );
}
```

## Detail Page Pattern

```tsx
"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTenant } from "@/context/tenant-context";

export default function BanDetailPage() {
  const { banId } = useParams();             // ← Extract dynamic segment
  const router = useRouter();
  const { activeTenant } = useTenant();

  const [ban, setBan] = useState<Ban | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchBan = useCallback(async () => {
    if (!activeTenant || !banId) return;
    try {
      const data = await bansService.getBanById(activeTenant, banId as string);
      setBan(data);
    } catch (err) {
      toast.error("Failed to load ban");
      router.push("/bans");                   // ← Redirect on failure
    } finally {
      setLoading(false);
    }
  }, [activeTenant, banId, router]);

  useEffect(() => { fetchBan(); }, [fetchBan]);

  if (loading) return <BanDetailSkeleton />;
  if (!ban) return <NotFoundMessage />;

  return (
    <div className="space-y-6 p-4">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => router.push("/bans")}>
          <ArrowLeft className="mr-2 h-4 w-4" /> Back
        </Button>
        <div className="flex gap-2">
          <Button variant="destructive" onClick={() => setDeleteDialogOpen(true)}>
            Delete
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader><CardTitle>Ban Details</CardTitle></CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            {/* Detail fields */}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
```

## Parallel Data Fetching Pattern

For pages requiring multiple data sources:

```tsx
useEffect(() => {
  if (!activeTenant) return;

  setLoading(true);
  Promise.all([
    charactersService.getAll(activeTenant),
    accountsService.getAllAccounts(activeTenant),
    fetchTenantConfiguration(activeTenant.id),
  ])
    .then(([chars, accts, config]) => {
      setCharacters(chars);
      setAccounts(accts);
      setTenantConfig(config);
    })
    .catch((err) => {
      const errorInfo = createErrorFromUnknown(err);
      setError(errorInfo.message);
    })
    .finally(() => setLoading(false));
}, [activeTenant, fetchTenantConfiguration]);
```

## Root Layout

The root layout is the **only server component** — it wraps the entire app with providers:

```tsx
// app/layout.tsx
export const metadata: Metadata = {
  title: "Atlas UI",
  description: "Administrative Web UI",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <TenantProvider>
          <QueryProvider>
            <ThemeProvider attribute="class" defaultTheme="dark">
              <SidebarProvider>
                <AppSidebar />
                <SidebarInset>
                  <header>
                    <SidebarTrigger />
                    <Separator orientation="vertical" />
                    <BreadcrumbBar />
                    <ThemeToggle />
                  </header>
                  <main className="flex-1 overflow-auto p-4">
                    {children}
                  </main>
                </SidebarInset>
              </SidebarProvider>
              <Toaster />
            </ThemeProvider>
          </QueryProvider>
        </TenantProvider>
      </body>
    </html>
  );
}
```

## Error Boundary Pattern

```tsx
// app/error.tsx
"use client";

export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  useEffect(() => {
    errorLogger.logError(error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center gap-4 p-8">
      <Alert variant="destructive">
        <AlertTitle>Something went wrong</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>

      {process.env.NODE_ENV === 'development' && (
        <pre className="text-xs">{error.stack}</pre>
      )}

      <div className="flex gap-2">
        <Button onClick={reset}>Try Again</Button>
        <Button variant="outline" asChild>
          <Link href="/">Go Home</Link>
        </Button>
      </div>
    </div>
  );
}
```

## Navigation Patterns

- **Sidebar:** Static navigation groups defined in `app-sidebar.tsx`
- **Breadcrumbs:** Dynamic resolution via `useBreadcrumbs()` hook with tenant context
- **Back navigation:** `router.push("/parent-route")` for explicit back, `window.history.back()` for browser back
- **Post-action redirect:** `router.push()` after successful delete, `window.location.replace()` after create
