/* Só o núcleo do editor e os tokenizadores (monarch): sem os language services, que exigiriam
   workers de TS/JSON/CSS/HTML e não servem para um visor read-only. */
import "monaco-editor/esm/vs/editor/edcore.main";
import "monaco-editor/esm/vs/basic-languages/monaco.contribution";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

self.MonacoEnvironment = {
  getWorker: () => new EditorWorker()
};

monaco.editor.defineTheme("oddeye", {
  base: "vs-dark",
  inherit: true,
  rules: [],
  colors: {
    "editor.background": "#16171b",
    "editorGutter.background": "#16171b",
    "minimap.background": "#131417",
    "editor.lineHighlightBackground": "#ffffff0a",
    "editor.lineHighlightBorder": "#00000000",
    "editor.foreground": "#d4d4d4",
    "editorLineNumber.foreground": "#6e7681",
    "editorLineNumber.activeForeground": "#c6c6c6"
  }
});

const extraByName: Record<string, string> = {
  dockerfile: "dockerfile",
  makefile: "shell",
  "schema.prisma": "graphql",
  "go.mod": "go"
};

export function languageFor(path: string): string {
  const name = path.split("/").pop()!.toLowerCase();
  if (extraByName[name]) return extraByName[name];
  if (name.endsWith(".prisma")) return "graphql";
  const ext = name.includes(".") ? "." + name.split(".").pop() : "";
  for (const lang of monaco.languages.getLanguages()) {
    if (lang.filenames?.some((f) => f.toLowerCase() === name)) return lang.id;
    if (ext && lang.extensions?.includes(ext)) return lang.id;
  }
  return "plaintext";
}

export { monaco };
