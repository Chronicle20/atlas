package crimsonbalrog

import (
	"atlas-events/event/registry"
	"atlas-events/kafka/message"
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"
	"context"
	"fmt"
)

// Start orchestrates the side effects of a newly created occurrence, in the
// attack maps of THIS channel only (FR-B11). Order matters: the visual first,
// so a player sees the enemy ship before its monsters materialize. atlas-events
// constructs no packets here (FR-B12) — the visual name and the state/subState
// bytes ride on the event as gameplay content; atlas-channel turns them into
// ContiMoveBody. Spawns go to the attack map only (FR-B13); the related
// (cabin) maps got their non-visual scope row already, in Evaluate.
func (h *Handler) Start(ctx context.Context, o registry.Occurrence) (registry.Progress, error) {
	oc, err := DecodeOccurrenceContext(o.Context)
	if err != nil {
		return registry.Progress{}, err
	}

	err = message.Emit(h.l, ctx)(func(buf *message.Buffer) error {
		for _, am := range oc.AttackMaps {
			if err := buf.Put(event.EnvEventTopicEventVisual, showVisualEventProvider(
				o.Id, oc.WorldId, oc.ChannelId, am.MapId,
				oc.Visual.Name, oc.Visual.ShowState, oc.Visual.ShowSubState, oc.BackgroundMusic,
			)); err != nil {
				return err
			}

			// config.Validate rejects a definition whose attack maps carry
			// fewer spawn positions than monsterCount (config.go), so every
			// AttackMap here is guaranteed to have at least oc.MonsterCount
			// positions. This check only guards against that invariant
			// somehow not holding (e.g. a stored occurrence context from a
			// definition version that predates the guard) — it fails closed
			// rather than reusing or wrapping around a position no one
			// configured.
			if uint32(len(am.SpawnPositions)) < oc.MonsterCount {
				return fmt.Errorf("crimsonbalrog: attack map %d has %d spawn positions, need %d", am.MapId, len(am.SpawnPositions), oc.MonsterCount)
			}

			for i := uint32(0); i < oc.MonsterCount; i++ {
				pos := am.SpawnPositions[i]
				if err := buf.Put(monster.EnvCommandTopic, spawnFieldCommandProvider(
					oc.WorldId, oc.ChannelId, am.MapId, oc.MonsterId, pos, o.Id,
				)); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return registry.Progress{}, err
	}

	// No NextTransitionAt: completion is externally driven — every monster
	// dies, or the vessel arrives (FR-B17). Nothing about this occurrence is
	// timed.
	return registry.Progress{Stage: StageAttacking}, nil
}
