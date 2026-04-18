import { cn } from "@/lib/utils"

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'circular' | 'rectangular';
  animation?: 'pulse' | 'wave' | 'none';
}

function Skeleton({
  className,
  variant = 'default',
  animation = 'pulse',
  ...props
}: SkeletonProps) {
  return (
    <div
      className={cn(
        'bg-muted',
        {
          'animate-pulse': animation === 'pulse',
          'animate-shimmer': animation === 'wave',
          'rounded-md': variant === 'default',
          'rounded-full': variant === 'circular',
          'rounded-sm': variant === 'rectangular',
        },
        className
      )}
      {...props}
    />
  )
}

export { Skeleton }
