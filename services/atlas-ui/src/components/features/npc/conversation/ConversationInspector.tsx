import { useEffect, useState } from "react";
import {
  AlertTriangle,
  CornerUpLeft,
  Play,
  Plus,
  Share2,
  Square,
  Trash2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type {
  Conversation,
  ConversationState,
  DialogueChoice,
  DialogueState,
} from "@/types/models/conversation";
import { buildStateIndex, getTransitions } from "./transitions";
import { type GraphAnalysis } from "./graphAnalysis";
import { STATE_TYPE_META } from "./stateMeta";
import { idIsTaken as idIsTakenOp } from "./editorOps";

interface ConversationInspectorProps {
  conversation: Conversation;
  selectedStateId: string | null;
  onSelect: (stateId: string) => void;
  analysis: GraphAnalysis;
  onUpdateState: (id: string, next: ConversationState) => void;
  onRename: (oldId: string, newId: string) => void;
  onDelete: (id: string) => void;
  readOnly: boolean;
}

export function ConversationInspector({
  conversation,
  selectedStateId,
  onSelect,
  analysis,
  onUpdateState,
  onRename,
  onDelete,
  readOnly,
}: ConversationInspectorProps) {
  if (!selectedStateId) {
    return (
      <div className="flex items-center justify-center h-full text-xs text-muted-foreground px-4 text-center">
        Select a state to inspect.
      </div>
    );
  }
  const state = buildStateIndex(conversation).get(selectedStateId);
  if (!state) {
    return (
      <div className="p-4 text-xs text-muted-foreground">
        State <code className="font-mono">{selectedStateId}</code> not found.
      </div>
    );
  }

  const meta = STATE_TYPE_META[state.type];
  const transitions = getTransitions(state);
  const isStart = conversation.attributes.startState === state.id;
  const isTerminal = analysis.terminals.has(state.id);
  const inbound = analysis.inboundCount.get(state.id) ?? 0;
  const predecessors = findPredecessors(conversation, state.id);

  const brokenOutbound = analysis.brokenRefs.filter(r => r.source === state.id);
  const isUnreachable = !analysis.reachable.has(state.id) && !isStart;
  const isDuplicate = analysis.duplicateIds.includes(state.id);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="px-4 py-3 border-b flex flex-col gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span
            className={`text-[10px] px-1.5 py-[1px] rounded-sm border ${meta.accent}`}
          >
            {meta.label}
          </span>
          {isStart && (
            <Badge className="text-[10px] px-1.5 py-0 gap-1">
              <Play className="h-2.5 w-2.5 fill-current" /> start
            </Badge>
          )}
          {isTerminal && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0 gap-1">
              <Square className="h-2.5 w-2.5 fill-current" /> terminal
            </Badge>
          )}
          {inbound >= 3 && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0 gap-1">
              <Share2 className="h-2.5 w-2.5" /> {inbound} inbound
            </Badge>
          )}
          {(brokenOutbound.length > 0 || isDuplicate || isUnreachable) && (
            <Badge variant="destructive" className="text-[10px] px-1.5 py-0 gap-1">
              <AlertTriangle className="h-2.5 w-2.5" /> issues
            </Badge>
          )}
        </div>
        <IdRow
          conversation={conversation}
          stateId={state.id}
          onRename={onRename}
          readOnly={readOnly}
          onDelete={() => onDelete(state.id)}
          canDelete={!readOnly && !isStart}
        />
      </div>

      <div className="flex-1 overflow-y-auto divide-y">
        <TypeFields
          state={state}
          onUpdateState={onUpdateState}
          readOnly={readOnly}
          conversation={conversation}
        />
        <TransitionsSection
          transitions={transitions}
          analysis={analysis}
          onSelect={onSelect}
        />
        <PredecessorsSection predecessors={predecessors} onSelect={onSelect} />
        {(brokenOutbound.length > 0 || isDuplicate || isUnreachable) && (
          <WarningsSection
            brokenOutbound={brokenOutbound}
            isDuplicate={isDuplicate}
            isUnreachable={isUnreachable}
          />
        )}
      </div>
    </div>
  );
}

