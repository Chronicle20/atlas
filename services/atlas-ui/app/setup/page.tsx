"use client"

import { useRef } from "react";
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
  Package,
  HelpCircle,
  FileArchive,
  FileCode,
  FileText,
  AlertTriangle,
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
  useUploadWzFiles,
  useRunWzExtraction,
  useRunDataProcessing,
  useWzInputStatus,
  useExtractionStatus,
  useDataStatus,
} from "@/lib/hooks/api/useSeed";

interface SeedButtonProps {
  label: string;
  description: string;
  icon: React.ReactNode;
  isPending: boolean;
  onClick: () => void;
}

function SeedButton({ label, description, icon, isPending, onClick }: SeedButtonProps) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between p-4">
        <div className="flex items-center gap-3">
          <div className="text-muted-foreground">{icon}</div>
          <div>
            <p className="font-medium text-sm">{label}</p>
            <p className="text-xs text-muted-foreground">{description}</p>
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={onClick} disabled={isPending}>
          {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Seed"}
        </Button>
      </CardContent>
    </Card>
  );
}

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

function formatCount(n: number): string {
  return new Intl.NumberFormat().format(n);
}

function pluralize(n: number, singular: string, plural: string): string {
  return n === 1 ? singular : plural;
}

interface GameDataRowProps {
  icon: React.ReactNode;
  label: string;
  badge: React.ReactNode;
  action: React.ReactNode;
  warning?: React.ReactNode;
}

function GameDataRow({ icon, label, badge, action, warning }: GameDataRowProps) {
  return (
    <div className="flex flex-col gap-2 border-b last:border-0 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="text-muted-foreground">{icon}</div>
          <div>
            <p className="font-medium text-sm">{label}</p>
            <p
              className="text-xs text-muted-foreground"
              aria-live="polite"
            >
              {badge}
            </p>
          </div>
        </div>
        {action}
      </div>
      {warning}
    </div>
  );
}

