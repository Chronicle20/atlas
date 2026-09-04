package script

import (
	"atlas-map-actions/validation"
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/condition"
)

type ConditionEvaluator struct {
	l           logrus.FieldLogger
	ctx         context.Context
	validationP validation.Processor
}

func NewConditionEvaluator(l logrus.FieldLogger, ctx context.Context) *ConditionEvaluator {
	return &ConditionEvaluator{
		l:           l,
		ctx:         ctx,
		validationP: validation.NewProcessor(l, ctx),
	}
}

func (e *ConditionEvaluator) EvaluateCondition(f field.Model, characterId uint32, cond condition.Model) (bool, error) {
	e.l.Debugf("Evaluating condition [%s] for character [%d].", cond.Type(), characterId)

	switch cond.Type() {
	case "map_id":
		return e.evaluateMapId(f, cond)
	default:
		return e.evaluateViaQueryAggregator(f, characterId, cond)
	}
}

func (e *ConditionEvaluator) evaluateMapId(f field.Model, cond condition.Model) (bool, error) {
	expectedMapId, err := strconv.ParseUint(cond.Value(), 10, 32)
	if err != nil {
		return false, fmt.Errorf("invalid map_id value [%s]: %w", cond.Value(), err)
	}

	actualMapId := uint64(f.MapId())

	switch cond.Operator() {
	case "=", "==":
		return actualMapId == expectedMapId, nil
	case "!=":
		return actualMapId != expectedMapId, nil
	case ">":
		return actualMapId > expectedMapId, nil
	case "<":
		return actualMapId < expectedMapId, nil
	case ">=":
		return actualMapId >= expectedMapId, nil
	case "<=":
		return actualMapId <= expectedMapId, nil
	default:
		return false, fmt.Errorf("unsupported operator [%s] for map_id condition", cond.Operator())
	}
}

func (e *ConditionEvaluator) evaluateViaQueryAggregator(f field.Model, characterId uint32, cond condition.Model) (bool, error) {
	// Parse the scalar value only when one is present. An `in` condition
	// carries its payload in values and leaves value empty; eagerly Atoi-ing
	// it here made `in` unreachable from a map-action document.
	var intValue int
	if valueStr := cond.Value(); valueStr != "" {
		parsed, err := strconv.Atoi(valueStr)
		if err != nil {
			return false, fmt.Errorf("invalid condition value [%s]: %w", valueStr, err)
		}
		intValue = parsed
	}

	var intValues []int
	for _, v := range cond.Values() {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return false, fmt.Errorf("invalid condition values entry [%s]: %w", v, err)
		}
		intValues = append(intValues, parsed)
	}

	var resolvedReferenceId uint32
	referenceIdRaw := cond.ReferenceIdRaw()
	if referenceIdRaw != "" {
		refIdInt, convErr := strconv.ParseUint(referenceIdRaw, 10, 32)
		if convErr != nil {
			return false, fmt.Errorf("referenceId [%s] is not a valid uint32", referenceIdRaw)
		}
		resolvedReferenceId = uint32(refIdInt)
	}

	// worldId/channelId come from the field the entry event carried, never from
	// the document — a seed is version-scoped, not world-scoped.
	validationCondition := validation.ConditionInput{
		Type:            cond.Type(),
		Operator:        cond.Operator(),
		Value:           intValue,
		Values:          intValues,
		ReferenceId:     resolvedReferenceId,
		Step:            cond.Step(),
		WorldId:         f.WorldId(),
		ChannelId:       f.ChannelId(),
		IncludeEquipped: cond.IncludeEquipped(),
		ValueString:     cond.ValueString(),
	}

	result, err := e.validationP.ValidateCharacterState(characterId, []validation.ConditionInput{validationCondition})
	if err != nil {
		e.l.WithError(err).Errorf("Failed to validate character state for condition [%s].", cond.Type())
		return false, err
	}

	e.l.Debugf("Condition [%s] evaluated to [%t] for character [%d].", cond.Type(), result.Passed(), characterId)
	return result.Passed(), nil
}
