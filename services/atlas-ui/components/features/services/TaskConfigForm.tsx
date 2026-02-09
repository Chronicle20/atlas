"use client";

import { Plus, Trash2 } from "lucide-react";

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
import { Card, CardContent } from "@/components/ui/card";

import type { TaskConfig, ServiceType } from "@/types/models/service";
import { TASK_TYPES_BY_SERVICE } from "@/types/models/service";

interface TaskConfigFormProps {
  tasks: TaskConfig[];
  serviceType: ServiceType;
  onChange: (tasks: TaskConfig[]) => void;
  disabled?: boolean;
}

/**
 * Form component for configuring service tasks.
 *
 * Provides:
 * - Dropdown of valid task types (filtered by service type)
 * - Interval input in milliseconds
 * - Duration input in milliseconds
 * - Add/remove task functionality
 */
export function TaskConfigForm({
  tasks,
  serviceType,
  onChange,
  disabled = false,
}: TaskConfigFormProps) {
  const availableTaskTypes = TASK_TYPES_BY_SERVICE[serviceType] || [];

  const handleAddTask = () => {
    const defaultType = availableTaskTypes[0] || "";
    onChange([
      ...tasks,
      {
        type: defaultType,
        interval: 30000, // 30 seconds default
        duration: 60000, // 60 seconds default
      },
    ]);
  };

  const handleRemoveTask = (index: number) => {
    onChange(tasks.filter((_, i) => i !== index));
  };

  const handleTaskChange = (
    index: number,
    field: keyof TaskConfig,
    value: string | number
  ) => {
    const updated = tasks.map((task, i) => {
      if (i !== index) return task;
      if (field === "type") {
        return { ...task, type: value as string };
      } else {
        return {
          ...task,
          [field]: typeof value === "string" ? parseInt(value, 10) || 0 : value,
        };
      }
    });
    onChange(updated);
  };

  // Helper to format milliseconds for display
  const formatMs = (ms: number): string => {
    if (ms >= 60000) {
      const mins = ms / 60000;
      return `${mins} min${mins !== 1 ? "s" : ""}`;
    }
    if (ms >= 1000) {
      const secs = ms / 1000;
      return `${secs} sec${secs !== 1 ? "s" : ""}`;
    }
    return `${ms} ms`;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Label className="text-base font-medium">Tasks</Label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleAddTask}
          disabled={disabled || availableTaskTypes.length === 0}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Task
        </Button>
      </div>

      {availableTaskTypes.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No task types available for this service type.
        </p>
      )}

      {tasks.length === 0 && availableTaskTypes.length > 0 && (
        <p className="text-sm text-muted-foreground">
          No tasks configured. Click &quot;Add Task&quot; to add one.
        </p>
      )}

      <div className="space-y-3">
        {tasks.map((task, index) => (
          <Card key={index}>
            <CardContent className="pt-4">
              <div className="grid gap-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 grid gap-2">
                    <Label htmlFor={`task-type-${index}`}>Task Type</Label>
                    <Select
                      value={task.type}
                      onValueChange={(value) =>
                        handleTaskChange(index, "type", value)
                      }
                      disabled={disabled}
                    >
                      <SelectTrigger id={`task-type-${index}`}>
                        <SelectValue placeholder="Select task type" />
                      </SelectTrigger>
                      <SelectContent>
                        {availableTaskTypes.map((type) => (
                          <SelectItem key={type} value={type}>
                            {type}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => handleRemoveTask(index)}
                    disabled={disabled}
                    className="mt-6"
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor={`task-interval-${index}`}>
                      Interval (ms)
                    </Label>
                    <Input
                      id={`task-interval-${index}`}
                      type="number"
                      min={0}
                      value={task.interval}
                      onChange={(e) =>
                        handleTaskChange(index, "interval", e.target.value)
                      }
                      disabled={disabled}
                    />
                    <p className="text-xs text-muted-foreground">
                      {formatMs(task.interval)}
                    </p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor={`task-duration-${index}`}>
                      Duration (ms)
                    </Label>
                    <Input
                      id={`task-duration-${index}`}
                      type="number"
                      min={0}
                      value={task.duration}
                      onChange={(e) =>
                        handleTaskChange(index, "duration", e.target.value)
                      }
                      disabled={disabled}
                    />
                    <p className="text-xs text-muted-foreground">
                      {formatMs(task.duration)}
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
