package main

import (
	"sync"
	"time"
)

var (
	rateMu      sync.Mutex
	rateBuckets = map[string]time.Time{}
)

func checkRateLimit(userID, action string, minInterval time.Duration) error {
	rateMu.Lock()
	defer rateMu.Unlock()
	key := userID + "|" + action
	if last, ok := rateBuckets[key]; ok && time.Since(last) < minInterval {
		return rpcError("too many requests", 9)
	}
	rateBuckets[key] = time.Now()
	return nil
}
