# libs/atlas-rest — gotchas

## JSON:API target structs MUST implement the relationship interfaces if the upstream response has any `relationships` block

`requests.GetRequest[T]` decodes responses with `api2go/jsonapi.Unmarshal`. If the upstream response includes a `relationships: {...}` block (most non-trivial atlas-data resources do), api2go walks every relationship and **errors out** unless the target struct implements `UnmarshalToOneRelations` and/or `UnmarshalToManyRelations` — even when the caller doesn't care about the relationship payload.

The error from api2go reads `"struct *YourModel does not implement UnmarshalToManyRelations"`. By the time it bubbles through the request wrapper and the calling client's error mapping, it usually surfaces as a generic "not found" or "lookup failed" — making the real cause invisible.

This already bit task-037 twice (atlas-configurations and atlas-character-factory each shipping equipment clients without the stubs).

### Required boilerplate for every JSON:API target struct

```go
type EquipmentRestModel struct {
    Id uint32 `json:"-"`
    // …attributes…
}

func (e EquipmentRestModel) GetName() string { return "statistics" } // jsonapi resource type
func (e EquipmentRestModel) GetID() string   { return strconv.Itoa(int(e.Id)) }
func (e *EquipmentRestModel) SetID(id string) error { /* parse */ return nil }

// Required even when you don't care about relationships:
func (e *EquipmentRestModel) SetToOneReferenceID(_, _ string) error             { return nil }
func (e *EquipmentRestModel) SetToManyReferenceIDs(_ string, _ []string) error  { return nil }
```

If you DO care about a relationship, populate the in-memory struct from the relationship name and IDs (see `services/atlas-npc-shops/atlas.com/npc/shops/rest.go` for an example that materializes commodities from a toMany relationship).

### How to be sure you got it right

Add an httptest-backed integration test for any new external client. Have the test serve a real fixture response (including the `relationships` block) and assert the decoded struct is non-zero. The `FakeClient` mocks under `mock/` packages bypass the unmarshal path and won't catch this.

### Why the wrapper doesn't fix it for you

`requests.GetRequest[T]` returns `(zero, err)` on any decode failure. The wrapper can't safely "ignore" the relationship error because the caller may genuinely need the relationship data. A future change could allow opt-out via a request option, but for now: implement the stubs.

## Distinguishing 404 from decode failure

`requests.ErrNotFound` is returned ONLY when the upstream returns HTTP 404. Any other error — connection refused, decode failure, 5xx, malformed JSON — is a different `error` value. Callers that map all errors to "resource doesn't exist" lose this distinction and hide deploy-time bugs as data bugs.

Pattern:

```go
if _, err := requestEquipmentById(id)(c.l, ctx); err != nil {
    if errors.Is(err, requests.ErrNotFound) {
        return ItemInfo{}, MyDomainNotFoundErr
    }
    c.l.WithError(err).Warnf("equipment lookup [%d] failed (non-404)", id)
    return ItemInfo{}, err
}
```
