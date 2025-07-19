"use client"

import * as React from "react"
import { Control, FieldPath, FieldValues, ControllerRenderProps } from "react-hook-form"
import {
  FormControl,
  FormDescription,
  FormField as ShadcnFormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface BaseFormFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> {
  control: Control<TFieldValues>
  name: TName
  label: string
  description?: string
  className?: string
  disabled?: boolean
}

interface TextFormFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> extends BaseFormFieldProps<TFieldValues, TName> {
  type?: "text" | "email" | "password" | "url"
  placeholder?: string
  inputProps?: React.ComponentPropsWithoutRef<typeof Input>
}

interface NumberFormFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> extends BaseFormFieldProps<TFieldValues, TName> {
  type: "number"
  placeholder?: string
  min?: number
  max?: number
  step?: number
  inputProps?: React.ComponentPropsWithoutRef<typeof Input>
}

interface CustomFormFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> extends BaseFormFieldProps<TFieldValues, TName> {
  render: (field: ControllerRenderProps<TFieldValues, TName>) => React.ReactNode
}

type FormFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
> = 
  | TextFormFieldProps<TFieldValues, TName>
  | NumberFormFieldProps<TFieldValues, TName> 
  | CustomFormFieldProps<TFieldValues, TName>

/**
 * A reusable form field component that wraps the shadcn/ui FormField pattern.
 * Reduces boilerplate by handling the common FormItem -> FormLabel -> FormControl -> FormMessage structure.
 * 
 * @example Text field:
 * <FormField
 *   control={form.control}
 *   name="username"
 *   label="Username"
 *   placeholder="Enter username"
 *   description="Choose a unique username"
 * />
 * 
 * @example Number field:
 * <FormField
 *   control={form.control}
 *   name="age"
 *   label="Age"
 *   type="number"
 *   min={0}
 *   max={120}
 * />
 * 
 * @example Custom field:
 * <FormField
 *   control={form.control}
 *   name="isActive"
 *   label="Active Status"
 *   render={({ field }) => (
 *     <Switch {...field} checked={field.value} onCheckedChange={field.onChange} />
 *   )}
 * />
 */
export function FormField<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>
>(props: FormFieldProps<TFieldValues, TName>) {
  const { control, name, label, description, className, disabled } = props

  return (
    <ShadcnFormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className={cn(className)}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            {"render" in props ? (
              props.render(field)
            ) : props.type === "number" ? (
              <Input
                type="number"
                placeholder={props.placeholder}
                min={props.min}
                max={props.max}
                step={props.step}
                disabled={disabled}
                {...field}
                {...props.inputProps}
                onChange={(e) => {
                  const value = e.target.value === "" ? undefined : Number(e.target.value)
                  field.onChange(value)
                }}
                value={field.value ?? ""}
              />
            ) : (
              <Input
                type={props.type || "text"}
                placeholder={props.placeholder}
                disabled={disabled}
                {...field}
                {...props.inputProps}
                value={field.value ?? ""}
              />
            )}
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

FormField.displayName = "FormField"