function IdRow({
  conversation,
  stateId,
  onRename,
  readOnly,
  onDelete,
  canDelete,
}: {
  conversation: Conversation;
  stateId: string;
  onRename: (oldId: string, newId: string) => void;
  readOnly: boolean;
  onDelete: () => void;
  canDelete: boolean;
}) {
  const [value, setValue] = useState(stateId);
  useEffect(() => setValue(stateId), [stateId]);

  const trimmed = value.trim();
  const isEmpty = trimmed === "";
  const unchanged = trimmed === stateId;
  const collides = !unchanged && idIsTakenOp(conversation, trimmed, stateId);
  const canRename = !readOnly && !unchanged && !isEmpty && !collides;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        <Input
          value={value}
          onChange={e => setValue(e.target.value)}
          disabled={readOnly}
          className="font-mono text-sm h-8 flex-1"
        />
        <Button
          size="sm"
          variant="outline"
          disabled={!canRename}
          onClick={() => onRename(stateId, trimmed)}
          title={
            readOnly
              ? "Read-only"
              : collides
                ? "ID already in use"
                : unchanged
                  ? "No change"
                  : "Rename and rewire references"
          }
        >
          Rename
        </Button>
        <Button
          size="sm"
          variant="ghost"
          className="h-8 w-8 p-0 text-destructive hover:text-destructive"
          disabled={!canDelete}
          onClick={onDelete}
          title={canDelete ? "Delete state" : "Cannot delete the start state"}
          aria-label="Delete state"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      {collides && (
        <p className="text-[11px] text-destructive">
          Another state already uses this ID.
        </p>
      )}
    </div>
  );
}

function Section({
  title,
  children,
  action,
}: {
  title: string;
  children: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <section className="px-4 py-3 flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <h4 className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          {title}
        </h4>
        {action}
      </div>
      {children}
    </section>
  );
}

function KV({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="grid grid-cols-[90px_1fr] gap-2 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono break-all" : "break-words"}>
        {value}
      </span>
    </div>
  );
}

