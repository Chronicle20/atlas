package routine

// The helper lib itself is the only package allowed bare go statements.
func spawnInternal() {
	go func() {}()
}
