---
description: Review codebase for TODOs, missing implementations, and unimplemented functions, then update docs/TODO.md
---

You are a codebase analyst performing a comprehensive review of the Atlas project to identify incomplete work.

## Instructions

### Phase 1: Discovery (Run in Parallel)

Launch three parallel exploration tasks:

1. **Find all TODO markers**
   - Search the entire codebase for: TODO, FIXME, XXX, HACK, and similar markers
   - For each finding, note: file path, line number, content, and surrounding context
   - Check all file types: Go, TypeScript, JavaScript, JSON, YAML, etc.

2. **Find unimplemented/stub code**
   - Search for patterns indicating incomplete implementations:
     - Functions returning nil/null without doing work
     - Empty function bodies or placeholder implementations
     - "not implemented" or "NotImplemented" strings/errors
     - Panic statements with "not implemented" messages
     - Functions that log warnings about missing implementation
     - Commented-out code blocks that might indicate incomplete work
   - Focus on Go files but also check TypeScript/JavaScript
   - Note the file, function name, and what appears to be missing

3. **Analyze project structure**
   - Identify the different domains/services in the codebase
   - Understand service directory structure and purposes
   - This enables organizing findings by domain

### Phase 2: Analysis

After discovery completes:

1. **Categorize findings by domain/service**
   - Group all TODOs and incomplete implementations by the service they belong to
   - Identify cross-cutting concerns that affect multiple services

2. **Prioritize findings**
   - **Critical**: Core gameplay functionality broken or missing
   - **High Priority**: Features incomplete but not blocking basic gameplay
   - **Medium Priority**: Quality/polish issues, performance optimizations
   - **Low Priority**: Minor cosmetic issues, documentation

3. **Identify patterns**
   - Note areas with concentrated incomplete work
   - Identify systemic issues vs one-off TODOs

### Phase 3: Update TODO.md

Read the existing `docs/TODO.md` file, then update it with the comprehensive findings.

**Structure for docs/TODO.md:**

```markdown
# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Item** - Brief description

### High Priority (Feature Incomplete)
- [ ] **Item** - Brief description

---

## Services

### Service Name
- [ ] Description of TODO (`file/path.go:123`)
- [ ] Description of TODO (`file/path.go:456`)

#### Subsection (if service has many related items)
- [ ] Item 1
- [ ] Item 2

[Continue for all services alphabetically]

---

## Libraries

### library-name
- [ ] Description (`file/path.go:123`)

---

## Notes

- Summary statistics
- Important context
```

**Guidelines for updating:**
- Preserve any manually-curated items that aren't from inline TODOs
- Include file path and line number for each inline TODO
- Use bold for critical/blocking items
- Group related items under subsections when a service has many items (e.g., "Character Attack System")
- Keep descriptions concise but informative
- Add summary statistics at the end (total TODOs, most concentrated areas)

### Output

After updating the file, provide a brief summary:
- Total number of TODOs/incomplete items found
- Top 3-5 most critical items
- Services with the most incomplete work
- Any new items added since the last review (if determinable)
