import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { TenantProvider } from "@/context/tenant-context";
import { Toaster } from "@/components/ui/sonner";
import { RouteErrorBoundary } from "@/components/common/error-boundary";
import { NotFoundPage } from "@/components/common/not-found-page";
import { AppShell } from "@/components/features/navigation/app-shell";
import { PageLoader } from "@/components/common/PageLoader";

// Dashboard is eagerly imported — always the first route a user sees.
import { DashboardPage } from "@/pages/DashboardPage";

// Everything else is split into its own chunk.
const AccountsPage = lazy(() => import("@/pages/AccountsPage").then(m => ({ default: m.AccountsPage })));
const AccountDetailPage = lazy(() => import("@/pages/AccountDetailPage").then(m => ({ default: m.AccountDetailPage })));
const BansPage = lazy(() => import("@/pages/BansPage").then(m => ({ default: m.BansPage })));
const BanDetailPage = lazy(() => import("@/pages/BanDetailPage").then(m => ({ default: m.BanDetailPage })));
const CharactersPage = lazy(() => import("@/pages/CharactersPage").then(m => ({ default: m.CharactersPage })));
const CharacterDetailPage = lazy(() => import("@/pages/CharacterDetailPage").then(m => ({ default: m.CharacterDetailPage })));
const GachaponsPage = lazy(() => import("@/pages/GachaponsPage").then(m => ({ default: m.GachaponsPage })));
const GachaponDetailPage = lazy(() => import("@/pages/GachaponDetailPage").then(m => ({ default: m.GachaponDetailPage })));
const GuildsPage = lazy(() => import("@/pages/GuildsPage").then(m => ({ default: m.GuildsPage })));
const GuildDetailPage = lazy(() => import("@/pages/GuildDetailPage").then(m => ({ default: m.GuildDetailPage })));
const ItemsPage = lazy(() => import("@/pages/ItemsPage").then(m => ({ default: m.ItemsPage })));
const ItemDetailPage = lazy(() => import("@/pages/ItemDetailPage").then(m => ({ default: m.ItemDetailPage })));
const LoginHistoryPage = lazy(() => import("@/pages/LoginHistoryPage").then(m => ({ default: m.LoginHistoryPage })));
const MapsPage = lazy(() => import("@/pages/MapsPage").then(m => ({ default: m.MapsPage })));
const MapDetailPage = lazy(() => import("@/pages/MapDetailPage").then(m => ({ default: m.MapDetailPage })));
const PortalDetailPage = lazy(() => import("@/pages/PortalDetailPage").then(m => ({ default: m.PortalDetailPage })));
const MerchantsPage = lazy(() => import("@/pages/MerchantsPage").then(m => ({ default: m.MerchantsPage })));
const MerchantDetailPage = lazy(() => import("@/pages/MerchantDetailPage").then(m => ({ default: m.MerchantDetailPage })));
const MonstersPage = lazy(() => import("@/pages/MonstersPage").then(m => ({ default: m.MonstersPage })));
const MonsterDetailPage = lazy(() => import("@/pages/MonsterDetailPage").then(m => ({ default: m.MonsterDetailPage })));
const NpcsPage = lazy(() => import("@/pages/NpcsPage").then(m => ({ default: m.NpcsPage })));
const NpcDetailPage = lazy(() => import("@/pages/NpcDetailPage").then(m => ({ default: m.NpcDetailPage })));
const NpcShopPage = lazy(() => import("@/pages/NpcShopPage").then(m => ({ default: m.NpcShopPage })));
const NpcConversationPage = lazy(() => import("@/pages/NpcConversationPage").then(m => ({ default: m.NpcConversationPage })));
const QuestsPage = lazy(() => import("@/pages/QuestsPage").then(m => ({ default: m.QuestsPage })));
const QuestDetailPage = lazy(() => import("@/pages/QuestDetailPage").then(m => ({ default: m.QuestDetailPage })));
const ReactorsPage = lazy(() => import("@/pages/ReactorsPage").then(m => ({ default: m.ReactorsPage })));
const ReactorDetailPage = lazy(() => import("@/pages/ReactorDetailPage").then(m => ({ default: m.ReactorDetailPage })));
const ServicesPage = lazy(() => import("@/pages/ServicesPage").then(m => ({ default: m.ServicesPage })));
const ServiceDetailPage = lazy(() => import("@/pages/ServiceDetailPage").then(m => ({ default: m.ServiceDetailPage })));
const SetupPage = lazy(() => import("@/pages/SetupPage").then(m => ({ default: m.SetupPage })));
const TemplatesPage = lazy(() => import("@/pages/TemplatesPage").then(m => ({ default: m.TemplatesPage })));
const TemplateDetailPage = lazy(() => import("@/pages/TemplateDetailPage").then(m => ({ default: m.TemplateDetailPage })));
const TemplatesHandlersPage = lazy(() => import("@/pages/TemplatesHandlersPage").then(m => ({ default: m.TemplatesHandlersPage })));
const TemplatesWorldsPage = lazy(() => import("@/pages/TemplatesWorldsPage").then(m => ({ default: m.TemplatesWorldsPage })));
const TemplatesWritersPage = lazy(() => import("@/pages/TemplatesWritersPage").then(m => ({ default: m.TemplatesWritersPage })));
const TemplatesPropertiesPage = lazy(() => import("@/pages/TemplatesPropertiesPage").then(m => ({ default: m.TemplatesPropertiesPage })));
const TemplatesCharacterTemplatesPage = lazy(() => import("@/pages/TemplatesCharacterTemplatesPage").then(m => ({ default: m.TemplatesCharacterTemplatesPage })));
const TenantsPage = lazy(() => import("@/pages/TenantsPage").then(m => ({ default: m.TenantsPage })));
const TenantDetailPage = lazy(() => import("@/pages/TenantDetailPage").then(m => ({ default: m.TenantDetailPage })));
const TenantsHandlersPage = lazy(() => import("@/pages/TenantsHandlersPage").then(m => ({ default: m.TenantsHandlersPage })));
const TenantsWorldsPage = lazy(() => import("@/pages/TenantsWorldsPage").then(m => ({ default: m.TenantsWorldsPage })));
const TenantsWritersPage = lazy(() => import("@/pages/TenantsWritersPage").then(m => ({ default: m.TenantsWritersPage })));
const TenantsPropertiesPage = lazy(() => import("@/pages/TenantsPropertiesPage").then(m => ({ default: m.TenantsPropertiesPage })));
const TenantsCharacterTemplatesPage = lazy(() => import("@/pages/TenantsCharacterTemplatesPage").then(m => ({ default: m.TenantsCharacterTemplatesPage })));

