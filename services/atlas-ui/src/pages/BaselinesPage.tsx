import { useRef, useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Copy,
  FileArchive,
  FileText,
  Loader2,
  Send,
  Upload,
} from "lucide-react";
import { Toaster, toast } from "sonner";
import { SetupRow } from "@/components/features/setup/SetupRow";
import {
  formatCount,
  pluralize,
} from "@/components/features/setup/setup-format";
import { BaselineTargetPicker } from "@/components/features/baselines/BaselineTargetPicker";
import {
  useBaselines,
  useCanonicalDataStatus,
  useCanonicalWzInputStatus,
  usePublishCanonicalBaseline,
  useRunCanonicalProcessing,
  useUploadCanonicalWz,
} from "@/lib/hooks/api/useCanonicalData";
import { formatBytes } from "@/lib/format";
import type { CanonicalSelection } from "@/lib/headers";

export function BaselinesPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [sel, setSel] = useState<CanonicalSelection | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const baselinesQuery = useBaselines();
  const wzInput = useCanonicalWzInputStatus(sel);
  const dataStatus = useCanonicalDataStatus(sel);
  const uploadWz = useUploadCanonicalWz(sel);
  const runProcessing = useRunCanonicalProcessing(sel);
  const publish = usePublishCanonicalBaseline(sel);

  const baselines = baselinesQuery.data ?? [];
  const wzData = wzInput.data;
  const docData = dataStatus.data;

  const existingBaseline = sel
    ? baselines.find(
        (b) =>
          b.region === sel.region &&
          b.majorVersion === sel.majorVersion &&
          b.minorVersion === sel.minorVersion,
      )
    : undefined;

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.name.toLowerCase().endsWith(".zip")) {
      toast.error("Please select a .zip file");
      return;
    }
    const size = file.size;
    uploadWz.mutate(file, {
      onSuccess: () => {
        toast.success(`WZ files uploaded (${formatBytes(size)})`);
      },
    });
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
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

  const doPublish = () => {
    publish.mutate(undefined, {
      onSuccess: () => {
        toast.success("Canonical baseline published");
      },
      onError: (error) => {
        toast.error(`Baseline publish failed: ${error.message}`);
      },
    });
  };

  const handlePublish = () => {
    if (existingBaseline) {
      setConfirmOpen(true);
      return;
    }
    doPublish();
  };

  const handleCopySha = (sha: string) => {
    void navigator.clipboard
      .writeText(sha)
      .then(() => toast.success("SHA-256 copied"));
  };

  const wzBadge = !sel
    ? "—"
    : !wzData
      ? "—"
      : wzData.fileCount === 0
        ? "0 .wz files"
        : `${formatCount(wzData.fileCount)} ${pluralize(wzData.fileCount, ".wz file", ".wz files")}, ${formatBytes(wzData.totalBytes)}`;

  const docBadge = !sel
    ? "—"
    : !docData
      ? "—"
      : `${formatCount(docData.documentCount)} ${pluralize(docData.documentCount, "document loaded", "documents loaded")}`;

  const processDisabled =
    !sel ||
    !wzData ||
    wzData.fileCount === 0 ||
    uploadWz.isPending ||
    runProcessing.isPending;
  const publishDisabled =
    !sel || !docData || docData.documentCount === 0 || publish.isPending;

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Baselines</h2>
        <p className="text-muted-foreground">
          Manage canonical game-data baselines shared by all tenants of a region
          and version.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Published Baselines</CardTitle>
          <CardDescription>
            Canonical baselines new tenants restore their game data from.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {baselinesQuery.isError ? (
            <p className="text-sm text-destructive">
              Failed to load baselines: {baselinesQuery.error?.message}
            </p>
          ) : baselines.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No canonical baselines published yet. A baseline is a published
              snapshot of processed canonical game data for one region/version;
              publish one from the workflow below and new tenants of that
              version will restore from it.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Region</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>SHA-256</TableHead>
                  <TableHead>Published</TableHead>
                  <TableHead>Size</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {baselines.map((b) => (
                  <TableRow
                    key={`${b.region}/${b.majorVersion}.${b.minorVersion}`}
                  >
                    <TableCell>{b.region}</TableCell>
                    <TableCell>{`${b.majorVersion}.${b.minorVersion}`}</TableCell>
                    <TableCell>
                      {b.sha256 ? (
                        <span className="inline-flex items-center gap-1 font-mono text-xs">
                          {`${b.sha256.slice(0, 12)}…`}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            aria-label={`Copy SHA-256 for ${b.region} ${b.majorVersion}.${b.minorVersion}`}
                            onClick={() => handleCopySha(b.sha256)}
                          >
                            <Copy className="h-3 w-3" />
                          </Button>
                        </span>
                      ) : (
                        "—"
                      )}
                    </TableCell>
                    <TableCell>
                      {new Date(b.publishedAt).toLocaleString()}
                    </TableCell>
                    <TableCell>{formatBytes(b.sizeBytes)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Canonical Workflow</CardTitle>
          <CardDescription>
            Pick a region and version, then upload a WZ zip, process it, and
            publish the baseline. No tenant is involved — this works before any
            tenant of the version exists.
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
            <BaselineTargetPicker value={sel} onChange={setSel} />
          </div>

          <SetupRow
            icon={<FileArchive className="h-5 w-5" />}
            label="Upload WZ"
            badge={wzBadge}
            action={
              <Button
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={!sel || uploadWz.isPending}
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
            badge={docBadge}
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handleRunProcessing}
                disabled={processDisabled}
                title={
                  sel && wzData && wzData.fileCount === 0
                    ? "Upload WZ files first"
                    : undefined
                }
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

          <SetupRow
            icon={<Send className="h-5 w-5" />}
            label="Publish Baseline"
            badge={
              existingBaseline?.sha256
                ? `current sha256:${existingBaseline.sha256.slice(0, 12)}…`
                : existingBaseline
                  ? "published (sha unavailable)"
                  : "not yet published"
            }
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handlePublish}
                disabled={publishDisabled}
              >
                {publish.isPending ? (
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
        </CardContent>
      </Card>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Replace existing baseline?</AlertDialogTitle>
            <AlertDialogDescription>
              This will replace the shared canonical baseline for {sel?.region}{" "}
              v{sel?.majorVersion}.{sel?.minorVersion}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={doPublish}
            >
              Replace Baseline
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Toaster richColors />
    </div>
  );
}
