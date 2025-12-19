
---
title: REST and JSON:API Pattern
description: Handler and transport conventions for JSON:API-compliant endpoints using server.RegisterHandler and api2go integration.
---

# REST and JSON:API Pattern

## Principles
- Use `server.RegisterHandler` and `server.RegisterInputHandler` for automatic tenant parsing, tracing, and JSON:API deserialization
- REST models implement JSON:API interface methods (`GetName()`, `GetID()`, `SetID()`)
- Handlers are thin - delegate business logic to processors
- Use `server.MarshalResponse` for success responses
- Map domain errors to HTTP status codes explicitly

---

## Resource File Structure


### Route Registration
Use `InitializeRoutes` to register all domain routes with the shared handler registration functions:

```go
func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			// CRUD endpoints
			router.HandleFunc("/users", server.RegisterHandler(l)(si)("get-users", listUsersHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/users", server.RegisterInputHandler[CreateRequest](l)(si)("create-user", createUserHandler(db))).Methods(http.MethodPost)
			router.HandleFunc("/users/{id}", server.RegisterHandler(l)(si)("get-user", getUserHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/users/{id}", server.RegisterInputHandler[UpdateRequest](l)(si)("update-user", updateUserHandler(db))).Methods(http.MethodPatch)
			router.HandleFunc("/users/{id}", server.RegisterHandler(l)(si)("delete-user", deleteUserHandler(db))).Methods(http.MethodDelete)

			// Relationship endpoints
			router.HandleFunc("/users/{id}/relationships/household", server.RegisterInputHandler[AssociateHouseholdRequest](l)(si)("associate-household", associateHouseholdHandler(db))).Methods(http.MethodPost)
		}

	}
}
```

**Key Points:**
- Return a curried function `func(db *gorm.DB) server.RouteInitializer` for dependency injection

- Use `server.RegisterHandler` for GET/DELETE (no request body)

- Use `server.RegisterInputHandler[T]` for POST/PATCH (with typed request model)
- Handler names (e.g., "get-users") are used for tracing and logging

---

## Handler Patterns


### GET Handler (No Request Body)
```go

func getUserHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return ParseId(d.Logger(), func(userId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				model, err := NewProcessor(d.Logger(), r.Context(), db).GetById(userId)()
				if err != nil {
					if errors.Is(err, ErrUserNotFound) {
						d.Logger().WithError(err).Error("User not found")
						w.WriteHeader(http.StatusNotFound)

						return
					}
					d.Logger().WithError(err).Error("Failed to fetch user")
					w.WriteHeader(http.StatusInternalServerError)
					return

				}

				res, err := ops.Map(Transform)(ops.FixedProvider(model))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(res)
			}
		})
	}
}
```

### POST/PATCH Handler (With Request Body)
```go
func createUserHandler(db *gorm.DB) server.InputHandler[CreateRequest] {
	return func(d *server.HandlerDependency, c *server.HandlerContext, req CreateRequest) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			input := CreateInput{
				Email:       req.Email,
				DisplayName: req.DisplayName,
				HouseholdId: req.HouseholdId,
			}

			model, err := NewProcessor(d.Logger(), r.Context(), db).Create(input)()
			if err != nil {
				if errors.Is(err, ErrEmailAlreadyExists) {
					d.Logger().WithError(err).Errorf("Email already exists.")

					w.WriteHeader(http.StatusConflict)

					return
				}
				if errors.Is(err, ErrEmailRequired) || errors.Is(err, ErrEmailInvalid) ||
					errors.Is(err, ErrDisplayNameRequired) || errors.Is(err, ErrDisplayNameEmpty) {
					d.Logger().WithError(err).Errorf("Validation failed.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				d.Logger().WithError(err).Errorf("Failed to create user.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := ops.Map(Transform)(ops.FixedProvider(model))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(res)
		}
	}
}

```

**Handler Dependency Benefits:**
- `d.Logger()` - Pre-configured logger with tenant and trace context
- `d.Context()` - Context with tenant information already parsed
- `c.ServerInformation()` - JSON:API server configuration

---

## REST Model Structure

### Response Models
Implement JSON:API interface methods for all response models:

```go
type RestModel struct {
	Id          uuid.UUID `json:"-"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	HouseholdId *string   `json:"household_id,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// GetName uses value receiver - required by api2go interface
func (r RestModel) GetName() string {
	return "users"
}

// GetID uses value receiver - read-only operation
func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID uses pointer receiver - mutates the model
func (r *RestModel) SetID(idStr string) error {

	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

```

