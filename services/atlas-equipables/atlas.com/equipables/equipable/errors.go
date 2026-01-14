package equipable

import "errors"

var (
	// ErrEquipableNotFound is returned when an equipable cannot be found by ID
	ErrEquipableNotFound = errors.New("equipable not found")

	// ErrInvalidItemId is returned when an invalid item ID is provided
	ErrInvalidItemId = errors.New("invalid item ID")

	// ErrTemplateNotFound is returned when the equipable template cannot be found
	ErrTemplateNotFound = errors.New("equipable template not found")

	// ErrCreateFailed is returned when equipable creation fails
	ErrCreateFailed = errors.New("failed to create equipable")

	// ErrUpdateFailed is returned when equipable update fails
	ErrUpdateFailed = errors.New("failed to update equipable")

	// ErrDeleteFailed is returned when equipable deletion fails
	ErrDeleteFailed = errors.New("failed to delete equipable")
)
