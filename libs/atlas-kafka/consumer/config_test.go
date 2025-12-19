package consumer

import (
	"github.com/Chronicle20/atlas-model/model"
	"testing"
	"time"
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
