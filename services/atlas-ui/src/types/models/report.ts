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

/**
 * Claim reason codes. These are the `nType` byte the client puts on the wire,
 * taken from the item data of the reason dropdown built in `CUIClaim::OnCreate`
 * (v83 @0x7db17d, IDA-verified — see
 * docs/tasks/task-145-player-reports/packet-findings.md §2). Codes 3–8 are the
 * chat-claim list; 9 is the only entry offered for a non-chat claim.
 *
 * There is no equivalent enum for `sue` reports: `CField::SendChatMsgSlash`
 * (v87 @0x55350b) derives the sue flag from `atoi()` on a slash-command
 * argument, so it is a player-typed number with no client-side meaning.
 */
export const ClaimReasonType = {
  CurseOrInappropriateContent: 3,
  Advertising: 4,
  Fraud: 5,
  CashTrade: 6,
  ImpersonatingGm: 7,
  ExposingPersonalInfo: 8,
  IllegalPrograms: 9,
} as const;
export type ClaimReasonType =
  (typeof ClaimReasonType)[keyof typeof ClaimReasonType];

export const ClaimReasonLabels: Record<ClaimReasonType, string> = {
  [ClaimReasonType.CurseOrInappropriateContent]:
    "Cursing / inappropriate content",
  [ClaimReasonType.Advertising]: "Advertising",
  [ClaimReasonType.Fraud]: "Fraud",
  [ClaimReasonType.CashTrade]: "Cash trade",
  [ClaimReasonType.ImpersonatingGm]: "Impersonating a GM",
  [ClaimReasonType.ExposingPersonalInfo]: "Exposing personal information",
  [ClaimReasonType.IllegalPrograms]: "Illegal programs",
};

/**
 * Renders the stored `reasonType` byte for display. The column holds a
 * different thing per kind, so the label is kind-aware: a named claim reason,
 * or the raw sue category the reporter typed. An unrecognised claim code keeps
 * the number visible rather than hiding it behind a guess.
 */
export function formatReportReason(
  kind: ReportKind,
  reasonType: number,
): string {
  if (kind !== ReportKind.Claim) {
    return `Category ${reasonType}`;
  }
  return (
    ClaimReasonLabels[reasonType as ClaimReasonType] ??
    `Unknown (${reasonType})`
  );
}

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
