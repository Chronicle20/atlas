package writer

import (
	"testing"

	"github.com/google/uuid"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

func TestTenantWriterOptionsLifecycle(t *testing.T) {
	tid := uuid.New()
	other := uuid.New()
	RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
		{OpCode: "0xEA", Writer: "CharacterSkillCooldown", Options: map[string]interface{}{
			"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": float64(5221999)},
		}},
		{OpCode: "0x20", Writer: "CharacterBuffGive"}, // nil options
	})
	t.Cleanup(func() { EvictTenantWriterOptions(tid) })

	opts, ok := TenantWriterOptions(tid, "CharacterSkillCooldown")
	if !ok {
		t.Fatal("expected options for CharacterSkillCooldown")
	}
	if _, hasSkills := opts["skills"]; !hasSkills {
		t.Fatal("expected skills table in options")
	}

	if _, ok := TenantWriterOptions(tid, "CharacterBuffGive"); ok {
		t.Fatal("writer with nil options should report ok=false")
	}
	if _, ok := TenantWriterOptions(tid, "NoSuchWriter"); ok {
		t.Fatal("unknown writer should report ok=false")
	}
	if _, ok := TenantWriterOptions(other, "CharacterSkillCooldown"); ok {
		t.Fatal("unregistered tenant should report ok=false")
	}

	EvictTenantWriterOptions(tid)
	if _, ok := TenantWriterOptions(tid, "CharacterSkillCooldown"); ok {
		t.Fatal("evicted tenant should report ok=false")
	}
}