**Critical Receiver Type Requirements:**
- `GetName()` **MUST** use value receiver `(r RestModel)` - required by api2go interface
- `GetID()` **SHOULD** use value receiver `(r RestModel)` - read-only operation
- `SetID()` **MUST** use pointer receiver `(r *RestModel)` - mutates the model

### Request Models
Request models also implement the JSON:API interface:

```go
type CreateRequest struct {
	Id          uuid.UUID  `json:"-"`

	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	HouseholdId *uuid.UUID `json:"household_id,omitempty"`
}


// GetName uses value receiver - required by api2go interface
func (r CreateRequest) GetName() string {
	return "users"
}


// GetID uses value receiver - read-only operation
func (r CreateRequest) GetID() string {
	return r.Id.String()
}


// SetID uses pointer receiver - mutates the model
func (r *CreateRequest) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}
```

**Key Points:**
- ID field tagged with `json:"-"` (set via SetID)
- Pointer fields for optional attributes (omitempty)
- Flat structure (no nested Data/Type/Attributes)
- `jsonapi.Unmarshal` handles JSON:API envelope automatically

---

## Transform Functions

Convert domain models to REST representations:


```go

func Transform(m Model) (RestModel, error) {

	var householdId *string
	if m.HouseholdId() != nil {
		hid := m.HouseholdId().String()
		householdId = &hid
	}


	return RestModel{

		Id:          m.Id(),
		Email:       m.Email(),
		DisplayName: m.DisplayName(),

		HouseholdId: householdId,
		CreatedAt:   m.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   m.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func TransformSlice(models []Model) ([]RestModel, error) {
	restModels := make([]RestModel, len(models))
	for i, model := range models {
		restModel, err := Transform(model)
		if err != nil {
			return nil, err
		}

		restModels[i] = restModel

	}
	return restModels, nil
}
```

---

## Error Handling


Map domain errors to HTTP status codes explicitly:

```go
if err != nil {
	// Specific domain errors
	if errors.Is(err, ErrUserNotFound) {
		d.Logger().WithError(err).Error("User not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrEmailAlreadyExists) {
		d.Logger().WithError(err).Error("Email already exists")
		w.WriteHeader(http.StatusConflict)
		return
	}
	if errors.Is(err, ErrEmailRequired) || errors.Is(err, ErrEmailInvalid) {
		d.Logger().WithError(err).Error("Validation failed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generic error
	d.Logger().WithError(err).Error("Internal error")
	w.WriteHeader(http.StatusInternalServerError)
	return
}
```

**Status Code Guidelines:**
- `400 Bad Request` - Validation errors, malformed input
- `404 Not Found` - Resource not found
- `409 Conflict` - Business rule violations (e.g., duplicate email)
- `500 Internal Server Error` - Unexpected errors

---

## Relationship Endpoints


For relationship endpoints (e.g., `/users/{id}/relationships/household`):

```go
type AssociateHouseholdRequest struct {
	Id uuid.UUID `json:"-"`
}

// GetName uses value receiver - required by api2go interface
func (r AssociateHouseholdRequest) GetName() string {
	return "households"  // Related resource type
}

// GetID uses value receiver - read-only operation
func (r AssociateHouseholdRequest) GetID() string {
	return r.Id.String()
}


// SetID uses pointer receiver - mutates the model
func (r *AssociateHouseholdRequest) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}
```


**Key Points:**
- `GetName()` returns the related resource type (e.g., "households")
- JSON:API request body contains the related resource ID

- Handler receives the parent resource ID from URL path


---

## Anti-Patterns to Avoid


❌ **Manual JSON encoding/decoding:**
```go
// DON'T DO THIS
var req struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`

	} `json:"data"`
}
json.NewDecoder(r.Body).Decode(&req)
```

✅ **Use server.RegisterInputHandler with flat request models:**
```go
// DO THIS
type CreateRequest struct {
	Id   uuid.UUID `json:"-"`
	Name string    `json:"name"`
}
// server.RegisterInputHandler automatically unmarshals JSON:API envelope
```

❌ **Manual tenant parsing:**
```go
// DON'T DO THIS
tenantID := r.Header.Get("X-Tenant-ID")
```

✅ **Use server.RegisterHandler which parses tenant automatically:**
```go
// DO THIS
// Tenant is available in d.Context() automatically
```


---

## Validation Guidelines


- Validate required fields in processor layer, not handlers
- Return typed domain errors (e.g., `ErrEmailRequired`)
- Map domain errors to HTTP status in handler
- Log errors with context using `d.Logger().WithError(err)`
