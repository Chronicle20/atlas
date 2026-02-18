package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type AccountKey struct {
	Tenant    tenant.Model
	AccountId uint32
}

type Service string

const (
	ServiceLogin   = "LOGIN"
	ServiceChannel = "CHANNEL"
)

type StateValue struct {
	State     State
	UpdatedAt time.Time
}

type ServiceKey struct {
	SessionId uuid.UUID
	Service   Service
}

type sessionEntry struct {
	State     State     `json:"state"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type sessionsData map[string]sessionEntry

type Registry struct {
	reg    *atlas.TenantRegistry[uint32, sessionsData]
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, sessionsData](client, "account-session", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		client: client,
	}
}

func GetRegistry() *Registry {
	return registry
}

func serviceKeyStr(sk ServiceKey) string {
	return fmt.Sprintf("%s:%s", sk.SessionId.String(), string(sk.Service))
}

func parseServiceKey(s string) (ServiceKey, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return ServiceKey{}, errors.New("invalid service key")
	}
	id, err := uuid.Parse(s[:idx])
	if err != nil {
		return ServiceKey{}, err
	}
	return ServiceKey{SessionId: id, Service: Service(s[idx+1:])}, nil
}

func (r *Registry) tenantSetKey() string {
	return fmt.Sprintf("atlas:%s:_tenants", r.reg.Namespace())
}

func (r *Registry) getSessions(ctx context.Context, key AccountKey) (sessionsData, error) {
	return r.reg.Get(ctx, key.Tenant, key.AccountId)
}

func (r *Registry) putSessions(ctx context.Context, key AccountKey, sm sessionsData) error {
	err := r.reg.Put(ctx, key.Tenant, key.AccountId, sm)
	if err != nil {
		return err
	}
	t := key.Tenant
	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)
	return nil
}

func (r *Registry) GetStates(ctx context.Context, key AccountKey) map[ServiceKey]StateValue {
	sm, err := r.getSessions(ctx, key)
	if err != nil {
		return map[ServiceKey]StateValue{}
	}

	result := make(map[ServiceKey]StateValue)
	for k, v := range sm {
		sk, err := parseServiceKey(k)
		if err != nil {
			continue
		}
		result[sk] = StateValue{State: v.State, UpdatedAt: v.UpdatedAt}
	}
	return result
}

func (r *Registry) MaximalState(ctx context.Context, key AccountKey) State {
	sm, err := r.getSessions(ctx, key)
	if err != nil || len(sm) == 0 {
		return StateNotLoggedIn
	}

	var maximalState = uint8(99)
	for _, entry := range sm {
		if uint8(entry.State) < maximalState {
			maximalState = uint8(entry.State)
		}
	}
	return State(maximalState)
}

func (r *Registry) IsLoggedIn(ctx context.Context, key AccountKey) bool {
	return r.MaximalState(ctx, key) > 0
}

func (r *Registry) Login(ctx context.Context, key AccountKey, sk ServiceKey) error {
	sm, err := r.getSessions(ctx, key)
	if err != nil {
		sm = make(sessionsData)
	}

	keyStr := serviceKeyStr(sk)

	if sk.Service == ServiceLogin {
		for _, entry := range sm {
			if entry.State > 0 {
				return errors.New("already logged in")
			}
		}
		sm[keyStr] = sessionEntry{State: StateLoggedIn, UpdatedAt: time.Now()}
	} else if sk.Service == ServiceChannel {
		var transition bool
		for _, entry := range sm {
			if entry.State > 1 {
				transition = true
			}
		}
		if !transition {
			return errors.New("no other service transitioning")
		}
		sm = sessionsData{keyStr: sessionEntry{State: StateLoggedIn, UpdatedAt: time.Now()}}
	} else {
		return errors.New("undefined service")
	}

	return r.putSessions(ctx, key, sm)
}

func (r *Registry) Transition(ctx context.Context, key AccountKey, sk ServiceKey) error {
	sm, err := r.getSessions(ctx, key)
	if err != nil {
		return errors.New("not logged in")
	}

	keyStr := serviceKeyStr(sk)
	if entry, ok := sm[keyStr]; ok {
		if entry.State > 0 {
			sm[keyStr] = sessionEntry{State: StateTransition, UpdatedAt: time.Now()}
			return r.putSessions(ctx, key, sm)
		}
	}
	return errors.New("not logged in")
}

func (r *Registry) ExpireTransition(ctx context.Context, key AccountKey, timeout time.Duration) {
	sm, err := r.getSessions(ctx, key)
	if err != nil {
		return
	}

	changed := false
	for k, entry := range sm {
		if entry.State == StateTransition && time.Since(entry.UpdatedAt) > timeout {
			delete(sm, k)
			changed = true
		}
	}

	if changed {
		_ = r.putSessions(ctx, key, sm)
	}
}

func (r *Registry) Logout(ctx context.Context, key AccountKey, sk ServiceKey) bool {
	sm, err := r.getSessions(ctx, key)
	if err != nil {
		return true
	}

	keyStr := serviceKeyStr(sk)
	if entry, ok := sm[keyStr]; ok {
		if entry.State == StateTransition {
			return false
		}
	}
	delete(sm, keyStr)
	_ = r.putSessions(ctx, key, sm)
	return true
}

func (r *Registry) Terminate(ctx context.Context, key AccountKey) bool {
	_ = r.reg.Remove(ctx, key.Tenant, key.AccountId)
	return true
}

func (r *Registry) GetExpiredInTransition(ctx context.Context, timeout time.Duration) []AccountKey {
	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return nil
	}

	var accounts []AccountKey
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}

		vals, err := r.reg.GetAllValues(ctx, t)
		if err != nil {
			continue
		}

		// We need to reconstruct account IDs from the keys, but GetAllValues only returns values.
		// Instead, scan the keys directly.
		pattern := fmt.Sprintf("atlas:%s:%s:*", r.reg.Namespace(), atlas.TenantKey(t))
		prefix := fmt.Sprintf("atlas:%s:%s:", r.reg.Namespace(), atlas.TenantKey(t))
		var cursor uint64
		for {
			keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			for _, k := range keys {
				suffix := strings.TrimPrefix(k, prefix)
				if strings.HasPrefix(suffix, "_") {
					continue
				}
				accountId, err := strconv.ParseUint(suffix, 10, 32)
				if err != nil {
					continue
				}
				ak := AccountKey{Tenant: t, AccountId: uint32(accountId)}
				sm, err := r.getSessions(ctx, ak)
				if err != nil {
					continue
				}
				for _, entry := range sm {
					if entry.State == StateTransition && time.Since(entry.UpdatedAt) > timeout {
						accounts = append(accounts, ak)
						break
					}
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		_ = vals // GetAllValues not needed, using SCAN instead
	}
	return accounts
}

func (r *Registry) Tenants(ctx context.Context) []tenant.Model {
	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return nil
	}

	var tenants []tenant.Model
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants
}
