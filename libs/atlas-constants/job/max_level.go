package job

// MaxLevelFor returns the level cap for the job's line. All lines currently
// return 200: services/atlas-character/atlas.com/character/character/experience_table.go
// defines a single flat MaxLevel = 200, and atlas-character never emits a
// level above it for any job. The per-line switch exists so the Cygnus
// figure is a one-line edit if a later verification confirms a different cap.
func MaxLevelFor(jobId Id) byte {
	switch GetType(jobId) {
	case TypeExplorer:
		return 200
	case TypeCygnus:
		return 200
	case TypeLegend:
		return 200
	default:
		return 200
	}
}
