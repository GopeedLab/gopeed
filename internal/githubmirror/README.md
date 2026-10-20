# GitHub mirror execution

This package owns GitHub URL rewriting and execution strategies. It depends only
on the standard library and does not depend on Downloader, HTTP clients, Git, or
storage. Downloader persists `Config` at its top-level `githubMirror` field;
`pkg/base` preserves the existing configuration names through aliases.

## Race: resource reads

```go
body, err := githubmirror.Race(ctx, cfg.GithubMirror, rawURL,
    githubmirror.Options[[]byte]{
        Kind:    githubmirror.Resource,
        Timeout: 30 * time.Second,
    },
    func(ctx context.Context, candidate string) ([]byte, error) {
        return fetchAndValidate(ctx, client, candidate)
    },
)
```

The callback owns HTTP requests, size limits, and content validation. The first
completed valid result wins. Other operations are canceled and drained. There is
no HEAD-probe assumption or client-specific HTTP behavior inside the executor.

## Fallback: repository operations

```go
repo, err := githubmirror.Fallback(ctx, cfg.GithubMirror, officialURL,
    githubmirror.Options[*git.Repository]{
        Kind: githubmirror.Repository,
    },
    func(ctx context.Context, candidate string) (*git.Repository, error) {
        // Use a clean temporary checkout for each attempt.
        return clone(ctx, candidate)
    },
)
```

The official URL is attempted first; mirrors follow their configured order.
jsDelivr is excluded for repositories. Stop on the first success. Perform
persistent installation or other business state changes after success, rather
than inside a potentially repeated attempt.

## Results with resources

For temporary files, set `Options[*Download].Discard` to remove the file. Return
the partial `*Download` alongside errors, so cleanup also covers failed reads or
validation. The executor discards every failure and every race loser exactly
once. The caller owns and must track/release the winning result.

- `Timeout == 0` uses 30 seconds per resource attempt and no overall deadline
  for repository attempts; a negative timeout is invalid. Extension Git HTTP
  transport limits connections/TLS handshakes to 10 seconds and resets a 60-second
  read deadline as data arrives. Slow continuous transfers can finish, while
  stalled sources fail and allow ordered fallback. Parent cancellation still applies.
- Attempts must honor the supplied context; draining waits for them to return.
- Attempt contexts are canceled after return, so callbacks must finish reading
  HTTP bodies before returning. Returning a still-streaming response is unsupported.
- Discard callbacks must accept zero values, must not panic, and run serially.
- Parent cancellation wins over a result selected while the parent is canceled;
  the selected result is then discarded as well.
- Errors preserve their causes through wrapping/joining for `errors.Is`.
- `github.com` and `raw.githubusercontent.com` expand into eligible mirrors.
  jsDelivr is only eligible for raw source resources. `api.github.com` is never
  mirrored and is rejected by the public mirror candidates API.
- Client application update checks request the official GitHub API directly,
  falling back to the independent Gopeed website API on request or validation
  failure. Only package downloads obtain URLs from
  `POST /api/v1/mirrors/github/candidates`. Candidate generation never performs
  HTTP requests.
- Disabled mirrors and other URLs execute only the original URL.
- Config is a caller-owned immutable snapshot for the duration of a call.
