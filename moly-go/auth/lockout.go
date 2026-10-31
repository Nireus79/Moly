package auth

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// attemptGuard slows down guessing of passwords and recovery codes. After maxFails wrong answers in a row for one name the answers are refused for
// a while, and the wait doubles with every further round (up to maxLock). A right answer clears the count. Names that do not exist count too,
// so the refusal tells nothing about which names exist. The counts live in memory: a restart of the app clears them (the rate limiter in front of
// the routes still applies).
type attemptGuard struct {
	mu    sync.Mutex
	state map[string]*failState
	now   func() time.Time
}

type failState struct {
	fails       int
	lockedUntil time.Time
	rounds      int
}

const (
	maxFails = 5
	baseLock = 5 * time.Minute
	maxLock  = 30 * time.Minute
)

func newAttemptGuard() *attemptGuard {
	return &attemptGuard{state: map[string]*failState{}, now: time.Now}
}

func guardKey(kind, name string) string { return kind + ":" + strings.ToLower(strings.TrimSpace(name)) }

// wait returns how long the caller must wait before another try (0: may try).
func (g *attemptGuard) wait(key string) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	s := g.state[key]
	if s == nil {
		return 0
	}
	if d := s.lockedUntil.Sub(g.now()); d > 0 {
		return d
	}
	return 0
}

// fail records a wrong answer.
func (g *attemptGuard) fail(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.state) > 5000 { // names typed by anyone must not grow the map without end
		g.state = map[string]*failState{}
	}
	s := g.state[key]
	if s == nil {
		s = &failState{}
		g.state[key] = s
	}
	s.fails++
	if s.fails >= maxFails {
		lock := baseLock << s.rounds
		if lock > maxLock {
			lock = maxLock
		}
		s.lockedUntil = g.now().Add(lock)
		s.rounds++
		s.fails = 0
	}
}

// ok clears the count after a right answer.
func (g *attemptGuard) ok(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.state, key)
}

// lockedMessage is what the person is told.
func lockedMessage(d time.Duration) string {
	minutes := int((d + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	unit := "minutes"
	if minutes == 1 {
		unit = "minute"
	}
	return fmt.Sprintf("Too many wrong tries. Wait %d %s and try again.", minutes, unit)
}
