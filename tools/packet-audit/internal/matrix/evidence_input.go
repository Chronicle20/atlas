package matrix

import (
	"fmt"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
)

// BuildEvidenceInputs loads the ledger and grades freshness per record.
// problems collects --check failures: hash drift, unresolvable citations,
// dangling versions (no export on disk).
func BuildEvidenceInputs(evidenceDir string, exportPaths map[string]string) (map[EvKey]EvidenceStatus, []string, error) {
	recs, err := evidence.LoadDir(evidenceDir)
	if err != nil {
		return nil, nil, err
	}
	out := map[EvKey]EvidenceStatus{}
	var problems []string
	for _, r := range recs {
		k := EvKey{Packet: r.Packet, Version: r.Version}
		exp, ok := exportPaths[r.Version]
		if !ok {
			out[k] = EvidenceStatus{Exists: true, Fresh: false, Address: r.IDA.Address,
				Note: "no IDA export for " + r.Version}
			problems = append(problems, fmt.Sprintf("evidence %s×%s: no export for version", r.Packet, r.Version))
			continue
		}
		h, err := evidence.FunctionHash(exp, r.IDA.Function)
		if err != nil {
			out[k] = EvidenceStatus{Exists: true, Fresh: false, Address: r.IDA.Address,
				Note: "citation unresolvable: " + r.IDA.Function}
			problems = append(problems, fmt.Sprintf("evidence %s×%s: %v", r.Packet, r.Version, err))
			continue
		}
		fresh := h == r.IDA.DecompileSHA256
		if !fresh {
			problems = append(problems, fmt.Sprintf("evidence %s×%s: decompile hash drift (re-pin after review)", r.Packet, r.Version))
		}
		out[k] = EvidenceStatus{Exists: true, Fresh: fresh, Address: r.IDA.Address}
	}
	return out, problems, nil
}
