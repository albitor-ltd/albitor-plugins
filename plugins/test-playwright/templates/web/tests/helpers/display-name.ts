// Starting point (test-playwright).

// Seeded records show up in the app's screens, and the walk and the screenshots capture them, so a
// name with a UUID in it reads as a defect in a delivered app. `displayName` makes a short,
// realistic one: the caller's label, then two ordinary words that identify the scope, then a small
// counter. Isolation still comes from the ids the API returns and from `uniqueKey` (the long form,
// for a value that must be unique but is never drawn: a slug, an email local part, a code).
// Two words from 40 each give 1,600 scopes per label; a collision needs two parallel scopes to draw
// the same pair and the same counter, and cleanup deletes by id, never by name.
const WORDS_A = [
	"Amber", "Birch", "Cedar", "Coral", "Dune", "Elm", "Fern", "Frost", "Garnet", "Hazel",
	"Indigo", "Ivory", "Jade", "Juniper", "Kestrel", "Linen", "Maple", "Marble", "Nectar", "Olive",
	"Opal", "Pebble", "Quartz", "Willow", "Rowan", "Saffron", "Slate", "Sorrel", "Thistle", "Umber",
	"Violet", "Walnut", "Wren", "Yarrow", "Zinnia", "Copper", "Meadow", "Harbour", "Summit", "Orchard",
] as const;
const WORDS_B = [
	"Lane", "Court", "Row", "Gate", "Mill", "Bridge", "Green", "Hill", "Wharf", "Yard",
	"Field", "Brook", "Cross", "Park", "Rise", "Walk", "Close", "Mews", "Quay", "Terrace",
	"Grove", "Fold", "Reach", "Vale", "Cove", "Ridge", "Steps", "Place", "Square", "Common",
	"Heath", "Wood", "Bank", "Dale", "Holt", "Ford", "Moor", "Sands", "Point", "Chase",
] as const;

// First letter upper-case; the rest as the caller wrote it.
function capital(text: string): string {
	return text.charAt(0).toUpperCase() + text.slice(1);
}

// "Claim Amber Court 3": the label, two words drawn from `namespace` (a UUID), the call counter.
export function displayName(namespace: string, sequence: number, prefix = "item", maxLength = 80): string {
	const hex = namespace.replace(/[^0-9a-f]/gi, "");
	const a = WORDS_A[parseInt(hex.slice(0, 4) || "0", 16) % WORDS_A.length];
	const b = WORDS_B[parseInt(hex.slice(4, 8) || "0", 16) % WORDS_B.length];
	const tail = ` ${a} ${b} ${sequence}`;
	const label = capital((prefix || "item").trim() || "item");
	return `${label.slice(0, Math.max(1, maxLength - tail.length))}${tail}`;
}
