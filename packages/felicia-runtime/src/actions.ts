export const characterActions = ["idle", "travel", "arrive", "pickup", "carry", "place", "celebrate"] as const

export type CharacterAction = (typeof characterActions)[number]

export const sceneActions = ["depart", "reveal", "inspect", "resume", "return-home", "archive"] as const

export type SceneAction = (typeof sceneActions)[number]

export type SemanticAction = { kind: "character"; action: CharacterAction; targetId?: string } | { kind: "scene"; action: SceneAction; targetId?: string }
