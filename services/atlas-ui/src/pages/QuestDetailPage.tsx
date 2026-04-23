import { useTenant } from "@/context/tenant-context";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuest } from "@/lib/hooks/api/useQuests";
import { useQuestConversation } from "@/lib/hooks/api/useQuestConversation";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import {
  QuestConversationProvider,
  QuestConversationToolbar,
  QuestConversationMachineEditor,
} from "@/components/features/quests/conversation/QuestConversationCard";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  ArrowLeft,
  ArrowRight,
  AlertTriangle,
  Hash,
  Clock,
  Zap,
  CheckCircle,
} from "lucide-react";
import { RequirementGrid } from "@/components/features/quests/RequirementGrid";
import { RewardGrid } from "@/components/features/quests/RewardGrid";
import { Toaster } from "sonner";

function QuestDetailSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <Skeleton className="h-8 w-32" />
      <Skeleton className="h-12 w-96" />
      <Skeleton className="h-48" />
      <Skeleton className="h-72" />
      <Skeleton className="h-72" />
    </div>
  );
}

export function QuestDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const navigate = useNavigate();
  const questId = params.id as string;

  const questQuery = useQuest(activeTenant, questId);
  const quest = questQuery.data ?? null;
  const loading = questQuery.isLoading;
  const error = questQuery.error?.message ?? null;

  const conversationQuery = useQuestConversation(Number(questId));

  if (loading) return <QuestDetailSkeleton />;

  if (error) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center space-y-4 p-10">
        <p className="text-destructive">{error}</p>
        <Button variant="outline" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Go Back
        </Button>
      </div>
    );
  }

  if (!quest) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center space-y-4 p-10">
        <p className="text-muted-foreground">Quest not found</p>
        <Button variant="outline" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Go Back
        </Button>
      </div>
    );
  }

  const attrs = quest.attributes;
  const conversation = conversationQuery.data ?? null;

  const page = (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <QuestHeader
        questId={questId}
        name={attrs.name}
        parent={attrs.parent}
        onBack={() => navigate(-1)}
      />

      <InformationCard quest={quest} />

      {attrs.endActions.nextQuest && (
        <QuestChainRow nextQuestId={attrs.endActions.nextQuest} />
      )}

      {conversation && <QuestConversationToolbar />}

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Start</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <Section title="Requirements">
            <RequirementGrid
              requirements={attrs.startRequirements}
              phase="start"
            />
          </Section>
          <Divider />
          <Section title="Actions">
            <RewardGrid actions={attrs.startActions} phase="start" />
          </Section>
          <Divider />
          <Section title="State machine">
            {conversationQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-11/12" />
              </div>
            ) : conversationQuery.error ? (
              <ErrorDisplay
                error={conversationQuery.error}
                retry={() => conversationQuery.refetch()}
              />
            ) : conversation ? (
              <QuestConversationMachineEditor machine="start" />
            ) : (
              <p className="text-sm text-muted-foreground">
                No start conversation defined.
              </p>
            )}
          </Section>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Completion</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <Section title="Requirements">
            <RequirementGrid
              requirements={attrs.endRequirements}
              phase="end"
            />
          </Section>
          <Divider />
          <Section title="Actions">
            <RewardGrid actions={attrs.endActions} phase="end" omitNextQuest />
          </Section>
          <Divider />
          <Section title="State machine">
            {conversationQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-11/12" />
              </div>
            ) : conversationQuery.error ? null : conversation &&
              conversation.attributes.endStateMachine ? (
              <QuestConversationMachineEditor machine="end" />
            ) : (
              <p className="text-sm text-muted-foreground">
                No completion conversation defined.
              </p>
            )}
          </Section>
        </CardContent>
      </Card>

      <Toaster richColors />
    </div>
  );

  if (conversation) {
    return (
      <QuestConversationProvider conversation={conversation}>
        {page}
      </QuestConversationProvider>
    );
  }
  return page;
}

interface QuestHeaderProps {
  questId: string;
  name?: string | undefined;
  parent?: string | undefined;
  onBack: () => void;
}

