import { wrap, type WrapOptions } from "svelte-spa-router/wrap"
import JourneyList from "./views/JourneyList.svelte"
import JourneyDetail from "./views/JourneyDetail.svelte"
import MementoEditor from "./views/MementoEditor.svelte"
import SiteDeploy from "./views/SiteDeploy.svelte"
import RouteView from "./views/RouteView.svelte"
import type { Locale } from "./i18n"

export interface StudioState {
  readonly locale: Locale
  readonly desktop: boolean
}

export function createRoutes(studio: StudioState) {
  // The library passes URL parameters as `params`. This adapter forwards them
  // to the pages' existing props and keeps shared shell settings reactive.
  function page(component: NonNullable<WrapOptions["component"]>) {
    return wrap({
      component: RouteView,
      props: { component, studio },
      conditions: ({ params }) => !params || Object.values(params).every((value) => typeof value === "string" && value.length > 0),
    })
  }

  return {
    "/": page(JourneyList),
    "/journey/:id": page(JourneyDetail),
    "/journey/:journeyId/memento/:id": page(MementoEditor),
    "/site": page(SiteDeploy),
    "*": page(JourneyList),
  }
}
