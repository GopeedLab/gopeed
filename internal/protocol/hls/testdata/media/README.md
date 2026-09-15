Media fixtures generated locally with FFmpeg 7.0.2 (testsrc + sine sources,
no third-party content):

- sample-00.ts / sample-01.ts: one continuous MPEG-2 TS encode split at a
  packet boundary, so continuity counters and PCR are consistent across the
  segment join exactly like segments cut from a single real-time stream.
- fmp4-init.mp4: ftyp+moov init section of a fragmented MP4.
- fmp4-seg1.m4s .. fmp4-seg3.m4s: moof+mdat media fragments (styp omitted by
  the muxer, which is legal) split at moof boundaries from one fragmented MP4
  (-movflags +frag_keyframe+empty_moov+default_base_moof -frag_duration 500000).
