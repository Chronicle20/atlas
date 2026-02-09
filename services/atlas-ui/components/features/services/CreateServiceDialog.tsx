"use client";

import { useState } from "react";
import { Loader2, Server, Globe, Package } from "lucide-react";
import { toast } from "sonner";

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
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

import { TaskConfigForm } from "./TaskConfigForm";
import { LoginTenantConfig } from "./LoginTenantConfig";
import { ChannelTenantConfig } from "./ChannelTenantConfig";

import { useCreateService } from "@/lib/hooks/api/useServices";
import type {
  ServiceType,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
  CreateServiceInput,
} from "@/types/models/service";
import { TASK_TYPES_BY_SERVICE } from "@/types/models/service";

// UUID validation regex
const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const isValidUuid = (value: string): boolean => UUID_REGEX.test(value);

interface CreateServiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (serviceId: string) => void;
}

type WizardStep = "type" | "config" | "tenants";

const SERVICE_TYPE_OPTIONS: {
  type: ServiceType;
  label: string;
  description: string;
  icon: React.ReactNode;
}[] = [
  {
    type: "login-service",
    label: "Login Service",
    description: "Handles player authentication and login",
    icon: <Server className="h-6 w-6" />,
  },
  {
    type: "channel-service",
    label: "Channel Service",
    description: "Manages game world channels and player sessions",
    icon: <Globe className="h-6 w-6" />,
  },
  {
    type: "drops-service",
    label: "Drops Service",
    description: "Handles item drop management and expiration",
    icon: <Package className="h-6 w-6" />,
  },
];

/**
 * Multi-step dialog for creating a new service configuration.
 *
 * Steps:
 * 1. Select service type + optional ID
 * 2. Configure tasks
 * 3. Configure tenant associations (Login/Channel only)
 */
