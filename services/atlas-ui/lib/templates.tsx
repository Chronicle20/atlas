import type {ApiListResponse, ApiSingleResponse} from "@/types/api/responses";
import type {Template as TemplateModel, TemplateAttributes as TemplateAttributesModel, CharacterTemplate as CharacterTemplateModel} from "@/types/models/template";

export interface CharacterTemplate {
    jobIndex: number;
    subJobIndex: number;
    gender: number;
    mapId: number;
    faces: number[];
    hairs: number[];
    hairColors: number[];
    skinColors: number[];
    tops: number[];
    bottoms: number[];
    shoes: number[];
    weapons: number[];
    items: number[];
    skills: number[];
}

export interface TemplateAttributes {
    region: string;
    majorVersion: number;
    minorVersion: number;
    usesPin: boolean;
    characters: {
        templates: CharacterTemplate[];
    };
    npcs: {
        npcId: number;
        impl: string;
    }[];
    socket: {
        handlers: {
            opCode: string;
            validator: string;
            handler: string;
            options: unknown;
        }[];
        writers: {
            opCode: string;
            writer: string;
            options: unknown;
        }[];
    }
    worlds: {
        name: string;
        flag: string;
        serverMessage: string;
        eventMessage: string;
        whyAmIRecommended: string;
    }[];
}

export interface Template {
    id: string;
    attributes: TemplateAttributes;
}

export async function fetchTemplates(): Promise<Template[]> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/configurations/templates", {
        method: "GET",
        headers: {
            "Content-Type": "application/json",
        },
    });
    if (!response.ok) {
        throw new Error("Failed to fetch templates.");
    }
    const responseData: ApiListResponse<Template> = await response.json();
    return responseData.data
        .map((template: Template) => ({
            ...template,
            attributes: {
                ...template.attributes,
                socket: {
                    ...template.attributes.socket,
                    handlers: [...template.attributes.socket.handlers].sort(
                        (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
                    ),
                    writers: [...template.attributes.socket.writers].sort(
                        (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
                    ),
                },
            },
        }))
        .sort((a: Template, b: Template) => {
            if (a.attributes.region !== b.attributes.region) {
                return a.attributes.region.localeCompare(b.attributes.region);
            }
            if (a.attributes.majorVersion !== b.attributes.majorVersion) {
                return a.attributes.majorVersion - b.attributes.majorVersion;
            }
            return a.attributes.minorVersion - b.attributes.minorVersion;
        });
}


export const updateTemplate = async (template: Template | undefined, updatedAttributes: Partial<TemplateAttributes>) => {
    if (!template) return;

    const input = {
        data: {
            id: template.id,
            type: "templates",
            attributes: {
                ...template.attributes,
                ...updatedAttributes,
            },
        },
    };

    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(`${rootUrl}/api/configurations/templates/${template.id}`, {
        method: "PATCH",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify(input),
    });

    if (!response.ok) throw new Error("Failed to submit data.");

    // If the request is successful, update the local template state
    return {...template, attributes: {...template.attributes, ...updatedAttributes}};
};

export const deleteTemplate = async (templateId: string) => {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(`${rootUrl}/api/configurations/templates/${templateId}`, {
        method: "DELETE",
        headers: {"Content-Type": "application/json"},
    });

    if (!response.ok) throw new Error("Failed to delete template.");

    return true;
};

export const createTemplate = async (attributes: TemplateAttributes) => {
    const input = {
        data: {
            type: "templates",
            attributes: attributes,
        },
    };

    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(`${rootUrl}/api/configurations/templates`, {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify(input),
    });

    if (!response.ok) throw new Error("Failed to create template.");

    const responseData: ApiSingleResponse<Template> = await response.json();
    return responseData.data;
};

export const cloneTemplate = (template: Template): TemplateAttributes => {
    // Create a deep copy of the template attributes
    const clonedAttributes: TemplateAttributes = JSON.parse(JSON.stringify(template.attributes));

    // Clear the region, majorVersion, and minorVersion as required
    clonedAttributes.region = "";
    clonedAttributes.majorVersion = 0;
    clonedAttributes.minorVersion = 0;

    return clonedAttributes;
};
