package githubmirror

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Options configures execution, not URL persistence. Discard releases results
// from failed attempts and successful race losers, including partial results
// returned together with an error. The caller owns the winning result.
type Options[T any] struct {
	Kind Kind
	// Timeout limits each attempt. Zero uses 30 seconds for resources and no
	// overall deadline for repositories, whose transport owns idle timeouts.
	Timeout time.Duration
	Discard func(T)
}

// Attempt must honor ctx and return a completed, validated result. Its context
// is canceled after it returns, so live response streams must not be returned.
// Return any partially acquired resources along with errors so Discard can
// release them. Failed attempts must not mutate persistent business state.
type Attempt[T any] func(context.Context, string) (T, error)

const defaultTimeout = 30 * time.Second

func prepare[T any](ctx context.Context, config *Config, raw string, options Options[T], attempt Attempt[T]) ([]string, time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if raw == "" {
		return nil, 0, errors.New("empty GitHub source URL")
	}
	if options.Kind != Resource && options.Kind != Repository {
		return nil, 0, errors.New("invalid GitHub resource kind")
	}
	if options.Timeout < 0 {
		return nil, 0, errors.New("invalid GitHub attempt timeout")
	}
	if attempt == nil {
		return nil, 0, errors.New("GitHub mirror attempt is required")
	}
	if config != nil {
		if err := config.Validate(); err != nil {
			return nil, 0, err
		}
	}
	timeout := options.Timeout
	if timeout == 0 && options.Kind == Resource {
		timeout = defaultTimeout
	}
	return Candidates(config, raw, options.Kind), timeout, nil
}
func invoke[T any](ctx context.Context, timeout time.Duration, candidate string, attempt Attempt[T]) (value T, err error) {
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("GitHub mirror attempt panicked: %v", recovered)
		}
	}()
	value, err = attempt(ctx, candidate)
	if err == nil {
		err = ctx.Err()
	}
	return
}
func discard[T any](options Options[T], value T) {
	if options.Discard != nil {
		options.Discard(value)
	}
}

// Race executes the official URL and eligible mirrors concurrently. The first
// completed successful result wins; losing attempts are canceled and drained
// before return so no temporary files or other resources outlive the call.
// HTTP status/content validation is the attempt's responsibility.
func Race[T any](ctx context.Context, config *Config, raw string, options Options[T], attempt Attempt[T]) (T, error) {
	var zero T
	candidates, timeout, err := prepare(ctx, config, raw, options, attempt)
	if err != nil {
		return zero, err
	}
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type outcome struct {
		index int
		value T
		err   error
	}
	outcomes := make(chan outcome, len(candidates))
	for index, candidate := range candidates {
		go func(index int, candidate string) {
			value, err := invoke(raceCtx, timeout, candidate, attempt)
			outcomes <- outcome{index, value, err}
		}(index, candidate)
	}
	var winner T
	won := false
	failures := make([]error, len(candidates))
	for range candidates {
		outcome := <-outcomes
		if !won && outcome.err == nil && ctx.Err() == nil {
			winner = outcome.value
			won = true
			cancel()
		} else {
			discard(options, outcome.value)
			if outcome.err != nil {
				failures[outcome.index] = fmt.Errorf("GitHub source %d: %w", outcome.index+1, outcome.err)
			}
			if ctx.Err() != nil {
				cancel()
			}
		}
	}
	if err := ctx.Err(); err != nil {
		if won {
			discard(options, winner)
		}
		return zero, err
	}
	if won {
		return winner, nil
	}
	return zero, fmt.Errorf("all GitHub sources failed: %w", errors.Join(failures...))
}

// Fallback executes the official URL first, then eligible mirrors in their
// configured order. It stops immediately on success and cleans every failure.
func Fallback[T any](ctx context.Context, config *Config, raw string, options Options[T], attempt Attempt[T]) (T, error) {
	var zero T
	candidates, timeout, err := prepare(ctx, config, raw, options, attempt)
	if err != nil {
		return zero, err
	}
	failures := make([]error, 0, len(candidates))
	for index, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		value, err := invoke(ctx, timeout, candidate, attempt)
		if err == nil && ctx.Err() == nil {
			return value, nil
		}
		discard(options, value)
		if parentErr := ctx.Err(); parentErr != nil {
			return zero, parentErr
		}
		failures = append(failures, fmt.Errorf("GitHub source %d: %w", index+1, err))
	}
	return zero, fmt.Errorf("all GitHub sources failed: %w", errors.Join(failures...))
}
