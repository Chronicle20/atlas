package request

import "github.com/google/uuid"

type Handler func(uuid.UUID, Reader)
