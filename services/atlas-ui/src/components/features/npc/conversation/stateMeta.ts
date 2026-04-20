import type { ConversationState, ConversationStateType } from "@/types/models/conversation";

export interface StateTypeMeta {
  label: string;
  accent: string;
}

export const STATE_TYPE_META: Record<ConversationStateType, StateTypeMeta> = {
  dialogue: {
    label: "Dialogue",
    accent: "bg-sky-500/15 text-sky-700 dark:text-sky-300 border-sky-500/30",
  },
  genericAction: {
    label: "Action",
    accent: "bg-violet-500/15 text-violet-700 dark:text-violet-300 border-violet-500/30",
  },
  craftAction: {
    label: "Craft",
    accent: "bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/30",
  },
  listSelection: {
    label: "List",
    accent: "bg-teal-500/15 text-teal-700 dark:text-teal-300 border-teal-500/30",
  },
  askNumber: {
    label: "Number",
    accent: "bg-indigo-500/15 text-indigo-700 dark:text-indigo-300 border-indigo-500/30",
  },
  askStyle: {
    label: "Style",
    accent: "bg-pink-500/15 text-pink-700 dark:text-pink-300 border-pink-500/30",
  },
  askSlideMenu: {
    label: "Slide Menu",
    accent: "bg-teal-500/15 text-teal-700 dark:text-teal-300 border-teal-500/30",
  },
  transportAction: {
    label: "Transport",
    accent: "bg-cyan-500/15 text-cyan-700 dark:text-cyan-300 border-cyan-500/30",
  },
  partyQuestAction: {
    label: "Party Quest",
    accent: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
  },
  partyQuestBonusAction: {
    label: "PQ Bonus",
    accent: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
  },
  gachaponAction: {
    label: "Gachapon",
    accent: "bg-fuchsia-500/15 text-fuchsia-700 dark:text-fuchsia-300 border-fuchsia-500/30",
  },
};

function truncate(text: string, max = 120): string {
  if (!text) return "";
  if (text.length <= max) return text;
  return text.slice(0, max - 1).trimEnd() + "…";
}

export function describeState(state: ConversationState): string {
  switch (state.type) {
    case "dialogue":
      return truncate(state.dialogue?.text ?? "", 140);
    case "genericAction": {
      const ops = state.genericAction?.operations ?? [];
      const outcomes = state.genericAction?.outcomes ?? [];
      const opLabels = ops.slice(0, 2).map(o => o.type).join(", ");
      const opsPart = opLabels
        ? `${opLabels}${ops.length > 2 ? `, +${ops.length - 2}` : ""}`
        : `${ops.length} op${ops.length === 1 ? "" : "s"}`;
      return `${opsPart} · ${outcomes.length} outcome${outcomes.length === 1 ? "" : "s"}`;
    }
    case "craftAction":
      return state.craftAction
        ? `item ${state.craftAction.itemId} · ${state.craftAction.materials?.length ?? 0} mats`
        : "";
    case "listSelection":
      return state.listSelection
        ? state.listSelection.title
          ? `${truncate(state.listSelection.title, 90)} · ${state.listSelection.choices?.length ?? 0} items`
          : `${state.listSelection.choices?.length ?? 0} items`
        : "";
    case "askSlideMenu": {
      const m = state.askSlideMenu;
      if (!m) return "";
      const n = m.choices?.length ?? 0;
      return m.title ? `${truncate(m.title, 90)} · ${n} options` : `${n} options`;
    }
    case "askNumber": {
      const a = state.askNumber;
      if (!a) return "";
      const range =
        a.minValue !== undefined && a.maxValue !== undefined
          ? `[${a.minValue}–${a.maxValue}]`
          : "";
      return [truncate(a.text ?? "", 100), range].filter(Boolean).join(" ");
    }
    case "askStyle": {
      const a = state.askStyle;
      if (!a) return "";
      const n = a.styles?.length ?? 0;
      return [truncate(a.text ?? "", 100), n ? `${n} styles` : ""]
        .filter(Boolean)
        .join(" · ");
    }
    case "transportAction":
      return state.transportAction?.routeName
        ? `route ${state.transportAction.routeName}`
        : "";
    case "partyQuestAction":
      return state.partyQuestAction?.questId
        ? `quest ${state.partyQuestAction.questId}`
        : "";
    case "partyQuestBonusAction":
      return "PQ bonus warp";
    case "gachaponAction":
      return state.gachaponAction
        ? `${state.gachaponAction.gachaponId} · ticket ${state.gachaponAction.ticketItemId}`
        : "";
    default:
      return "";
  }
}
