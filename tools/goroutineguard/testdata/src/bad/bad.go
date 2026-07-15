package bad

func named() {}

func spawn() {
	go func() {}() // want `goroutineguard: bare go statement`
	go named()     // want `goroutineguard: bare go statement`

	//goroutine-guard:allow
	go named() // want `goroutineguard: allow marker requires a justification`
}
