package metrics

import (
	"sync/atomic"
)

var requestCount int64

func IncrRequest() { atomic.AddInt64(&requestCount, 1) }
func GetRequests() int64 { return atomic.LoadInt64(&requestCount) }
func ResetRequests() { atomic.StoreInt64(&requestCount, 0) }
