package app

// registrars lists one registration per feature, one line each, in the order they mount. A
// feature owns a file pair, <feature>.go (handlers) and <feature>_routes.go (its
// register<Feature> registration); adding a feature adds exactly one line here and edits no
// other shared file.
//
// Keep this file to the list: no declarations other than registrars, and no code of any kind,
// because the platform's rebase resolver unions concurrent appends to a registry file only when
// the file has none. Append a line; never reorder, reflow or edit another feature's line.
//
// There is no init() registration and no generator; a feature that is not listed here is not
// served.
var registrars = []registrar{
	// one line per feature, appended below
}
