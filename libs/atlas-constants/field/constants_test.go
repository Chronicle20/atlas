package field

import "testing"

func TestParseObjectKind(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    ObjectKind
		wantErr string
	}{
		{name: "empty defaults to environment", s: "", want: ObjectKindEnvironment, wantErr: ""},
		{name: "environment", s: "ENVIRONMENT", want: ObjectKindEnvironment, wantErr: ""},
		{name: "obstacle", s: "OBSTACLE", want: ObjectKindObstacle, wantErr: ""},
		{name: "lowercase environment", s: "environment", want: ObjectKindEnvironment, wantErr: ""},
		{name: "lowercase obstacle", s: "obstacle", want: ObjectKindObstacle, wantErr: ""},
		{name: "mixed case obstacle", s: "Obstacle", want: ObjectKindObstacle, wantErr: ""},
		{name: "unknown", s: "GATE", want: ObjectKind(""), wantErr: "unrecognized object kind [GATE]"},
		{name: "whitespace only", s: "   ", want: ObjectKindEnvironment, wantErr: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseObjectKind(tt.s)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("expected error message %q, got %q", tt.wantErr, err.Error())
				}
			}

			if got != tt.want {
				t.Errorf("expected ObjectKind %q, got %q", tt.want, got)
			}
		})
	}
}

func TestObjectKindConstants(t *testing.T) {
	if string(ObjectKindEnvironment) != "ENVIRONMENT" {
		t.Errorf("expected ObjectKindEnvironment to be %q, got %q", "ENVIRONMENT", string(ObjectKindEnvironment))
	}
	if string(ObjectKindObstacle) != "OBSTACLE" {
		t.Errorf("expected ObjectKindObstacle to be %q, got %q", "OBSTACLE", string(ObjectKindObstacle))
	}
}
