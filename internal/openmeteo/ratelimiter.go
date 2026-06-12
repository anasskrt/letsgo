package openmeteo

import "time"

// rateLimiter est un petit "token bucket" maison pour limiter le nombre
// d'appels par seconde. Une goroutine rajoute un jeton a intervalle
// regulier. wait() bloque tant qu'aucun jeton n'est dispo.
type rateLimiter struct {
	tokens chan struct{}
}

func newRateLimiter(perSec int) *rateLimiter {
	if perSec < 1 {
		perSec = 1
	}
	rl := &rateLimiter{tokens: make(chan struct{}, perSec)}
	// on remplit le seau au depart
	for i := 0; i < perSec; i++ {
		rl.tokens <- struct{}{}
	}
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(perSec))
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// seau plein, on ne fait rien
			}
		}
	}()
	return rl
}

func (rl *rateLimiter) wait() {
	<-rl.tokens
}
