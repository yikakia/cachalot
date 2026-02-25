package cache

type CallOptConfig struct {
	CustomField map[string]any
}

type CallOption func(*CallOptConfig)

func ApplyOptions(opts ...CallOption) *CallOptConfig {
	c := &CallOptConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithOptionCustomField(key string, value any) CallOption {
	return func(c *CallOptConfig) {
		c.SetCustomField(key, value)
	}
}

func (c *CallOptConfig) GetCustomField(key string) (any, bool) {
	if c.CustomField == nil {
		c.CustomField = make(map[string]any)
	}
	v, ok := c.CustomField[key]
	return v, ok
}

func (c *CallOptConfig) SetCustomField(key string, value any) {
	if c.CustomField == nil {
		c.CustomField = make(map[string]any)
	}
	c.CustomField[key] = value
}
