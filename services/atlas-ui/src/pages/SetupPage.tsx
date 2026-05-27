
import { useRef, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Loader2,
  Upload,
  Database,
  MessageSquare,
  Store,
  DoorOpen,
  Zap,
  Map,
  Package,
  HelpCircle,
  FileArchive,
  FileText,
  RotateCcw,
  Send,
} from "lucide-react";
import { Toaster, toast } from "sonner";
import {
  useSeedDrops,
  useSeedGachapons,
  useSeedNpcConversations,
  useSeedQuestConversations,
  useSeedNpcShops,
  useSeedPortalScripts,
  useSeedReactorScripts,
  useSeedMapActionScripts,
  useUploadWzFiles,
  useRunDataProcessing,
  useWzInputStatus,
  useDataStatus,
  useDropsSeedStatus,
  useGachaponsSeedStatus,
  useNpcConversationsSeedStatus,
  useQuestConversationsSeedStatus,
  useNpcShopsSeedStatus,
  usePortalScriptsSeedStatus,
  useReactorScriptsSeedStatus,
  useMapActionScriptsSeedStatus,
} from "@/lib/hooks/api/useSeed";
import type {
  DropsSeedStatus,
  GachaponsSeedStatus,
  NpcConversationsSeedStatus,
  QuestConversationsSeedStatus,
  NpcShopsSeedStatus,
  PortalScriptsSeedStatus,
  ReactorScriptsSeedStatus,
  MapActionScriptsSeedStatus,
} from "@/services/api/seed.service";
import { SetupRow, formatCount, pluralize } from "@/components/features/setup/SetupRow";
import { ScopeToggle, type Scope } from "@/components/features/setup/ScopeToggle";
import { useRestoreBaseline, usePublishBaseline } from "@/lib/hooks/api/useBaseline";
import { useTenant } from "@/context/tenant-context";

function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  const formatted = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value >= 10 || unit === 0 ? 0 : 1,
  }).format(value);
  return `${formatted} ${units[unit]}`;
}

