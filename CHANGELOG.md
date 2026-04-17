# Changelog

## v1.0.0

First public release.

### Containers

Read and write support for:

- MP3, AAC / ADTS, AC-3, DTS, AMR / AMR-WB
- FLAC, Ogg Vorbis / Opus / Speex / FLAC-in-Ogg
- MP4 / M4A / M4B (including Nero `chpl` and native QuickTime chapters,
  `ilst` items, freeform `----:` atoms, and `mdta`)
- WAV / RIFF / RF64 / BW64, AIFF / AIFC, Wave64 (`.w64`)
- Monkey's Audio (`.ape`), WavPack (`.wv`), Musepack (`.mpc`),
  TrueAudio (`.tta`), TAK (`.tak`)
- DSF (`.dsf`), DSDIFF / DFF (`.dff`)
- ASF / WMA, Matroska / WebM (`.mka`, `.mkv`, `.webm`)
- RealAudio / RealMedia (`.ra`, `.rm`)
- Tracker modules (`.mod`, `.s3m`, `.xm`, `.it`)
- Core Audio Format (`.caf`), OpenMG / ATRAC (`.oma`, `.aa3`, `.at3`)

### Polymorphic API

- Text fields: `Title`, `Artist`, `Album`, `AlbumArtist`, `Composer`,
  `Copyright`, `Publisher`, `EncodedBy`, `Comment`, `Lyrics`.
- Numbering and dates: `Track`, `TrackTotal`, `Disc`, `DiscTotal`,
  `Year`, `RecordingTime`.
- Images: `Images`, `ImageSummaries`, `AddImage`, `SetCoverArt`,
  `RemoveImages`.
- Chapters: `Chapters`, `SetChapters`, `RemoveChapters`.
- Custom fields: `CustomValue`, `CustomValues`, `SetCustomValues`,
  `RemoveCustom`.
- ReplayGain: `ReplayGainTrack` / `ReplayGainAlbum` (with MP3
  LAME/Xing summary fallback).
- MusicBrainz: `MusicBrainzID` / `SetMusicBrainzID` across recording,
  track, release, release-group, artist, album-artist, work, and
  release-type identifiers.
- Rating and play count: `Rating`, `RatingEmail`, `SetRating`,
  `PlayCount`, `SetPlayCount`, with POPM / PCNT synchronisation.
- Compilation, podcast flags, and podcast text (`TCAT`, `TDES`,
  `TGID`, `WFED`).
- iTunes `iTunNORM` parsing.
- TIPL / IPLS / TMCL credits through `InvolvedPeople` and
  `MusicianCredits`.

### Native stores and container extras

- `ID3v1()`, `ID3v2()`, `APEv2()`, polymorphic `Tag(kind)` / `Tags()`.
- Matroska attachments (`Attachments`, `AttachmentSummaries`).
- FLAC cue sheet (`CueSheet`).
- Broadcast Wave (`BroadcastWave`) with `bext` / `iXML` / `aXML` /
  `cart` / `UMID` read and write.
- OpenMG / ATRAC helpers (`OpenMG`, `SetOpenMGTrack`, etc.).
- Subpackages `id3v1`, `id3v2`, `flac`, `mp4`, `ape` expose the raw
  types and encoders.

### Save paths

- In-place patching when the on-disk layout permits it.
- Sibling temp-file rewrite with atomic rename for path-backed opens.
- Stream-back through `WriteAt` / `Truncate` for writable sources.
- Preserves file mode, modified time, and (on Linux) extended
  attributes across rewrites.
- `SaveFormats(Format)` / `SaveWith(Format)` for exact ID3 layout
  control.
- `SaveContext(ctx)` for cancellable long rewrites.

### Open options

- `WithReadOnly`, `WithSkipV1`, `WithSkipV2`, `WithSkipPictures`,
  `WithSkipAttachments`, `WithLeadingJunkScan`, `WithSyncV1toV2`,
  `WithCreateV2OnV1Only`, `WithGenreSyncStrategy`, `WithPaddingBudget`,
  `WithMaxFileSize`, `WithAudioPropertiesStyle`.

### Quality

- CI on `linux/amd64`, `linux/arm64`, `darwin/arm64`,
  `windows/amd64`, plus cross-build smoke tests for `linux/386`,
  `linux/arm`, and `darwin/amd64`.
- Fuzz tests for `id3v1`, `id3v2`, `flac`, `ape`, and `mp4` parsers.
- Portable smoke test suite that uses fixtures under
  `tests/testdata` and requires no external corpus or tools.
