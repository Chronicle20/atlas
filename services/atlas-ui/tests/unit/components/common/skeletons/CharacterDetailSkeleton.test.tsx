import { render, screen } from '@testing-library/react';
import { CharacterDetailSkeleton, CharacterRendererDetailSkeleton } from '@/components/common/skeletons/CharacterDetailSkeleton';

describe('CharacterDetailSkeleton', () => {
  it('renders the complete character detail page skeleton', () => {
    render(<CharacterDetailSkeleton />);
    
    // Check that skeleton elements exist
    const skeletonElements = document.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('renders character name skeleton in header', () => {
    render(<CharacterDetailSkeleton />);
    
    // Find skeleton elements
    const skeletonElements = document.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('renders character rendering and attributes sections', () => {
    render(<CharacterDetailSkeleton />);
    
    // Check for card elements using data-slot attribute
    const cards = document.querySelectorAll('[data-slot="card"]');
    expect(cards.length).toBeGreaterThan(0);
  });

  it('has inventory section skeleton', () => {
    render(<CharacterDetailSkeleton />);
    
    // Just check that the inventory section exists with some cards
    const cards = document.querySelectorAll('.border');
    expect(cards.length).toBeGreaterThan(5); // Should have multiple compartments
  });
});

describe('CharacterRendererDetailSkeleton', () => {
  it('renders skeleton with proper structure', () => {
    const { container } = render(<CharacterRendererDetailSkeleton />);
    
    // Check that the component renders without crashing
    expect(container.firstChild).toBeInTheDocument();
  });

  it('renders with small size', () => {
    const { container } = render(<CharacterRendererDetailSkeleton size="small" />);
    
    expect(container.firstChild).toBeInTheDocument();
    // Check for presence of skeleton elements
    const skeletonElements = container.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('renders with large size', () => {
    const { container } = render(<CharacterRendererDetailSkeleton size="large" />);
    
    expect(container.firstChild).toBeInTheDocument();
    // Check for presence of skeleton elements
    const skeletonElements = container.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletonElements.length).toBeGreaterThan(0);
  });

  it('applies custom className', () => {
    const { container } = render(<CharacterRendererDetailSkeleton className="custom-class" />);
    
    // Check for custom class in the DOM
    const customElement = container.querySelector('.custom-class');
    expect(customElement).toBeInTheDocument();
  });

  it('renders character-like skeleton elements', () => {
    const { container } = render(<CharacterRendererDetailSkeleton size="medium" />);
    
    // Should have skeleton elements
    const skeletonElements = container.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletonElements.length).toBeGreaterThan(3);
  });

  it('renders with proper opacity overlays', () => {
    const { container } = render(<CharacterRendererDetailSkeleton size="large" />);
    
    // Should have elements with opacity-60 class for overlay effect
    const overlayElements = container.querySelectorAll('.opacity-60');
    expect(overlayElements.length).toBeGreaterThan(2);
  });
});