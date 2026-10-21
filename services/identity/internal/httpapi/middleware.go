package httpapi

import (
	"context"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"borrowhub/services/identity/internal/auth"
	"github.com/google/uuid"
)

type userContextKey uint8

const (
	userIDKey userContextKey = iota
	userEmailKey
)

func UserID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(userIDKey).(uuid.UUID)
	return val, ok
}

func UserEmail(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userEmailKey).(string)
	return val, ok
}

func AuthMiddleware(tokenManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication token", nil)
				return
			}

			claims, err := tokenManager.ValidateAccessToken(tokenStr)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token", nil)
				return
			}

			uid, err := uuid.Parse(claims.UserID)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user subject in token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, uid)
			ctx = context.WithValue(ctx, userEmailKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}

	return ""
}

func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   15 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   30 * 24 * 3600,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAuthCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

type ipBucket struct {
	tokens     float64
	lastRefill time.Time
}

const defaultMaxEntries = 10000

type RateLimiter struct {
	mu         sync.Mutex
	rate       float64 // tokens per second
	capacity   float64 // burst capacity
	maxEntries int     // maximum tracked IPs
	buckets    map[string]*ipBucket
	lastPrune  time.Time
}

// NewRateLimiter creates a token bucket rate limiter with specified rate per minute and burst capacity.
func NewRateLimiter(ratePerMinute float64, burst float64) *RateLimiter {
	return &RateLimiter{
		rate:       ratePerMinute / 60.0,
		capacity:   burst,
		maxEntries: defaultMaxEntries,
		buckets:    make(map[string]*ipBucket),
		lastPrune:  time.Now(),
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	allowed, _ := rl.AllowWithWait(ip)
	return allowed
}

func (rl *RateLimiter) AllowWithWait(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Periodic prune to prevent unbounded memory growth
	if now.Sub(rl.lastPrune) > 5*time.Minute {
		for k, b := range rl.buckets {
			if now.Sub(b.lastRefill) > 10*time.Minute {
				delete(rl.buckets, k)
			}
		}
		rl.lastPrune = now
	}

	b, ok := rl.buckets[ip]
	if !ok {
		if rl.maxEntries > 0 && len(rl.buckets) >= rl.maxEntries {
			var oldestKey string
			var oldestTime time.Time
			hasOldest := false

			for k, bucket := range rl.buckets {
				if now.Sub(bucket.lastRefill) > 5*time.Minute {
					delete(rl.buckets, k)
				} else if !hasOldest || bucket.lastRefill.Before(oldestTime) {
					oldestKey = k
					oldestTime = bucket.lastRefill
					hasOldest = true
				}
			}
			if len(rl.buckets) >= rl.maxEntries && hasOldest {
				delete(rl.buckets, oldestKey)
			}
		}

		rl.buckets[ip] = &ipBucket{
			tokens:     rl.capacity - 1.0,
			lastRefill: now,
		}
		return true, 0
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = b.tokens + elapsed*rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, 0
	}

	needed := 1.0 - b.tokens
	var waitSec float64
	if rl.rate > 0 {
		waitSec = needed / rl.rate
	} else {
		waitSec = 60
	}
	waitDur := time.Duration(math.Ceil(waitSec)) * time.Second
	if waitDur < time.Second {
		waitDur = time.Second
	}

	return false, waitDur
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		allowed, waitDur := rl.AllowWithWait(ip)
		if !allowed {
			waitSec := int(math.Ceil(waitDur.Seconds()))
			if waitSec < 1 {
				waitSec = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(waitSec))
			WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "rate limit exceeded", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
