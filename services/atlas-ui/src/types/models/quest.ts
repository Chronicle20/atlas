// Quest definition types (from atlas-data)

export interface QuestDefinition {
    id: string;
    type: "quests";
    attributes: QuestAttributes;
}

export interface QuestAttributes {
    name: string;
    parent?: string;           // Category
    area: number;
    order?: number;
    autoStart: boolean;
    autoPreComplete: boolean;
    autoComplete: boolean;
    timeLimit?: number;
    timeLimit2?: number;
    selectedMob?: boolean;
    summary?: string;
    demandSummary?: string;
    rewardSummary?: string;
    startRequirements: QuestRequirements;
    endRequirements: QuestRequirements;
    startActions: QuestActions;
    endActions: QuestActions;
}

export interface QuestRequirements {
    npcId?: number;
    levelMin?: number;
    levelMax?: number;
    fameMin?: number;
    mesoMin?: number;
    mesoMax?: number;
    jobs?: number[];
    quests?: QuestRequirement[];
    items?: ItemRequirement[];
    mobs?: MobRequirement[];
    fieldEnter?: number[];
    pet?: number[];
    petTamenessMin?: number;
    dayOfWeek?: string;
    start?: string;
    end?: string;
    interval?: number;
    startScript?: string;
    endScript?: string;
    infoNumber?: number;
    normalAutoStart?: boolean;
    completionCount?: number;
}

export interface QuestRequirement {
    id: number;
    state: number;  // 0 = not started, 1 = started, 2 = completed
}

export interface ItemRequirement {
    id: number;
    count: number;  // Can be negative for removal
}

export interface MobRequirement {
    id: number;
    count: number;
}

export interface QuestActions {
    npcId?: number;
    exp?: number;
    money?: number;
    fame?: number;
    items?: ItemReward[];
    skills?: SkillReward[];
    nextQuest?: number;
    buffItemId?: number;
    interval?: number;
    levelMin?: number;
}

export interface ItemReward {
    id: number;
    count: number;
    job?: number;
    gender?: number;   // -1 = any, 0 = male, 1 = female
    prop?: number;     // Probability (-1 = guaranteed, 0+ = chance)
    period?: number;   // Duration in minutes (0 = permanent)
    dateExpire?: string;
    var?: number;
}

export interface SkillReward {
    id: number;
    level?: number;    // -1 = remove skill
    masterLevel?: number;
    jobs?: number[];
}

// Character quest status types (from atlas-quest)

export type QuestState = 0 | 1 | 2;  // 0 = not started, 1 = started, 2 = completed

export const QuestStateLabels: Record<QuestState, string> = {
    0: "Not Started",
    1: "Started",
    2: "Completed",
};

export interface CharacterQuestStatus {
    id: string;
    type: "quest-status";
    attributes: CharacterQuestStatusAttributes;
}

export interface CharacterQuestStatusAttributes {
    characterId: number;
    questId: number;
    state: QuestState;
    startedAt: string;
    completedAt?: string;
    expirationTime?: string;
    completedCount: number;
    forfeitCount: number;
    progress: QuestProgress[];
}

export interface QuestProgress {
    infoNumber: number;
    progress: string;
}

// API Response types
export interface QuestDefinitionResponse {
    data: QuestDefinition;
}

export interface QuestDefinitionsResponse {
    data: QuestDefinition[];
}

export interface CharacterQuestStatusResponse {
    data: CharacterQuestStatus;
}

export interface CharacterQuestStatusesResponse {
    data: CharacterQuestStatus[];
}
