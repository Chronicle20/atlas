package saga

import "testing"

// TestExtractDerivesTheOutcomeFromTheSteps pins the derivation reconciliation
// completes a trade on. The orchestrator's saga resource carries no
// saga-level status field — only per-step Status — so this table IS the oracle,
// and every wrong answer is a real trade either credited twice or cancelled
// after it executed.
func TestExtractDerivesTheOutcomeFromTheSteps(t *testing.T) {
	for name, tc := range map[string]struct {
		steps []StepRestModel
		want  Outcome
	}{
		"every step completed": {
			steps: []StepRestModel{{Status: stepCompleted}, {Status: stepCompleted}},
			want:  OutcomeSucceeded,
		},
		"one step failed": {
			steps: []StepRestModel{{Status: stepCompleted}, {Status: stepFailed}},
			want:  OutcomeFailed,
		},
		"a failure anywhere wins over the completions after it": {
			steps: []StepRestModel{{Status: stepFailed}, {Status: stepCompleted}, {Status: stepCompleted}},
			want:  OutcomeFailed,
		},
		"still running": {
			steps: []StepRestModel{{Status: stepCompleted}, {Status: stepPending}},
			want:  OutcomeRunning,
		},
		// An unexpanded composite has no steps at all. Reading "no failures" as
		// success there would complete a trade that has moved nothing.
		"not yet expanded": {
			steps: nil,
			want:  OutcomeRunning,
		},
		// An unrecognised status is not evidence. Neither terminal answer may be
		// reached by guessing at it.
		"unrecognised status": {
			steps: []StepRestModel{{Status: stepCompleted}, {Status: "wat"}},
			want:  OutcomeRunning,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Extract(RestModel{Steps: tc.steps})
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			if got != tc.want {
				t.Errorf("outcome: got %s, want %s", got, tc.want)
			}
		})
	}
}
