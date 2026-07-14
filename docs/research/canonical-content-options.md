# Research — canonical content and media options

> 2026-07-13. This note records community-established options considered for
> memento templates, rich content, external links, embeds, and observability.
> It informs ADR 0010 and does not add dependencies by itself.

## Templates and user input

The current repository-owned YAML registry is the right first implementation:
it keeps the five existing consumers close together (form, validation, stub,
i18n, and kind selection) and makes ticket-like mementos easy to author.

JSON Schema 2020-12 is the community-standard interoperability option for a
future user-template surface. It has broad tooling and explicit validation
vocabularies ([JSON Schema specification](https://json-schema.org/specification)).
The likely evolution is to keep a constrained Felicia template format as the
authoritative source and generate JSON Schema for clients, rather than allowing
arbitrary schemas to become an accidental UI/ETL language.

Open questions for user-created templates:

- repository-owned only, or signed/imported user templates later;
- template version stored with each memento;
- media slots and repeatable fields;
- custom field validation versus the closed primitive type catalog.

## Essays and embedded content

Markdown remains the smallest stable authored format. If the admin editor needs
structured blocks, two established families are worth a focused spike:

- **Portable Text** stores JSON blocks, spans, marks, and custom objects. Its
  custom blocks naturally represent images, video embeds, and other memento
  content ([Portable Text](https://www.portabletext.org/)).
- **ProseMirror/Tiptap JSON** uses a strict schema and serializes the editor
  document as JSON; it is powerful for rich editing and custom nodes
  ([ProseMirror guide](https://prosemirror.net/docs/guide/),
  [Tiptap concepts](https://tiptap.dev/docs/editor/core-concepts/introduction)).

Recommendation: retain Markdown plus attached `MediaAsset` values for the
first write slice. Do not invent a custom block schema until the authoring UX
requires it. If that need appears, Portable Text is the lighter content
interchange candidate; Tiptap/ProseMirror is the stronger editor candidate.

## Links and embeds

Treat a pasted URL as a link first. Resolve it to an embed only when the
provider is allowed and the provider response is safe. oEmbed is the natural
community protocol for this flow: it supports discovery, metadata, link,
photo, video, and rich responses ([oEmbed](https://oembed.com/)).

Canonical storage should contain:

```text
original_url, media_kind, provider, title, thumbnail_url, embed_url,
width, height, fetched_at
```

It should not store arbitrary provider HTML. Rendering should use an allowlist,
HTTPS-only URLs, sandboxed iframes where needed, and a restrictive CSP. Private
or unsupported URLs remain ordinary links.

## Go quality and observability

The community-favored interoperability path is OpenTelemetry for metrics and
traces; the Go documentation currently lists traces and metrics as stable and
logs as beta ([OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)).
Felicia should therefore keep `log/slog` as its application logging API and
design bounded metric names/labels so an OTel metrics exporter can be added
without changing domain code.

No telemetry SDK belongs in the pure domain package.
