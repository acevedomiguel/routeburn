//go:build windows

package credentials

// PermissionsAreEnforced reports whether the OS exposes POSIX permission bits.
// Windows synthesises 0666 for every file, so a 0600 check there proves nothing.
const PermissionsAreEnforced = false
