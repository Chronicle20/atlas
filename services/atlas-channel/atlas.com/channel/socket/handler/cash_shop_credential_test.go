package handler

import "testing"

// TestCredentialMatches covers the pure decision credentialMatches makes,
// with no session and no processor involved.
func TestCredentialMatches(t *testing.T) {
	tests := []struct {
		name            string
		usesPIC         bool
		storedPIC       string
		storedBirthDate uint32
		spw             string
		birthday        uint32
		expect          bool
	}{
		{
			name:            "pic matches",
			usesPIC:         true,
			storedPIC:       "5678",
			storedBirthDate: 19940203,
			spw:             "5678",
			birthday:        0,
			expect:          true,
		},
		{
			name:            "pic mismatches",
			usesPIC:         true,
			storedPIC:       "5678",
			storedBirthDate: 19940203,
			spw:             "1234",
			birthday:        0,
			expect:          false,
		},
		{
			name:            "pic empty passes",
			usesPIC:         true,
			storedPIC:       "",
			storedBirthDate: 19940203,
			spw:             "1234",
			birthday:        0,
			expect:          true,
		},
		{
			name:            "pic empty and empty spw passes",
			usesPIC:         true,
			storedPIC:       "",
			storedBirthDate: 0,
			spw:             "",
			birthday:        0,
			expect:          true,
		},
		{
			name:            "birthday matches",
			usesPIC:         false,
			storedPIC:       "5678",
			storedBirthDate: 19940203,
			spw:             "",
			birthday:        19940203,
			expect:          true,
		},
		{
			name:            "birthday mismatches",
			usesPIC:         false,
			storedPIC:       "5678",
			storedBirthDate: 19940203,
			spw:             "",
			birthday:        19700101,
			expect:          false,
		},
		{
			name:            "birthday unset passes",
			usesPIC:         false,
			storedPIC:       "5678",
			storedBirthDate: 0,
			spw:             "",
			birthday:        19700101,
			expect:          true,
		},
		{
			name:            "pre-95 ignores pic entirely",
			usesPIC:         false,
			storedPIC:       "5678",
			storedBirthDate: 19940203,
			spw:             "wrong",
			birthday:        19940203,
			expect:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := credentialMatches(tc.usesPIC, tc.storedPIC, tc.storedBirthDate, tc.spw, tc.birthday)
			if got != tc.expect {
				t.Errorf("credentialMatches(%v, %q, %d, %q, %d) = %v, want %v",
					tc.usesPIC, tc.storedPIC, tc.storedBirthDate, tc.spw, tc.birthday, got, tc.expect)
			}
		})
	}
}
