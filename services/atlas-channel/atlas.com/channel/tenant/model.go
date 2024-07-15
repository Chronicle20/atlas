package tenant

import "github.com/google/uuid"

type Model struct {
	Id           uuid.UUID `json:"id"`
	Region       string    `json:"region"`
	MajorVersion uint16    `json:"majorVersion"`
	MinorVersion uint16    `json:"minorVersion"`
}
