Source: https://github.com/braheezy/shine-mp3, tag v0.2.0.

This local copy keeps the upstream license and applies one correction:

- MPEG-1 mono has 17 bytes of side information (plus a 4-byte header), not 9.
  The upstream constructor reserves 13 bytes total. With 48 kHz mono input,
  that emits 392-byte frames whose header advertises 384 bytes. Standard MP3
  decoders reject the stream. Reserve 21 bytes for MPEG-1 mono instead.

Aoikane uses EncodeBufferInterleaved with full channel-aware chunks instead of
the upstream Write helper, and adds a final silence frame to drain filter delay.
