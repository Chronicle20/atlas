import * as React from "react"
import * as TooltipPrimitive from "@radix-ui/react-tooltip"
import { Copy } from "lucide-react"
import { toast } from "sonner"

import { cn } from "@/lib/utils"

function TooltipProvider({
  delayDuration = 0,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Provider>) {
  return (
    <TooltipPrimitive.Provider
      data-slot="tooltip-provider"
      delayDuration={delayDuration}
      {...props}
    />
  )
}

function Tooltip({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Root>) {
  return (
    <TooltipProvider>
      <TooltipPrimitive.Root data-slot="tooltip" {...props} />
    </TooltipProvider>
  )
}

function TooltipTrigger({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Trigger>) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />
}

function TooltipContent({
  className,
  sideOffset = 0,
  children,
  copyable = false,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Content> & { copyable?: boolean }) {
  const copyToClipboard = async (text: string) => {
    if (!text) return

    const notifySuccess = () =>
      toast.success("Copied", { description: text, duration: 2000 })
    const notifyFailure = (err: unknown) => {
      console.error("Failed to copy text: ", err)
      toast.error("Failed to copy")
    }

    // Clipboard API is only available on secure contexts (HTTPS or localhost).
    // Fall back to the legacy execCommand path when it's unavailable so the UI
    // keeps working on plain-HTTP LAN deployments.
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text)
        notifySuccess()
        return
      } catch (err) {
        // fall through to the execCommand fallback before giving up
        console.warn("navigator.clipboard.writeText failed, falling back", err)
      }
    }

    if (typeof document === "undefined") return

    const textarea = document.createElement("textarea")
    textarea.value = text
    textarea.setAttribute("readonly", "")
    textarea.style.position = "fixed"
    textarea.style.top = "0"
    textarea.style.left = "0"
    textarea.style.opacity = "0"
    textarea.style.pointerEvents = "none"
    document.body.appendChild(textarea)
    try {
      textarea.select()
      const ok = document.execCommand("copy")
      if (ok) {
        notifySuccess()
      } else {
        notifyFailure(new Error("document.execCommand('copy') returned false"))
      }
    } catch (err) {
      notifyFailure(err)
    } finally {
      document.body.removeChild(textarea)
    }
  }

  // Extract text content from children for copying
  const getTextContent = (children: React.ReactNode): string => {
    if (typeof children === 'string') return children;
    if (typeof children === 'number') return children.toString();
    if (Array.isArray(children)) return children.map(getTextContent).join('');
    if (React.isValidElement<{ children?: React.ReactNode }>(children)) {
      return getTextContent(children.props.children);
    }
    return '';
  };

  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Content
        data-slot="tooltip-content"
        sideOffset={sideOffset}
        className={cn(
          "bg-primary text-primary-foreground animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 w-fit rounded-md px-3 py-1.5 text-xs text-balance",
          className
        )}
        {...props}
      >
        <div className={cn("flex items-center", copyable ? "gap-2" : "")}>
          {copyable && (
            <button
              type="button"
              onPointerDown={(e) => {
                e.preventDefault()
                void copyToClipboard(getTextContent(children))
              }}
              className="text-primary-foreground opacity-70 hover:opacity-100 hover:bg-primary-foreground/20 hover:scale-110 p-1 rounded cursor-pointer transition-all duration-200"
              aria-label="Copy to clipboard"
            >
              <Copy className="size-3.5" />
            </button>
          )}
          <div>{children}</div>
        </div>
        <TooltipPrimitive.Arrow className="bg-primary fill-primary z-50 size-2.5 translate-y-[calc(-50%_-_2px)] rotate-45 rounded-[2px]" />
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  )
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider }
