/**
 * Intersection Observer hook for lazy loading and visibility detection
 */

import { useEffect, useRef, useState } from 'react';

interface UseIntersectionObserverOptions {
  threshold?: number | number[];
  root?: Element | null;
  rootMargin?: string;
  triggerOnce?: boolean;
  enabled?: boolean;
}

interface IntersectionObserverResult<T extends Element = Element> {
  isIntersecting: boolean;
  entry: IntersectionObserverEntry | null;
  ref: React.RefObject<T>;
}

export function useIntersectionObserver<T extends Element = Element>(
  options: UseIntersectionObserverOptions = {}
): IntersectionObserverResult<T> {
  const {
    threshold = 0.1,
    root = null,
    rootMargin = '50px',
    triggerOnce = true,
    enabled = true,
  } = options;

  const elementRef = useRef<T>(null);
  const [isIntersecting, setIsIntersecting] = useState(false);
  const [entry, setEntry] = useState<IntersectionObserverEntry | null>(null);
  const [hasTriggered, setHasTriggered] = useState(false);

  useEffect(() => {
    if (!enabled || !elementRef.current) return;

    // Don't create observer if triggerOnce and already triggered
    if (triggerOnce && hasTriggered) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry) {
          setEntry(entry);
          setIsIntersecting(entry.isIntersecting);

          if (entry.isIntersecting && triggerOnce) {
            setHasTriggered(true);
          }
        }
      },
      {
        threshold,
        root,
        rootMargin,
      }
    );

    observer.observe(elementRef.current);

    return () => {
      observer.disconnect();
    };
  }, [threshold, root, rootMargin, triggerOnce, enabled, hasTriggered]);

  return {
    isIntersecting: triggerOnce ? (hasTriggered || isIntersecting) : isIntersecting,
    entry,
    ref: elementRef as React.RefObject<T>,
  };
}

/**
 * Hook for lazy loading that only enables queries when element is visible
 */
export function useLazyLoad<T extends Element = HTMLDivElement>(options?: UseIntersectionObserverOptions) {
  const { isIntersecting, ref } = useIntersectionObserver<T>({
    triggerOnce: true,
    threshold: 0.1,
    rootMargin: '100px', // Load slightly before entering viewport
    ...options,
  });

  return {
    shouldLoad: isIntersecting,
    ref,
  };
}