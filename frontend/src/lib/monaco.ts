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
  /* a sintaxe herda as cores do vs-dark (keywords, strings, tipos); só as superfícies seguem o tema do app */
  inherit: true,
  rules: [],
  colors: {
    "editorIndentGuide.background1": "#1c1c1c",
    "editorIndentGuide.activeBackground1": "#333333",
    "editorCursor.foreground": "#ededed",
    "editor.selectionBackground": "#3a3a3a",
    "editor.inactiveSelectionBackground": "#2a2a2a",
    "editorOverviewRuler.border": "#00000000",
    "editor.background": "#111111",
    "editorGutter.background": "#111111",
    "editor.lineHighlightBackground": "#ffffff0a",
    "editor.lineHighlightBorder": "#00000000",
    "editor.foreground": "#c4c4c4",
    "editorLineNumber.foreground": "#8f8f8f",
    "editorLineNumber.activeForeground": "#ededed",
    "editorStickyScroll.background": "#171717",
    "editorStickyScrollHover.background": "#1d1d1d",
    "editorHoverWidget.background": "#171717",
    "editorHoverWidget.border": "#333333",
    "scrollbarSlider.background": "#33333380",
    "scrollbarSlider.hoverBackground": "#333333",
    "scrollbarSlider.activeBackground": "#8f8f8f"
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
