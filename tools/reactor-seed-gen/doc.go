package main

// scriptDoc is the bare script object that becomes data.attributes.
type scriptDoc struct {
	ReactorId   string
	Description string
	HitRules    []ruleDoc
	ActRules    []ruleDoc
}

// ruleDoc is one rule. Conditions are ANDed; an empty slice always matches.
type ruleDoc struct {
	Id         string
	Conditions []condDoc
	Operations []opDoc
}

type condDoc struct {
	Type     string // "reactor_state" | "pq_custom_data"
	Operator string // "=" "!=" ">" "<" ">=" "<="
	Value    string
	Step     string // pq_custom_data only; omitted when empty
}

type opDoc struct {
	Type   string
	Params map[string]string // nil means "emit no params key at all"
}
