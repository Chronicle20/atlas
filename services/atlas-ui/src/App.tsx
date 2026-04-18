import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { TenantProvider } from "@/context/tenant-context";
import { Toaster } from "@/components/ui/sonner";
import { RouteErrorBoundary } from "@/components/common/error-boundary";
import { NotFoundPage } from "@/components/common/not-found-page";
import { AppShell } from "@/components/features/navigation/app-shell";

import { DashboardPage } from "@/pages/DashboardPage";
import { AccountsPage } from "@/pages/AccountsPage";
import { AccountDetailPage } from "@/pages/AccountDetailPage";
import { BansPage } from "@/pages/BansPage";
import { BanDetailPage } from "@/pages/BanDetailPage";
import { CharactersPage } from "@/pages/CharactersPage";
import { CharacterDetailPage } from "@/pages/CharacterDetailPage";
import { GachaponsPage } from "@/pages/GachaponsPage";
import { GachaponDetailPage } from "@/pages/GachaponDetailPage";
import { GuildsPage } from "@/pages/GuildsPage";
import { GuildDetailPage } from "@/pages/GuildDetailPage";
import { ItemsPage } from "@/pages/ItemsPage";
import { ItemDetailPage } from "@/pages/ItemDetailPage";
import { LoginHistoryPage } from "@/pages/LoginHistoryPage";
import { MapsPage } from "@/pages/MapsPage";
import { MapDetailPage } from "@/pages/MapDetailPage";
import { PortalDetailPage } from "@/pages/PortalDetailPage";
import { MerchantsPage } from "@/pages/MerchantsPage";
import { MerchantDetailPage } from "@/pages/MerchantDetailPage";
import { MonstersPage } from "@/pages/MonstersPage";
import { MonsterDetailPage } from "@/pages/MonsterDetailPage";
import { NpcsPage } from "@/pages/NpcsPage";
import { NpcDetailPage } from "@/pages/NpcDetailPage";
import { NpcShopPage } from "@/pages/NpcShopPage";
import { NpcConversationPage } from "@/pages/NpcConversationPage";
import { QuestsPage } from "@/pages/QuestsPage";
import { QuestDetailPage } from "@/pages/QuestDetailPage";
import { ReactorsPage } from "@/pages/ReactorsPage";
import { ReactorDetailPage } from "@/pages/ReactorDetailPage";
import { ServicesPage } from "@/pages/ServicesPage";
import { ServiceDetailPage } from "@/pages/ServiceDetailPage";
import { SetupPage } from "@/pages/SetupPage";
import { TemplatesPage } from "@/pages/TemplatesPage";
import { TemplateDetailPage } from "@/pages/TemplateDetailPage";
import { TemplatesHandlersPage } from "@/pages/TemplatesHandlersPage";
import { TemplatesWorldsPage } from "@/pages/TemplatesWorldsPage";
import { TemplatesWritersPage } from "@/pages/TemplatesWritersPage";
import { TemplatesPropertiesPage } from "@/pages/TemplatesPropertiesPage";
import { TemplatesCharacterTemplatesPage } from "@/pages/TemplatesCharacterTemplatesPage";
import { TenantsPage } from "@/pages/TenantsPage";
import { TenantDetailPage } from "@/pages/TenantDetailPage";
import { TenantsHandlersPage } from "@/pages/TenantsHandlersPage";
import { TenantsWorldsPage } from "@/pages/TenantsWorldsPage";
import { TenantsWritersPage } from "@/pages/TenantsWritersPage";
import { TenantsPropertiesPage } from "@/pages/TenantsPropertiesPage";
import { TenantsCharacterTemplatesPage } from "@/pages/TenantsCharacterTemplatesPage";

export function App() {
  return (
    <BrowserRouter>
      <QueryProvider>
        <ThemeProvider>
          <TenantProvider>
            <Toaster />
            <RouteErrorBoundary>
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
            </RouteErrorBoundary>
          </TenantProvider>
        </ThemeProvider>
      </QueryProvider>
    </BrowserRouter>
  );
}
