package characterimage

import "errors"

var (
	ErrUnknownTemplateId = errors.New("characterimage: unknown templateId")
	ErrInvalidStance     = errors.New("characterimage: invalid stance")
	ErrFrameOutOfRange   = errors.New("characterimage: frame out of range")
	ErrAssetsMissing     = errors.New("characterimage: assets missing")
)
