import { useEffect, useState, type ReactNode } from "react";
import { useTenant } from "@/context/tenant-context";
import { useNavigate, useParams } from "react-router-dom";
import { useReport } from "@/lib/hooks/api/useReports";
import { ReportKindLabels } from "@/types/models/report";
import { ReportStatusBadge } from "@/components/features/reports/ReportStatusBadge";
import { UpdateReportStatusDialog } from "@/components/features/reports/UpdateReportStatusDialog";
import { Toaster, toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ArrowLeft, Flag } from "lucide-react";

function ReportDetailSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-4">
        <Skeleton className="h-9 w-9" />
        <Skeleton className="h-8 w-48" />
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-6 w-32" />
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-2/3" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

export function ReportDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const navigate = useNavigate();
  const reportId = params.reportId as string;

  const reportQuery = useReport(activeTenant, reportId);

  const report = reportQuery.data ?? null;
  const loading = reportQuery.isLoading;
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);

  // Surface a genuine fetch failure (network error, 500, etc.) as a toast —
  // distinct from a report id that legitimately doesn't exist. The GM
  // navigated here via a URL, so we don't auto-bounce them away like
  // BanDetailPage does; we just make sure they can tell "backend is down"
  // apart from "this report doesn't exist" (see the two render branches
  // below).
  useEffect(() => {
    if (reportQuery.error) {
      const errorInfo = createErrorFromUnknown(
        reportQuery.error,
        "Failed to load report",
      );
      toast.error(errorInfo.message);
    }
  }, [reportQuery.error]);

  let content: ReactNode;

  if (loading) {
    content = <ReportDetailSkeleton />;
  } else if (reportQuery.error) {
    content = (
      <div className="flex flex-col flex-1 items-center justify-center p-10">
        <p className="text-muted-foreground">Failed to load report</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => navigate("/reports")}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Reports
        </Button>
      </div>
    );
  } else if (!report) {
    content = (
      <div className="flex flex-col flex-1 items-center justify-center p-10">
        <p className="text-muted-foreground">Report not found</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => navigate("/reports")}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Reports
        </Button>
      </div>
    );
  } else {
    const { attributes } = report;
    content = (
      <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate("/reports")}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div className="flex items-center gap-2">
              <Flag className="h-6 w-6" />
              <h2 className="text-2xl font-bold tracking-tight">
                {ReportKindLabels[attributes.kind]} Report #{report.id}
              </h2>
              <ReportStatusBadge status={attributes.status} />
            </div>
          </div>
          <Button onClick={() => setStatusDialogOpen(true)}>
            Update Status
          </Button>
        </div>

        <div className="text-sm text-muted-foreground space-y-1">
          <p>
            Reported by{" "}
            <span className="font-medium text-foreground">
              {attributes.reporterName}
            </span>{" "}
            against{" "}
            <span className="font-medium text-foreground">
              {attributes.accusedName}
            </span>
          </p>
          <p>
            Created {new Date(attributes.createdAt).toLocaleString()} &middot;
            Updated {new Date(attributes.updatedAt).toLocaleString()}
          </p>
        </div>

        <div className="grid gap-4 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Description</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="whitespace-pre-wrap text-sm">
                {attributes.description}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Client Chat Log</CardTitle>
            </CardHeader>
            <CardContent>
              {attributes.chatLog ? (
                <pre className="whitespace-pre-wrap text-sm">
                  {attributes.chatLog}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No chat log submitted.
                </p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Server Transcript</CardTitle>
            </CardHeader>
            <CardContent>
              {attributes.serverTranscript &&
              attributes.serverTranscript.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Time</TableHead>
                      <TableHead>Sender</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Message</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {attributes.serverTranscript.map((line, i) => (
                      <TableRow key={i}>
                        <TableCell className="whitespace-nowrap text-xs">
                          {new Date(line.timestamp).toLocaleString()}
                        </TableCell>
                        <TableCell className="text-xs">
                          {line.senderName}
                        </TableCell>
                        <TableCell className="text-xs">
                          {line.chatType}
                        </TableCell>
                        <TableCell className="whitespace-pre-wrap text-xs">
                          {line.text}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No server transcript captured.
                </p>
              )}
            </CardContent>
          </Card>
        </div>

        <UpdateReportStatusDialog
          report={report}
          open={statusDialogOpen}
          onOpenChange={setStatusDialogOpen}
        />
      </div>
    );
  }

  return (
    <>
      {content}
      <Toaster richColors />
    </>
  );
}