export default function SetupPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const seedDrops = useSeedDrops();
  const seedGachapons = useSeedGachapons();
  const seedNpcConversations = useSeedNpcConversations();
  const seedQuestConversations = useSeedQuestConversations();
  const seedNpcShops = useSeedNpcShops();
  const seedPortalScripts = useSeedPortalScripts();
  const seedReactorScripts = useSeedReactorScripts();

  const uploadWz = useUploadWzFiles();
  const runExtraction = useRunWzExtraction();
  const runProcessing = useRunDataProcessing();

  const wzInput = useWzInputStatus();
  const extraction = useExtractionStatus();
  const dataStatus = useDataStatus();

  const wzInputData = wzInput.data;
  const extractionData = extraction.data;
  const dataStatusData = dataStatus.data;

  const anyMutationPending =
    uploadWz.isPending || runExtraction.isPending || runProcessing.isPending;

  const extractDisabledReason = !wzInputData
    ? null
    : wzInputData.fileCount === 0
      ? "Upload WZ files first"
      : null;
  const extractDisabled =
    !wzInputData || wzInputData.fileCount === 0 || anyMutationPending;

  const ingestDisabledReason = !extractionData
    ? null
    : extractionData.fileCount === 0
      ? "Run extraction first"
      : null;
  const ingestDisabled =
    !extractionData ||
    extractionData.fileCount === 0 ||
    runExtraction.isPending ||
    runProcessing.isPending;

  const staleWarning =
    wzInputData?.updatedAt &&
    extractionData?.updatedAt &&
    wzInputData.updatedAt > extractionData.updatedAt;

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
    uploadWz.mutate(file, {
      onSuccess: () => {
        toast.success(`WZ files uploaded (${formatBytes(size)})`);
      },
      onError: (error) => {
        const err = error as Error & { status?: number };
        if (err.status === 409) {
          toast.error("Another upload or extraction is in progress for this tenant. Try again in a moment.");
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

  const handleRunExtraction = () => {
    runExtraction.mutate(undefined, {
      onSuccess: () => {
        toast.success("Extraction complete");
      },
      onError: (error) => {
        toast.error(`Extraction failed: ${error.message}`);
      },
    });
  };

  const handleRunProcessing = () => {
    runProcessing.mutate(undefined, {
      onSuccess: () => {
        toast.success("Data processing started");
      },
      onError: (error) => {
        toast.error(`Data processing failed: ${error.message}`);
      },
    });
  };

  const wzInputBadge = !wzInputData ? (
    "—"
  ) : wzInputData.fileCount === 0 ? (
    "0 .wz files"
  ) : (
    `${formatCount(wzInputData.fileCount)} ${pluralize(wzInputData.fileCount, ".wz file", ".wz files")}, ${formatBytes(wzInputData.totalBytes)}`
  );

  const extractionBadge = !extractionData
    ? "—"
    : `${formatCount(extractionData.fileCount)} ${pluralize(extractionData.fileCount, "XML extracted", "XMLs extracted")}`;

  const dataStatusBadge = !dataStatusData
    ? "—"
    : `${formatCount(dataStatusData.documentCount)} ${pluralize(dataStatusData.documentCount, "document loaded", "documents loaded")}`;

  const seedActions = [
    { label: "Monster & Reactor Drops", description: "Seed drop tables for monsters and reactors", icon: <Database className="h-5 w-5" />, mutation: seedDrops },
    { label: "Gachapons", description: "Seed gachapon machine configurations", icon: <Package className="h-5 w-5" />, mutation: seedGachapons },
    { label: "NPC Conversations", description: "Seed NPC conversation scripts", icon: <MessageSquare className="h-5 w-5" />, mutation: seedNpcConversations },
    { label: "Quest Conversations", description: "Seed quest conversation scripts", icon: <HelpCircle className="h-5 w-5" />, mutation: seedQuestConversations },
    { label: "NPC Shops", description: "Seed NPC shop inventories", icon: <Store className="h-5 w-5" />, mutation: seedNpcShops },
    { label: "Portal Scripts", description: "Seed portal action scripts", icon: <DoorOpen className="h-5 w-5" />, mutation: seedPortalScripts },
    { label: "Reactor Scripts", description: "Seed reactor action scripts", icon: <Zap className="h-5 w-5" />, mutation: seedReactorScripts },
  ];

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Bootstrap</h2>
        <p className="text-muted-foreground">Upload game data, run extraction and ingest, then seed service databases.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Game Data</CardTitle>
          <CardDescription>
            Upload a WZ zip, extract it into XMLs, then ingest the XMLs into atlas-data. Each step is independent.
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

          <GameDataRow
            icon={<FileArchive className="h-5 w-5" />}
            label="Upload WZ"
            badge={wzInputBadge}
            action={
              <Button
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadWz.isPending}
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

          <GameDataRow
            icon={<FileCode className="h-5 w-5" />}
            label="Extract"
            badge={extractionBadge}
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handleRunExtraction}
                disabled={extractDisabled}
                title={extractDisabledReason ?? (runExtraction.isPending ? "Extracting…" : undefined)}
              >
                {runExtraction.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Extracting…
                  </>
                ) : (
                  "Run Extraction"
                )}
              </Button>
            }
          />

          {staleWarning && (
            <div
              role="status"
              className="flex items-start gap-2 rounded-md border border-yellow-500/40 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-700 dark:text-yellow-400"
            >
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>
                Uploaded WZ files are newer than the last extraction. Re-run extraction before ingest to avoid stale data.
              </span>
            </div>
          )}

          <GameDataRow
            icon={<FileText className="h-5 w-5" />}
            label="Ingest"
            badge={dataStatusBadge}
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handleRunProcessing}
                disabled={ingestDisabled}
                title={ingestDisabledReason ?? (runProcessing.isPending ? "Ingesting…" : undefined)}
              >
                {runProcessing.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Ingesting…
                  </>
                ) : (
                  "Process Data"
                )}
              </Button>
            }
          />
        </CardContent>
      </Card>

      <div>
        <h3 className="text-lg font-semibold mb-3">Seed Data</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Populate individual service databases from their configured data sources.
        </p>
        <div className="grid gap-3">
          {seedActions.map((action) => (
            <SeedButton
              key={action.label}
              label={action.label}
              description={action.description}
              icon={action.icon}
              isPending={action.mutation.isPending}
              onClick={() => handleSeed(action.mutation, action.label)}
            />
          ))}
        </div>
      </div>

      <Toaster richColors />
    </div>
  );
}
