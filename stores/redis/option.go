package redis

type Option func(*Store)

func WithStoreName(name string) Option {
	return func(s *Store) {
		s.name = name
	}
}
