# Third-source spike: manual YAML

Date: 2026-07-13

Task 7 uses a local YAML document as a realistic third source. It represents
both a synchronized external-style record (`source_system` plus stable
`external_id`) and Felicia-owned manual input. The adapter is intentionally
thin: [`internal/manualyaml`](../../internal/manualyaml) parses the document
into `domain.Observation` and `domain.MementoCandidate`; it does not write the
database or invent template behavior.

## Findings

| YAML concern                 | Canonical mapping                                | Result                                                     |
| ---------------------------- | ------------------------------------------------ | ---------------------------------------------------------- |
| stable record ID             | `SourceIdentity{system, external_id}`            | clean; idempotent through task 3                           |
| observed/occurred timestamps | `Observation.ObservedAt`, candidate `OccurredAt` | clean after RFC3339 parsing                                |
| confidence                   | observation and provenance confidence            | clean; bounded to 0–1                                      |
| point/line geometry          | candidate `Geom`                                 | clean for Point/LineString; domain validates ranges/anchor |
| kind-specific fields         | candidate `KindData`                             | clean when matching a repository template                  |
| missing author essay         | no candidate field                               | intentionally authored later                               |
| publication state            | not in source document                           | intentionally assigned by write side                       |
| media attachments/embeds     | not in this spike                                | remains a separate controlled media seam                   |

The canonical layer survived the third source without a generic mapping DSL or
provider types leaking into the domain. The practical boundary is clear:
source adapters normalize identity, timestamps, confidence, geometry, and
template-shaped `kind_data`; Felicia owns authored prose, media policy,
publication state, and persistence. This supports the task-1 decision to keep
adapters explicit until a third source demonstrates genuinely repeated mapping
rules.

## Example

```yaml
source_system: felicia-yaml
records:
  - external_id: ticket-1
    kind: goods
    occurred_at: 2026-03-20T10:00:00Z
    occurred_tz: Asia/Tokyo
    kind_data:
      name: 絵はがき
      price: { amount: 500, currency: JPY }
```
