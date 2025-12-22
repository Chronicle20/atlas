"use client";

import { useState, useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";

import {
  templatesService,
  onboardingService,
  ConfigurationCreationError,
  TenantCreationError,
  TemplateNotFoundError,
  type TemplateOption,
} from "@/services/api";
import {
  createTenantSchema,
  type CreateTenantFormData,
  createTenantDefaults,
} from "@/lib/schemas/tenant.schema";

/**
 * Props for the CreateTenantDialog component.
 */
interface CreateTenantDialogProps {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when the dialog open state changes */
  onOpenChange: (open: boolean) => void;
  /** Optional callback called after successful tenant creation */
  onSuccess?: () => void;
}

/**
 * Dialog component for creating a new tenant from a template.
 *
 * This dialog provides a form with cascading dropdowns for selecting:
 * 1. Region (e.g., GMS, JMS)
 * 2. Major Version (filtered by region)
 * 3. Minor Version (filtered by region + major version)
 *
 * On submission, it creates both:
 * - A tenant entry in atlas-tenants (with generated UUID)
 * - A configuration entry in atlas-configurations (with full template data)
 *
 * @example
 * ```tsx
 * <CreateTenantDialog
 *   open={isOpen}
 *   onOpenChange={setIsOpen}
 *   onSuccess={() => refreshTenants()}
 * />
 * ```
 */
export function CreateTenantDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateTenantDialogProps) {
  // Form state
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateTenantFormData>({
    resolver: zodResolver(createTenantSchema),
    defaultValues: createTenantDefaults,
  });

  // Template options state
  const [templateOptions, setTemplateOptions] = useState<TemplateOption[]>([]);
  const [isLoadingOptions, setIsLoadingOptions] = useState(false);
  const [optionsError, setOptionsError] = useState<string | null>(null);

  // Watch form values for cascading dropdown logic
  const selectedRegion = watch("region");
  const selectedMajorVersion = watch("majorVersion");

  // Fetch template options when dialog opens
  useEffect(() => {
    if (open) {
      setIsLoadingOptions(true);
      setOptionsError(null);

      templatesService
        .getTemplateOptions()
        .then((options) => {
          setTemplateOptions(options);
          if (options.length === 0) {
            setOptionsError("No templates available. Please create a template first.");
          }
        })
        .catch((err) => {
          console.error("Failed to fetch template options:", err);
          setOptionsError("Failed to load template options. Please try again.");
        })
        .finally(() => {
          setIsLoadingOptions(false);
        });
    }
  }, [open]);

  // Compute unique regions from template options
  const availableRegions = useMemo(() => {
    const regions = new Set(templateOptions.map((t) => t.attributes.region));
    return Array.from(regions).sort();
  }, [templateOptions]);

  // Compute major versions available for selected region
  const availableMajorVersions = useMemo(() => {
    if (!selectedRegion) return [];
    const versions = new Set(
      templateOptions
        .filter((t) => t.attributes.region === selectedRegion)
        .map((t) => t.attributes.majorVersion)
    );
    return Array.from(versions).sort((a, b) => a - b);
  }, [templateOptions, selectedRegion]);

  // Compute minor versions available for selected region + major version
  const availableMinorVersions = useMemo(() => {
    if (!selectedRegion || selectedMajorVersion === undefined) return [];
    const versions = new Set(
      templateOptions
        .filter(
          (t) =>
            t.attributes.region === selectedRegion &&
            t.attributes.majorVersion === selectedMajorVersion
        )
        .map((t) => t.attributes.minorVersion)
    );
    return Array.from(versions).sort((a, b) => a - b);
  }, [templateOptions, selectedRegion, selectedMajorVersion]);

  // Reset dependent fields when region changes
  const handleRegionChange = (region: string) => {
    setValue("region", region);
    setValue("majorVersion", 0);
    setValue("minorVersion", 0);
  };

  // Reset minor version when major version changes
  const handleMajorVersionChange = (majorVersion: string) => {
    setValue("majorVersion", parseInt(majorVersion, 10));
    setValue("minorVersion", 0);
  };

  // Handle minor version selection
  const handleMinorVersionChange = (minorVersion: string) => {
    setValue("minorVersion", parseInt(minorVersion, 10));
  };

  // Handle form submission
  const onSubmit = async (data: CreateTenantFormData) => {
    try {
      const result = await onboardingService.onboardTenantByVersion(
        data.name,
        data.region,
        data.majorVersion,
        data.minorVersion
      );

      toast.success("Tenant created successfully with full configuration");

      // Reset form and close dialog
      reset(createTenantDefaults);
      onOpenChange(false);
      onSuccess?.();

      // Navigate to the new tenant
      window.location.replace(`/tenants/${result.tenant.id}/properties`);
    } catch (error) {
      console.error("Failed to create tenant:", error);

      if (error instanceof TemplateNotFoundError) {
        toast.error("Selected template version no longer exists. Please select a different version.");
      } else if (error instanceof ConfigurationCreationError) {
        toast.error(
          `Tenant created but configuration failed. Tenant ID: ${error.tenantId}. Please retry configuration manually.`
        );
      } else if (error instanceof TenantCreationError) {
        toast.error("Failed to create tenant. Please try again.");
      } else {
        toast.error("An unexpected error occurred. Please try again.");
      }
    }
  };

  // Handle dialog close
  const handleOpenChange = (newOpen: boolean) => {
    if (!isSubmitting) {
      onOpenChange(newOpen);
      if (!newOpen) {
        reset(createTenantDefaults);
      }
    }
  };

  // Check if form can be submitted
  const canSubmit =
    !isLoadingOptions &&
    !optionsError &&
    selectedRegion &&
    selectedMajorVersion !== undefined &&
    selectedMajorVersion > 0 &&
    watch("minorVersion") !== undefined;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit(onSubmit)}>
          <DialogHeader>
            <DialogTitle>Create New Tenant</DialogTitle>
            <DialogDescription>
              Create a new tenant based on an existing template. The tenant will be
              configured with all settings from the selected template.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Tenant Name */}
            <div className="grid gap-2">
              <Label htmlFor="name">Tenant Name</Label>
              <Input
                id="name"
                placeholder="Enter tenant name"
                {...register("name")}
                disabled={isSubmitting}
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>

            {/* Loading state for template options */}
            {isLoadingOptions && (
              <div className="space-y-4">
                <div className="grid gap-2">
                  <Label>Region</Label>
                  <Skeleton className="h-9 w-full" />
                </div>
                <div className="grid gap-2">
                  <Label>Major Version</Label>
                  <Skeleton className="h-9 w-full" />
                </div>
                <div className="grid gap-2">
                  <Label>Minor Version</Label>
                  <Skeleton className="h-9 w-full" />
                </div>
              </div>
            )}

            {/* Error state for template options */}
            {optionsError && !isLoadingOptions && (
              <div className="rounded-md bg-destructive/10 p-4 text-sm text-destructive">
                {optionsError}
              </div>
            )}

            {/* Template selection dropdowns */}
            {!isLoadingOptions && !optionsError && (
              <>
                {/* Region Select */}
                <div className="grid gap-2">
                  <Label htmlFor="region">Region</Label>
                  <Select
                    value={selectedRegion || ""}
                    onValueChange={handleRegionChange}
                    disabled={isSubmitting}
                  >
                    <SelectTrigger id="region">
                      <SelectValue placeholder="Select region" />
                    </SelectTrigger>
                    <SelectContent>
                      {availableRegions.map((region) => (
                        <SelectItem key={region} value={region}>
                          {region}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {errors.region && (
                    <p className="text-sm text-destructive">{errors.region.message}</p>
                  )}
                </div>

                {/* Major Version Select */}
                <div className="grid gap-2">
                  <Label htmlFor="majorVersion">Major Version</Label>
                  <Select
                    value={selectedMajorVersion ? selectedMajorVersion.toString() : ""}
                    onValueChange={handleMajorVersionChange}
                    disabled={isSubmitting || !selectedRegion}
                  >
                    <SelectTrigger id="majorVersion">
                      <SelectValue placeholder="Select major version" />
                    </SelectTrigger>
                    <SelectContent>
                      {availableMajorVersions.map((version) => (
                        <SelectItem key={version} value={version.toString()}>
                          {version}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {errors.majorVersion && (
                    <p className="text-sm text-destructive">
                      {errors.majorVersion.message}
                    </p>
                  )}
                </div>

                {/* Minor Version Select */}
                <div className="grid gap-2">
                  <Label htmlFor="minorVersion">Minor Version</Label>
                  <Select
                    value={watch("minorVersion")?.toString() || ""}
                    onValueChange={handleMinorVersionChange}
                    disabled={
                      isSubmitting ||
                      !selectedRegion ||
                      !selectedMajorVersion
                    }
                  >
                    <SelectTrigger id="minorVersion">
                      <SelectValue placeholder="Select minor version" />
                    </SelectTrigger>
                    <SelectContent>
                      {availableMinorVersions.map((version) => (
                        <SelectItem key={version} value={version.toString()}>
                          {version}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {errors.minorVersion && (
                    <p className="text-sm text-destructive">
                      {errors.minorVersion.message}
                    </p>
                  )}
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting || !canSubmit}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Tenant"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
