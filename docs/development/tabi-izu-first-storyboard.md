# Tabi first storyboard: Izu return-home trip

This is the first concrete storyboard for the Tabi memory diorama. It uses the
sanitized published fixture at
[`publication/journeys/izu-trip-2026-08-01/`](../../publication/journeys/izu-trip-2026-08-01/).
It is a design input, not an implementation or a new authored-data format.

## Source facts

The fixture contains:

- 101 GPX track points;
- origin `[139.7074, 35.8163]`;
- destination `[139.7126, 35.817]`;
- ten selected stop candidates, including seven derived-route candidates and
  one manually authored beach stop;
- three published mementos with media: Kujuppama Beach, Omuroyama Lift, and
  Jogasaki Coast.

The origin and destination are different coordinates in the same Tokyo area,
and the route contains the actual outbound and return travel. Tabi may therefore
play a real return-home leg for this fixture. It must preserve both endpoints
and must not replace the route with a straight line between them.

## First three-stop route

The first vertical slice follows only stops with published mementos. The other
selected candidates remain on the route but do not interrupt the first demo.

| Order | Stop            | Recorded time    | Memento                  | Event meaning                                                |
| ----- | --------------- | ---------------- | ------------------------ | ------------------------------------------------------------ |
| 1     | Kujuppama Beach | 2026-08-01 16:12 | souvenir + beach photo   | pause by the sea and collect a shared shoreline memory       |
| 2     | Omuroyama Lift  | 2026-08-02 11:43 | ticket + viewpoint photo | collect the lift ticket and inspect the mountain-view memory |
| 3     | Jogasaki Coast  | 2026-08-02 15:08 | souvenir + coast photo   | collect the coastal walk memory before the return journey    |

These are authored mementos, not automatically chosen “important” places. The
storyboard uses the existing `stop_key` association:

```text
manual:kujuppama-beach  -> Kujuppama Beach memento
derived-route:cluster-005 -> Omuroyama Lift ticket
derived-route:cluster-007 -> Jogasaki Coast memento
```

## Playback sequence

```text
HOME_READY
  -> DEPARTING
  -> TRAVELLING
  -> ARRIVING_AT_STOP(kujuppama-beach)
  -> REVEALING_MEMENTO(kujuppama-beach)
  -> INSPECTING_MEMORY(kujuppama-beach)
  -> RESUMING
  -> TRAVELLING
  -> ARRIVING_AT_STOP(omuroyama-lift)
  -> REVEALING_MEMENTO(omuroyama-lift)
  -> INSPECTING_MEMORY(omuroyama-lift)
  -> RESUMING
  -> TRAVELLING
  -> ARRIVING_AT_STOP(jogasaki-coast)
  -> REVEALING_MEMENTO(jogasaki-coast)
  -> INSPECTING_MEMORY(jogasaki-coast)
  -> RESUMING
  -> DESTINATION_REACHED
  -> RETURNING_HOME
  -> HOME_ARCHIVE
```

Travel between authored stops may be compressed into a board-friendly montage,
but the character's path must remain derived from the source route. The
overnight gap between the first and second stops is a good place to test a
chapter transition rather than trying to animate every recorded hour.

## Stop interaction grammar

Automatic travel pauses at each selected stop. The reader then has one clear
manual action:

1. The character arrives and the camera frames the stop.
2. The memento appears as a physical diorama object.
3. Selecting the object reveals its photo and short essay.
4. Selecting continue closes the memory and resumes travel.

The first slice does not add inventory management, scoring, optional side
quests, dice, combat, or branching route choices. Those would obscure whether
the route, stop association, and memento reveal are working.

## Contract mapping

The storyboard exercises the skeleton as follows:

- `JourneyScene.origin` and `.destination` preserve the GPX endpoints;
- `JourneyScene.route` supplies the character's travel path;
- `JourneyScene.stops` contains the three memento-bearing stops for the first
  authored sequence;
- `JourneyEvent.arrive`, `.reveal`, `.inspect`, `.resume`,
  `.destination-reached`, `.return-home`, and `.archive` describe the narrative;
- `SemanticAction` maps those events to character, artifact, camera, and board
  presentation without making those visuals part of the shared contract;
- `TabiCompositeModel` composes the board, stop markers, artifact, and archive.

## First-slice acceptance criteria

The first implementation is ready for review when a reader can:

- see the route begin at the real GPX origin;
- watch or advance through the three selected stops in recorded order;
- retrieve and inspect each published memento once;
- see the route reach the real GPX destination;
- see a return-home closure backed by the actual final route segment;
- open the completed home archive without changing the existing named themes.

The first implementation does not need to render all ten selected candidates,
all GPX timestamps, or a full 3D asset library.

## Review questions before implementation

- Is the first board framed as a single continuous tabletop or as three authored
  chapters connected by a folded route?
- Should the character carry the ticket/artifact between stops, or should each
  collected memory move directly into the home archive?
- Is the Tokyo endpoint best named “home,” or should the first screen use a
  neutral “start” label and reserve “home” for the archive state?
