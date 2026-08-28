package position

// PodiumPosition returns the slot position for rank on a podium map, laid
// out in groups of step slots per platform: platform = rank/step, relative
// = (rank%step)+1. platformX is -50 for platform 0, -170 for platform 1,
// and 70 for every other platform; platformY is -47 for platform 0 and 40
// otherwise. step == 0 returns ErrInvalidStep rather than dividing by zero.
func PodiumPosition(rank uint32, step byte) (Point, error) {
	if step == 0 {
		return Point{}, ErrInvalidStep
	}

	platform := rank / uint32(step)
	relative := (rank % uint32(step)) + 1

	var platformX, platformY int32
	switch platform {
	case 0:
		platformX, platformY = -50, -47
	case 1:
		platformX, platformY = -170, 40
	default:
		platformX, platformY = 70, 40
	}

	x := platformX + int32(100*relative)/(int32(step)+1)
	return Point{X: int16(x), CY: int16(platformY)}, nil
}

// EncodePodiumState packs a podium's current step and occupant count into a
// single wire value: count*32 + step.
func EncodePodiumState(step byte, count uint32) uint32 {
	return count*32 + uint32(step)
}

// DecodePodiumState reverses EncodePodiumState.
func DecodePodiumState(state uint32) (step byte, count uint32) {
	step = byte(state % 32)
	count = state / 32
	return step, count
}

// RaisePodiumStep raises step by one when count reaches 3*step occupants,
// signalling that the podium needs re-organization at the finer step.
// Raising never proceeds past t.AreaSteps: once step is already at the
// bound and count still warrants a raise, it returns ErrMapFull instead of
// exceeding it.
func RaisePodiumStep(t Tuning, step byte, count uint32) (newStep byte, raised bool, err error) {
	warrants := count >= 3*uint32(step)

	if step >= t.AreaSteps {
		if warrants {
			return step, false, ErrMapFull
		}
		return step, false, nil
	}

	if warrants {
		return step + 1, true, nil
	}
	return step, false, nil
}
