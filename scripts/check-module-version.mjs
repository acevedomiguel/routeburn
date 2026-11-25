#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

export function checkModuleVersion(modulePath, version) {
  const match = /^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.exec(version);
  if (!match) {
    throw new Error(`Invalid release version ${JSON.stringify(version)}; expected semantic versioning.`);
  }

  const suffix = /\/v(\d+)$/.exec(modulePath);
  const expectedMajor = suffix ? Number(suffix[1]) : 1;
  const releaseMajor = Number(match[1]);
  if (releaseMajor !== expectedMajor) {
    throw new Error(
      `Release v${version.replace(/^v/, "")} is incompatible with module ${modulePath}, which requires major v${expectedMajor}. ` +
      `A coordinated module-path change is required before any v${releaseMajor} tag.`,
    );
  }
}

function main() {
  const goMod = readFileSync(resolve(process.cwd(), "go.mod"), "utf8");
  const moduleMatch = /^module\s+(\S+)\s*$/m.exec(goMod);
  if (!moduleMatch) {
    throw new Error("Could not determine the module path from go.mod.");
  }
  const version = process.argv[2];
  checkModuleVersion(moduleMatch[1], version);
  console.log(`Module ${moduleMatch[1]} is compatible with release v${version.replace(/^v/, "")}.`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