export function App() {
  return (
    <BrowserRouter>
      <QueryProvider>
        <ThemeProvider>
          <TenantProvider>
            <Toaster />
            <RouteErrorBoundary>
              <Suspense fallback={<PageLoader />}>
                <Routes>
                  <Route element={<AppShell />}>
                    <Route index element={<DashboardPage />} />
                    <Route path="/accounts" element={<AccountsPage />} />
                    <Route path="/accounts/:id" element={<AccountDetailPage />} />
                    <Route path="/bans" element={<BansPage />} />
                    <Route path="/bans/:banId" element={<BanDetailPage />} />
                    <Route path="/characters" element={<CharactersPage />} />
                    <Route path="/characters/:id" element={<CharacterDetailPage />} />
                    <Route path="/gachapons" element={<GachaponsPage />} />
                    <Route path="/gachapons/:id" element={<GachaponDetailPage />} />
                    <Route path="/guilds" element={<GuildsPage />} />
                    <Route path="/guilds/:id" element={<GuildDetailPage />} />
                    <Route path="/items" element={<ItemsPage />} />
                    <Route path="/items/:id" element={<ItemDetailPage />} />
                    <Route path="/login-history" element={<LoginHistoryPage />} />
                    <Route path="/maps" element={<MapsPage />} />
                    <Route path="/maps/:id" element={<MapDetailPage />} />
                    <Route path="/maps/:id/portals/:portalId" element={<PortalDetailPage />} />
                    <Route path="/merchants" element={<MerchantsPage />} />
                    <Route path="/merchants/:id" element={<MerchantDetailPage />} />
                    <Route path="/monsters" element={<MonstersPage />} />
                    <Route path="/monsters/:id" element={<MonsterDetailPage />} />
                    <Route path="/npcs" element={<NpcsPage />} />
                    <Route path="/npcs/:id" element={<NpcDetailPage />} />
                    <Route path="/npcs/:id/conversations" element={<NpcConversationPage />} />
                    <Route path="/npcs/:id/shop" element={<NpcShopPage />} />
                    <Route path="/quests" element={<QuestsPage />} />
                    <Route path="/quests/:id" element={<QuestDetailPage />} />
                    <Route path="/reactors" element={<ReactorsPage />} />
                    <Route path="/reactors/:id" element={<ReactorDetailPage />} />
                    <Route path="/services" element={<ServicesPage />} />
                    <Route path="/services/:id" element={<ServiceDetailPage />} />
                    <Route path="/setup" element={<SetupPage />} />
                    <Route path="/templates" element={<TemplatesPage />} />
                    <Route path="/templates/:id" element={<TemplateDetailPage />} />
                    <Route path="/templates/:id/handlers" element={<TemplatesHandlersPage />} />
                    <Route path="/templates/:id/worlds" element={<TemplatesWorldsPage />} />
                    <Route path="/templates/:id/writers" element={<TemplatesWritersPage />} />
                    <Route path="/templates/:id/properties" element={<TemplatesPropertiesPage />} />
                    <Route path="/templates/:id/character/templates" element={<TemplatesCharacterTemplatesPage />} />
                    <Route path="/tenants" element={<TenantsPage />} />
                    <Route path="/tenants/:id" element={<TenantDetailPage />} />
                    <Route path="/tenants/:id/handlers" element={<TenantsHandlersPage />} />
                    <Route path="/tenants/:id/worlds" element={<TenantsWorldsPage />} />
                    <Route path="/tenants/:id/writers" element={<TenantsWritersPage />} />
                    <Route path="/tenants/:id/properties" element={<TenantsPropertiesPage />} />
                    <Route path="/tenants/:id/character/templates" element={<TenantsCharacterTemplatesPage />} />
                  </Route>
                  <Route path="*" element={<NotFoundPage />} />
                </Routes>
              </Suspense>
            </RouteErrorBoundary>
          </TenantProvider>
        </ThemeProvider>
      </QueryProvider>
    </BrowserRouter>
  );
}
