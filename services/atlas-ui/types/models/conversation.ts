// NPC Conversation domain model types
// Re-exported from lib/npc-conversations.tsx to centralize type definitions

export interface DialogueChoice {
    text: string;
    nextState: string | null;
    context?: Record<string, string>;
}

export interface DialogueState {
    dialogueType: "sendOk" | "sendYesNo" | "sendSimple" | "sendNext";
    text: string;
    choices: DialogueChoice[];
    exit?: boolean;
}

export interface GenericActionOperation {
    type: string;
    params?: Record<string, never>;
}

export interface Condition {
    type: string;
    operator: string;
    value: string;
}

export interface GenericActionOutcome {
    conditions: Condition[];
    operations: GenericActionOperation[];
}

export interface GenericAction {
    type: string;
    outcomes: GenericActionOutcome[];
}

export interface ConversationState {
    id: string;
    dialogue?: DialogueState;
    actions?: GenericAction[];
}

export interface Conversation {
    id: string;
    type: string;
    attributes: ConversationAttributes;
}

export interface ConversationAttributes {
    npcId: number;
    initial: string;
    states: ConversationState[];
}