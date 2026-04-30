import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { PropertiesForm } from "@/pages/templates-properties-form";

export function TemplatesPropertiesPage() {
    return (
        <TemplateDetailLayout>
            <PropertiesForm />
        </TemplateDetailLayout>
    );
}
