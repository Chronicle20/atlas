package good

import "context"

func named() {}

// spawnLike mimics routine.Go's shape; ordinary calls are not go statements.
func spawnLike(fn func(context.Context)) { fn(context.Background()) }

//go:generate echo "go func() in a directive must not match"

const doc = "go func() inside a string literal must not match"

// A comment mentioning go func() and go named() must not match.

func fine() {
	spawnLike(func(context.Context) {})

	//goroutine-guard:allow fixture: marker on the line above with justification
	go named()

	go named() //goroutine-guard:allow fixture: trailing marker with justification
}
