import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { TemplatesPresetsForm } from "@/pages/templates-character-presets-form";

export function TemplatesCharacterPresetsPage() {
    return (
        <TemplateDetailLayout>
            <TemplatesPresetsForm />
        </TemplateDetailLayout>
    );
}
