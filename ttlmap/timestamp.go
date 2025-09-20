package ttlmap

import "time"

type timestamp time.Duration

func (t timestamp) Before(o timestamp) bool {
	return t < o
}

func fromDuration(t time.Duration) timestamp {
	return timestamp(t)
}

func toDuration(t timestamp) time.Duration {
	return time.Duration(t)
}

func fromTime(t time.Time) timestamp {
	return timestamp(t.UnixNano())
}

func getNow() timestamp {
	return fromTime(time.Now())
}
