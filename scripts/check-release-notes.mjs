#!/usr/bin/env node
// Guards the release pipeline against a dependency-resolution drift that
// breaks changelog rendering only at release time.
//
// `conventional-changelog-conventionalcommits@10` refuses to render against
// `conventional-changelog-writer@8`, and the error surfaces as an opaque
// "Missing helper" from handlebars. The combination only fails when the
// release runs, so this script renders notes from a fixed set of commits and
// fails fast in CI when the tooling cannot produce them.
import { generateNotes } from "@semantic-release/release-notes-generator";

const commits = [
  {
    hash: "0123456789abcdef0123456789abcdef01234567",
    message: "feat: add the dashboard\n\nA longer body.",
  },
  {
    hash: "89abcdef0123456789abcdef0123456789abcdef",
    message: "fix: correct the timezone",
  },
];

const notes = await generateNotes(
  { preset: "conventionalcommits" },
  {
    options: { repositoryUrl: "https://github.com/acevedomiguel/routeburn.git" },
    lastRelease: {},
    nextRelease: { version: "1.0.0", gitTag: "v1.0.0", gitHead: commits[0].hash },
    commits,
    logger: { log: () => {} },
    cwd: process.cwd(),
  },
);

const failures = [];
if (!notes.includes("### Features")) failures.push("the Features section is missing");
if (!notes.includes("### Bug Fixes")) failures.push("the Bug Fixes section is missing");
if (!notes.includes("add the dashboard")) failures.push("the feature summary is missing");
if (!notes.includes("correct the timezone")) failures.push("the fix summary is missing");
if (notes.includes("Missing helper")) failures.push("the writer guard fired");

if (failures.length > 0) {
  console.error("Release notes could not be rendered:");
  for (const failure of failures) console.error(`  - ${failure}`);
  console.error("\nRendered notes:\n");
  console.error(notes);
  process.exit(1);
}

console.log("Release notes render correctly.");
