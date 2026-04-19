import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { EquipmentAttributes } from "@/types/models/item";
import { formatReqJob } from "./formatReqJob";

interface EquipmentRequirementsCardProps {
  attributes: EquipmentAttributes;
}

export function EquipmentRequirementsCard({ attributes }: EquipmentRequirementsCardProps) {
  const { reqLevel, reqJob, reqStr, reqDex, reqInt, reqLuk, reqPop, reqFame } = attributes;

  const hasStatReq =
    reqLevel > 0 || reqStr > 0 || reqDex > 0 || reqInt > 0 || reqLuk > 0 || reqPop > 0 || reqFame > 0;
  const hasJobReq = reqJob > 0;

  if (!hasStatReq && !hasJobReq) return null;

  const jobs = formatReqJob(reqJob);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Requirements</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
          <RequirementField label="Level" value={reqLevel} />
          {jobs.length > 0 && (
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Job</p>
              <div className="flex flex-wrap gap-1">
                {jobs.map((job) => (
                  <Badge key={job} variant="outline">
                    {job}
                  </Badge>
                ))}
              </div>
            </div>
          )}
          {reqStr > 0 && <RequirementField label="STR" value={reqStr} />}
          {reqDex > 0 && <RequirementField label="DEX" value={reqDex} />}
          {reqInt > 0 && <RequirementField label="INT" value={reqInt} />}
          {reqLuk > 0 && <RequirementField label="LUK" value={reqLuk} />}
          {reqPop > 0 && <RequirementField label="POP" value={reqPop} />}
          {reqFame > 0 && <RequirementField label="Fame" value={reqFame} />}
        </div>
      </CardContent>
    </Card>
  );
}

function RequirementField({ label, value }: { label: string; value: number }) {
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="text-sm font-medium">{value}</p>
    </div>
  );
}
