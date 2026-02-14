package cache

import (
	"fmt"
)

var ErrNotFound = fmt.Errorf("item not exist")
var ErrTypeMissMatch = fmt.Errorf("type mismatch")
