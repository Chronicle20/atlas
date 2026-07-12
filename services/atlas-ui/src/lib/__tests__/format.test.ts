import { describe, it, expect } from 'vitest';
import { formatBytes } from '@/lib/format';

describe('formatBytes', () => {
  it('formats zero', () => {
    expect(formatBytes(0)).toBe('0 B');
  });
  it('formats bytes without decimals', () => {
    expect(formatBytes(512)).toBe('512 B');
  });
  it('formats small unit values with one decimal', () => {
    expect(formatBytes(1536)).toBe('1.5 KB');
  });
  it('formats values >= 10 without decimals', () => {
    expect(formatBytes(10 * 1024 * 1024)).toBe('10 MB');
  });
});
