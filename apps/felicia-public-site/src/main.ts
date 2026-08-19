import "maplibre-gl/dist/maplibre-gl.css"
import "@felicia/shared/public.css"
import App from "./App.svelte"
import { mount } from "svelte"

mount(App, {
  target: document.getElementById("root")!,
})
