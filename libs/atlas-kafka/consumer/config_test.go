package consumer

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestConfig(t *testing.T) {
	c := NewConfig([]string{"test"}, "test", "test_topic", "test_group")

	if len(c.brokers) != 1 {
		t.Fatalf("Invalid broker count.")
	}

	if c.maxWait != time.Millisecond*50 {
		t.Fatalf("Invalid broker max wait.")
	}

	c, err := model.Decorate(model.Decorators(SetMaxWait(time.Second * 60)))(c)
	if err != nil || c.maxWait != time.Second*60 {
		t.Fatalf("Invalid broker max wait.")
	}
}

func TestFetchTimeoutDefaultsAndOverride(t *testing.T) {
	c := NewConfig([]string{"test"}, "test", "test_topic", "test_group")

	if c.fetchTimeout != 5*time.Minute {
		t.Fatalf("expected default fetchTimeout=5m, got %v", c.fetchTimeout)
	}
	if c.maxConsecutiveTimeouts != 3 {
		t.Fatalf("expected default maxConsecutiveTimeouts=3, got %d", c.maxConsecutiveTimeouts)
	}

	c, err := model.Decorate(model.Decorators(SetFetchTimeout(20*time.Minute)))(c)
	if err != nil || c.fetchTimeout != 20*time.Minute {
		t.Fatalf("expected SetFetchTimeout to override to 20m, got %v (err=%v)", c.fetchTimeout, err)
	}

	c, err = model.Decorate(model.Decorators(SetMaxConsecutiveTimeouts(7)))(c)
	if err != nil || c.maxConsecutiveTimeouts != 7 {
		t.Fatalf("expected SetMaxConsecutiveTimeouts to override to 7, got %d (err=%v)", c.maxConsecutiveTimeouts, err)
	}
}
