// Package topic is a stub of libs/atlas-kafka/topic for tools/topicguard's
// analysistest fixtures, which run inside a synthetic GOPATH that cannot
// resolve the real module. Kept in sync in shape (not content) with
// libs/atlas-kafka/topic/token.go: only the named type the fixtures need to
// import.
package topic

// Token mirrors libs/atlas-kafka/topic.Token for fixture purposes.
type Token string
