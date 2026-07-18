# Media support matrix

This is the implementation boundary exposed by the current local importer and
static publisher. “Media” is intentionally broader in the intake vocabulary,
but the public package currently has a photo-shaped attachment contract.

| Media class                  | Current local/package path                                | Public preview                     | Decision                             |
| ---------------------------- | --------------------------------------------------------- | ---------------------------------- | ------------------------------------ |
| JPEG / PNG / WebP image      | Supported as `memento_photos.object_key`                  | `<img>` gallery and stub fallback  | v0.1                                 |
| SVG image                    | Bytes can be copied, but not yet allowlisted or sanitized | Browser-dependent                  | Validate before enabling             |
| HEIC / RAW                   | Not decoded or transformed                                | Not supported                      | Convert through Immich preview JPEG  |
| Video                        | Not modeled; no duration/poster fields                    | Not supported                      | Add a separate attachment kind       |
| Audio                        | Not modeled; no duration/waveform fields                  | Not supported                      | Add a separate attachment kind       |
| PDF / document / ticket scan | Can be treated as bytes, but importer calls it a photo    | Not supported as a document viewer | Add download/document projection     |
| External URL / embed         | Not accepted by the package importer                      | Not supported                      | Add explicit trusted-provider policy |

The current demo deliberately exercises multiple image files and repeated use
of one image. Repeated assets must remain valid: attachment keys are identified
by memento plus sequence, not only by the URL/object key.

The v1 local authoring attachment shape now permits
`kind=image|video|audio|document|embed`, `mime`, `visibility=public|private`,
`caption`, and `path`. This is descriptive intake metadata, not a promise that
the current publisher can render every kind. The public package boundary accepts
only public local JPEG/PNG/WebP images; private attachments, unsupported kinds,
raw unsupported files, and external URLs are rejected rather than silently
published. A future `memento_media` model still needs `object_key`,
`content_hash`, `width`, `height`, `duration_ms`, `poster_key`, and `seq` for
non-image publication.
