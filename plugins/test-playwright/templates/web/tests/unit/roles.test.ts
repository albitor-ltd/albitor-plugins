// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import {
	localRoleIdentity,
	parseDeclaredRoles,
	readDeclaredRoles,
	readRoleCredentials,
	roleAccountUsername,
} from "../helpers/roles.ts";

describe("parseDeclaredRoles", () => {
	it("reads a flow list", () => {
		const yaml = "name: x\nservices:\n  auth:\n    provider: cognito\n    groups: [admin, field-officer]\n";
		assert.deepEqual(parseDeclaredRoles(yaml), ["admin", "field-officer"]);
	});
	it("reads a block list, indented or not", () => {
		assert.deepEqual(parseDeclaredRoles("services:\n  auth:\n    groups:\n      - admin\n      - 'reviewer' # who checks\n"), ["admin", "reviewer"]);
		assert.deepEqual(parseDeclaredRoles("services:\n  auth:\n    groups:\n    - admin\n    posture: private\n"), ["admin"]);
	});
	it("ignores groups anywhere but services.auth", () => {
		assert.deepEqual(parseDeclaredRoles("groups: [nope]\nservices:\n  data:\n    groups: [nope]\n  auth:\n    posture: private\n"), []);
	});
	it("has no roles without a groups list", () => {
		assert.deepEqual(parseDeclaredRoles("services:\n  auth:\n    provider: cognito\n"), []);
		assert.deepEqual(parseDeclaredRoles(""), []);
	});
	it("drops anything that is not a group name, and duplicates", () => {
		assert.deepEqual(parseDeclaredRoles("services:\n  auth:\n    groups: [admin, Admin, a, 'x y', admin, ../etc]\n"), ["admin"]);
	});
	it("stops reading a block list at the next key", () => {
		assert.deepEqual(parseDeclaredRoles("services:\n  auth:\n    groups:\n      - admin\n    admin_group: admin\n    other:\n      - zzz\n"), ["admin"]);
	});
});

describe("readDeclaredRoles", () => {
	it("has no roles when there is no albitor.yaml", () => {
		assert.deepEqual(readDeclaredRoles(mkdtempSync(join(tmpdir(), "albitor-roles-"))), []);
	});
	it("reads the file at the repository root", () => {
		const dir = mkdtempSync(join(tmpdir(), "albitor-roles-"));
		writeFileSync(join(dir, "albitor.yaml"), "services:\n  auth:\n    groups: [manager]\n");
		assert.deepEqual(readDeclaredRoles(dir), ["manager"]);
	});
});

describe("role accounts", () => {
	it("names the seeded account, the variable and the local identity from the role", () => {
		assert.equal(roleAccountUsername("field-officer"), "field-officer@albitor-role.invalid");
		assert.equal(localRoleIdentity("manager"), "role-manager");
	});
	it("reads a role's password only from the credentials file", () => {
		const dir = mkdtempSync(join(tmpdir(), "albitor-roles-"));
		const file = join(dir, "creds.json");
		writeFileSync(file, JSON.stringify({ password: "demo", roles: { manager: { password: "from-file" }, other: { username: "o@x.invalid", password: "p2" }, empty: { password: "" } } }));
		const env = { ALBITOR_JOURNEY_CREDENTIALS: file, ALBITOR_ROLE_MANAGER_ACCOUNT_PASSWORD: "from-env", ALBITOR_ROLE_EMPTY_ACCOUNT_PASSWORD: "env-empty" };
		assert.deepEqual(readRoleCredentials(env, "manager"), { username: "manager@albitor-role.invalid", password: "from-file" });
		assert.deepEqual(readRoleCredentials(env, "other"), { username: "o@x.invalid", password: "p2" });
		assert.equal(readRoleCredentials(env, "empty"), null);
		assert.equal(readRoleCredentials(env, "absent"), null);
	});
	it("ignores the environment: no file, or an unreadable one, means no account", () => {
		const env = { ALBITOR_ROLE_MANAGER_ACCOUNT_PASSWORD: "from-env" };
		assert.equal(readRoleCredentials(env, "manager"), null);
		assert.equal(readRoleCredentials({ ...env, ALBITOR_JOURNEY_CREDENTIALS: "/nonexistent/creds.json" }, "manager"), null);
	});
	it("gives no account to a role that is not a group name", () => {
		assert.equal(readRoleCredentials({ ALBITOR_ROLE_X_ACCOUNT_PASSWORD: "pw" }, "X"), null);
		assert.equal(readRoleCredentials({}, "../x"), null);
	});
});
