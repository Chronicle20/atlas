package configuration

import (
	"atlas-channel/configuration/tenant"
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var once sync.Once
var configMu sync.RWMutex
var serviceConfig *RestModel
var tenantConfig map[uuid.UUID]tenant.RestModel

// readyCh is closed once PublishSnapshot has populated serviceConfig for
// the first time. main.go registers ~20 Kafka consumer groups before
// WaitCaughtUp returns; if a message arrives in that window and the
// handler calls GetServiceConfig / GetTenantConfig, the legacy log.Fatalf
// path used to crash the pod. Get* now blocks on readyCh instead, bounded
// by readyTimeout.
var readyCh = make(chan struct{})
var readyOnce sync.Once

// readyTimeout caps how long a Get* call waits for the projection's first
// PublishSnapshot. Long enough to outlast catch-up in a fresh PR env,
// short enough that a wedged projection surfaces as request errors rather
// than goroutine pileup.
const readyTimeout = 60 * time.Second

// ErrNotReady is returned by Get* when the projection has not yet
// published a snapshot within readyTimeout. Callers should log and skip;
// /readyz keeps the pod out of service until catch-up completes.
var ErrNotReady = errors.New("configuration: projection snapshot not yet published")

// ErrTenantNotConfigured is returned by GetTenantConfig when the requested
// tenant is absent from the current snapshot.
var ErrTenantNotConfigured = errors.New("configuration: tenant not configured")

func waitReady() error {
	select {
	case <-readyCh:
		return nil
	case <-time.After(readyTimeout):
		return ErrNotReady
	}
}

func GetServiceConfig() (*RestModel, error) {
	if err := waitReady(); err != nil {
		return nil, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	if serviceConfig == nil {
		return nil, ErrNotReady
	}
	return serviceConfig, nil
}

func GetTenantConfigs() map[uuid.UUID]tenant.RestModel {
	_ = waitReady()
	configMu.RLock()
	defer configMu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(tenantConfig))
	for k, v := range tenantConfig {
		out[k] = v
	}
	return out
}

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return tenant.RestModel{}, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	val, ok := tenantConfig[tenantId]
	if !ok {
		return tenant.RestModel{}, ErrTenantNotConfigured
	}
	return val, nil
}

func Init(l logrus.FieldLogger) func(ctx context.Context) func(serviceId uuid.UUID) {
	return func(ctx context.Context) func(serviceId uuid.UUID) {
		return func(serviceId uuid.UUID) {
			once.Do(func() {
				tenantConfig = make(map[uuid.UUID]tenant.RestModel)
				c, err := requestByService(serviceId)(l, ctx)
				if err != nil {
					log.Fatalf("Could not retrieve configuration.")
				}
				serviceConfig = &c

				for _, t := range c.Tenants {
					tenantId := uuid.MustParse(t.Id)
					tc, err := requestForTenant(tenantId)(l, ctx)
					if err != nil {
						log.Fatalf("Could not retrieve tenant configuration.")
					}
					tenantConfig[tenantId] = tc
				}
			})
		}
	}
}

// PublishSnapshot replaces the package-level service+tenant config with
// the snapshot taken from the kafka-backed projection. Called by main.go
// after CaughtUp fires (and again from the projection apply loop on each
// observed change) so legacy callers of GetServiceConfig / GetTenantConfig
// (handlers, the session timeout task, kafka consumers) see the same data
// the listener registry was built from. Both args are taken by value-copy
// so the caller's projection State can mutate independently after the
// call.
func PublishSnapshot(svc *RestModel, tenants map[uuid.UUID]tenant.RestModel) {
	configMu.Lock()
	if svc != nil {
		c := *svc
		serviceConfig = &c
	} else {
		serviceConfig = nil
	}
	next := make(map[uuid.UUID]tenant.RestModel, len(tenants))
	for k, v := range tenants {
		next[k] = v
	}
	tenantConfig = next
	configMu.Unlock()

	if svc != nil {
		readyOnce.Do(func() { close(readyCh) })
	}
}
