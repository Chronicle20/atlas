// Guild domain model types
// Re-exported from lib/guilds.tsx to centralize type definitions

export interface Guild {
    id: string;
    attributes: GuildAttributes;
}

export interface GuildAttributes {
    worldId: number;
    name: string;
    notice: string;
    points: number;
    capacity: number;
    logo: number;
    logoColor: number;
    logoBackground: number;
    logoBackgroundColor: number;
    leaderId: number;
    members: GuildMember[];
    titles: GuildTitle[];
}

export interface GuildMember {
    characterId: number;
    name: string;
    jobId: number;
    level: number;
    title: number;
    online: boolean;
    allianceTitle: number;
}

export interface GuildTitle {
    name: string;
    index: number;
}