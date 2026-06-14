export interface DescSegment {
  text: string;
  /** Set to "highlight" for #c...# regions; the page renders plain text in v1. */
  color?: string;
}

export interface FormattedDescription {
  /** Outer array = visual lines (split on newlines); inner = styled segments. */
  lines: DescSegment[][];
  /** Parsed from a leading [Master Level : N]; the header is removed from `lines`. */
  masterLevelHeader?: number;
}

const MASTER_LEVEL_RE = /^\s*\[Master Level\s*:\s*(\d+)\]\s*(?:\r\n|\r|\n)?/;
const HIGHLIGHT = "highlight";

function tokenizeLine(line: string): DescSegment[] {
  const segments: DescSegment[] = [];
  let buf = "";
  let color: string | undefined;
  // True while inside a #x...# directive region; a '#' that follows is its reset.
  let active = false;

  const flush = () => {
    if (buf.length > 0) {
      segments.push(color ? { text: buf, color } : { text: buf });
      buf = "";
    }
  };

  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === "#") {
      const next = line[i + 1];
      // Treat '#<letter>' as an opener only when not already inside a region;
      // a '#' inside an active region is always the reset that closes it (even
      // when the following character is a letter, e.g. "#cred#then").
      if (!active && next !== undefined && /[a-zA-Z]/.test(next)) {
        // opener like #c, #e, #z... — start a new segment, set color only for #c
        flush();
        color = next.toLowerCase() === "c" ? HIGHLIGHT : undefined;
        active = true;
        i += 1; // consume the letter; the loop's i++ consumes the '#'
        continue;
      }
      // bare '#' reset
      flush();
      color = undefined;
      active = false;
      continue;
    }
    buf += ch;
  }
  flush();
  if (segments.length === 0) segments.push({ text: "" });
  return segments;
}

export function formatSkillDescription(raw: string | undefined): FormattedDescription {
  if (raw == null || raw.trim() === "") {
    return { lines: [] };
  }

  let body = raw;
  let masterLevelHeader: number | undefined;
  const m = body.match(MASTER_LEVEL_RE);
  if (m) {
    masterLevelHeader = Number(m[1]);
    body = body.slice(m[0].length);
  }

  if (body.trim() === "") {
    return masterLevelHeader !== undefined ? { lines: [], masterLevelHeader } : { lines: [] };
  }

  const lines = body.split(/\r\n|\r|\n/).map(tokenizeLine);
  return masterLevelHeader !== undefined ? { lines, masterLevelHeader } : { lines };
}
