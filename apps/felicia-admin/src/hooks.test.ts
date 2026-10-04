import { expect, test } from "vitest"
import { reroute } from "./hooks"

test("malformed hash encoding recovers before framework route decoding", async () => {
  for (const hash of ["#/journey/%ZZ", "#/journey/a/memento/%E0%A4"]) {
    expect(await reroute({ url: new URL(`http://localhost/${hash}`), fetch })).toBe("/")
  }
})

test("valid encoded IDs and query strings stay under framework routing", async () => {
  for (const hash of ["", "#/", "#/journey/%61bc/?tab=overview", "#/journey/a%2Fb/memento/c%20d"]) {
    expect(await reroute({ url: new URL(`http://localhost/${hash}`), fetch })).toBeUndefined()
  }
})
