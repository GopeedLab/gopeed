//go:build !windows || (!amd64 && !arm64)

package ffmpeg

import "context"

func withLinearMemory(ctx context.Context) (context.Context, func()) { return ctx, func() {} }
