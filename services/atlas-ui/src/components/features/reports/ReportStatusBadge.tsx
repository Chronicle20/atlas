import { Badge } from "@/components/ui/badge";
import { ReportStatus, ReportStatusLabels } from "@/types/models/report";

const statusVariant: Record<ReportStatus, "default" | "secondary" | "outline"> =
  {
    [ReportStatus.Open]: "default",
    [ReportStatus.Reviewed]: "secondary",
    [ReportStatus.Actioned]: "outline",
  };

export function ReportStatusBadge({ status }: { status: ReportStatus }) {
  return (
    <Badge variant={statusVariant[status] ?? "default"}>
      {ReportStatusLabels[status] ?? status}
    </Badge>
  );
}
