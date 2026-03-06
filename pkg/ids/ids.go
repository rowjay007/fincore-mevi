package ids

import "github.com/google/uuid"

type ID string

func New() ID {
	return ID(uuid.NewString())
}

func (id ID) String() string { return string(id) }
