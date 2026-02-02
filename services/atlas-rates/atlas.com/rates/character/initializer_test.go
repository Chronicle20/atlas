package character

import (
	"atlas-rates/rate"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func createTestTenantForInitializer() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestIsInitialized_False(t *testing.T) {
	ResetInitializedForTesting()
	ten := createTestTenantForInitializer()

	if IsInitialized(ten, 12345) {
		t.Error("IsInitialized() = true for uninitialized character")
	}
}

func TestMarkInitialized(t *testing.T) {
	ResetInitializedForTesting()
	ten := createTestTenantForInitializer()

	MarkInitialized(ten, 12345)

	if !IsInitialized(ten, 12345) {
		t.Error("IsInitialized() = false after MarkInitialized()")
	}
}

func TestClearInitialized(t *testing.T) {
	ResetInitializedForTesting()
	ten := createTestTenantForInitializer()

	MarkInitialized(ten, 12345)
	ClearInitialized(ten, 12345)

	if IsInitialized(ten, 12345) {
		t.Error("IsInitialized() = true after ClearInitialized()")
	}
}

func TestInitializedTenantIsolation(t *testing.T) {
	ResetInitializedForTesting()

	t1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	t2, _ := tenant.Create(uuid.New(), "KMS", 1, 2)

	MarkInitialized(t1, 12345)

	if !IsInitialized(t1, 12345) {
		t.Error("IsInitialized(t1) = false, want true")
	}
	if IsInitialized(t2, 12345) {
		t.Error("IsInitialized(t2) = true, want false (tenant isolation)")
	}
}

func TestGetRateTypeFromTemplateId(t *testing.T) {
	tests := []struct {
		name       string
		templateId uint32
		expected   rate.Type
	}{
		// EXP coupons (521xxxx)
		{"exp coupon min", 5210000, rate.TypeExp},
		{"exp coupon mid", 5210001, rate.TypeExp},
		{"exp coupon max", 5219999, rate.TypeExp},

		// Drop coupons (536xxxx)
		{"drop coupon min", 5360000, rate.TypeItemDrop},
		{"drop coupon mid", 5360001, rate.TypeItemDrop},
		{"drop coupon max", 5369999, rate.TypeItemDrop},

		// Out of range
		{"below exp range", 5209999, ""},
		{"above exp range", 5220000, ""},
		{"below drop range", 5359999, ""},
		{"above drop range", 5370000, ""},
		{"unrelated item", 1000000, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRateTypeFromTemplateId(tt.templateId)
			if got != tt.expected {
				t.Errorf("GetRateTypeFromTemplateId(%d) = %v, want %v", tt.templateId, got, tt.expected)
			}
		})
	}
}
