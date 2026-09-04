import { Suspense } from "react";
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  useParams,
} from "react-router-dom";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { TenantProvider } from "@/context/tenant-context";
import { Toaster } from "@/components/ui/sonner";
import { RouteErrorBoundary } from "@/components/common/error-boundary";
import { NotFoundPage } from "@/components/common/not-found-page";
import { AppShell } from "@/components/features/navigation/app-shell";
import { PageLoader } from "@/components/common/PageLoader";
import { lazyWithReload } from "@/lib/utils/lazy-with-reload";

// Dashboard is eagerly imported — always the first route a user sees.
import { DashboardPage } from "@/pages/DashboardPage";

// Everything else is split into its own chunk. lazyWithReload (not React.lazy)
// so a tab left open across a redeploy recovers from the now-missing
// content-hashed chunk instead of dead-ending in the error boundary.
const AccountsPage = lazyWithReload(() =>
  import("@/pages/AccountsPage").then((m) => ({ default: m.AccountsPage })),
);
const AccountDetailPage = lazyWithReload(() =>
  import("@/pages/AccountDetailPage").then((m) => ({
    default: m.AccountDetailPage,
  })),
);
const BansPage = lazyWithReload(() =>
  import("@/pages/BansPage").then((m) => ({ default: m.BansPage })),
);
const BanDetailPage = lazyWithReload(() =>
  import("@/pages/BanDetailPage").then((m) => ({ default: m.BanDetailPage })),
);
const ReportsPage = lazyWithReload(() =>
  import("@/pages/ReportsPage").then((m) => ({ default: m.ReportsPage })),
);
const ReportDetailPage = lazyWithReload(() =>
  import("@/pages/ReportDetailPage").then((m) => ({
    default: m.ReportDetailPage,
  })),
);
const BaselinesPage = lazyWithReload(() =>
  import("@/pages/BaselinesPage").then((m) => ({ default: m.BaselinesPage })),
);
const CharactersPage = lazyWithReload(() =>
  import("@/pages/CharactersPage").then((m) => ({ default: m.CharactersPage })),
);
const CharacterDetailPage = lazyWithReload(() =>
  import("@/pages/CharacterDetailPage").then((m) => ({
    default: m.CharacterDetailPage,
  })),
);
const RewardPoolsPage = lazyWithReload(() =>
  import("@/pages/RewardPoolsPage").then((m) => ({
    default: m.RewardPoolsPage,
  })),
);
const RewardPoolDetailPage = lazyWithReload(() =>
  import("@/pages/RewardPoolDetailPage").then((m) => ({
    default: m.RewardPoolDetailPage,
  })),
);
const CouponsPage = lazyWithReload(() =>
  import("@/pages/CouponsPage").then((m) => ({ default: m.CouponsPage })),
);
const EventDefinitionsPage = lazyWithReload(() =>
  import("@/pages/EventDefinitionsPage").then((m) => ({
    default: m.EventDefinitionsPage,
  })),
);
const EventOccurrencesPage = lazyWithReload(() =>
  import("@/pages/EventOccurrencesPage").then((m) => ({
    default: m.EventOccurrencesPage,
  })),
);
const EventOccurrenceDetailPage = lazyWithReload(() =>
  import("@/pages/EventOccurrenceDetailPage").then((m) => ({
    default: m.EventOccurrenceDetailPage,
  })),
);
const CouponDetailPage = lazyWithReload(() =>
  import("@/pages/CouponDetailPage").then((m) => ({
    default: m.CouponDetailPage,
  })),
);
const GuildsPage = lazyWithReload(() =>
  import("@/pages/GuildsPage").then((m) => ({ default: m.GuildsPage })),
);
const GuildDetailPage = lazyWithReload(() =>
  import("@/pages/GuildDetailPage").then((m) => ({
    default: m.GuildDetailPage,
  })),
);
const ItemsPage = lazyWithReload(() =>
  import("@/pages/ItemsPage").then((m) => ({ default: m.ItemsPage })),
);
const ItemDetailPage = lazyWithReload(() =>
  import("@/pages/ItemDetailPage").then((m) => ({ default: m.ItemDetailPage })),
);
const JobsPage = lazyWithReload(() =>
  import("@/pages/JobsPage").then((m) => ({ default: m.JobsPage })),
);
const LoginHistoryPage = lazyWithReload(() =>
  import("@/pages/LoginHistoryPage").then((m) => ({
    default: m.LoginHistoryPage,
  })),
);
const MapsPage = lazyWithReload(() =>
  import("@/pages/MapsPage").then((m) => ({ default: m.MapsPage })),
);
const MapDetailPage = lazyWithReload(() =>
  import("@/pages/MapDetailPage").then((m) => ({ default: m.MapDetailPage })),
);
const PortalDetailPage = lazyWithReload(() =>
  import("@/pages/PortalDetailPage").then((m) => ({
    default: m.PortalDetailPage,
  })),
);
const FieldsPage = lazyWithReload(() =>
  import("@/pages/FieldsPage").then((m) => ({ default: m.FieldsPage })),
);
const MerchantsPage = lazyWithReload(() =>
  import("@/pages/MerchantsPage").then((m) => ({ default: m.MerchantsPage })),
);
const MarketplacePage = lazyWithReload(() =>
  import("@/pages/MarketplacePage").then((m) => ({
    default: m.MarketplacePage,
  })),
);
const MerchantDetailPage = lazyWithReload(() =>
  import("@/pages/MerchantDetailPage").then((m) => ({
    default: m.MerchantDetailPage,
  })),
);
const MonstersPage = lazyWithReload(() =>
  import("@/pages/MonstersPage").then((m) => ({ default: m.MonstersPage })),
);
const MonsterDetailPage = lazyWithReload(() =>
  import("@/pages/MonsterDetailPage").then((m) => ({
    default: m.MonsterDetailPage,
  })),
);
const NpcsPage = lazyWithReload(() =>
  import("@/pages/NpcsPage").then((m) => ({ default: m.NpcsPage })),
);
const NpcDetailPage = lazyWithReload(() =>
  import("@/pages/NpcDetailPage").then((m) => ({ default: m.NpcDetailPage })),
);
const RankingsPage = lazyWithReload(() =>
  import("@/pages/RankingsPage").then((m) => ({
    default: m.RankingsPage,
  })),
);
const QuestsPage = lazyWithReload(() =>
  import("@/pages/QuestsPage").then((m) => ({ default: m.QuestsPage })),
);
const QuestDetailPage = lazyWithReload(() =>
  import("@/pages/QuestDetailPage").then((m) => ({
    default: m.QuestDetailPage,
  })),
);
const ReactorsPage = lazyWithReload(() =>
  import("@/pages/ReactorsPage").then((m) => ({ default: m.ReactorsPage })),
);
const ReactorDetailPage = lazyWithReload(() =>
  import("@/pages/ReactorDetailPage").then((m) => ({
    default: m.ReactorDetailPage,
  })),
);
const ServicesPage = lazyWithReload(() =>
  import("@/pages/ServicesPage").then((m) => ({ default: m.ServicesPage })),
);
const ServiceDetailPage = lazyWithReload(() =>
  import("@/pages/ServiceDetailPage").then((m) => ({
    default: m.ServiceDetailPage,
  })),
);
const SetupPage = lazyWithReload(() =>
  import("@/pages/SetupPage").then((m) => ({ default: m.SetupPage })),
);
const PacketMatrixPage = lazyWithReload(() =>
  import("@/pages/PacketMatrixPage").then((m) => ({
    default: m.PacketMatrixPage,
  })),
);
const TemplatesPage = lazyWithReload(() =>
  import("@/pages/TemplatesPage").then((m) => ({ default: m.TemplatesPage })),
);
const TemplateDetailPage = lazyWithReload(() =>
  import("@/pages/TemplateDetailPage").then((m) => ({
    default: m.TemplateDetailPage,
  })),
);
const TemplatesHandlersPage = lazyWithReload(() =>
  import("@/pages/TemplatesHandlersPage").then((m) => ({
    default: m.TemplatesHandlersPage,
  })),
);
const TemplatesWorldsPage = lazyWithReload(() =>
  import("@/pages/TemplatesWorldsPage").then((m) => ({
    default: m.TemplatesWorldsPage,
  })),
);
const TemplatesWritersPage = lazyWithReload(() =>
  import("@/pages/TemplatesWritersPage").then((m) => ({
    default: m.TemplatesWritersPage,
  })),
);
const TemplatesPropertiesPage = lazyWithReload(() =>
  import("@/pages/TemplatesPropertiesPage").then((m) => ({
    default: m.TemplatesPropertiesPage,
  })),
);
const TemplatesCharacterTemplatesPage = lazyWithReload(() =>
  import("@/pages/TemplatesCharacterTemplatesPage").then((m) => ({
    default: m.TemplatesCharacterTemplatesPage,
  })),
);
const TemplatesCharacterPresetsPage = lazyWithReload(() =>
  import("@/pages/TemplatesCharacterPresetsPage").then((m) => ({
    default: m.TemplatesCharacterPresetsPage,
  })),
);
const TemplatesMapleLifePage = lazyWithReload(() =>
  import("@/pages/TemplatesMapleLifePage").then((m) => ({
    default: m.TemplatesMapleLifePage,
  })),
);
const TenantsPage = lazyWithReload(() =>
  import("@/pages/TenantsPage").then((m) => ({ default: m.TenantsPage })),
);
const TenantDetailPage = lazyWithReload(() =>
  import("@/pages/TenantDetailPage").then((m) => ({
    default: m.TenantDetailPage,
  })),
);
const TenantsHandlersPage = lazyWithReload(() =>
  import("@/pages/TenantsHandlersPage").then((m) => ({
    default: m.TenantsHandlersPage,
  })),
);
const TenantsWorldsPage = lazyWithReload(() =>
  import("@/pages/TenantsWorldsPage").then((m) => ({
    default: m.TenantsWorldsPage,
  })),
);
const TenantsWritersPage = lazyWithReload(() =>
  import("@/pages/TenantsWritersPage").then((m) => ({
    default: m.TenantsWritersPage,
  })),
);
const TenantsPropertiesPage = lazyWithReload(() =>
  import("@/pages/TenantsPropertiesPage").then((m) => ({
    default: m.TenantsPropertiesPage,
  })),
);
const TenantsCharacterTemplatesPage = lazyWithReload(() =>
  import("@/pages/TenantsCharacterTemplatesPage").then((m) => ({
    default: m.TenantsCharacterTemplatesPage,
  })),
);
const TenantsCharacterPresetsPage = lazyWithReload(() =>
  import("@/pages/TenantsCharacterPresetsPage").then((m) => ({
    default: m.TenantsCharacterPresetsPage,
  })),
);
const TenantsMapleLifePage = lazyWithReload(() =>
  import("@/pages/TenantsMapleLifePage").then((m) => ({
    default: m.TenantsMapleLifePage,
  })),
);
const TenantsDiagnosticsPage = lazyWithReload(() =>
  import("@/pages/TenantsDiagnosticsPage").then((m) => ({
    default: m.TenantsDiagnosticsPage,
  })),
);
const TenantsMtsConfigPage = lazyWithReload(() =>
  import("@/pages/TenantsMtsConfigPage").then((m) => ({
    default: m.TenantsMtsConfigPage,
  })),
);
const TransportsPage = lazyWithReload(() =>
  import("@/pages/TransportsPage").then((m) => ({
    default: m.TransportsPage,
  })),
);
const TransportRouteDetailPage = lazyWithReload(() =>
  import("@/pages/TransportRouteDetailPage").then((m) => ({
    default: m.TransportRouteDetailPage,
  })),
);

