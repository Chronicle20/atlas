import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { useUpdateReportStatus } from "@/lib/hooks/api/useReports";
import { ReportStatus, ReportStatusLabels } from "@/types/models/report";
import type { Report } from "@/types/models/report";
import { Loader2 } from "lucide-react";

interface UpdateReportStatusDialogProps {
  report: Report;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UpdateReportStatusDialog({
  report,
  open,
  onOpenChange,
}: UpdateReportStatusDialogProps) {
  const [status, setStatus] = useState<ReportStatus>(report.attributes.status);
  // Reset the selected status whenever the dialog transitions from closed to
  // open (adjust state during render per https://react.dev/learn/you-might-not-need-an-effect
  // instead of a useEffect) — the dialog never unmounts, so without this a
  // status picked and then cancelled on a prior open would stick around and
  // silently arm the Save button on the next open. Mirrors PoolFormDialog.tsx.
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setStatus(report.attributes.status);
  }
  const updateStatus = useUpdateReportStatus();

  const handleSubmit = () => {
    updateStatus.mutate(
      { id: report.id, status },
      {
        onSuccess: () => {
          toast.success(`Report marked ${ReportStatusLabels[status]}`);
          onOpenChange(false);
        },
        onError: (error) => {
          const errorInfo = createErrorFromUnknown(
            error,
            "Failed to update report status",
          );
          toast.error(errorInfo.message);
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Update Report Status</DialogTitle>
          <DialogDescription>
            Set the review state for the report against{" "}
            {report.attributes.accusedName}.
          </DialogDescription>
        </DialogHeader>
        <Select
          value={status}
          onValueChange={(v) => setStatus(v as ReportStatus)}
          disabled={updateStatus.isPending}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {Object.values(ReportStatus).map((s) => (
              <SelectItem key={s} value={s}>
                {ReportStatusLabels[s]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={updateStatus.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={
              updateStatus.isPending || status === report.attributes.status
            }
          >
            {updateStatus.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
