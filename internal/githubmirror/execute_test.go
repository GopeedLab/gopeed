package githubmirror

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const testSource = "https://github.com/owner/repo.git"

func testConfig() *Config {
	return &Config{Enabled: true, Mirrors: []Mirror{
		{Type: "ghProxy", URL: "https://first.example"},
		{Type: "ghProxy", URL: "https://second.example"},
	}}
}
func TestRaceWinsAndDrainsOwnedResults(t *testing.T) {
	config := testConfig()
	candidates := Candidates(config, testSource, Resource)
	var started sync.WaitGroup
	started.Add(len(candidates))
	discarded := []string{}
	winner, err := Race(context.Background(), config, testSource, Options[string]{
		Discard: func(value string) { discarded = append(discarded, value) },
	}, func(ctx context.Context, address string) (string, error) {
		started.Done()
		started.Wait()
		switch address {
		case candidates[0]:
			return address, errors.New("official offline")
		case candidates[1]:
			return address, nil
		default:
			<-ctx.Done()
			return address, ctx.Err()
		}
	})
	if err != nil || winner != candidates[1] {
		t.Fatalf("winner=%s err=%v", winner, err)
	}
	if len(discarded) != 2 || !contains(discarded, candidates[0]) || !contains(discarded, candidates[2]) {
		t.Fatalf("discarded %v", discarded)
	}
}
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
func TestRaceCleansSimultaneousSuccessfulLosers(t *testing.T) {
	config := testConfig()
	var started sync.WaitGroup
	started.Add(3)
	cleaned := []string{}
	winner, err := Race(context.Background(), config, testSource, Options[string]{Discard: func(value string) { cleaned = append(cleaned, value) }}, func(_ context.Context, address string) (string, error) {
		started.Done()
		started.Wait()
		return address, nil
	})
	if err != nil || winner == "" || len(cleaned) != 2 || contains(cleaned, winner) {
		t.Fatalf("winner=%s discarded=%v err=%v", winner, cleaned, err)
	}
}
func TestRaceParentCancellationCleansWinnerToo(t *testing.T) {
	config := testConfig()
	config.Mirrors = config.Mirrors[:1]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cleaned := []string{}
	winner, err := Race(ctx, config, testSource, Options[string]{Discard: func(value string) { cleaned = append(cleaned, value) }}, func(ctx context.Context, address string) (string, error) {
		if address == testSource {
			return address, nil
		}
		<-ctx.Done()
		cancel()
		return address, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) || winner != "" || len(cleaned) != 2 || !contains(cleaned, testSource) {
		t.Fatalf("winner=%s cleanup=%v err=%v", winner, cleaned, err)
	}
}
func TestRaceTimeoutAndErrorAggregation(t *testing.T) {
	expected := errors.New("offline")
	config := testConfig()
	config.Mirrors = config.Mirrors[:1]
	discarded := 0
	_, err := Race(context.Background(), config, testSource, Options[string]{Timeout: 10 * time.Millisecond, Discard: func(string) { discarded++ }}, func(ctx context.Context, address string) (string, error) {
		if address == testSource {
			return address, expected
		}
		<-ctx.Done()
		return address, ctx.Err()
	})
	if !errors.Is(err, expected) || !errors.Is(err, context.DeadlineExceeded) || discarded != 2 {
		t.Fatalf("discarded=%d err=%v", discarded, err)
	}
}
func TestFallbackPreservesOrderAndStopsOnSuccess(t *testing.T) {
	config := testConfig()
	candidates := Candidates(config, testSource, Repository)
	tried := []string{}
	cleaned := []string{}
	winner, err := Fallback(context.Background(), config, testSource, Options[string]{Kind: Repository, Discard: func(value string) { cleaned = append(cleaned, value) }}, func(_ context.Context, address string) (string, error) {
		tried = append(tried, address)
		if address == candidates[1] {
			return address, nil
		}
		return address, errors.New("offline")
	})
	if err != nil || winner != candidates[1] || !reflect.DeepEqual(tried, candidates[:2]) || !reflect.DeepEqual(cleaned, candidates[:1]) {
		t.Fatalf("winner=%s tried=%v discarded=%v err=%v", winner, tried, cleaned, err)
	}
	tried = nil
	_, err = Fallback(context.Background(), config, testSource, Options[string]{Kind: Repository}, func(_ context.Context, address string) (string, error) {
		tried = append(tried, address)
		return address, nil
	})
	if err != nil || !reflect.DeepEqual(tried, candidates[:1]) {
		t.Fatalf("tried=%v err=%v", tried, err)
	}
}
func TestFallbackTimeoutContinuesWithFreshContext(t *testing.T) {
	discarded := []string{}
	winner, err := Fallback(context.Background(), testConfig(), testSource, Options[string]{Kind: Repository, Timeout: 10 * time.Millisecond, Discard: func(value string) { discarded = append(discarded, value) }}, func(ctx context.Context, address string) (string, error) {
		if address == testSource {
			<-ctx.Done()
			return address, ctx.Err()
		}
		if err := ctx.Err(); err != nil {
			return address, err
		}
		return address, nil
	})
	if err != nil || !strings.HasPrefix(winner, "https://first.example/") || !reflect.DeepEqual(discarded, []string{testSource}) {
		t.Fatalf("winner=%s cleanup=%v err=%v", winner, discarded, err)
	}
}
func TestFallbackCancellationStopsBeforeNextMirror(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	cleaned := 0
	_, err := Fallback(ctx, testConfig(), testSource, Options[string]{Kind: Repository, Discard: func(string) { cleaned++ }}, func(ctx context.Context, address string) (string, error) {
		calls++
		cancel()
		return address, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) || calls != 1 || cleaned != 1 {
		t.Fatalf("calls=%d discarded=%d err=%v", calls, cleaned, err)
	}
}
func TestExecutorsDoNotStartInvalidOrCanceledRequests(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range []struct {
		ctx     context.Context
		config  *Config
		options Options[string]
		attempt Attempt[string]
	}{
		{ctx: canceled, options: Options[string]{}, attempt: func(context.Context, string) (string, error) { t.Fatal("called after cancellation"); return "", nil }},
		{ctx: context.Background(), options: Options[string]{Kind: 99}, attempt: func(context.Context, string) (string, error) { t.Fatal("invalid kind"); return "", nil }},
		{ctx: context.Background(), options: Options[string]{Timeout: -time.Second}, attempt: func(context.Context, string) (string, error) { t.Fatal("invalid timeout"); return "", nil }},
		{ctx: context.Background(), options: Options[string]{}, attempt: nil},
		{ctx: context.Background(), config: &Config{Mirrors: []Mirror{{Type: "invalid", URL: "https://mirror.example"}}}, attempt: func(context.Context, string) (string, error) { t.Fatal("invalid config"); return "", nil }},
	} {
		if _, err := Race(test.ctx, test.config, testSource, test.options, test.attempt); err == nil {
			t.Fatal("invalid race accepted")
		}
		if _, err := Fallback(test.ctx, test.config, testSource, test.options, test.attempt); err == nil {
			t.Fatal("invalid fallback accepted")
		}
	}
}
func TestAttemptPanicBecomesError(t *testing.T) {
	attempt := func(context.Context, string) (string, error) { panic("broken operation") }
	if _, err := Race(context.Background(), nil, testSource, Options[string]{}, attempt); err == nil || !strings.Contains(err.Error(), "broken operation") {
		t.Fatal(err)
	}
	if _, err := Fallback(context.Background(), nil, testSource, Options[string]{}, attempt); err == nil || !strings.Contains(err.Error(), "broken operation") {
		t.Fatal(err)
	}
}
func TestNonGithubResourcesAndDisabledMirrorsUseOnlyOriginal(t *testing.T) {
	for _, test := range []struct {
		config *Config
		raw    string
	}{{testConfig(), "https://example.com/resource"}, {&Config{Enabled: false, Mirrors: testConfig().Mirrors}, testSource}} {
		calls := 0
		got, err := Race(context.Background(), test.config, test.raw, Options[string]{}, func(_ context.Context, address string) (string, error) { calls++; return address, nil })
		if err != nil || got != test.raw || calls != 1 {
			t.Fatalf("got=%s calls=%d err=%v", got, calls, err)
		}
	}
}

func TestRepositoryAttemptsHaveNoDefaultDeadline(t *testing.T) {
	value, err := Fallback(context.Background(), nil, testSource, Options[string]{Kind: Repository}, func(ctx context.Context, raw string) (string, error) {
		if _, ok := ctx.Deadline(); ok {
			t.Fatal("repository attempt has a total deadline")
		}
		return raw, nil
	})
	if err != nil || value != testSource {
		t.Fatalf("value=%s error=%v", value, err)
	}
}
