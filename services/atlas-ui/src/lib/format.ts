export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  const formatted = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value >= 10 || unit === 0 ? 0 : 1,
  }).format(value);
  return `${formatted} ${units[unit]}`;
}
