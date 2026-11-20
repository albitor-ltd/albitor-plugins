// Starting point (test-playwright).

// Seeded records show up in the app's screens, and the walk and the screenshots capture them, so a
// name with a UUID in it reads as a defect in a delivered app. `displayName` makes a short,
// realistic one: the caller's label, two ordinary words and a four-digit number drawn from the
// scope's namespace. Isolation still comes from the ids the API returns and from `uniqueKey` (the long form,
// for a value that must be unique but is never drawn: a slug, an email local part, a code).
// Two words from 40 each and a number from 1000-9999 give 14.4 million names per label, so two
// scopes with the same label collide with probability about 1 in 14 million (not zero; tests that
// need certainty use uniqueKey). Within one scope the number steps with each call, so the first
// 9,000 calls never repeat. Cleanup deletes by id, never by name.
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

// "Claim Amber Court 4821": the label, two words and a number drawn from `namespace` (a UUID; its
// first 12 hex digits are used), the number stepping with the call `sequence` (1, 2, ...).
export function displayName(namespace: string, sequence: number, prefix = "item", maxLength = 80): string {
	const hex = namespace.replace(/[^0-9a-f]/gi, "");
	const part = (from: number) => parseInt(hex.slice(from, from + 4) || "0", 16);
	const a = WORDS_A[part(0) % WORDS_A.length];
	const b = WORDS_B[part(4) % WORDS_B.length];
	const number = 1000 + ((part(8) + sequence) % 9000);
	const tail = ` ${a} ${b} ${number}`;
	const label = capital((prefix || "item").trim() || "item");
	return `${label.slice(0, Math.max(1, maxLength - tail.length))}${tail}`;
}
