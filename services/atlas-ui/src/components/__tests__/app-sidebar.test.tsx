import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AppSidebar } from '@/components/app-sidebar';
import { sidebarItems } from '@/components/app-sidebar-items';
import { SidebarProvider } from '@/components/ui/sidebar';
import { isDeploymentRoute } from '@/lib/deployment-routes';

vi.mock('@/components/app-tenant-switcher', () => ({
  TenantSwitcher: () => <div data-testid="tenant-switcher-stub" />,
}));

beforeAll(() => {
  // SidebarProvider's mobile detection needs matchMedia, absent in jsdom.
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function renderSidebar(initialPath = '/') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <SidebarProvider>
        <AppSidebar />
      </SidebarProvider>
    </MemoryRouter>,
  );
}

describe('AppSidebar', () => {
  it('declares groups in blast-radius order with the Deployment children ordered', () => {
    expect(sidebarItems.map((g) => g.title)).toEqual(['Operations', 'Security', 'Setup', 'Deployment']);
    const deployment = sidebarItems[3]!;
    expect(deployment.children.map((c) => c.title)).toEqual(['Templates', 'Tenants', 'Services', 'Baselines']);
    expect(deployment.separated).toBe(true);
    expect(deployment.caption).toBe('Applies to all tenants');
    const setup = sidebarItems[2]!;
    expect(setup.children).toEqual([{ title: 'Setup', url: '/setup' }]);
  });

  it('keeps the sidebar and the route predicate in sync', () => {
    const deployment = sidebarItems.find((g) => g.title === 'Deployment')!;
    for (const child of deployment.children) {
      expect(isDeploymentRoute(child.url), `${child.url} must be a Deployment route`).toBe(true);
    }
    for (const group of sidebarItems.filter((g) => g.title !== 'Deployment')) {
      for (const child of group.children) {
        expect(isDeploymentRoute(child.url), `${child.url} must NOT be a Deployment route`).toBe(false);
      }
    }
  });

  it('renders the Deployment caption', () => {
    renderSidebar();
    expect(screen.getByText('Applies to all tenants')).toBeInTheDocument();
  });

  it('renders the Baselines link', () => {
    renderSidebar('/baselines');
    expect(screen.getByRole('link', { name: 'Baselines' })).toHaveAttribute('href', '/baselines');
  });
});
