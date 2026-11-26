package main

import "runtime/debug"

// engineModule is the shared provider engine this binary embeds.
const engineModule = "github.com/acevedomiguel/burnblocks"

// engineVersion reports the Burn Blocks version this binary was built against.
// RouteBurn is the interface; the engine behind it lives in another module, so a
// support report needs to name which one the binary carries. It returns
// "unknown" rather than an empty string when the binary has no module
// information, so the line is always renderable.
func engineVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path != engineModule {
			continue
		}
		version := dep.Version
		if dep.Replace != nil && dep.Replace.Version != "" {
			version = dep.Replace.Version
		}
		if version == "" {
			return "unknown"
		}
		return version
	}
	return "unknown"
}
