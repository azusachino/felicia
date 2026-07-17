---
title: "OpenStreetMap demo data"
status: "verified"
date: "2026-07-17"
---

# OpenStreetMap demo data

The one-click preview uses a small raw journey package so a clean checkout has
real geographic content instead of an empty map. The authored journey and
memento story are Felicia fixture content; the geographic points below are
OpenStreetMap-derived facts.

| Place                        | OSM element        | Coordinates               | Retrieved  |
| ---------------------------- | ------------------ | ------------------------- | ---------- |
| Narita International Airport | relation `3182579` | `140.3933101, 35.7758714` | 2026-07-17 |
| Naritasan Shinshoji          | way `273746634`    | `140.3172238, 35.7863311` | 2026-07-17 |

The raw package keeps each OSM element reference in `kind_data.osm`, so the
provenance survives import and static compilation. The route GPX remains a
sample recorded track; it is not presented as an OSM-contributed personal
trace.

OSM data is provided under the Open Database License. Public maps and derived
data must show readable OpenStreetMap attribution; the public app already uses
OpenStreetMap attribution in its map styles. See the [OSM attribution
guidelines](https://osmfoundation.org/wiki/Licence/Attribution_Guidelines) and
[tile usage policy](https://operations.osmfoundation.org/policies/tiles/) before
switching the demo to live OSM tiles or geocoding.
