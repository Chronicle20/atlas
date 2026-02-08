package ban

import "testing"

func TestIpMatchesCIDR(t *testing.T) {
	if !ipMatchesCIDR("192.168.1.50", "192.168.1.0/24") {
		t.Error("192.168.1.50 should match 192.168.1.0/24")
	}

	if ipMatchesCIDR("10.0.0.1", "192.168.1.0/24") {
		t.Error("10.0.0.1 should not match 192.168.1.0/24")
	}

	if !ipMatchesCIDR("10.0.0.1", "10.0.0.0/8") {
		t.Error("10.0.0.1 should match 10.0.0.0/8")
	}

	if ipMatchesCIDR("invalidip", "192.168.1.0/24") {
		t.Error("Invalid IP should not match")
	}

	if ipMatchesCIDR("192.168.1.1", "invalidcidr") {
		t.Error("Invalid CIDR should not match")
	}
}

func TestIsCIDR(t *testing.T) {
	if !isCIDR("192.168.1.0/24") {
		t.Error("192.168.1.0/24 should be a valid CIDR")
	}

	if !isCIDR("10.0.0.0/8") {
		t.Error("10.0.0.0/8 should be a valid CIDR")
	}

	if isCIDR("192.168.1.1") {
		t.Error("192.168.1.1 should not be a valid CIDR")
	}

	if isCIDR("notacidr") {
		t.Error("notacidr should not be a valid CIDR")
	}

	if isCIDR("") {
		t.Error("Empty string should not be a valid CIDR")
	}
}
