package projection

import (
	"atlas-login/configuration"
	"atlas-login/configuration/tenant"
	"atlas-login/listener"

	"github.com/google/uuid"
)

// OpKind enumerates the actions ComputeOps can emit.
type OpKind int

const (
	// OpAdd means a listener for Key should be created. Cfg carries the
	// fields the apply loop needs to call listener.Registry.Add.
	OpAdd OpKind = iota
	// OpDrain means a listener for Key should be drained and removed.
	OpDrain
)

// ListenerConfig carries everything an Add op needs. Captured at diff
// time so the apply loop's call into listener.Registry.Add doesn't have
// to re-resolve from State.
type ListenerConfig struct {
	Port         int
	Region       string
	MajorVersion uint16
	MinorVersion uint16
}

// Op is the unit of work emitted by ComputeOps and consumed by the apply
// loop. For OpDrain, Cfg is zero — only Key is needed.
type Op struct {
	Kind OpKind
	Key  listener.Key
	Cfg  ListenerConfig
}

// desired is the flattened, comparable form of a listener spec derived
// from the (service, tenant) snapshot. Two desired entries with equal
// values mean no work is needed; any field difference triggers a
// drain-and-readd.
type desired struct {
	Key listener.Key
	Cfg ListenerConfig
}

// ComputeOps diffs prev against next and returns the ops needed to
// reconcile prev → next. A PORT CHANGE on the same Key emits Drain then
// Add (two ops); the apply loop must execute them in order.
//
// Tenants present in service.Tenants but absent from tenantConfigs are
// skipped — they need both topics to land. Tombstoned service drains
// every listener (returns Drain for every prev key, no adds).
func ComputeOps(prevSvc *configuration.RestModel, prevTenants map[uuid.UUID]tenant.RestModel,
	nextSvc *configuration.RestModel, nextTenants map[uuid.UUID]tenant.RestModel) []Op {

	prevDesired := flatten(prevSvc, prevTenants)
	nextDesired := flatten(nextSvc, nextTenants)

	var ops []Op
	// Drains first: any key in prev not in next, OR present in both but
	// with a different Cfg.
	for k, p := range prevDesired {
		n, stillThere := nextDesired[k]
		if !stillThere || n.Cfg != p.Cfg {
			ops = append(ops, Op{Kind: OpDrain, Key: k})
		}
	}
	// Adds: any key in next not in prev, OR present in both with a
	// different Cfg (the drain above plus this add gives the requested
	// drain-then-add semantics).
	for k, n := range nextDesired {
		p, was := prevDesired[k]
		if !was || n.Cfg != p.Cfg {
			ops = append(ops, Op{Kind: OpAdd, Key: k, Cfg: n.Cfg})
		}
	}
	return ops
}

// flatten walks a (service, tenants) pair and yields one desired entry
// per tenant. Tenants without a matching tenant config are skipped — both
// topics must agree before we admit a listener.
func flatten(svc *configuration.RestModel, tenants map[uuid.UUID]tenant.RestModel) map[listener.Key]desired {
	out := make(map[listener.Key]desired)
	if svc == nil {
		return out
	}
	for _, st := range svc.Tenants {
		tid, err := uuid.Parse(st.Id)
		if err != nil {
			continue
		}
		tc, ok := tenants[tid]
		if !ok {
			continue
		}
		k := listener.Key{TenantId: tid}
		out[k] = desired{
			Key: k,
			Cfg: ListenerConfig{
				Port:         st.Port,
				Region:       tc.Region,
				MajorVersion: tc.MajorVersion,
				MinorVersion: tc.MinorVersion,
			},
		}
	}
	return out
}
