package tenant

type Model struct {
	id           string
	region       string
	majorVersion uint16
	minorVersion uint16
}

func (m Model) Id() string {
	return m.id
}

func (m Model) Region() string {
	return m.region
}

func (m Model) MajorVersion() uint16 {
	return m.majorVersion
}

func (m Model) MinorVersion() uint16 {
	return m.minorVersion
}
