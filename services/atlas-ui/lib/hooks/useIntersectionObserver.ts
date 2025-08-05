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

interface IntersectionObserverResult {
  isIntersecting: boolean;
  entry: IntersectionObserverEntry | null;
  ref: React.RefObject<Element>;
}

export function useIntersectionObserver(
  options: UseIntersectionObserverOptions = {}
): IntersectionObserverResult {
  const {
    threshold = 0.1,
    root = null,
    rootMargin = '50px',
    triggerOnce = true,
    enabled = true,
  } = options;

  const elementRef = useRef<Element>(null);
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
        setEntry(entry);
        setIsIntersecting(entry.isIntersecting);

        if (entry.isIntersecting && triggerOnce) {
          setHasTriggered(true);
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
    ref: elementRef,
  };
}

/**
 * Hook for lazy loading that only enables queries when element is visible
 */
export function useLazyLoad(options?: UseIntersectionObserverOptions) {
  const { isIntersecting, ref } = useIntersectionObserver({
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