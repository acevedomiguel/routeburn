import assert from "node:assert/strict";
import test from "node:test";

import { checkModuleVersion } from "./check-module-version.mjs";

const modulePath = "github.com/acevedomiguel/routeburn";

test("accepts v1 releases for the unsuffixed module path", () => {
  assert.doesNotThrow(() => checkModuleVersion(modulePath, "1.5.1"));
  assert.doesNotThrow(() => checkModuleVersion(modulePath, "1.6.0"));
});

test("rejects a v2 release before the module path changes", () => {
  assert.throws(
    () => checkModuleVersion(modulePath, "2.0.0"),
    /coordinated module-path change.*before any v2 tag/,
  );
});

test("requires the release major to match an explicit module suffix", () => {
  assert.doesNotThrow(() => checkModuleVersion(`${modulePath}/v2`, "2.0.0"));
  assert.throws(
    () => checkModuleVersion(`${modulePath}/v2`, "1.9.0"),
    /coordinated module-path change/,
  );
});

test("rejects malformed release versions", () => {
  for (const version of ["", "1", "1.2", "not-a-version"]) {
    assert.throws(
      () => checkModuleVersion(modulePath, version),
      /Invalid release version/,
    );
  }
});
