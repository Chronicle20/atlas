package setup

type Model struct {
	recoveryHP uint32
	recoveryMP uint32
}

func (m Model) RecoveryHP() uint32 {
	return m.recoveryHP
}

func (m Model) RecoveryMP() uint32 {
	return m.recoveryMP
}