function QuestHeader({ questId, name, parent, onBack }: QuestHeaderProps) {
  return (
    <div className="flex items-center gap-4">
      <Button variant="ghost" size="icon" onClick={onBack}>
        <ArrowLeft className="h-4 w-4" />
      </Button>
      <div className="flex items-center gap-3 flex-wrap">
        <h2 className="text-2xl font-bold tracking-tight">
          {name || "(Unnamed Quest)"}
        </h2>
        {parent && <Badge variant="outline">{parent}</Badge>}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex items-center gap-1 text-xs text-muted-foreground rounded px-1.5 py-0.5 border hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                aria-label="Quest id"
              >
                <Hash className="h-3 w-3" />
                {questId}
              </button>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{questId}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}

interface InformationCardProps {
  quest: {
    attributes: {
      autoStart: boolean;
      autoComplete: boolean;
      timeLimit?: number | undefined;
      area: number;
      order?: number | undefined;
      summary?: string | undefined;
      demandSummary?: string | undefined;
      rewardSummary?: string | undefined;
    };
  };
}

function InformationCard({ quest }: InformationCardProps) {
  const attrs = quest.attributes;
  const hasSummary =
    attrs.summary || attrs.demandSummary || attrs.rewardSummary;
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetaField label="Auto Start">
            {attrs.autoStart ? (
              <Badge className="gap-1">
                <Zap className="h-3 w-3" />
                Yes
              </Badge>
            ) : (
              <span className="text-sm">No</span>
            )}
          </MetaField>
          <MetaField label="Auto Complete">
            {attrs.autoComplete ? (
              <Badge variant="secondary" className="gap-1">
                <CheckCircle className="h-3 w-3" />
                Yes
              </Badge>
            ) : (
              <span className="text-sm">No</span>
            )}
          </MetaField>
          <MetaField label="Time Limit">
            {attrs.timeLimit && attrs.timeLimit > 0 ? (
              <Badge variant="outline" className="gap-1">
                <Clock className="h-3 w-3" />
                {formatTime(attrs.timeLimit)}
              </Badge>
            ) : (
              <span className="text-sm">None</span>
            )}
          </MetaField>
          <MetaField label="Area / Order">
            <span className="text-sm">
              {attrs.area || 0} / {attrs.order || 0}
            </span>
          </MetaField>
        </div>
        {hasSummary && (
          <div className="mt-4 pt-4 border-t space-y-2">
            {attrs.summary && (
              <SummaryLine label="Summary" value={attrs.summary} />
            )}
            {attrs.demandSummary && (
              <SummaryLine label="Demand" value={attrs.demandSummary} />
            )}
            {attrs.rewardSummary && (
              <SummaryLine label="Reward" value={attrs.rewardSummary} />
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function MetaField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <div className="flex items-center gap-1">{children}</div>
    </div>
  );
}

function SummaryLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-sm font-medium">{label}</p>
      <p className="text-sm text-muted-foreground">{value}</p>
    </div>
  );
}

interface QuestChainRowProps {
  nextQuestId: number;
}

function QuestChainRow({ nextQuestId }: QuestChainRowProps) {
  const { activeTenant } = useTenant();
  const q = useQuest(activeTenant, String(nextQuestId));
  const loading = q.isLoading;
  const err = q.error;
  const name = q.data?.attributes.name;
  const broken = !!err;

  return (
    <Card>
      <CardContent className="py-4 flex items-center gap-3 flex-wrap">
        <span className="text-sm text-muted-foreground">Next quest →</span>
        {loading ? (
          <Skeleton className="h-8 w-40" />
        ) : broken ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span
                  tabIndex={0}
                  className="inline-flex items-center gap-2 rounded px-2 py-1 text-sm text-destructive border border-destructive/40 cursor-help"
                >
                  <AlertTriangle className="h-4 w-4" />
                  Quest #{nextQuestId}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <p>Target quest not found</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : (
          <Button asChild variant="outline" size="sm">
            <Link to={`/quests/${nextQuestId}`}>
              {name ?? `Quest #${nextQuestId}`}
              <ArrowRight className="h-4 w-4 ml-1" />
            </Link>
          </Button>
        )}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex items-center gap-1 text-xs text-muted-foreground rounded px-1.5 py-0.5 border hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                aria-label="Next quest id"
              >
                <Hash className="h-3 w-3" />
                {nextQuestId}
              </button>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{String(nextQuestId)}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </CardContent>
    </Card>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
        {title}
      </h3>
      {children}
    </section>
  );
}

function Divider() {
  return <div className="border-t" />;
}

function formatTime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}
