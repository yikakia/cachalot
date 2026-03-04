package cache

import (
	"fmt"
)

var ErrNotFound = fmt.Errorf("item not exist")
var ErrTypeMismatch = fmt.Errorf("type mismatch")
var ErrInvalidTTL = fmt.Errorf("invalid ttl")
