import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { TenantProvider } from "@/context/tenant-context";
import { Toaster } from "@/components/ui/sonner";
import { RouteErrorBoundary } from "@/components/common/error-boundary";
import { NotFoundPage } from "@/components/common/not-found-page";

export function App() {
  return (
    <BrowserRouter>
      <QueryProvider>
        <ThemeProvider>
          <TenantProvider>
            <Toaster />
            <RouteErrorBoundary>
              <Routes>
                <Route path="/" element={<div className="p-4">AtlasMS (Vite scaffold)</div>} />
                <Route path="*" element={<NotFoundPage />} />
              </Routes>
            </RouteErrorBoundary>
          </TenantProvider>
        </ThemeProvider>
      </QueryProvider>
    </BrowserRouter>
  );
}
