## Open items

- [ ] **Release artefacts for v1.0.0.**
      - create a release commit for the current tree;
      - tag `v1.0.0` and push so `go get github.com/tommyo123/mtag@v1.0.0`
        resolves to the released code;
      - verify the pkg.go.dev render after the tag propagates.

- [ ] **Portability hardening.**
      - [x] preserve file mode bits on temp-file rewrite paths;
      - [x] preserve modified time on path-backed rewrite paths;
      - [x] preserve Linux xattrs on temp-file rewrite paths;
      - audit ACL handling on Unix-like systems where it is not exposed through xattrs;
      - [x] audit remaining size-to-`int` conversions for 32-bit builds and large-file paths;
      - [x] add CI coverage for `linux/amd64`, `linux/arm64`, and `darwin/arm64`;
      - [x] add one lightweight cross-platform test pass that avoids corpus-heavy sweeps.

## Recently completed

- [x] Read chapters from VorbisComment `CHAPTER###` fields in FLAC and Ogg.
- [x] Read chapters from WAV `cue ` + `LIST/adtl/labl`.
- [x] Read ReplayGain from MP3 LAME/Xing summary data with CRC validation.
- [x] Persist and remove trailing APEv2 on raw audio files that already carry it, including `TTA` and `AAC/ADTS`.
- [x] Read, preserve, and update MP4 `mdta` metadata beside classic `ilst`, with `ilst` taking precedence for polymorphic field reads.
- [x] Preserve non-file-level Matroska tag targets on save without flattening them into file-level `SimpleTag` values.
- [x] Add public chapter mutation APIs and native write paths for ID3v2, MP4, Matroska, VorbisComment chapter tags, and WAV cue/adtl markers.
- [x] Update MP3 LAME/Xing ReplayGain summary data on save when a writable summary block is present.
- [x] Add TAK detection, audio-property parsing, and APEv2 metadata reuse.
- [x] Add Sony Wave64 support with native summary metadata, audio properties, and streaming-safe rewrites.

