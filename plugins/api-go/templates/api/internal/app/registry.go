package app

// Features wire in here and nowhere else. A feature is its own Go package, internal/<feature>
// (for example internal/widgets), with an exported Register(*http.ServeMux). Adding one appends
// exactly two lines: an import line just below this comment, and the Register line in the list.
//
//	import "webapp/internal/widgets"
//
//	widgets.Register,
//
// Write each import as its own one-line import statement (no block, no alias), so that two
// features added at once only add lines. Keep this file to the imports and the list: no
// declarations other than registrars, and no code of any kind, because the platform's rebase
// resolver unions concurrent appends to a registry file only when it has none. Append; never
// reorder, reflow or edit another feature's line.
//
// There is no init() registration and no generator; a feature that is not listed here is not
// served.

// one import line per feature, appended below

// registrars is the list of every feature's Register, mounted in this order by Routes
// (routes.go). Each feature appends one line inside the braces and nothing else.
// The comment lines here are padding as much as prose: a rebase merges two changes into one
// when fewer than four unchanged lines separate them, and that would tangle an import line
// with the list below. Keep these lines where they are.
var registrars = []registrar{
	// one line per feature, appended below
}
