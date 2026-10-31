// Starting point (test-playwright). The pure half of auth@2: which roles an app declares, and
// where a role's deployed account and password come from. No Playwright here, so it is unit tested.
import { readFileSync } from "node:fs";
import { join } from "node:path";

// A role is a group of services.auth.groups, so it is a group name as the user pool takes it.
export const ROLE_PATTERN = /^[a-z][a-z0-9-]{1,31}$/;

const unquote = (s: string) => s.trim().replace(/^(['"])(.*)\1$/, "$2");

/**
 * The roles an albitor.yaml declares: services.auth.groups, in order, each a valid group name.
 * Reads the YAML subset the manifest uses (block or flow list under services > auth > groups); an
 * app with no groups, or no auth, has no roles. Anything that is not a group name is dropped, so a
 * role is always safe to put in a username, an environment variable name or a path.
 */
export function parseDeclaredRoles(yaml: string): string[] {
  const stack: { indent: number; key: string }[] = [];
  const found: string[] = [];
  let listIndent = -1; // the indent of a `groups:` key whose block list is being read
  for (const raw of yaml.split(/\r?\n/)) {
    const line = raw.replace(/\s+#.*$/, "");
    const text = line.trimStart();
    if (text === "" || text.startsWith("#") || text === "---" || text === "...") continue;
    const indent = line.length - text.length;
    if (listIndent >= 0) {
      if (text === "-" || text.startsWith("- ")) {
        found.push(unquote(text.slice(1)));
        continue;
      }
      listIndent = -1;
    }
    const m = /^("[^"]*"|'[^']*'|[A-Za-z0-9_-]+):(\s|$)/.exec(text);
    if (!m) continue;
    while (stack.length > 0 && stack[stack.length - 1].indent >= indent) stack.pop();
    stack.push({ indent, key: unquote(m[1]) });
    if (stack.map((s) => s.key).join(".") !== "services.auth.groups") continue;
    const rest = text.slice(m[0].length).trim();
    if (rest === "") {
      listIndent = indent;
    } else if (rest.startsWith("[") && rest.endsWith("]")) {
      for (const item of rest.slice(1, -1).split(",")) found.push(unquote(item));
    }
  }
  return [...new Set(found.filter((r) => ROLE_PATTERN.test(r)))];
}

/** The roles of the app at `root` (the repository root, above web/), none when it has no albitor.yaml. */
export function readDeclaredRoles(root: string): string[] {
  try {
    return parseDeclaredRoles(readFileSync(join(root, "albitor.yaml"), "utf8"));
  } catch {
    return [];
  }
}

export type RoleCredentials = { username: string; password: string };

/** The username of a role's seeded account: deploy-byoa-aws's seed creates exactly this one. */
export function roleAccountUsername(role: string): string {
  return `${role}@albitor-role.invalid`;
}

/** The identity the local issuer registers for a role (api-go's localauth.RoleIdentityName). */
export function localRoleIdentity(role: string): `role-${string}` {
  return `role-${role}`;
}

/**
 * A role's deployed account, or null when none reached this run.
 *
 * Credentials reach the runner only as the execution-scoped journey credentials file
 * (ALBITOR_JOURNEY_CREDENTIALS, mode 0600, deleted when the walk ends), never as environment
 * variables. Besides the demo account's top-level { username, password, sign_in_route }, the
 * platform writes one entry per declared role:
 *
 *   { "username": "...", "password": "...", "sign_in_route": "/sign-in",
 *     "roles": { "<role>": { "username": "<role>@albitor-role.invalid", "password": "..." } } }
 *
 * `roles.<role>.password` is required; `roles.<role>.username` is optional and defaults to the
 * account the deploy seeds (roleAccountUsername). No file, an unreadable one, or a role without a
 * password means no account. A role that is not a group name has no account.
 */
export function readRoleCredentials(env: Record<string, string | undefined>, role: string): RoleCredentials | null {
  if (!ROLE_PATTERN.test(role)) return null;
  const path = env.ALBITOR_JOURNEY_CREDENTIALS || "";
  if (!path) return null;
  try {
    const parsed = JSON.parse(readFileSync(path, "utf8")) as {
      roles?: Record<string, { username?: unknown; password?: unknown } | undefined>;
    };
    const entry = parsed.roles?.[role];
    if (!entry || typeof entry.password !== "string" || !entry.password) return null;
    return {
      username: typeof entry.username === "string" && entry.username ? entry.username : roleAccountUsername(role),
      password: entry.password,
    };
  } catch {
    return null;
  }
}
