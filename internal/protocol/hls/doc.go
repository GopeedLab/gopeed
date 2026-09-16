// Package hls implements native HTTP Live Streaming downloads for m3u8
// playlists. A task produces exactly one merged, playable output file:
// MPEG-2 TS segments are concatenated as-is, fMP4 streams are merged as
// init section followed by the media fragments, and AES-128 encrypted
// segments are decrypted before they are stored.
//
// Supported subset:
//   - VOD media playlists (EXT-X-ENDLIST present) and master playlists whose
//     selected variant carries its own audio
//   - AES-128 with the identity KEYFORMAT, explicit IV or media-sequence
//     derived IV, including key rotation mid-playlist
//   - EXT-X-BYTERANGE (including offset inheritance) and fMP4 init sections
//     via a single, constant EXT-X-MAP
//
// Everything the merge pipeline cannot reproduce faithfully is rejected with
// an explicit error instead of silently producing a broken output file:
//   - live streams (no EXT-X-ENDLIST) and SAMPLE-AES / DRM encryption
//   - EXT-X-GAP (would leave a hole) and EXT-X-DISCONTINUITY (broken timeline)
//   - changing init sections (a repeated EXT-X-MAP is compared by all its
//     attributes, not just the URI) and encrypted EXT-X-MAP without an
//     explicit IV
//   - separate audio renditions referenced by the selected variant (the
//     merged output would have no audio) and non-identity KEYFORMATs
package hls
