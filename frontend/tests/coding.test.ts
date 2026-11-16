import { strict as assert } from "node:assert";
import { changedDirs, decodeBase64, folderName, nextActive, nextHomeTab, nextIndex, shortcutFor, shortPath } from "../src/lib/coding/coding";

const k = (key: string, mods: Partial<KeyboardEvent> = {}) =>
  ({ key, ctrlKey: true, metaKey: false, altKey: false, shiftKey: false, ...mods }) as KeyboardEvent;

assert.equal(shortcutFor(k("e")), "editor");
assert.equal(shortcutFor(k("E")), "editor");
assert.equal(shortcutFor(k("t")), "new");
assert.equal(shortcutFor(k("w")), "close");
assert.equal(shortcutFor(k("s")), "save");
assert.deepEqual(shortcutFor(k("3")), { tab: 2 });
assert.equal(shortcutFor(k("0")), null);
assert.equal(shortcutFor(k("e", { ctrlKey: false })), null);
assert.equal(shortcutFor(k("e", { ctrlKey: false, metaKey: true })), "editor");
assert.equal(shortcutFor(k("e", { shiftKey: true })), null);
assert.equal(shortcutFor(k("c")), null);
assert.equal(shortcutFor(k("T", { shiftKey: true })), "shell");
assert.equal(shortcutFor(k("Tab")), "next");
assert.equal(shortcutFor(k("Tab", { shiftKey: true })), "home");
assert.equal(shortcutFor(k("Tab", { ctrlKey: false })), null);
assert.equal(shortcutFor(k("ISO_Left_Tab", { shiftKey: true })), "home");
assert.equal(shortcutFor({ ...k("Unidentified", { shiftKey: true }), code: "Tab" } as KeyboardEvent), "home");
assert.equal(shortcutFor(k("c", { shiftKey: true })), null);
assert.equal(shortcutFor(k("v", { shiftKey: true })), null);

assert.equal(nextIndex(3, 0), 1);
assert.equal(nextIndex(3, 2), 0);
assert.equal(nextIndex(0, 0), 0);
assert.equal(nextHomeTab("prs"), "board");
assert.equal(nextHomeTab("board"), "coding");
assert.equal(nextHomeTab("coding"), "prs");

assert.equal(folderName("/home/u/repos/oddeye"), "oddeye");
assert.equal(folderName("/home/u/repos/oddeye/"), "oddeye");
assert.equal(folderName("C:\\src\\app"), "app");
assert.equal(shortPath("/home/u/repos/x"), "~/repos/x");
assert.equal(shortPath("/Users/ana/x"), "~/x");
assert.equal(shortPath("C:\\Users\\ana\\src"), "~\\src");
assert.equal(shortPath("/opt/x"), "/opt/x");

assert.deepEqual([...changedDirs(["a/b/c.go", "a/d.go", "e.go"])].sort(), ["a", "a/b"]);

assert.deepEqual([...decodeBase64(btoa("oi"))], [111, 105]);

assert.equal(nextActive(["a", "b", "c"], 1), "c");
assert.equal(nextActive(["a", "b", "c"], 2), "b");
assert.equal(nextActive(["a"], 0), null);

console.log("coding: ok");
