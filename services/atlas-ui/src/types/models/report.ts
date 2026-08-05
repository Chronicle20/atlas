// Report domain model types (player sue/claim reports reviewed by GMs).

/**
 * Const-object enums rather than TS `enum` so the type-only strip the
 * Vite/ESBuild toolchain performs is lossless (see `erasableSyntaxOnly`
 * in tsconfig.app.json).
 */
export const ReportStatus = {
  Open: "open",
  Reviewed: "reviewed",
  Actioned: "actioned",
} as const;
export type ReportStatus = (typeof ReportStatus)[keyof typeof ReportStatus];

export const ReportStatusLabels: Record<ReportStatus, string> = {
  [ReportStatus.Open]: "Open",
  [ReportStatus.Reviewed]: "Reviewed",
  [ReportStatus.Actioned]: "Actioned",
};

export const ReportKind = {
  Sue: "sue",
  Claim: "claim",
} as const;
export type ReportKind = (typeof ReportKind)[keyof typeof ReportKind];

export const ReportKindLabels: Record<ReportKind, string> = {
  [ReportKind.Sue]: "Sue",
  [ReportKind.Claim]: "Claim",
};

export interface TranscriptLine {
  timestamp: number;
  senderId: number;
  senderName: string;
  chatType: string;
  text: string;
}

export interface ReportAttributes {
  kind: ReportKind;
  reporterId: number;
  reporterName: string;
  accusedId: number;
  accusedName: string;
  reasonType: number;
  description: string;
  chatLog: string | null;
  serverTranscript: TranscriptLine[] | null;
  status: ReportStatus;
  createdAt: string;
  updatedAt: string;
}

export interface Report {
  id: string;
  type: "reports";
  attributes: ReportAttributes;
}
