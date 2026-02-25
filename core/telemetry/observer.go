package telemetry

type Observable struct {
	Metrics
	Logger
}

func DefaultObservable() *Observable {
	return &Observable{
		Metrics: NoopMetrics(),
		Logger:  SlogLogger(),
	}
}
