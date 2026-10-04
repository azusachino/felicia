export { cn } from "cn"

export type WithElementRef<T, E extends HTMLElement = HTMLElement> = T & { ref?: E | null }
export type WithoutChildren<T> = Omit<T, "children">
export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">
