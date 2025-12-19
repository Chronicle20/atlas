"use client"

import * as React from "react"
import { Control, FieldPath, FieldValues } from "react-hook-form"
import {
  FormControl,
  FormDescription,
  FormField as ShadcnFormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

interface FormSelectProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> {
  control: Control<TFieldValues>
  name: TName
  label: string
  description?: string
  placeholder?: string
  options: SelectOption[]
  className?: string
  disabled?: boolean
  emptyMessage?: string
}

/**
 * A reusable form select component that wraps the shadcn/ui Select with FormField pattern.
 * Reduces boilerplate by handling the common FormItem -> FormLabel -> FormControl -> FormMessage structure.
 * 
 * @example Basic select:
 * <FormSelect
 *   control={form.control}
 *   name="status"
 *   label="Status"
 *   placeholder="Select a status"
 *   options={[
 *     { value: "active", label: "Active" },
 *     { value: "inactive", label: "Inactive" },
 *     { value: "pending", label: "Pending", disabled: true }
 *   ]}
 *   description="Choose the current status"
 * />
 * 
 * @example With empty message:
 * <FormSelect
 *   control={form.control}
 *   name="category"
 *   label="Category"
 *   options={categories}
 *   emptyMessage="No categories available"
 * />
 */
export function FormSelect<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
>({
  control,
  name,
  label,
  description,
  placeholder = "Select an option",
  options,
  className,
  disabled,
  emptyMessage = "No options available"
}: FormSelectProps<TFieldValues, TName>) {
  return (
    <ShadcnFormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className={cn(className)}>
          <FormLabel>{label}</FormLabel>
          <Select
            onValueChange={field.onChange}
            defaultValue={field.value}
            disabled={disabled ?? false}
          >
            <FormControl>
              <SelectTrigger>
                <SelectValue placeholder={placeholder} />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              {options.length === 0 ? (
                <div className="py-6 text-center text-sm text-muted-foreground">
                  {emptyMessage}
                </div>
              ) : (
                options.map((option) => (
                  <SelectItem
                    key={option.value}
                    value={option.value}
                    disabled={option.disabled ?? false}
                  >
                    {option.label}
                  </SelectItem>
                ))
              )}
            </SelectContent>
          </Select>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

FormSelect.displayName = "FormSelect"