function GachaponRedirect() {
  const { id } = useParams();
  return <Navigate to={`/reward-pools/${id}`} replace />;
}

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
                    <Route
                      path="/accounts/:id"
                      element={<AccountDetailPage />}
                    />
                    <Route path="/bans" element={<BansPage />} />
                    <Route path="/bans/:banId" element={<BanDetailPage />} />
                    <Route path="/reports" element={<ReportsPage />} />
                    <Route
                      path="/reports/:reportId"
                      element={<ReportDetailPage />}
                    />
                    <Route path="/baselines" element={<BaselinesPage />} />
                    <Route path="/characters" element={<CharactersPage />} />
                    <Route
                      path="/characters/:id"
                      element={<CharacterDetailPage />}
                    />
                    <Route path="/coupons" element={<CouponsPage />} />
                    <Route
                      path="/coupons/:couponId"
                      element={<CouponDetailPage />}
                    />
                    <Route
                      path="/events/definitions"
                      element={<EventDefinitionsPage />}
                    />
                    <Route
                      path="/events/occurrences"
                      element={<EventOccurrencesPage />}
                    />
                    <Route
                      path="/events/occurrences/:id"
                      element={<EventOccurrenceDetailPage />}
                    />
                    <Route path="/reward-pools" element={<RewardPoolsPage />} />
                    <Route
                      path="/reward-pools/:id"
                      element={<RewardPoolDetailPage />}
                    />
                    <Route
                      path="/gachapons"
                      element={<Navigate to="/reward-pools" replace />}
                    />
                    <Route
                      path="/gachapons/:id"
                      element={<GachaponRedirect />}
                    />
                    <Route path="/guilds" element={<GuildsPage />} />
                    <Route path="/guilds/:id" element={<GuildDetailPage />} />
                    <Route path="/items" element={<ItemsPage />} />
                    <Route path="/items/:id" element={<ItemDetailPage />} />
                    <Route path="/jobs" element={<JobsPage />} />
                    <Route path="/jobs/:jobId" element={<JobsPage />} />
                    <Route
                      path="/login-history"
                      element={<LoginHistoryPage />}
                    />
                    <Route path="/maps" element={<MapsPage />} />
                    <Route path="/maps/:id" element={<MapDetailPage />} />
                    <Route
                      path="/maps/:id/portals/:portalId"
                      element={<PortalDetailPage />}
                    />
                    <Route path="/fields" element={<FieldsPage />} />
                    <Route path="/transports" element={<TransportsPage />} />
                    <Route
                      path="/transports/routes/:routeId"
                      element={<TransportRouteDetailPage />}
                    />
                    <Route path="/merchants" element={<MerchantsPage />} />
                    <Route
                      path="/merchants/:id"
                      element={<MerchantDetailPage />}
                    />
                    <Route path="/marketplace" element={<MarketplacePage />} />
                    <Route path="/monsters" element={<MonstersPage />} />
                    <Route
                      path="/monsters/:id"
                      element={<MonsterDetailPage />}
                    />
                    <Route path="/npcs" element={<NpcsPage />} />
                    <Route path="/npcs/:id" element={<NpcDetailPage />} />
                    <Route path="/rankings" element={<RankingsPage />} />
                    <Route path="/quests" element={<QuestsPage />} />
                    <Route path="/quests/:id" element={<QuestDetailPage />} />
                    <Route path="/reactors" element={<ReactorsPage />} />
                    <Route
                      path="/reactors/:id"
                      element={<ReactorDetailPage />}
                    />
                    <Route path="/services" element={<ServicesPage />} />
                    <Route
                      path="/services/:id"
                      element={<ServiceDetailPage />}
                    />
                    <Route path="/setup" element={<SetupPage />} />
                    <Route path="/templates" element={<TemplatesPage />} />
                    <Route
                      path="/templates/:id"
                      element={<TemplateDetailPage />}
                    />
                    <Route
                      path="/templates/:id/handlers"
                      element={<TemplatesHandlersPage />}
                    />
                    <Route
                      path="/templates/:id/worlds"
                      element={<TemplatesWorldsPage />}
                    />
                    <Route
                      path="/templates/:id/writers"
                      element={<TemplatesWritersPage />}
                    />
                    <Route
                      path="/templates/:id/properties"
                      element={<TemplatesPropertiesPage />}
                    />
                    <Route
                      path="/templates/:id/character/templates"
                      element={<TemplatesCharacterTemplatesPage />}
                    />
                    <Route
                      path="/templates/:id/character/presets"
                      element={<TemplatesCharacterPresetsPage />}
                    />
                    <Route
                      path="/templates/:id/character/maple-life"
                      element={<TemplatesMapleLifePage />}
                    />
                    <Route path="/tenants" element={<TenantsPage />} />
                    <Route
                      path="/packet-matrix"
                      element={<PacketMatrixPage />}
                    />
                    <Route path="/tenants/:id" element={<TenantDetailPage />} />
                    <Route
                      path="/tenants/:id/handlers"
                      element={<TenantsHandlersPage />}
                    />
                    <Route
                      path="/tenants/:id/worlds"
                      element={<TenantsWorldsPage />}
                    />
                    <Route
                      path="/tenants/:id/writers"
                      element={<TenantsWritersPage />}
                    />
                    <Route
                      path="/tenants/:id/properties"
                      element={<TenantsPropertiesPage />}
                    />
                    <Route
                      path="/tenants/:id/character/templates"
                      element={<TenantsCharacterTemplatesPage />}
                    />
                    <Route
                      path="/tenants/:id/character/presets"
                      element={<TenantsCharacterPresetsPage />}
                    />
                    <Route
                      path="/tenants/:id/character/maple-life"
                      element={<TenantsMapleLifePage />}
                    />
                    <Route
                      path="/tenants/:id/mts-config"
                      element={<TenantsMtsConfigPage />}
                    />
                    <Route
                      path="/tenants/:id/diagnostics"
                      element={<TenantsDiagnosticsPage />}
                    />
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
