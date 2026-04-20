import { useEffect, useState } from "react";
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  CornerDownRight,
  CornerUpLeft,
  GitBranch,
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
  AskNumberState,
  AskSlideMenuState,
  AskStyleState,
  Condition,
  Conversation,
  ConversationState,
  ConversationStateType,
  CraftActionState,
  DialogueChoice,
  DialogueState,
  DialogueType,
  GachaponActionState,
  GenericActionOperation,
  GenericActionOutcome,
  GenericActionState,
  ListSelectionState,
  PartyQuestActionState,
  PartyQuestBonusActionState,
  TransportActionState,
} from "@/types/models/conversation";
import { buildStateIndex, getTransitions, type Transition } from "./transitions";
import { type GraphAnalysis } from "./graphAnalysis";
import { STATE_TYPE_META } from "./stateMeta";
import { canAddChild, idIsTaken as idIsTakenOp } from "./editorOps";

interface ConversationInspectorProps {
  conversation: Conversation;
  selectedStateId: string | null;
  onSelect: (stateId: string) => void;
  analysis: GraphAnalysis;
  onUpdateState: (id: string, next: ConversationState) => void;
  onRename: (oldId: string, newId: string) => void;
  onDelete: (id: string) => void;
  onSwitchType: (id: string, nextType: ConversationStateType) => void;
  onAddChild: (id: string) => void;
  onInsertBetween: (
    sourceId: string,
    kind: Transition["kind"],
    ordinal: number,
  ) => void;
  onInsertBefore: (targetId: string) => void;
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
  onSwitchType,
  onAddChild,
  onInsertBetween,
  onInsertBefore,
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
        <TypePicker
          currentType={state.type}
          onSwitch={next => onSwitchType(state.id, next)}
          disabled={readOnly}
        />
        {!readOnly && (
          <Button
            size="sm"
            variant="outline"
            className="self-start"
            onClick={() => onInsertBefore(state.id)}
            title="Create a new state before this one (rewires inbound refs)"
          >
            <CornerDownRight className="h-3.5 w-3.5 -rotate-90" />
            Insert before
          </Button>
        )}
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
          onAddChild={
            !readOnly && canAddChild(state.type)
              ? () => onAddChild(state.id)
              : null
          }
          onInsertBetween={
            readOnly
              ? null
              : (kind, ordinal) => onInsertBetween(state.id, kind, ordinal)
          }
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

function TypePicker({
  currentType,
  onSwitch,
  disabled,
}: {
  currentType: ConversationStateType;
  onSwitch: (next: ConversationStateType) => void;
  disabled: boolean;
}) {
  return (
    <div className="flex items-center gap-2">
      <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground shrink-0">
        Type
      </Label>
      <Select
        value={currentType}
        onValueChange={v => {
          if (v !== currentType) onSwitch(v as ConversationStateType);
        }}
        disabled={disabled}
      >
        <SelectTrigger className="h-8 text-xs flex-1">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {(Object.keys(STATE_TYPE_META) as ConversationStateType[]).map(t => (
            <SelectItem key={t} value={t}>
              {STATE_TYPE_META[t].label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
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
      return readOnly ? (
        <Section title="List Selection">
          {state.listSelection ? (
            <KV
              label="title"
              value={
                state.listSelection.title || (
                  <em className="text-muted-foreground">(none)</em>
                )
              }
            />
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <ListSelectionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "askSlideMenu":
      return readOnly ? (
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
            </>
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <AskSlideMenuForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "askNumber":
      return readOnly ? (
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
            </>
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <AskNumberForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "askStyle":
      return readOnly ? (
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
            </>
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <AskStyleForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "craftAction":
      return readOnly ? (
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
            </>
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <CraftActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "transportAction":
      return readOnly ? (
        <Section title="Transport">
          {state.transportAction ? (
            <KV label="routeName" value={state.transportAction.routeName} mono />
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <TransportActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "partyQuestAction":
      return readOnly ? (
        <Section title="Party Quest">
          {state.partyQuestAction ? (
            <KV label="questId" value={state.partyQuestAction.questId} mono />
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <PartyQuestActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "partyQuestBonusAction":
      return readOnly ? (
        <Section title="PQ Bonus">
          <p className="text-xs text-muted-foreground">
            Warps party to the bonus stage.
          </p>
        </Section>
      ) : (
        <PartyQuestBonusActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "gachaponAction":
      return readOnly ? (
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
            </>
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <GachaponActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
      );
    case "genericAction":
      return readOnly ? (
        <Section title="Generic Action">
          {state.genericAction ? (
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
          ) : (
            <Empty />
          )}
        </Section>
      ) : (
        <GenericActionForm
          state={state}
          conversation={conversation}
          onUpdateState={onUpdateState}
        />
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

function ChoicesEditor({
  choices,
  onChange,
  otherIds,
  label = "Choices",
  itemLabel = "Choice",
}: {
  choices: DialogueChoice[];
  onChange: (next: DialogueChoice[]) => void;
  otherIds: string[];
  label?: string;
  itemLabel?: string;
}) {
  const updateChoice = (i: number, nextChoice: DialogueChoice) => {
    const next = [...choices];
    next[i] = nextChoice;
    onChange(next);
  };
  const addChoice = () =>
    onChange([...choices, { text: "", nextState: null }]);
  const removeChoice = (i: number) => {
    const next = [...choices];
    next.splice(i, 1);
    onChange(next);
  };

  return (
    <>
      <div className="flex items-center justify-between mt-2">
        <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          {label} ({choices.length})
        </Label>
        <Button size="sm" variant="outline" onClick={addChoice}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      <div className="flex flex-col gap-2">
        {choices.map((choice, i) => (
          <ChoiceRow
            key={i}
            index={i}
            itemLabel={itemLabel}
            choice={choice}
            otherIds={otherIds}
            onChange={next => updateChoice(i, next)}
            onRemove={() => removeChoice(i)}
          />
        ))}
      </div>
    </>
  );
}

function ChoiceRow({
  index,
  itemLabel,
  choice,
  otherIds,
  onChange,
  onRemove,
}: {
  index: number;
  itemLabel: string;
  choice: DialogueChoice;
  otherIds: string[];
  onChange: (next: DialogueChoice) => void;
  onRemove: () => void;
}) {
  const ctxKeys = choice.context ? Object.keys(choice.context) : [];
  const [expanded, setExpanded] = useState(false);

  const replaceContext = (ctx: Record<string, string>) => {
    if (Object.keys(ctx).length > 0) {
      onChange({ ...choice, context: ctx });
    } else {
      const { context: _removed, ...rest } = choice;
      void _removed;
      onChange(rest);
    }
  };
  const setKey = (oldKey: string, newKey: string) => {
    if (oldKey === newKey) return;
    const ctx = { ...(choice.context ?? {}) };
    const v = ctx[oldKey];
    delete ctx[oldKey];
    if (newKey !== "") ctx[newKey] = v ?? "";
    replaceContext(ctx);
  };
  const setValue = (key: string, value: string) => {
    replaceContext({ ...(choice.context ?? {}), [key]: value });
  };
  const removeKey = (key: string) => {
    const ctx = { ...(choice.context ?? {}) };
    delete ctx[key];
    replaceContext(ctx);
  };
  const addKey = () => {
    let i = 1;
    const ctx = { ...(choice.context ?? {}) };
    while (ctx[`key${i}`] !== undefined) i += 1;
    ctx[`key${i}`] = "";
    replaceContext(ctx);
    setExpanded(true);
  };

  return (
    <div className="flex flex-col gap-0.5">
      <div className="grid grid-cols-[1fr_140px_auto_auto] gap-1.5 items-start">
        <Input
          value={choice.text}
          onChange={e => onChange({ ...choice, text: e.target.value })}
          placeholder={`${itemLabel} ${index + 1}`}
          className="h-8 text-xs"
        />
        <Select
          value={choice.nextState || "__end__"}
          onValueChange={v =>
            onChange({ ...choice, nextState: v === "__end__" ? null : v })
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
          className="h-8 w-8"
          onClick={() =>
            ctxKeys.length > 0 ? setExpanded(v => !v) : addKey()
          }
          title={
            ctxKeys.length > 0
              ? `${ctxKeys.length} context ${ctxKeys.length === 1 ? "key" : "keys"}`
              : "Add context"
          }
          aria-label="Toggle context"
        >
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" />
          )}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-destructive"
          onClick={onRemove}
          title={`Remove ${itemLabel}`}
          aria-label={`Remove ${itemLabel}`}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      {ctxKeys.length > 0 && !expanded && (
        <span
          className="text-[10px] text-muted-foreground italic truncate pl-1"
          title={`context: ${ctxKeys.join(", ")}`}
        >
          sets {ctxKeys.join(", ")}
        </span>
      )}
      {expanded && (
        <div className="ml-3 flex flex-col gap-1 pl-2 border-l border-border/40">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">
              context ({ctxKeys.length})
            </span>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 text-[10px]"
              onClick={addKey}
            >
              <Plus className="h-3 w-3" />
              Add key
            </Button>
          </div>
          {ctxKeys.length === 0 ? (
            <p className="text-[11px] text-muted-foreground italic">
              (no context)
            </p>
          ) : (
            ctxKeys.map(k => (
              <div key={k} className="grid grid-cols-[1fr_1fr_auto] gap-1">
                <Input
                  defaultValue={k}
                  onBlur={e => setKey(k, e.target.value.trim())}
                  placeholder="key"
                  className="h-7 text-[11px] font-mono"
                />
                <Input
                  value={choice.context?.[k] ?? ""}
                  onChange={e => setValue(k, e.target.value)}
                  placeholder="value"
                  className="h-7 text-[11px] font-mono"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-destructive"
                  onClick={() => removeKey(k)}
                  title="Remove key"
                  aria-label="Remove key"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
            ))
          )}
        </div>
      )}
    </div>
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

  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);

  return (
    <Section title="Dialogue">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">dialogueType</Label>
        <Select
          value={d.dialogueType}
          onValueChange={v => update({ dialogueType: v as DialogueType })}
        >
          <SelectTrigger className="h-8 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="sendOk">sendOk</SelectItem>
            <SelectItem value="sendYesNo">sendYesNo</SelectItem>
            <SelectItem value="sendNext">sendNext</SelectItem>
            <SelectItem value="sendNextPrev">sendNextPrev</SelectItem>
            <SelectItem value="sendPrev">sendPrev</SelectItem>
            <SelectItem value="sendAcceptDecline">sendAcceptDecline</SelectItem>
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
      <ChoicesEditor
        choices={d.choices ?? []}
        onChange={next => update({ choices: next })}
        otherIds={otherIds}
      />
    </Section>
  );
}

function ListSelectionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const l: ListSelectionState = state.listSelection ?? { title: "", choices: [] };
  const update = (patch: Partial<ListSelectionState>) =>
    onUpdateState(state.id, {
      ...state,
      listSelection: { ...l, ...patch },
    });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  return (
    <Section title="List Selection">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">title</Label>
        <Input
          value={l.title}
          onChange={e => update({ title: e.target.value })}
          placeholder="Menu title (optional)"
          className="h-8 text-xs"
        />
      </div>
      <ChoicesEditor
        choices={l.choices ?? []}
        onChange={next => update({ choices: next })}
        otherIds={otherIds}
        label="Items"
        itemLabel="Item"
      />
    </Section>
  );
}

function AskSlideMenuForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const m: AskSlideMenuState = state.askSlideMenu ?? {
    title: "",
    menuType: 0,
    choices: [],
  };
  const update = (patch: Partial<AskSlideMenuState>) =>
    onUpdateState(state.id, {
      ...state,
      askSlideMenu: { ...m, ...patch },
    });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  return (
    <Section title="Slide Menu">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">title</Label>
        <Input
          value={m.title}
          onChange={e => update({ title: e.target.value })}
          placeholder="Menu title (optional)"
          className="h-8 text-xs"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">menuType</Label>
        <Input
          type="number"
          value={m.menuType ?? 0}
          onChange={e => update({ menuType: Number(e.target.value) })}
          className="h-8 text-xs"
        />
      </div>
      <ChoicesEditor
        choices={m.choices ?? []}
        onChange={next => update({ choices: next })}
        otherIds={otherIds}
        label="Options"
        itemLabel="Option"
      />
    </Section>
  );
}

function AskNumberForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const a: AskNumberState = state.askNumber ?? {
    text: "",
    defaultValue: 0,
    minValue: 0,
    maxValue: 0,
    nextState: "",
  };
  const update = (patch: Partial<AskNumberState>) =>
    onUpdateState(state.id, { ...state, askNumber: { ...a, ...patch } });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  return (
    <Section title="Ask Number">
      <div className="grid grid-cols-[90px_1fr] gap-2">
        <Label className="text-xs text-muted-foreground pt-1">text</Label>
        <Textarea
          value={a.text}
          onChange={e => update({ text: e.target.value })}
          className="min-h-[72px] text-xs"
          placeholder="Prompt text"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr_1fr_1fr] gap-2 items-center">
        <span className="text-xs text-muted-foreground">range</span>
        <Input
          type="number"
          value={a.minValue}
          onChange={e => update({ minValue: Number(e.target.value) })}
          placeholder="min"
          className="h-8 text-xs"
        />
        <Input
          type="number"
          value={a.defaultValue}
          onChange={e => update({ defaultValue: Number(e.target.value) })}
          placeholder="default"
          className="h-8 text-xs"
        />
        <Input
          type="number"
          value={a.maxValue}
          onChange={e => update({ maxValue: Number(e.target.value) })}
          placeholder="max"
          className="h-8 text-xs"
        />
      </div>
      <OptionalTextField
        label="contextKey"
        value={a.contextKey}
        onChange={v => update({ contextKey: v })}
        onRemove={() => {
          const next = { ...a };
          delete next.contextKey;
          onUpdateState(state.id, { ...state, askNumber: next });
        }}
      />
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">answer →</Label>
        <StatePicker
          value={a.nextState}
          onChange={v => update({ nextState: v })}
          otherIds={otherIds}
        />
      </div>
    </Section>
  );
}

function AskStyleForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const a: AskStyleState = state.askStyle ?? { text: "", nextState: "" };
  const update = (patch: Partial<AskStyleState>) =>
    onUpdateState(state.id, { ...state, askStyle: { ...a, ...patch } });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  const stylesText = (a.styles ?? []).join(", ");
  return (
    <Section title="Ask Style">
      <div className="grid grid-cols-[90px_1fr] gap-2">
        <Label className="text-xs text-muted-foreground pt-1">text</Label>
        <Textarea
          value={a.text}
          onChange={e => update({ text: e.target.value })}
          className="min-h-[72px] text-xs"
          placeholder="Prompt text"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">styles</Label>
        <Input
          value={stylesText}
          onChange={e => {
            const parts = e.target.value
              .split(",")
              .map(s => s.trim())
              .filter(s => s.length > 0)
              .map(s => Number(s))
              .filter(n => !Number.isNaN(n));
            update({ styles: parts });
          }}
          placeholder="comma-separated style ids"
          className="h-8 text-xs font-mono"
        />
      </div>
      <OptionalTextField
        label="stylesContextKey"
        value={a.stylesContextKey}
        onChange={v => update({ stylesContextKey: v })}
        onRemove={() => {
          const next = { ...a };
          delete next.stylesContextKey;
          onUpdateState(state.id, { ...state, askStyle: next });
        }}
      />
      <OptionalTextField
        label="contextKey"
        value={a.contextKey}
        onChange={v => update({ contextKey: v })}
        onRemove={() => {
          const next = { ...a };
          delete next.contextKey;
          onUpdateState(state.id, { ...state, askStyle: next });
        }}
      />
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">selection →</Label>
        <StatePicker
          value={a.nextState}
          onChange={v => update({ nextState: v })}
          otherIds={otherIds}
        />
      </div>
    </Section>
  );
}

function TransportActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const t: TransportActionState = state.transportAction ?? {
    routeName: "",
    failureState: "",
  };
  const update = (patch: Partial<TransportActionState>) =>
    onUpdateState(state.id, { ...state, transportAction: { ...t, ...patch } });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  const removeKey = (key: keyof TransportActionState) => {
    const next = { ...t };
    delete next[key];
    onUpdateState(state.id, { ...state, transportAction: next });
  };

  return (
    <Section title="Transport">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">routeName</Label>
        <Input
          value={t.routeName}
          onChange={e => update({ routeName: e.target.value })}
          placeholder="route identifier"
          className="h-8 text-xs font-mono"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">failure →</Label>
        <StatePicker
          value={t.failureState}
          onChange={v => update({ failureState: v })}
          otherIds={otherIds}
        />
      </div>
      <OptionalSlot
        label="capacity full"
        value={t.capacityFullState}
        otherIds={otherIds}
        onChange={v => update({ capacityFullState: v })}
        onRemove={() => removeKey("capacityFullState")}
      />
      <OptionalSlot
        label="in transit"
        value={t.alreadyInTransitState}
        otherIds={otherIds}
        onChange={v => update({ alreadyInTransitState: v })}
        onRemove={() => removeKey("alreadyInTransitState")}
      />
      <OptionalSlot
        label="route missing"
        value={t.routeNotFoundState}
        otherIds={otherIds}
        onChange={v => update({ routeNotFoundState: v })}
        onRemove={() => removeKey("routeNotFoundState")}
      />
      <OptionalSlot
        label="service error"
        value={t.serviceErrorState}
        otherIds={otherIds}
        onChange={v => update({ serviceErrorState: v })}
        onRemove={() => removeKey("serviceErrorState")}
      />
    </Section>
  );
}

function PartyQuestActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const p: PartyQuestActionState = state.partyQuestAction ?? {
    questId: "",
    failureState: "",
  };
  const update = (patch: Partial<PartyQuestActionState>) =>
    onUpdateState(state.id, {
      ...state,
      partyQuestAction: { ...p, ...patch },
    });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  const removeKey = (key: keyof PartyQuestActionState) => {
    const next = { ...p };
    delete next[key];
    onUpdateState(state.id, { ...state, partyQuestAction: next });
  };

  return (
    <Section title="Party Quest">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">questId</Label>
        <Input
          value={p.questId}
          onChange={e => update({ questId: e.target.value })}
          placeholder="quest identifier"
          className="h-8 text-xs font-mono"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">failure →</Label>
        <StatePicker
          value={p.failureState}
          onChange={v => update({ failureState: v })}
          otherIds={otherIds}
        />
      </div>
      <OptionalSlot
        label="not in party"
        value={p.notInPartyState}
        otherIds={otherIds}
        onChange={v => update({ notInPartyState: v })}
        onRemove={() => removeKey("notInPartyState")}
      />
      <OptionalSlot
        label="not leader"
        value={p.notLeaderState}
        otherIds={otherIds}
        onChange={v => update({ notLeaderState: v })}
        onRemove={() => removeKey("notLeaderState")}
      />
    </Section>
  );
}

function PartyQuestBonusActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const p: PartyQuestBonusActionState =
    state.partyQuestBonusAction ?? { failureState: "" };
  const update = (patch: Partial<PartyQuestBonusActionState>) =>
    onUpdateState(state.id, {
      ...state,
      partyQuestBonusAction: { ...p, ...patch },
    });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  return (
    <Section title="PQ Bonus">
      <p className="text-xs text-muted-foreground">
        Warps the party to the bonus stage.
      </p>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">failure →</Label>
        <StatePicker
          value={p.failureState}
          onChange={v => update({ failureState: v })}
          otherIds={otherIds}
        />
      </div>
    </Section>
  );
}

function GachaponActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const g: GachaponActionState = state.gachaponAction ?? {
    gachaponId: "",
    ticketItemId: 0,
    failureState: "",
  };
  const update = (patch: Partial<GachaponActionState>) =>
    onUpdateState(state.id, { ...state, gachaponAction: { ...g, ...patch } });
  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);
  return (
    <Section title="Gachapon">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">gachaponId</Label>
        <Input
          value={g.gachaponId}
          onChange={e => update({ gachaponId: e.target.value })}
          placeholder="gachapon identifier"
          className="h-8 text-xs font-mono"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">ticketItemId</Label>
        <Input
          type="number"
          value={g.ticketItemId}
          onChange={e => update({ ticketItemId: Number(e.target.value) })}
          className="h-8 text-xs"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">failure →</Label>
        <StatePicker
          value={g.failureState}
          onChange={v => update({ failureState: v })}
          otherIds={otherIds}
        />
      </div>
    </Section>
  );
}

function OptionalTextField({
  label,
  value,
  onChange,
  onRemove,
}: {
  label: string;
  value: string | undefined;
  onChange: (v: string) => void;
  onRemove: () => void;
}) {
  const enabled = value !== undefined;
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          className="h-3.5 w-3.5"
          checked={enabled}
          onChange={e => {
            if (e.target.checked) onChange("");
            else onRemove();
          }}
        />
        <Label className="text-xs">{label}</Label>
      </div>
      {enabled && (
        <Input
          value={value ?? ""}
          onChange={e => onChange(e.target.value)}
          className="h-8 text-xs font-mono ml-5 w-[calc(100%-1.25rem)]"
        />
      )}
    </div>
  );
}

function OptionalSlot({
  label,
  value,
  otherIds,
  onChange,
  onRemove,
}: {
  label: string;
  value: string | undefined;
  otherIds: string[];
  onChange: (next: string) => void;
  onRemove: () => void;
}) {
  const enabled = value !== undefined;
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          className="h-3.5 w-3.5"
          checked={enabled}
          onChange={e => {
            if (e.target.checked) onChange("");
            else onRemove();
          }}
        />
        <Label className="text-xs">{label} →</Label>
      </div>
      {enabled && (
        <div className="ml-5">
          <StatePicker
            value={value ?? ""}
            onChange={onChange}
            otherIds={otherIds}
          />
        </div>
      )}
    </div>
  );
}

function StatePicker({
  value,
  onChange,
  otherIds,
  allowEnd = true,
}: {
  value: string | null;
  onChange: (next: string) => void;
  otherIds: string[];
  allowEnd?: boolean;
}) {
  return (
    <Select
      value={value || (allowEnd ? "__end__" : "")}
      onValueChange={v => onChange(v === "__end__" ? "" : v)}
    >
      <SelectTrigger className="h-8 text-xs">
        <SelectValue placeholder="select state" />
      </SelectTrigger>
      <SelectContent>
        {allowEnd && <SelectItem value="__end__">&lt;end&gt;</SelectItem>}
        {otherIds.map(id => (
          <SelectItem key={id} value={id}>
            {id}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function CraftActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const c: CraftActionState = state.craftAction ?? {
    itemId: 0,
    materials: [],
    quantities: [],
    mesoCost: 0,
    successState: "",
    failureState: "",
    missingMaterialsState: "",
  };
  const update = (patch: Partial<CraftActionState>) =>
    onUpdateState(state.id, {
      ...state,
      craftAction: { ...c, ...patch },
    });

  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);

  const rows = Math.max(c.materials?.length ?? 0, c.quantities?.length ?? 0);
  const materials = Array.from(
    { length: rows },
    (_, i) => c.materials?.[i] ?? 0,
  );
  const quantities = Array.from(
    { length: rows },
    (_, i) => c.quantities?.[i] ?? 0,
  );

  const setRow = (i: number, patch: { material?: number; quantity?: number }) => {
    const nextMats = [...materials];
    const nextQtys = [...quantities];
    if (patch.material !== undefined) nextMats[i] = patch.material;
    if (patch.quantity !== undefined) nextQtys[i] = patch.quantity;
    update({ materials: nextMats, quantities: nextQtys });
  };
  const addRow = () =>
    update({ materials: [...materials, 0], quantities: [...quantities, 0] });
  const removeRow = (i: number) => {
    const nextMats = [...materials];
    const nextQtys = [...quantities];
    nextMats.splice(i, 1);
    nextQtys.splice(i, 1);
    update({ materials: nextMats, quantities: nextQtys });
  };

  const stimulatorEnabled = c.stimulatorId !== undefined;

  return (
    <Section title="Craft">
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">itemId</Label>
        <Input
          type="number"
          value={c.itemId ?? 0}
          onChange={e => update({ itemId: Number(e.target.value) })}
          className="h-8 text-xs"
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">mesoCost</Label>
        <Input
          type="number"
          value={c.mesoCost ?? 0}
          onChange={e => update({ mesoCost: Number(e.target.value) })}
          className="h-8 text-xs"
        />
      </div>

      <div className="flex items-center justify-between mt-2">
        <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          Materials ({rows})
        </Label>
        <Button size="sm" variant="outline" onClick={addRow}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      <div className="flex flex-col gap-1.5">
        {rows === 0 && (
          <p className="text-[11px] text-muted-foreground italic">
            No materials required.
          </p>
        )}
        {Array.from({ length: rows }).map((_, i) => (
          <div
            key={i}
            className="grid grid-cols-[1fr_90px_auto] gap-1.5 items-center"
          >
            <Input
              type="number"
              value={materials[i]!}
              onChange={e =>
                setRow(i, { material: Number(e.target.value) })
              }
              placeholder="item id"
              className="h-8 text-xs"
            />
            <Input
              type="number"
              value={quantities[i]!}
              onChange={e =>
                setRow(i, { quantity: Number(e.target.value) })
              }
              placeholder="qty"
              className="h-8 text-xs"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive"
              onClick={() => removeRow(i)}
              title="Remove material"
              aria-label="Remove material"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-[90px_1fr] gap-2 items-center mt-2">
        <Label className="text-xs text-muted-foreground">success →</Label>
        <StatePicker
          value={c.successState}
          onChange={v => update({ successState: v })}
          otherIds={otherIds}
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">failure →</Label>
        <StatePicker
          value={c.failureState}
          onChange={v => update({ failureState: v })}
          otherIds={otherIds}
        />
      </div>
      <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
        <Label className="text-xs text-muted-foreground">missing →</Label>
        <StatePicker
          value={c.missingMaterialsState}
          onChange={v => update({ missingMaterialsState: v })}
          otherIds={otherIds}
        />
      </div>

      <div className="flex items-center gap-2 mt-3">
        <input
          id="stim-enable"
          type="checkbox"
          className="h-3.5 w-3.5"
          checked={stimulatorEnabled}
          onChange={e => {
            if (e.target.checked) {
              update({ stimulatorId: 0, stimulatorFailChance: 0 });
            } else {
              const next = { ...c };
              delete next.stimulatorId;
              delete next.stimulatorFailChance;
              onUpdateState(state.id, { ...state, craftAction: next });
            }
          }}
        />
        <Label htmlFor="stim-enable" className="text-xs">
          Has stimulator
        </Label>
      </div>
      {stimulatorEnabled && (
        <>
          <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
            <Label className="text-xs text-muted-foreground">stim item</Label>
            <Input
              type="number"
              value={c.stimulatorId ?? 0}
              onChange={e =>
                update({ stimulatorId: Number(e.target.value) })
              }
              className="h-8 text-xs"
            />
          </div>
          <div className="grid grid-cols-[90px_1fr] gap-2 items-center">
            <Label className="text-xs text-muted-foreground">fail %</Label>
            <Input
              type="number"
              value={c.stimulatorFailChance ?? 0}
              onChange={e =>
                update({ stimulatorFailChance: Number(e.target.value) })
              }
              className="h-8 text-xs"
            />
          </div>
        </>
      )}
    </Section>
  );
}

function GenericActionForm({
  state,
  conversation,
  onUpdateState,
}: {
  state: ConversationState;
  conversation: Conversation;
  onUpdateState: (id: string, next: ConversationState) => void;
}) {
  const g: GenericActionState = state.genericAction ?? {
    operations: [],
    outcomes: [],
  };
  const update = (patch: Partial<GenericActionState>) =>
    onUpdateState(state.id, {
      ...state,
      genericAction: { ...g, ...patch },
    });

  const otherIds = conversation.attributes.states
    .map(s => s.id)
    .filter(id => id !== state.id);

  const updateOperation = (
    i: number,
    next: GenericActionOperation,
  ) => {
    const ops = [...(g.operations ?? [])];
    ops[i] = next;
    update({ operations: ops });
  };
  const removeOperation = (i: number) => {
    const ops = [...(g.operations ?? [])];
    ops.splice(i, 1);
    update({ operations: ops });
  };
  const addOperation = () =>
    update({
      operations: [...(g.operations ?? []), { type: "" }],
    });

  const updateOutcome = (i: number, next: GenericActionOutcome) => {
    const outs = [...(g.outcomes ?? [])];
    outs[i] = next;
    update({ outcomes: outs });
  };
  const removeOutcome = (i: number) => {
    const outs = [...(g.outcomes ?? [])];
    outs.splice(i, 1);
    update({ outcomes: outs });
  };
  const addOutcome = () =>
    update({
      outcomes: [...(g.outcomes ?? []), { conditions: [], nextState: "" }],
    });

  return (
    <Section title="Generic Action">
      <div className="flex items-center justify-between">
        <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          Operations ({g.operations?.length ?? 0})
        </Label>
        <Button size="sm" variant="outline" onClick={addOperation}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      {(g.operations ?? []).length === 0 && (
        <p className="text-[11px] text-muted-foreground italic">
          No operations.
        </p>
      )}
      <div className="flex flex-col gap-2">
        {(g.operations ?? []).map((op, i) => (
          <OperationEditor
            key={i}
            op={op}
            onChange={next => updateOperation(i, next)}
            onRemove={() => removeOperation(i)}
          />
        ))}
      </div>

      <div className="flex items-center justify-between mt-3">
        <Label className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">
          Outcomes ({g.outcomes?.length ?? 0})
        </Label>
        <Button size="sm" variant="outline" onClick={addOutcome}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      {(g.outcomes ?? []).length === 0 && (
        <p className="text-[11px] text-muted-foreground italic">
          No outcomes. Add one to route the action.
        </p>
      )}
      <div className="flex flex-col gap-2">
        {(g.outcomes ?? []).map((outcome, i) => (
          <div
            key={i}
            className="rounded-md border bg-muted/30 p-2 flex flex-col gap-1.5"
          >
            <div className="flex items-center justify-between gap-2">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
                Outcome {i + 1}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-6 w-6 text-destructive"
                onClick={() => removeOutcome(i)}
                title="Remove outcome"
                aria-label="Remove outcome"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
            <ConditionsEditor
              conditions={outcome.conditions ?? []}
              onChange={next =>
                updateOutcome(i, { ...outcome, conditions: next })
              }
            />
            <div className="grid grid-cols-[60px_1fr] gap-2 items-center">
              <Label className="text-xs text-muted-foreground">→</Label>
              <StatePicker
                value={outcome.nextState}
                onChange={v =>
                  updateOutcome(i, { ...outcome, nextState: v })
                }
                otherIds={otherIds}
              />
            </div>
          </div>
        ))}
      </div>
    </Section>
  );
}

function ConditionsEditor({
  conditions,
  onChange,
}: {
  conditions: Condition[];
  onChange: (next: Condition[]) => void;
}) {
  const update = (i: number, patch: Partial<Condition>) => {
    const next = [...conditions];
    const current = next[i] ?? { type: "", operator: "", value: "" };
    next[i] = { ...current, ...patch };
    onChange(next);
  };
  const remove = (i: number) => {
    const next = [...conditions];
    next.splice(i, 1);
    onChange(next);
  };
  const add = () =>
    onChange([...conditions, { type: "", operator: "", value: "" }]);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between">
        <Label className="text-[10px] text-muted-foreground">
          Conditions ({conditions.length})
        </Label>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 text-[10px]"
          onClick={add}
        >
          <Plus className="h-3 w-3" />
          Add condition
        </Button>
      </div>
      {conditions.length === 0 ? (
        <p className="text-[11px] text-muted-foreground italic">
          No conditions (always fires)
        </p>
      ) : (
        <div className="flex flex-col gap-1">
          {conditions.map((c, i) => (
            <div
              key={i}
              className="grid grid-cols-[1fr_1fr_1fr_auto] gap-1 items-center"
            >
              <Input
                value={c.type}
                onChange={e => update(i, { type: e.target.value })}
                placeholder="type"
                className="h-7 text-[11px] font-mono"
              />
              <Input
                value={c.operator}
                onChange={e => update(i, { operator: e.target.value })}
                placeholder="op"
                className="h-7 text-[11px] font-mono"
              />
              <Input
                value={c.value}
                onChange={e => update(i, { value: e.target.value })}
                placeholder="value"
                className="h-7 text-[11px] font-mono"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-destructive"
                onClick={() => remove(i)}
                title="Remove condition"
                aria-label="Remove condition"
              >
                <Trash2 className="h-3 w-3" />
              </Button>
              {c.referenceId !== undefined ? (
                <Input
                  value={c.referenceId}
                  onChange={e =>
                    update(i, { referenceId: e.target.value })
                  }
                  placeholder="referenceId"
                  className="col-span-4 h-7 text-[11px] font-mono"
                />
              ) : (
                <button
                  type="button"
                  onClick={() => update(i, { referenceId: "" })}
                  className="col-span-4 text-[10px] text-muted-foreground hover:text-foreground italic text-left pl-0.5"
                >
                  + add referenceId
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function OperationEditor({
  op,
  onChange,
  onRemove,
}: {
  op: GenericActionOperation;
  onChange: (next: GenericActionOperation) => void;
  onRemove: () => void;
}) {
  const params = op.params ?? {};
  const entries = Object.entries(params);

  const setKey = (oldKey: string, newKey: string) => {
    if (newKey === oldKey) return;
    const next = { ...params };
    const v = next[oldKey];
    delete next[oldKey];
    if (newKey !== "") next[newKey] = v ?? "";
    onChange({ ...op, params: next });
  };
  const setValue = (key: string, value: string) => {
    onChange({ ...op, params: { ...params, [key]: value } });
  };
  const removeKey = (key: string) => {
    const next = { ...params };
    delete next[key];
    onChange({ ...op, params: next });
  };
  const addKey = () => {
    const placeholder = deriveUniqueParamKey(params);
    onChange({ ...op, params: { ...params, [placeholder]: "" } });
  };

  return (
    <div className="rounded-md border bg-muted/30 p-2 flex flex-col gap-1.5">
      <div className="grid grid-cols-[1fr_auto] gap-1.5 items-center">
        <Input
          value={op.type}
          onChange={e => onChange({ ...op, type: e.target.value })}
          placeholder="operation type"
          className="h-8 text-xs font-mono"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-destructive"
          onClick={onRemove}
          title="Remove operation"
          aria-label="Remove operation"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      <div className="flex items-center justify-between pl-1">
        <span className="text-[10px] text-muted-foreground">
          params ({entries.length})
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 text-[10px]"
          onClick={addKey}
        >
          <Plus className="h-3 w-3" />
          Add param
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className="text-[11px] text-muted-foreground italic pl-1">
          (no params)
        </p>
      ) : (
        <div className="flex flex-col gap-1">
          {entries.map(([k, v]) => (
            <div key={k} className="grid grid-cols-[1fr_1fr_auto] gap-1.5">
              <Input
                defaultValue={k}
                onBlur={e => setKey(k, e.target.value.trim())}
                placeholder="key"
                className="h-7 text-[11px] font-mono"
              />
              <Input
                value={v}
                onChange={e => setValue(k, e.target.value)}
                placeholder="value"
                className="h-7 text-[11px] font-mono"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-destructive"
                onClick={() => removeKey(k)}
                title="Remove param"
                aria-label="Remove param"
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function deriveUniqueParamKey(params: Record<string, string>): string {
  let i = 1;
  while (params[`key${i}`] !== undefined) i += 1;
  return `key${i}`;
}


function TransitionsSection({
  transitions,
  analysis,
  onSelect,
  onAddChild,
  onInsertBetween,
}: {
  transitions: ReturnType<typeof getTransitions>;
  analysis: GraphAnalysis;
  onSelect: (stateId: string) => void;
  onAddChild: (() => void) | null;
  onInsertBetween:
    | ((kind: Transition["kind"], ordinal: number) => void)
    | null;
}) {
  const action = onAddChild ? (
    <Button size="sm" variant="outline" onClick={onAddChild}>
      <Plus className="h-3 w-3" />
      Add child
    </Button>
  ) : undefined;
  if (transitions.length === 0) {
    return (
      <Section
        title="Transitions"
        {...(action && { action })}
      >
        <p className="text-xs text-muted-foreground">No outgoing transitions.</p>
      </Section>
    );
  }
  return (
    <Section
      title={`Transitions (${transitions.length})`}
      {...(action && { action })}
    >
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
              {onInsertBetween && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => onInsertBetween(t.kind, t.ordinal)}
                  title="Insert a new state between source and target"
                  aria-label="Insert between"
                >
                  <GitBranch className="h-3 w-3" />
                </Button>
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
