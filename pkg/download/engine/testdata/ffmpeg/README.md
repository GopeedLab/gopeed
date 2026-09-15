# Synthetic media fixtures

These files contain 0.3 seconds of a generated 32x32 black image (H.264) and
generated mono silence (AAC). They contain no third-party media. Generated
with the pinned go-ffmpreg FFmpeg binary using the equivalent commands:

```
ffmpeg -f lavfi -i color=c=black:s=32x32:r=10 -t 0.3 -c:v libx264 -preset ultrafast -movflags frag_keyframe+empty_moov -f mp4 video.mp4
ffmpeg -f lavfi -i anullsrc=r=48000:cl=mono -t 0.3 -c:a aac -movflags frag_keyframe+empty_moov -f mp4 audio.mp4
```

The backend tests also generate and probe media using the embedded FFmpeg and
FFprobe at test time, without relying on a system installation.

`video.webm` and `audio.webm` use the same black/silent sources, encoded once
with native FFmpeg 7.1 using `-c:v libvpx-vp9` and `-c:a libopus`. The committed
fixtures let the tests verify WASM stream-copy remuxing without exercising its
VP9 encoder or requiring native FFmpeg on developer/CI machines.
