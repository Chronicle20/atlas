import type { Template } from "@/types/models/template";
import type { TenantConfig } from "@/services/api/tenants.service";
import type {
  SocketConfig,
  SocketHandlerEntry,
  SocketWriterEntry,
} from "@/types/models/socket";
import { parseOpcode } from "@/lib/socket/opcode";
import type { Binding, SocketObject } from "@/lib/socket/model";

/**
 * Turns a fetched Template or Tenant configuration into the normalized
 * SocketObject the whole domain layer operates on. This is the ONLY place the
 * wire shape is read; nothing downstream touches socket.handlers directly.
 */
function build(
  key: string,
  source: SocketObject["source"],
  region: string,
  majorVersion: number,
  minorVersion: number,
  socket: SocketConfig | undefined,
): SocketObject {
  const handlers = new Map<string, Binding[]>();
  const writers = new Map<string, Binding[]>();

  (socket?.handlers ?? []).forEach((e: SocketHandlerEntry, index) => {
    push(handlers, e.handler, {
      opCode: e.opCode,
      opCodeValue: parseOpcode(e.opCode),
      validator: e.validator,
      services: e.services ?? [],
      options: e.options,
      ...(e.fname !== undefined ? { fname: e.fname } : {}),
      index,
    });
  });

  (socket?.writers ?? []).forEach((e: SocketWriterEntry, index) => {
    push(writers, e.writer, {
      opCode: e.opCode,
      opCodeValue: parseOpcode(e.opCode),
      services: e.services ?? [],
      options: e.options,
      ...(e.fname !== undefined ? { fname: e.fname } : {}),
      index,
    });
  });

  return {
    key,
    label: `${region} v${majorVersion}.${minorVersion}`,
    source,
    region,
    majorVersion,
    minorVersion,
    handlers,
    writers,
    unsupportedHandlers: new Set(socket?.unsupported?.handlers ?? []),
    unsupportedWriters: new Set(socket?.unsupported?.writers ?? []),
  };
}

function push(
  into: Map<string, Binding[]>,
  name: string,
  binding: Binding,
): void {
  const existing = into.get(name);
  if (existing) existing.push(binding);
  else into.set(name, [binding]);
}

export function fromTemplate(t: Template): SocketObject {
  const a = t.attributes;
  return build(
    t.id,
    "template",
    a.region,
    a.majorVersion,
    a.minorVersion,
    a.socket,
  );
}

export function fromTenantConfig(t: TenantConfig): SocketObject {
  const a = t.attributes;
  return build(
    t.id,
    "tenant",
    a.region,
    a.majorVersion,
    a.minorVersion,
    a.socket,
  );
}
