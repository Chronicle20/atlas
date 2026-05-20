package crypto

import "testing"

func TestCalculateVersionHash(t *testing.T) {
	tests := []struct {
		version          int
		wantEncrypted    uint16
		wantHashNonZero  bool
	}{
		{version: 83, wantHashNonZero: true},
		{version: 95, wantHashNonZero: true},
		{version: 176, wantHashNonZero: true},
		{version: 1, wantHashNonZero: true},
	}
	for _, tt := range tests {
		ev, hash := CalculateVersionHash(tt.version)
		if !tt.wantHashNonZero {
			continue
		}
		if hash == 0 {
			t.Errorf("CalculateVersionHash(%d): hash = 0, want non-zero", tt.version)
		}
		if ev == 0 {
			t.Errorf("CalculateVersionHash(%d): encryptedVersion = 0, want non-zero", tt.version)
		}
	}
}

func TestCalculateVersionHashDeterministic(t *testing.T) {
	ev1, hash1 := CalculateVersionHash(83)
	ev2, hash2 := CalculateVersionHash(83)
	if ev1 != ev2 || hash1 != hash2 {
		t.Errorf("CalculateVersionHash is not deterministic: (%d, %d) != (%d, %d)", ev1, hash1, ev2, hash2)
	}
}

func TestCalculateVersionHashDifferentVersions(t *testing.T) {
	ev1, hash1 := CalculateVersionHash(83)
	ev2, hash2 := CalculateVersionHash(95)
	if ev1 == ev2 && hash1 == hash2 {
		t.Errorf("CalculateVersionHash(83) == CalculateVersionHash(95), expected different results")
	}
}

func TestGetIVForEncryption(t *testing.T) {
	gms := GetIVForEncryption(EncryptionGMS)
	if len(gms) != 4 || gms[0] != 0x4D {
		t.Errorf("GetIVForEncryption(GMS) = %v, want [0x4D, ...]", gms)
	}

	kms := GetIVForEncryption(EncryptionKMS)
	if len(kms) != 4 || kms[0] != 0xB9 {
		t.Errorf("GetIVForEncryption(KMS) = %v, want [0xB9, ...]", kms)
	}

	none := GetIVForEncryption(EncryptionNone)
	if len(none) != 4 || none[0] != 0x00 || none[1] != 0x00 || none[2] != 0x00 || none[3] != 0x00 {
		t.Errorf("GetIVForEncryption(None) = %v, want [0, 0, 0, 0]", none)
	}
}

func TestAllEncryptionTypes(t *testing.T) {
	types := AllEncryptionTypes()
	if len(types) != 3 {
		t.Errorf("AllEncryptionTypes() returned %d types, want 3", len(types))
	}
}
