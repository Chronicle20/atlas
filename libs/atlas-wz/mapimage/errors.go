package mapimage

import "errors"

// ErrSkipEmpty is returned when a map has no back[] and no layer content
// (cash shop / system maps). Caller should treat as a non-fatal skip.
var ErrSkipEmpty = errors.New("empty map")

// ErrSkipTooLarge is returned when a map's world bounds exceed MaxPixels.
// Caller should treat as a non-fatal skip and log.
var ErrSkipTooLarge = errors.New("map bounds exceed MaxPixels")

// ErrNoMinimap is returned when a Map.img has no miniMap/canvas property.
var ErrNoMinimap = errors.New("no minimap")
