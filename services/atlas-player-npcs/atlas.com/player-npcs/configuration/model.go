package configuration

// Model is the tenant's player-npcs configuration (design §9.4, FR-4.7).
// Defaults apply when a tenant has no configuration, or the read fails —
// see rest.go's DefaultModel.
type Model struct {
	initialX          int16
	initialY          int16
	areaX             int16
	areaY             int16
	areaSteps         int
	organizeArea      bool
	autoDeployEnabled bool
}

func (m Model) InitialX() int16         { return m.initialX }
func (m Model) InitialY() int16         { return m.initialY }
func (m Model) AreaX() int16            { return m.areaX }
func (m Model) AreaY() int16            { return m.areaY }
func (m Model) AreaSteps() int          { return m.areaSteps }
func (m Model) OrganizeArea() bool      { return m.organizeArea }
func (m Model) AutoDeployEnabled() bool { return m.autoDeployEnabled }