export function SetupPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { activeTenant } = useTenant();
  const [scope, setScope] = useState<Scope>('tenant');

  const seedDrops = useSeedDrops();
  const seedGachapons = useSeedGachapons();
  const seedNpcConversations = useSeedNpcConversations();
  const seedQuestConversations = useSeedQuestConversations();
  const seedNpcShops = useSeedNpcShops();
  const seedPortalScripts = useSeedPortalScripts();
  const seedReactorScripts = useSeedReactorScripts();
  const seedMapActionScripts = useSeedMapActionScripts();

  const uploadWz = useUploadWzFiles();
  const runProcessing = useRunDataProcessing();

  const wzInput = useWzInputStatus();
  const dataStatus = useDataStatus();

  const dropsSeed = useDropsSeedStatus();
  const gachaponsSeed = useGachaponsSeedStatus();
  const npcConversationsSeed = useNpcConversationsSeedStatus();
  const questConversationsSeed = useQuestConversationsSeedStatus();
  const npcShopsSeed = useNpcShopsSeedStatus();
  const portalScriptsSeed = usePortalScriptsSeedStatus();
  const reactorScriptsSeed = useReactorScriptsSeedStatus();
  const mapActionScriptsSeed = useMapActionScriptsSeedStatus();

  const restoreMutation = useRestoreBaseline(activeTenant);
  const publishMutation = usePublishBaseline(activeTenant);

  const wzInputData = wzInput.data;
  const dataStatusData = dataStatus.data;

  const anyMutationPending = uploadWz.isPending || runProcessing.isPending;

  const ingestDisabledReason = !wzInputData
    ? null
    : wzInputData.fileCount === 0
      ? "Upload WZ files first"
      : null;
  const ingestDisabled =
    !wzInputData || wzInputData.fileCount === 0 || anyMutationPending;

  const handleSeed = (mutation: { mutate: () => void; }, label: string) => {
    mutation.mutate();
    toast.info(`Seeding ${label}...`);
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    if (!file.name.toLowerCase().endsWith('.zip')) {
      toast.error("Please select a .zip file");
      return;
    }

    const size = file.size;
    uploadWz.mutate({ file, scope }, {
      onSuccess: () => {
        toast.success(`WZ files uploaded (${formatBytes(size)})`);
      },
      onError: (error) => {
        const err = error as Error & { status?: number };
        if (err.status === 409) {
          toast.error("Another upload or processing job is in progress for this tenant. Try again in a moment.");
        } else if (err.status === 400) {
          toast.error(`Upload rejected: ${err.message}`);
        } else {
          toast.error(`Upload failed: ${err.message}`);
        }
      },
    });

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleRunProcessing = () => {
    runProcessing.mutate(scope, {
      onSuccess: () => {
        toast.success("Data processing started");
      },
      onError: (error) => {
        toast.error(`Data processing failed: ${error.message}`);
      },
    });
  };

  const handleRestoreBaseline = () => {
    if (!activeTenant) return;
    restoreMutation.mutate(
      {
        region: activeTenant.attributes.region,
        majorVersion: activeTenant.attributes.majorVersion,
        minorVersion: activeTenant.attributes.minorVersion,
        tenantId: activeTenant.id,
      },
      {
        onSuccess: () => {
          toast.success("Canonical baseline restored");
        },
        onError: (error) => {
          toast.error(`Baseline restore failed: ${error.message}`);
        },
      },
    );
  };

  const handlePublishBaseline = () => {
    if (!activeTenant) return;
    publishMutation.mutate(
      {
        region: activeTenant.attributes.region,
        majorVersion: activeTenant.attributes.majorVersion,
        minorVersion: activeTenant.attributes.minorVersion,
      },
      {
        onSuccess: () => {
          toast.success("Canonical baseline published");
        },
        onError: (error) => {
          toast.error(`Baseline publish failed: ${error.message}`);
        },
      },
    );
  };

  const wzInputBadge = !wzInputData ? (
    "—"
  ) : wzInputData.fileCount === 0 ? (
    "0 .wz files"
  ) : (
    `${formatCount(wzInputData.fileCount)} ${pluralize(wzInputData.fileCount, ".wz file", ".wz files")}, ${formatBytes(wzInputData.totalBytes)}`
  );

  const dataStatusBadge = !dataStatusData
    ? "—"
    : `${formatCount(dataStatusData.documentCount)} ${pluralize(dataStatusData.documentCount, "document loaded", "documents loaded")}`;

  const showRestoreRow = dataStatusData?.documentCount === 0;
  const showPublishRow =
    scope === 'shared' && !!dataStatusData && dataStatusData.documentCount > 0;

  const tenantRegion = activeTenant?.attributes.region ?? "";
  const tenantVersion = activeTenant
    ? `${activeTenant.attributes.majorVersion}.${activeTenant.attributes.minorVersion}`
    : "";

  const seedRows = [
    {
      label: "Monster & Reactor Drops",
      icon: <Database className="h-5 w-5" />,
      mutation: seedDrops,
      status: dropsSeed,
      formatBadge: (d?: DropsSeedStatus) =>
        !d
          ? "—"
          : `${formatCount(d.monsterDropCount)} ${pluralize(d.monsterDropCount, "monster drop", "monster drops")} / ${formatCount(d.continentDropCount)} ${pluralize(d.continentDropCount, "continent drop", "continent drops")} / ${formatCount(d.reactorDropCount)} ${pluralize(d.reactorDropCount, "reactor drop", "reactor drops")}`,
    },
    {
      label: "Gachapons",
      icon: <Package className="h-5 w-5" />,
      mutation: seedGachapons,
      status: gachaponsSeed,
      formatBadge: (d?: GachaponsSeedStatus) =>
        !d
          ? "—"
          : `${formatCount(d.gachaponCount)} ${pluralize(d.gachaponCount, "gachapon", "gachapons")} / ${formatCount(d.itemCount)} ${pluralize(d.itemCount, "item", "items")} / ${formatCount(d.globalItemCount)} ${pluralize(d.globalItemCount, "global item", "global items")}`,
    },
    {
      label: "NPC Conversations",
      icon: <MessageSquare className="h-5 w-5" />,
      mutation: seedNpcConversations,
      status: npcConversationsSeed,
      formatBadge: (d?: NpcConversationsSeedStatus) =>
        !d ? "—" : `${formatCount(d.conversationCount)} ${pluralize(d.conversationCount, "conversation", "conversations")}`,
    },
    {
      label: "Quest Conversations",
      icon: <HelpCircle className="h-5 w-5" />,
      mutation: seedQuestConversations,
      status: questConversationsSeed,
      formatBadge: (d?: QuestConversationsSeedStatus) =>
        !d ? "—" : `${formatCount(d.conversationCount)} ${pluralize(d.conversationCount, "conversation", "conversations")}`,
    },
    {
      label: "NPC Shops",
      icon: <Store className="h-5 w-5" />,
      mutation: seedNpcShops,
      status: npcShopsSeed,
      formatBadge: (d?: NpcShopsSeedStatus) =>
        !d ? "—" : `${formatCount(d.shopCount)} ${pluralize(d.shopCount, "shop", "shops")}`,
    },
    {
      label: "Portal Scripts",
      icon: <DoorOpen className="h-5 w-5" />,
      mutation: seedPortalScripts,
      status: portalScriptsSeed,
      formatBadge: (d?: PortalScriptsSeedStatus) =>
        !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
    },
    {
      label: "Reactor Scripts",
      icon: <Zap className="h-5 w-5" />,
      mutation: seedReactorScripts,
      status: reactorScriptsSeed,
      formatBadge: (d?: ReactorScriptsSeedStatus) =>
        !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
    },
    {
      label: "Map Action Scripts",
      icon: <Map className="h-5 w-5" />,
      mutation: seedMapActionScripts,
      status: mapActionScriptsSeed,
      formatBadge: (d?: MapActionScriptsSeedStatus) =>
        !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
    },
  ];

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Bootstrap</h2>
        <p className="text-muted-foreground">Upload a WZ zip and process it into atlas-data, then seed service databases.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Game Data</CardTitle>
          <CardDescription>
            Upload a WZ zip and process it into atlas-data. Choose the tenant scope, or canonical (shared) when publishing a baseline.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <input
            ref={fileInputRef}
            type="file"
            accept=".zip"
            className="hidden"
            onChange={handleFileUpload}
            aria-label="Upload WZ zip archive"
          />

          <div className="mb-4">
            <ScopeToggle
              value={scope}
              onChange={setScope}
              region={tenantRegion}
              version={tenantVersion}
            />
          </div>

          <SetupRow
            icon={<FileArchive className="h-5 w-5" />}
            label="Upload WZ"
            badge={wzInputBadge}
            action={
              <Button
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadWz.isPending || !activeTenant}
              >
                {uploadWz.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Uploading…
                  </>
                ) : (
                  <>
                    <Upload className="mr-2 h-4 w-4" />
                    Upload
                  </>
                )}
              </Button>
            }
          />

          <SetupRow
            icon={<FileText className="h-5 w-5" />}
            label="Process Data"
            badge={dataStatusBadge}
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handleRunProcessing}
                disabled={ingestDisabled || !activeTenant}
                title={ingestDisabledReason ?? (runProcessing.isPending ? "Processing…" : undefined)}
              >
                {runProcessing.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing…
                  </>
                ) : (
                  "Process Data"
                )}
              </Button>
            }
          />

          {showPublishRow && (
            <SetupRow
              icon={<Send className="h-5 w-5" />}
              label="Publish Canonical Baseline"
              badge={
                dataStatusData?.baselineSha256
                  ? `sha256:${dataStatusData.baselineSha256.slice(0, 12)}…`
                  : "—"
              }
              action={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handlePublishBaseline}
                  disabled={publishMutation.isPending || !activeTenant}
                >
                  {publishMutation.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Publishing…
                    </>
                  ) : (
                    "Publish Baseline"
                  )}
                </Button>
              }
            />
          )}

          {showRestoreRow && (
            <SetupRow
              icon={<RotateCcw className="h-5 w-5" />}
              label="Restore Canonical Baseline"
              badge={
                dataStatusData?.baselineRestoredAt
                  ? `restored ${dataStatusData.baselineRestoredAt}`
                  : "no baseline restored"
              }
              action={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleRestoreBaseline}
                  disabled={restoreMutation.isPending || !activeTenant}
                >
                  {restoreMutation.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Restoring…
                    </>
                  ) : (
                    "Restore Baseline"
                  )}
                </Button>
              }
            />
          )}
        </CardContent>
      </Card>

      <div>
        <h3 className="text-lg font-semibold mb-3">Seed Data</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Populate individual service databases from their configured data sources.
        </p>
        <div className="grid gap-0">
          {seedRows.map((row) => (
            <SetupRow
              key={row.label}
              icon={row.icon}
              label={row.label}
              badge={row.formatBadge(row.status.data as never)}
              action={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleSeed(row.mutation, row.label)}
                  disabled={row.mutation.isPending}
                >
                  {row.mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Seed"}
                </Button>
              }
            />
          ))}
        </div>
      </div>

      <Toaster richColors />
    </div>
  );
}
