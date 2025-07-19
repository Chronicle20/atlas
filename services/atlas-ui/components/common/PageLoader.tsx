import { LoadingSpinner } from './LoadingSpinner';

interface PageLoaderProps {
  className?: string;
}

export function PageLoader({ className }: PageLoaderProps = {}) {
  return (
    <div className={className || 'flex h-[50vh] items-center justify-center'}>
      <LoadingSpinner size="lg" />
    </div>
  );
}