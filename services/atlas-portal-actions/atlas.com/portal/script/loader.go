package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/sirupsen/logrus"
)

// jsonPortalScript represents the JSON structure of a portal script
type jsonPortalScript struct {
	PortalId    string     `json:"portalId"`
	MapId       uint32     `json:"mapId"`
	Description string     `json:"description"`
	Rules       []jsonRule `json:"rules"`
}

type jsonRule struct {
	Id         string          `json:"id"`
	Conditions []jsonCondition `json:"conditions"`
	OnMatch    jsonOutcome     `json:"onMatch"`
}

type jsonCondition struct {
	Type        string `json:"type"`
	Operator    string `json:"operator"`
	Value       string `json:"value"`
	ReferenceId string `json:"referenceId,omitempty"`
}

type jsonOutcome struct {
	Allow      bool            `json:"allow"`
	Operations []jsonOperation `json:"operations"`
}

type jsonOperation struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// Loader loads portal scripts from the filesystem
type Loader struct {
	l         logrus.FieldLogger
	scriptsDir string
	cache     map[string]PortalScript
	cacheMu   sync.RWMutex
}

// NewLoader creates a new script loader
func NewLoader(l logrus.FieldLogger, scriptsDir string) *Loader {
	return &Loader{
		l:         l,
		scriptsDir: scriptsDir,
		cache:     make(map[string]PortalScript),
	}
}

// LoadByPortalId loads a portal script by its ID
func (ld *Loader) LoadByPortalId(portalId string) (PortalScript, error) {
	// Check cache first
	ld.cacheMu.RLock()
	if script, ok := ld.cache[portalId]; ok {
		ld.cacheMu.RUnlock()
		return script, nil
	}
	ld.cacheMu.RUnlock()

	// Load from filesystem
	filePath := filepath.Join(ld.scriptsDir, portalId+".json")
	script, err := ld.loadFromFile(filePath)
	if err != nil {
		return PortalScript{}, err
	}

	// Cache the script
	ld.cacheMu.Lock()
	ld.cache[portalId] = script
	ld.cacheMu.Unlock()

	return script, nil
}

// loadFromFile loads a portal script from a JSON file
func (ld *Loader) loadFromFile(filePath string) (PortalScript, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return PortalScript{}, fmt.Errorf("failed to read script file [%s]: %w", filePath, err)
	}

	var jsonScript jsonPortalScript
	if err := json.Unmarshal(data, &jsonScript); err != nil {
		return PortalScript{}, fmt.Errorf("failed to parse script file [%s]: %w", filePath, err)
	}

	return ld.convertToModel(jsonScript)
}

// convertToModel converts JSON structure to domain model
func (ld *Loader) convertToModel(js jsonPortalScript) (PortalScript, error) {
	builder := NewPortalScriptBuilder().
		SetPortalId(js.PortalId).
		SetMapId(js.MapId).
		SetDescription(js.Description)

	for _, jr := range js.Rules {
		rule, err := ld.convertRule(jr)
		if err != nil {
			return PortalScript{}, fmt.Errorf("failed to convert rule [%s]: %w", jr.Id, err)
		}
		builder.AddRule(rule)
	}

	return builder.Build(), nil
}

// convertRule converts a JSON rule to domain model
func (ld *Loader) convertRule(jr jsonRule) (Rule, error) {
	rb := NewRuleBuilder().SetId(jr.Id)

	// Convert conditions
	for _, jc := range jr.Conditions {
		cond, err := ld.convertCondition(jc)
		if err != nil {
			return Rule{}, err
		}
		rb.AddCondition(cond)
	}

	// Convert outcome
	outcome, err := ld.convertOutcome(jr.OnMatch)
	if err != nil {
		return Rule{}, err
	}
	rb.SetOnMatch(outcome)

	return rb.Build(), nil
}

// convertCondition converts a JSON condition to domain model
func (ld *Loader) convertCondition(jc jsonCondition) (condition.Model, error) {
	builder := condition.NewBuilder().
		SetType(jc.Type).
		SetOperator(jc.Operator).
		SetValue(jc.Value)

	if jc.ReferenceId != "" {
		builder.SetReferenceId(jc.ReferenceId)
	}

	return builder.Build()
}

// convertOutcome converts a JSON outcome to domain model
func (ld *Loader) convertOutcome(jo jsonOutcome) (RuleOutcome, error) {
	ob := NewRuleOutcomeBuilder().SetAllow(jo.Allow)

	for _, jop := range jo.Operations {
		op, err := ld.convertOperation(jop)
		if err != nil {
			return RuleOutcome{}, err
		}
		ob.AddOperation(op)
	}

	return ob.Build(), nil
}

// convertOperation converts a JSON operation to domain model
func (ld *Loader) convertOperation(jo jsonOperation) (operation.Model, error) {
	builder := operation.NewBuilder().SetType(jo.Type)

	if jo.Params != nil {
		builder.SetParams(jo.Params)
	}

	return builder.Build()
}

// ClearCache clears the script cache
func (ld *Loader) ClearCache() {
	ld.cacheMu.Lock()
	ld.cache = make(map[string]PortalScript)
	ld.cacheMu.Unlock()
}

// Preload loads all scripts from the scripts directory into cache
func (ld *Loader) Preload() error {
	entries, err := os.ReadDir(ld.scriptsDir)
	if err != nil {
		return fmt.Errorf("failed to read scripts directory [%s]: %w", ld.scriptsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		portalId := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		if _, err := ld.LoadByPortalId(portalId); err != nil {
			ld.l.WithError(err).Warnf("Failed to preload script [%s]", portalId)
		}
	}

	ld.l.Infof("Preloaded %d portal scripts", len(ld.cache))
	return nil
}
