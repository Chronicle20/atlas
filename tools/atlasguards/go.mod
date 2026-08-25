module github.com/Chronicle20/atlas/tools/atlasguards

go 1.27.0

require (
	github.com/Chronicle20/atlas/tools/buffdurationguard v0.0.0
	github.com/Chronicle20/atlas/tools/goroutineguard v0.0.0
	github.com/Chronicle20/atlas/tools/outboxguard v0.0.0
	github.com/Chronicle20/atlas/tools/rediskeyguard v0.0.0
	github.com/Chronicle20/atlas/tools/scopeguard v0.0.0
	golang.org/x/tools v0.49.0
)

replace github.com/Chronicle20/atlas/tools/buffdurationguard => ../buffdurationguard

replace github.com/Chronicle20/atlas/tools/goroutineguard => ../goroutineguard

replace github.com/Chronicle20/atlas/tools/outboxguard => ../outboxguard

replace github.com/Chronicle20/atlas/tools/rediskeyguard => ../rediskeyguard

replace github.com/Chronicle20/atlas/tools/scopeguard => ../scopeguard
