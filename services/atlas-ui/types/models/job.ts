// Job domain model types
// Re-exported from lib/jobs.ts to centralize type definitions

// Job ID to name mapping
export interface JobNameMap {
    [key: number]: string;
}

export type JobId = number;
export type JobName = string;