package queue

type OptionFunc func(q Queue) error

func WithMetrics(m Metrics) OptionFunc {
	return func(t Queue) error {
		t.SetMetrics(m)
		return nil
	}
}
