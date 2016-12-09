package fastlimiter

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

//Options ...
type Options struct {
	Prefix        string
	Max           int32
	CleanDuration time.Duration
	Duration      time.Duration
}
type statusCacheItem struct {
	Index  int32
	Expire time.Time
}

// LimiterCacheItem of limiter
type limiterCacheItem struct {
	Total     int32
	Remaining int32
	Duration  time.Duration
	Expire    time.Time
}

//FastLimiter ...
type FastLimiter struct {
	lock    sync.RWMutex
	status  map[string]*statusCacheItem
	store   map[string]*limiterCacheItem
	ticker  *time.Ticker
	options *Options
}

// Result of limiter
type Result struct {
	Total     int
	Remaining int
	Duration  time.Duration
	Reset     time.Time
}

//New ...
func New(opts *Options) (limiter *FastLimiter) {
	limiter = &FastLimiter{
		options: opts,
	}
	if limiter.options.Duration == 0 {
		limiter.options.Duration = time.Minute
	}
	if limiter.options.Prefix == "" {
		limiter.options.Prefix = "limit:"
	}
	if limiter.options.Max == 0 {
		limiter.options.Max = 1000
	}
	limiter.store = make(map[string]*limiterCacheItem)
	limiter.status = make(map[string]*statusCacheItem)
	duration := 5 * time.Second
	if limiter.options.CleanDuration != 0 {
		duration = limiter.options.CleanDuration
	}
	limiter.ticker = time.NewTicker(duration)
	go limiter.cleanCache()
	return
}

//Get ...
func (l *FastLimiter) Get(id string, policy ...int32) (result Result, err error) {

	key := l.options.Prefix + id

	length := len(policy)
	if odd := length % 2; odd == 1 {
		return result, errors.New("fastlimiter: must be paired values")
	}
	if length == 0 {
		policy = []int32{l.options.Max, int32(l.options.Duration / time.Millisecond)}
	}
	return l.getResult(key, policy...)
}

//Remove ...
func (l *FastLimiter) Remove(id string) {
	key := l.options.Prefix + id
	statusKey := "{" + key + "}:S"

	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.store, key)
	delete(l.status, statusKey)
}

func (l *FastLimiter) getResult(id string, policy ...int32) (Result, error) {
	var result Result
	res := l.getLimit(id, policy...)

	remaining := atomic.LoadInt32(&res.Remaining)
	total := atomic.LoadInt32(&res.Total)
	result = Result{
		Remaining: int(remaining),
		Total:     int(total),
		Duration:  res.Duration,
		Reset:     res.Expire,
	}
	return result, nil
}
func (l *FastLimiter) getLimit(key string, args ...int32) (res *limiterCacheItem) {

	var ok bool
	if res, ok = l.getMapValue(key); ok {
		if res.Expire.Before(time.Now()) {
			res = l.initCacheItem(key, args...)
		} else {
			if atomic.LoadInt32(&res.Remaining) == -1 {
				return
			}
			atomic.AddInt32(&res.Remaining, -1)
		}
	} else {
		res = l.initCacheItem(key, args...)
	}
	return
}
func (l *FastLimiter) getMapValue(key string) (res *limiterCacheItem, ok bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok = l.store[key]
	return
}
func (l *FastLimiter) setMapValue(key string, res *limiterCacheItem) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.store[key] = res
}
func (l *FastLimiter) getStatusMapValue(key string) (res *statusCacheItem, ok bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok = l.status[key]
	return
}
func (l *FastLimiter) setStatusMapValue(key string, res *statusCacheItem) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.status[key] = res
}

func (l *FastLimiter) initCacheItem(key string, args ...int32) (res *limiterCacheItem) {

	policyCount := int32(len(args) / 2)
	statusKey := "{" + key + "}:S"
	total := args[0]
	duration := args[1]
	var statusItem *statusCacheItem

	if policyCount > 1 {
		var ok bool
		statusItem, ok = l.getStatusMapValue(statusKey)
		if !ok {
			l.setStatusMapValue(statusKey, &statusCacheItem{
				Index:  1,
				Expire: time.Now().Add(time.Duration(duration) * time.Millisecond * 2),
			})
		} else {
			index := atomic.LoadInt32(&statusItem.Index)
			if index >= policyCount {
				atomic.StoreInt32(&statusItem.Index, 1)
			} else {
				atomic.AddInt32(&statusItem.Index, 1)
			}
		}
	}
	if statusItem != nil {
		total = args[(statusItem.Index*2)-2]
		duration = args[(statusItem.Index*2)-1]
		l.setStatusMapValue(statusKey, &statusCacheItem{
			Index:  statusItem.Index,
			Expire: time.Now().Add(time.Duration(duration) * time.Millisecond * 2),
		})
	}
	res = &limiterCacheItem{
		Total:     total,
		Remaining: total - 1,
		Duration:  time.Duration(duration) * time.Millisecond,
		Expire:    time.Now().Add(time.Duration(duration) * time.Millisecond),
	}
	l.setMapValue(key, res)
	return res
}

//Count ...
func (l *FastLimiter) Count() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return len(l.store)
}

//CleanCache ...
func (l *FastLimiter) cleanCache() {
	for now := range l.ticker.C {
		var _ = now

	}
}

//Clean ...
func (l *FastLimiter) Clean() {
	for key, value := range l.store {
		if value.Expire.Before(time.Now()) {
			l.Remove(key)
		}
	}
}