function TypeFields({
  state,
  onUpdateState,
  readOnly,
  conversation,
}: {
  state: ConversationState;
  onUpdateState: (id: string, next: ConversationState) => void;
  readOnly: boolean;
  conversation: Conversation;
}) {
  switch (state.type) {
    case "dialogue":
      return readOnly ? (
        <DialogueReadOnly state={state} />
      ) : (
        <DialogueForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "listSelection":
      return (
        <Section title="List Selection">
          {state.listSelection ? (
            <>
              <KV
                label="title"
                value={
                  state.listSelection.title || (
                    <em className="text-muted-foreground">(none)</em>
                  )
                }
              />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "askSlideMenu":
      return (
        <Section title="Slide Menu">
          {state.askSlideMenu ? (
            <>
              <KV
                label="title"
                value={
                  state.askSlideMenu.title || (
                    <em className="text-muted-foreground">(none)</em>
                  )
                }
              />
              <KV
                label="menuType"
                value={state.askSlideMenu.menuType ?? 0}
                mono
              />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "askNumber":
      return (
        <Section title="Ask Number">
          {state.askNumber ? (
            <>
              <KV label="text" value={state.askNumber.text} />
              <KV label="default" value={state.askNumber.defaultValue} mono />
              <KV label="min" value={state.askNumber.minValue} mono />
              <KV label="max" value={state.askNumber.maxValue} mono />
              {state.askNumber.contextKey && (
                <KV label="contextKey" value={state.askNumber.contextKey} mono />
              )}
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "askStyle":
      return (
        <Section title="Ask Style">
          {state.askStyle ? (
            <>
              <KV label="text" value={state.askStyle.text} />
              <KV
                label="styles"
                value={state.askStyle.styles?.join(", ") ?? "—"}
                mono
              />
              {state.askStyle.contextKey && (
                <KV label="contextKey" value={state.askStyle.contextKey} mono />
              )}
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "craftAction":
      return (
        <Section title="Craft">
          {state.craftAction ? (
            <>
              <KV label="itemId" value={state.craftAction.itemId} mono />
              <KV
                label="materials"
                value={state.craftAction.materials?.join(", ") ?? "—"}
                mono
              />
              <KV
                label="quantities"
                value={state.craftAction.quantities?.join(", ") ?? "—"}
                mono
              />
              <KV label="mesoCost" value={state.craftAction.mesoCost ?? 0} mono />
              {state.craftAction.stimulatorId !== undefined && (
                <KV
                  label="stimulator"
                  value={`item ${state.craftAction.stimulatorId} · fail ${state.craftAction.stimulatorFailChance ?? 0}%`}
                  mono
                />
              )}
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "transportAction":
      return (
        <Section title="Transport">
          {state.transportAction ? (
            <>
              <KV label="routeName" value={state.transportAction.routeName} mono />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "partyQuestAction":
      return (
        <Section title="Party Quest">
          {state.partyQuestAction ? (
            <>
              <KV label="questId" value={state.partyQuestAction.questId} mono />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "partyQuestBonusAction":
      return (
        <Section title="PQ Bonus">
          <p className="text-xs text-muted-foreground">
            Warps party to the bonus stage.
          </p>
          <EditNotSupported />
        </Section>
      );
    case "gachaponAction":
      return (
        <Section title="Gachapon">
          {state.gachaponAction ? (
            <>
              <KV
                label="gachaponId"
                value={state.gachaponAction.gachaponId}
                mono
              />
              <KV
                label="ticketItem"
                value={state.gachaponAction.ticketItemId}
                mono
              />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    case "genericAction":
      return (
        <Section title="Generic Action">
          {state.genericAction ? (
            <>
              <KV
                label="operations"
                value={
                  <div className="flex flex-col gap-0.5">
                    {(state.genericAction.operations ?? []).length === 0 ? (
                      <em className="text-muted-foreground">(none)</em>
                    ) : (
                      (state.genericAction.operations ?? []).map((op, i) => (
                        <code key={i} className="font-mono text-[11px]">
                          {op.type}
                          {op.params && Object.keys(op.params).length > 0
                            ? ` (${Object.entries(op.params)
                                .map(([k, v]) => `${k}=${v}`)
                                .join(", ")})`
                            : ""}
                        </code>
                      ))
                    )}
                  </div>
                }
              />
              <EditNotSupported />
            </>
          ) : (
            <Empty />
          )}
        </Section>
      );
    default:
      return null;
  }
}

function DialogueReadOnly({ state }: { state: ConversationState }) {
  const d = state.dialogue;
  return (
    <Section title="Dialogue">
      {d ? (
        <>
          <KV label="dialogueType" value={d.dialogueType} mono />
          <KV
            label="text"
            value={
              <span className="whitespace-pre-wrap">
                {d.text || <em className="text-muted-foreground">(empty)</em>}
              </span>
            }
          />
        </>
      ) : (
        <Empty />
      )}
    </Section>
  );
}

function DialogueForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const d: DialogueState = state.dialogue ?? {
    dialogueType: "sendOk",
    text: "",
    choices: [],
  };

  const update = (patch: Partial<DialogueState>) =>
    onUpdateState(state.id, {
      ...state,
      dialogue: { ...d, ...patch },
    });

  const updateChoice = (i: number, patch: Partial<DialogueChoice>) => {
    const choices = [...(d.choices ?? [])];
    const current = choices[i] ?? { text: "", nextState: null };
    choices[i] = { ...current, ...patch };
    update({ choices });
  };

  const addChoice = () =>
    update({
      choices: [...(d.choices ?? []), { text: "", nextState: null }],
    });

  const removeChoice = (i: number) => {
    const choices = [...(d.choices ?? [])];
    choices.splice(i, 1);
    update({ choices });
  };

  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);

  return (
    <Section title="Dialogue">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">dialogueType</Label>
        <Select
          value={d.dialogueType}
          onValueChange={v =>
            update({ dialogueType: v as DialogueState["dialogueType"] })
          }
        >
          <SelectTrigger className="h-8 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="sendOk">sendOk</SelectItem>
            <SelectItem value="sendYesNo">sendYesNo</SelectItem>
            <SelectItem value="sendNext">sendNext</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2">
        <Label className="text-xs text-muted-foreground pt-1">text</Label>
        <Textarea
          value={d.text}
          onChange={e => update({ text: e.target.value })}
          className="min-h-[96px] text-xs"
          placeholder="What the NPC says…"
        />
      </div>

      <div className="flex items-center justify-between mt-2">
        <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          Choices ({d.choices?.length ?? 0})
        </Label>
        <Button size="sm" variant="outline" onClick={addChoice}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      <div className="flex flex-col gap-2">
        {(d.choices ?? []).map((choice, i) => (
          <div
            key={i}
            className="grid grid-cols-[1fr_140px_auto] gap-1.5 items-start"
          >
            <Input
              value={choice.text}
              onChange={e => updateChoice(i, { text: e.target.value })}
              placeholder={`Choice ${i + 1}`}
              className="h-8 text-xs"
            />
            <Select
              value={choice.nextState || "__end__"}
              onValueChange={v =>
                updateChoice(i, { nextState: v === "__end__" ? null : v })
              }
            >
              <SelectTrigger className="h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__end__">&lt;end&gt;</SelectItem>
                {otherIds.map(id => (
                  <SelectItem key={id} value={id}>
                    {id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive"
              onClick={() => removeChoice(i)}
              title="Remove Choice"
              aria-label="Remove Choice"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>
    </Section>
  );
}

function EditNotSupported() {
  return (
    <p className="text-[11px] text-muted-foreground italic">
      Field editing for this state type isn't in this prototype slice yet.
    </p>
  );
}

function TransitionsSection({
  transitions,
  analysis,
  onSelect,
}: {
  transitions: ReturnType<typeof getTransitions>;
  analysis: GraphAnalysis;
  onSelect: (stateId: string) => void;
}) {
  if (transitions.length === 0) {
    return (
      <Section title="Transitions">
        <p className="text-xs text-muted-foreground">No outgoing transitions.</p>
      </Section>
    );
  }
  return (
    <Section title={`Transitions (${transitions.length})`}>
      <ul className="flex flex-col divide-y divide-border/50 rounded-md border bg-muted/20">
        {transitions.map((t, i) => {
          const target = t.target;
          const isBack = target
            ? analysis.backEdges.has(`${t.source}->${target}`)
            : false;
          const broken =
            !!target &&
            analysis.brokenRefs.some(
              r => r.source === t.source && r.target === target,
            );
          return (
            <li key={i} className="flex items-center gap-2 px-2 py-1.5 text-xs">
              <span className="text-muted-foreground w-16 shrink-0 text-[10px] uppercase">
                {t.kind}
              </span>
              <span className="flex-1 truncate" title={t.label}>
                {t.label}
              </span>
              {target === null ? (
                <span className="italic text-muted-foreground text-[11px]">
                  &lt;end&gt;
                </span>
              ) : broken ? (
                <span className="font-mono text-[11px] text-destructive">
                  {target} (missing)
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => onSelect(target)}
                  className={`font-mono text-[11px] hover:underline flex items-center gap-1 ${
                    isBack ? "text-amber-600 dark:text-amber-400" : "text-primary"
                  }`}
                >
                  {isBack && <CornerUpLeft className="h-2.5 w-2.5" />}
                  {target}
                </button>
              )}
            </li>
          );
        })}
      </ul>
    </Section>
  );
}

function PredecessorsSection({
  predecessors,
  onSelect,
}: {
  predecessors: Array<{ source: string; label: string }>;
  onSelect: (stateId: string) => void;
}) {
  if (predecessors.length === 0) {
    return (
      <Section title="Inbound">
        <p className="text-xs text-muted-foreground">No inbound transitions.</p>
      </Section>
    );
  }
  return (
    <Section title={`Inbound (${predecessors.length})`}>
      <ul className="flex flex-col gap-0.5">
        {predecessors.map((p, i) => (
          <li
            key={i}
            className="flex items-center gap-2 text-xs text-muted-foreground"
          >
            <button
              type="button"
              onClick={() => onSelect(p.source)}
              className="font-mono text-[11px] text-primary hover:underline"
            >
              {p.source}
            </button>
            <span className="truncate">{p.label}</span>
          </li>
        ))}
      </ul>
    </Section>
  );
}

function WarningsSection({
  brokenOutbound,
  isDuplicate,
  isUnreachable,
}: {
  brokenOutbound: Array<{ source: string; target: string }>;
  isDuplicate: boolean;
  isUnreachable: boolean;
}) {
  return (
    <Section title="Issues">
      <ul className="flex flex-col gap-1 text-xs text-destructive">
        {isUnreachable && <li>Unreachable from start state.</li>}
        {isDuplicate && <li>Duplicate state ID in the conversation.</li>}
        {brokenOutbound.map((b, i) => (
          <li key={i}>
            Transition points at missing state{" "}
            <code className="font-mono">{b.target}</code>.
          </li>
        ))}
      </ul>
    </Section>
  );
}

function Empty() {
  return (
    <p className="text-xs text-muted-foreground">No configuration present.</p>
  );
}

function findPredecessors(
  conversation: Conversation,
  targetId: string,
): Array<{ source: string; label: string }> {
  const out: Array<{ source: string; label: string }> = [];
  for (const state of conversation.attributes.states) {
    for (const t of getTransitions(state)) {
      if (t.target === targetId) out.push({ source: state.id, label: t.label });
    }
  }
  return out;
}
