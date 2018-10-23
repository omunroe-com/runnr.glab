package helpers

import "time"

const (
	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1
)

// RoundDuration does exactly the same as time.Round in go.1.9+ since we are
// still on go1.8 we do not have this available. You can check the actual
// implementation in
// https://github.com/golang/go/blob/dev.boringcrypto.go1.9/src/time/time.go#L819-L841
// and the it can be found in go1.9 change log https://golang.org/doc/go1.9
func RoundDuration(d time.Duration, m time.Duration) time.Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if lessThanHalf(r, m) {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return minDuration // overflow
	}
	if lessThanHalf(r, m) {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return maxDuration // overflow
}

// lessThanHalf reports whether x+x < y but avoids overflow,
// assuming x and y are both positive (Duration is signed).
func lessThanHalf(x, y time.Duration) bool {
	return uint64(x)+uint64(x) < uint64(y)
}
