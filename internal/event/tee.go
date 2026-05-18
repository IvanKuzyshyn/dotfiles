package event

// Tee fans out events to multiple sinks.
func Tee(sinks ...Sink) Sink { return teeSink(sinks) }

type teeSink []Sink

func (t teeSink) Send(e Event) {
	for _, s := range t {
		s.Send(e)
	}
}
