export function formatCount(n: number): string {
  return new Intl.NumberFormat().format(n);
}

export function pluralize(n: number, singular: string, plural: string): string {
  return n === 1 ? singular : plural;
}
