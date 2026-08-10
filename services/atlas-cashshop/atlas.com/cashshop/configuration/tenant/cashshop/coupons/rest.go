package coupons

type RestModel struct {
	RateLimit RateLimitRestModel `json:"rateLimit"`
}

// RateLimitRestModel bounds coupon brute-forcing. Attempts is the number of
// FAILED redemption attempts one account may make inside WindowSeconds before
// further attempts short-circuit; WindowSeconds is the counter's TTL.
type RateLimitRestModel struct {
	Attempts      uint32 `json:"attempts"`
	WindowSeconds uint32 `json:"windowSeconds"`
}
