package script

type Model struct {
	onFirstUserEnter string
	onUserEnter      string
}

func (m Model) OnFirstUserEnter() string {
	return m.onFirstUserEnter
}

func (m Model) OnUserEnter() string {
	return m.onUserEnter
}
