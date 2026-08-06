import type { DefinitionKind, SocketObject } from "@/lib/socket/model";

/**
 * The "Open in <object>" destination: `/${scope.source}s/${scope.key}/${kind}s?def=<name>`.
 * Shared by `PacketMatrixPage` and `DefinitionGridPage` so the two never
 * drift on the URL shape a matrix cell's "Open in" action navigates to.
 */
export function buildOpenInPath(
  scope: SocketObject,
  kind: DefinitionKind,
  name: string,
): string {
  return `/${scope.source}s/${scope.key}/${kind}s?def=${encodeURIComponent(name)}`;
}
