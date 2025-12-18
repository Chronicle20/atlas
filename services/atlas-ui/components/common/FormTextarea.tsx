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
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

interface FormTextareaProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> {
  control: Control<TFieldValues>
  name: TName
  label: string
  description?: string
  placeholder?: string
  className?: string
  disabled?: boolean
  rows?: number
  maxLength?: number
  textareaProps?: React.ComponentPropsWithoutRef<typeof Textarea>
}

/**
 * A reusable form textarea component that wraps the shadcn/ui Textarea with FormField pattern.
 * Reduces boilerplate by handling the common FormItem -> FormLabel -> FormControl -> FormMessage structure.
 * 
 * @example Basic textarea:
 * <FormTextarea
 *   control={form.control}
 *   name="description"
 *   label="Description"
 *   placeholder="Enter a description"
 *   description="Provide a detailed description"
 * />
 * 
 * @example With character limit:
 * <FormTextarea
 *   control={form.control}
 *   name="bio"
 *   label="Biography"
 *   placeholder="Tell us about yourself"
 *   maxLength={500}
 *   rows={5}
 * />
 * 
 * @example With custom styling:
 * <FormTextarea
 *   control={form.control}
 *   name="notes"
 *   label="Notes"
 *   className="min-h-[120px]"
 *   textareaProps={{ resize: "vertical" }}
 * />
 */
export function FormTextarea<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
>({
  control,
  name,
  label,
  description,
  placeholder,
  className,
  disabled,
  rows,
  maxLength,
  textareaProps
}: FormTextareaProps<TFieldValues, TName>) {
  return (
    <ShadcnFormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className={cn(className)}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Textarea
              placeholder={placeholder}
              disabled={disabled}
              rows={rows}
              maxLength={maxLength}
              {...field}
              {...textareaProps}
              value={field.value ?? ""}
            />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          {maxLength && (
            <FormDescription className="text-right text-xs">
              {(field.value?.length ?? 0)}/{maxLength}
            </FormDescription>
          )}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

FormTextarea.displayName = "FormTextarea"