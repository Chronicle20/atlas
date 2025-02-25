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
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
    const response = await fetch(rootUrl + "/api/configurations/templates/", {
        method: "GET",
        headers: {
            "Content-Type": "application/json",
        },
    });
    if (!response.ok) {
        throw new Error("Failed to fetch templates.");
    }
    const responseData = await response.json();
    return responseData.data;
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

    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
    const response = await fetch(`${rootUrl}/api/configurations/templates/${template.id}`, {
        method: "PATCH",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify(input),
    });

    if (!response.ok) throw new Error("Failed to submit data.");

    // If the request is successful, update the local template state
    return {...template, attributes: {...template.attributes, ...updatedAttributes}};
};