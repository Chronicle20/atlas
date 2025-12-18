import { LoadingSpinner } from './LoadingSpinner';

interface PageLoaderProps {
  className?: string;
}

export function PageLoader({ className }: PageLoaderProps = {}) {
  return (
    <div 
      className={className || 'flex h-[50vh] items-center justify-center'}
      data-testid="page-loader"
    >
      <LoadingSpinner size="lg" />
    </div>
  );
}