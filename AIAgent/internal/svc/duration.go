package svc

import "time"

func durationMs(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