export function CreateServiceDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateServiceDialogProps) {
  const createService = useCreateService();

  // Wizard state
  const [step, setStep] = useState<WizardStep>("type");

  // Form state
  const [serviceType, setServiceType] = useState<ServiceType | null>(null);
  const [customId, setCustomId] = useState("");
  const [customIdError, setCustomIdError] = useState<string | null>(null);
  const [tasks, setTasks] = useState<TaskConfig[]>([]);
  const [loginTenants, setLoginTenants] = useState<LoginTenant[]>([]);
  const [channelTenants, setChannelTenants] = useState<ChannelTenant[]>([]);

  // Validate custom ID when it changes
  const handleCustomIdChange = (value: string) => {
    setCustomId(value);
    const trimmed = value.trim();
    if (trimmed && !isValidUuid(trimmed)) {
      setCustomIdError("Invalid UUID format. Must be like: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx");
    } else {
      setCustomIdError(null);
    }
  };

  const resetForm = () => {
    setStep("type");
    setServiceType(null);
    setCustomId("");
    setCustomIdError(null);
    setTasks([]);
    setLoginTenants([]);
    setChannelTenants([]);
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      resetForm();
    }
    onOpenChange(newOpen);
  };

  const handleTypeSelect = (type: ServiceType) => {
    setServiceType(type);
    // Initialize with default tasks for the type
    const defaultTaskTypes = TASK_TYPES_BY_SERVICE[type] || [];
    const firstTaskType = defaultTaskTypes[0];
    if (firstTaskType) {
      setTasks([
        {
          type: firstTaskType,
          interval: 30000,
          duration: 60000,
        },
      ]);
    } else {
      setTasks([]);
    }
    // Reset tenants when type changes
    setLoginTenants([]);
    setChannelTenants([]);
  };

  const handleNext = () => {
    if (step === "type" && serviceType) {
      setStep("config");
    } else if (step === "config") {
      if (serviceType === "drops-service") {
        // Drops service doesn't have tenants, submit directly
        handleSubmit();
      } else {
        setStep("tenants");
      }
    }
  };

  const handleBack = () => {
    if (step === "config") {
      setStep("type");
    } else if (step === "tenants") {
      setStep("config");
    }
  };

  const handleSubmit = async () => {
    if (!serviceType) return;

    const input: CreateServiceInput = {
      type: serviceType,
      tasks,
    };

    if (customId.trim()) {
      input.id = customId.trim();
    }

    if (serviceType === "login-service") {
      input.tenants = loginTenants;
    } else if (serviceType === "channel-service") {
      input.tenants = channelTenants;
    }

    try {
      const result = await createService.mutateAsync(input);
      toast.success("Service created successfully");
      handleOpenChange(false);
      onSuccess?.(result.id);
    } catch (error) {
      console.error("Failed to create service:", error);
      toast.error("Failed to create service. Please try again.");
    }
  };

  const isSubmitting = createService.isPending;

  const canProceedFromType = serviceType !== null && !customIdError;
  const canProceedFromConfig = tasks.length > 0 || serviceType === "drops-service";
  const canSubmitFromTenants =
    (serviceType === "login-service" && loginTenants.every((t) => t.id && t.port > 0)) ||
    (serviceType === "channel-service" &&
      channelTenants.every(
        (t) =>
          t.id &&
          t.ipAddress &&
          t.worlds.every((w) => w.channels.every((c) => c.port > 0))
      ));

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {step === "type" && "Create New Service"}
            {step === "config" && "Configure Tasks"}
            {step === "tenants" && "Configure Tenants"}
          </DialogTitle>
          <DialogDescription>
            {step === "type" && "Select the type of service you want to create."}
            {step === "config" && "Configure the background tasks for this service."}
            {step === "tenants" && "Associate tenants with this service."}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {/* Step 1: Type Selection */}
          {step === "type" && (
            <div className="space-y-4">
              <div className="grid gap-3">
                {SERVICE_TYPE_OPTIONS.map((option) => (
                  <Card
                    key={option.type}
                    className={cn(
                      "cursor-pointer transition-colors hover:bg-muted/50",
                      serviceType === option.type && "border-primary bg-primary/5"
                    )}
                    onClick={() => handleTypeSelect(option.type)}
                  >
                    <CardContent className="flex items-center gap-4 p-4">
                      <div
                        className={cn(
                          "p-2 rounded-lg",
                          serviceType === option.type
                            ? "bg-primary text-primary-foreground"
                            : "bg-muted"
                        )}
                      >
                        {option.icon}
                      </div>
                      <div>
                        <h3 className="font-medium">{option.label}</h3>
                        <p className="text-sm text-muted-foreground">
                          {option.description}
                        </p>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>

              <div className="grid gap-2 pt-4 border-t">
                <Label htmlFor="customId">
                  Service ID <span className="text-muted-foreground">(optional)</span>
                </Label>
                <Input
                  id="customId"
                  value={customId}
                  onChange={(e) => handleCustomIdChange(e.target.value)}
                  placeholder="Leave empty to auto-generate UUID"
                  disabled={isSubmitting}
                  className={customIdError ? "border-destructive" : ""}
                />
                {customIdError ? (
                  <p className="text-xs text-destructive">{customIdError}</p>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    If not provided, a UUID will be automatically generated.
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Step 2: Task Configuration */}
          {step === "config" && serviceType && (
            <TaskConfigForm
              tasks={tasks}
              serviceType={serviceType}
              onChange={setTasks}
              disabled={isSubmitting}
            />
          )}

          {/* Step 3: Tenant Configuration */}
          {step === "tenants" && serviceType === "login-service" && (
            <LoginTenantConfig
              tenants={loginTenants}
              onChange={setLoginTenants}
              disabled={isSubmitting}
            />
          )}

          {step === "tenants" && serviceType === "channel-service" && (
            <ChannelTenantConfig
              tenants={channelTenants}
              onChange={setChannelTenants}
              disabled={isSubmitting}
            />
          )}
        </div>

        <DialogFooter>
          {step !== "type" && (
            <Button
              type="button"
              variant="outline"
              onClick={handleBack}
              disabled={isSubmitting}
            >
              Back
            </Button>
          )}

          {step === "type" && (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button
                type="button"
                onClick={handleNext}
                disabled={!canProceedFromType || isSubmitting}
              >
                Next
              </Button>
            </>
          )}

          {step === "config" && (
            <Button
              type="button"
              onClick={handleNext}
              disabled={!canProceedFromConfig || isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : serviceType === "drops-service" ? (
                "Create Service"
              ) : (
                "Next"
              )}
            </Button>
          )}

          {step === "tenants" && (
            <Button
              type="button"
              onClick={handleSubmit}
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Service"
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
