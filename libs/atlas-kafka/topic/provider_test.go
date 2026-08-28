package topic

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestEnvProvider(t *testing.T) {
	const tok = "EVENT_TOPIC_PROVIDER_TEST"
	const wantErr = "topic token [EVENT_TOPIC_PROVIDER_TEST] has no value in the environment"

	l := logrus.New()
	l.SetOutput(io.Discard)

	tests := []struct {
		name    string
		setEnv  bool
		envVal  string
		want    string
		wantErr string
	}{
		{
			name:   "resolved",
			setEnv: true,
			envVal: "evt-resolved",
			want:   "evt-resolved",
		},
		{
			name:    "unset",
			setEnv:  false,
			want:    "",
			wantErr: wantErr,
		},
		{
			name:    "empty value",
			setEnv:  true,
			envVal:  "",
			want:    "",
			wantErr: wantErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tok, tt.envVal)
			}

			got, err := EnvProvider(l)(tok)()

			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("err = nil, want error containing %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("err.Error() = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
