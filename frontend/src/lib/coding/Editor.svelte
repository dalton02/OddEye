<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { monaco, languageFor } from "../monaco";
  import { api, errorMessage, type WsChanges, type WsEntry } from "../api";
  import { baseName, dirName, folderName } from "./coding";
  import FileTree from "./FileTree.svelte";
  import GitChanges from "./GitChanges.svelte";

  interface Props {
    root: string;
    /** a aba está na frente e no modo editor */
    visible: boolean;
  }

  let { root, visible }: Props = $props();

  /* ---------- árvore ---------- */
  let entries = $state(new Map<string, WsEntry[]>());
  let expanded = $state(new Set<string>());
  let changes = $state<WsChanges | null>(null);
  let changesKey = "";

  async function loadDir(rel: string): Promise<void> {
    try {
      const list = await api.wsListDir(root, rel);
      entries = new Map(entries).set(rel, list);
    } catch {
      /* pasta sumiu entre a listagem e o clique: some da árvore no próximo refresh */
    }
  }

  function toggleDir(path: string): void {
    const next = new Set(expanded);
    if (next.has(path)) next.delete(path);
    else {
      next.add(path);
      if (!entries.has(path)) void loadDir(path);
    }
    expanded = next;
  }

  /** abre as pastas até o arquivo, para ele aparecer selecionado na árvore */
  function reveal(path: string): void {
    const next = new Set(expanded);
    for (let d = dirName(path); d; d = dirName(d)) {
      if (!next.has(d)) {
        next.add(d);
        if (!entries.has(d)) void loadDir(d);
      }
    }
    expanded = next;
  }

  /* ---------- arquivos abertos ---------- */
  interface Doc {
    model: monaco.editor.ITextModel;
    saved: number;
    readOnly: boolean;
  }
  interface FileTab {
    key: string;
    path: string;
    diff: boolean;
  }

  const docs = new Map<string, Doc>(); // por caminho; a aba de diff e a normal compartilham o modelo
  const originals = new Map<string, monaco.editor.ITextModel>(); // conteúdo em HEAD, por caminho
  let unsupported = $state(new Map<string, string>()); // caminho -> por que não abre
  let tabs = $state<FileTab[]>([]);
  let current = $state<FileTab | null>(null);
  let dirty = $state(new Set<string>());
  let confirmClose = $state<string | null>(null);
  let status = $state("");
  let syncing = false;

  let codeHost: HTMLDivElement;
  let diffHost: HTMLDivElement;
  let code: monaco.editor.IStandaloneCodeEditor | undefined;
  let diff: monaco.editor.IStandaloneDiffEditor | undefined;

  const currentNotice = $derived(current ? (unsupported.get(current.path) ?? "") : "");
  let lockedPaths = $state(new Set<string>()); // apagados no disco: o diff mostra, mas não grava

  function markDirty(path: string): void {
    const d = docs.get(path);
    const isDirty = !!d && d.model.getAlternativeVersionId() !== d.saved;
    if (isDirty === dirty.has(path)) return;
    const next = new Set(dirty);
    if (isDirty) next.add(path);
    else next.delete(path);
    dirty = next;
  }

  function makeDoc(path: string, content: string, readOnly: boolean): Doc {
    const model = monaco.editor.createModel(content, languageFor(path));
    const doc = { model, saved: model.getAlternativeVersionId(), readOnly };
    model.onDidChangeContent(() => {
      if (!syncing) markDirty(path);
    });
    docs.set(path, doc);
    if (readOnly) setLocked(path, true);
    return doc;
  }

  function setLocked(path: string, on: boolean): void {
    if (on === lockedPaths.has(path)) return;
    const next = new Set(lockedPaths);
    if (on) next.add(path);
    else next.delete(path);
    lockedPaths = next;
  }

  function setUnsupported(path: string, msg: string): void {
    unsupported = new Map(unsupported).set(path, msg);
  }

  async function ensureDoc(path: string, wantDiff: boolean): Promise<void> {
    if (wantDiff) {
      const d = await api.wsFileDiff(root, path);
      if (d.binary) return setUnsupported(path, "Arquivo binário ou grande demais para mostrar o diff.");
      const orig = originals.get(path);
      if (orig) orig.setValue(d.original);
      else originals.set(path, monaco.editor.createModel(d.original, languageFor(path)));
      const deleted = changes?.files.find((f) => f.path === path)?.status === "D";
      if (!docs.has(path)) makeDoc(path, d.modified, deleted);
      return;
    }
    if (docs.has(path) || unsupported.has(path)) return;
    const f = await api.wsReadFile(root, path);
    if (f.binary) return setUnsupported(path, "Arquivo binário: o editor não abre.");
    if (f.tooBig) return setUnsupported(path, "Arquivo grande demais para o editor (mais de 2 MB).");
    makeDoc(path, f.content, false);
  }

  async function openFile(path: string, wantDiff: boolean): Promise<void> {
    const key = (wantDiff ? "diff:" : "") + path;
    try {
      await ensureDoc(path, wantDiff);
    } catch (e) {
      status = errorMessage(e);
      return;
    }
    let tab = tabs.find((t) => t.key === key);
    if (!tab) {
      tab = { key, path, diff: wantDiff };
      tabs = [...tabs, tab];
    }
    current = tab;
    confirmClose = null;
    status = "";
    reveal(path);
    show();
  }

  function show(): void {
    if (!current || !code || !diff) return;
    const doc = docs.get(current.path);
    if (!doc) return;
    if (current.diff) {
      const original = originals.get(current.path);
      if (!original) return;
      diff.setModel({ original, modified: doc.model });
      diff.getModifiedEditor().updateOptions({ readOnly: doc.readOnly });
      diff.getModifiedEditor().focus();
    } else {
      code.setModel(doc.model);
      code.updateOptions({ readOnly: doc.readOnly });
      code.focus();
    }
  }

  function closeTab(tab: FileTab): void {
    const stillUsed = tabs.some((t) => t !== tab && t.path === tab.path);
    if (!stillUsed && dirty.has(tab.path) && confirmClose !== tab.key) {
      confirmClose = tab.key;
      return;
    }
    confirmClose = null;
    const i = tabs.indexOf(tab);
    tabs = tabs.filter((t) => t !== tab);
    if (current === tab) {
      current = tabs[Math.min(i, tabs.length - 1)] ?? null;
      if (current) show();
      else {
        code?.setModel(null);
        diff?.setModel(null);
      }
    }
    if (!tabs.some((t) => t.path === tab.path)) {
      docs.get(tab.path)?.model.dispose();
      docs.delete(tab.path);
      if (unsupported.has(tab.path)) {
        unsupported.delete(tab.path);
        unsupported = new Map(unsupported);
      }
      setLocked(tab.path, false);
      if (dirty.delete(tab.path)) dirty = new Set(dirty);
    }
    if (!tabs.some((t) => t.path === tab.path && t.diff)) {
      originals.get(tab.path)?.dispose();
      originals.delete(tab.path);
    }
  }

  /** Ctrl+S: grava o arquivo da aba atual. */
  export async function save(): Promise<void> {
    if (!current) return;
    const path = current.path;
    const doc = docs.get(path);
    if (!doc || doc.readOnly) return;
    try {
      await api.wsWriteFile(root, path, doc.model.getValue());
      doc.saved = doc.model.getAlternativeVersionId();
      markDirty(path);
      status = `${baseName(path)} salvo`;
      void refresh();
    } catch (e) {
      status = `Não consegui salvar: ${errorMessage(e)}`;
    }
  }

  /* ---------- sincronização com o disco (o Claude edita os arquivos enquanto estão abertos) ---------- */
  function replaceContent(doc: Doc, text: string): void {
    if (doc.model.getValue() === text) return;
    syncing = true;
    doc.model.pushEditOperations([], [{ range: doc.model.getFullModelRange(), text }], () => null);
    syncing = false;
    doc.saved = doc.model.getAlternativeVersionId();
  }

  let refreshing = false;
  async function refresh(): Promise<void> {
    if (refreshing) return;
    refreshing = true;
    try {
      const next = await api.wsChanges(root).catch(() => null);
      if (next) {
        changes = next;
        const key = next.files.map((f) => f.status + f.path).join("|");
        if (key !== changesKey) {
          changesKey = key;
          await Promise.all(["", ...expanded].filter((d) => entries.has(d)).map(loadDir));
        }
      }
      const diffPaths = new Set(tabs.filter((t) => t.diff).map((t) => t.path));
      await Promise.all(
        [...docs].map(async ([path, doc]) => {
          if (dirty.has(path)) return;
          try {
            if (diffPaths.has(path)) {
              const d = await api.wsFileDiff(root, path);
              if (d.binary) return;
              const orig = originals.get(path);
              if (orig && orig.getValue() !== d.original) orig.setValue(d.original);
              replaceContent(doc, d.modified);
            } else {
              const f = await api.wsReadFile(root, path);
              if (!f.binary && !f.tooBig) replaceContent(doc, f.content);
            }
          } catch {
            /* apagado no disco: mantém o que está aberto */
          }
        })
      );
    } finally {
      refreshing = false;
    }
  }

  const editorOptions: monaco.editor.IEditorOptions = {
    automaticLayout: true,
    fontFamily: "'JetBrains Mono Variable', 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'SF Mono', Consolas, Menlo, monospace",
    fontSize: 13,
    fontLigatures: true,
    lineHeight: 20,
    lineNumbersMinChars: 4,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    smoothScrolling: false,
    renderLineHighlight: "line",
    stickyScroll: { enabled: true },
    bracketPairColorization: { enabled: true },
    guides: { bracketPairs: false, indentation: true },
    padding: { top: 8, bottom: 8 },
    scrollbar: { verticalScrollbarSize: 12, horizontalScrollbarSize: 12 },
    fixedOverflowWidgets: true
  };

  onMount(() => {
    code = monaco.editor.create(codeHost, { ...editorOptions, theme: "oddeye", model: null });
    diff = monaco.editor.createDiffEditor(diffHost, {
      ...editorOptions,
      theme: "oddeye",
      renderSideBySide: false,
      originalEditable: false,
      renderOverviewRuler: false
    });
    void document.fonts.ready.then(() => monaco.editor.remeasureFonts());
    void loadDir("");
    void refresh();

    const onFocus = () => visible && void refresh();
    window.addEventListener("focus", onFocus);
    const timer = setInterval(() => visible && void refresh(), 3000);
    return () => {
      clearInterval(timer);
      window.removeEventListener("focus", onFocus);
      code?.dispose();
      diff?.dispose();
      docs.forEach((d) => d.model.dispose());
      originals.forEach((m) => m.dispose());
    };
  });

  $effect(() => {
    if (visible) {
      untrack(() => void refresh());
      requestAnimationFrame(() => (current?.diff ? diff?.getModifiedEditor() : code)?.focus());
    }
  });

</script>

<div class="editor" class:hidden={!visible}>
  <FileTree
    title={folderName(root)}
    {entries}
    {expanded}
    changes={changes?.files ?? []}
    activePath={current?.path ?? null}
    ontoggle={toggleDir}
    onopen={(p) => void openFile(p, false)}
  />

  <div class="center">
    {#if tabs.length}
      <div class="ftabs" role="tablist" aria-label="Arquivos abertos">
        {#each tabs as t (t.key)}
          <div class="ftab" class:on={t === current} role="tab" aria-selected={t === current} title={t.path}>
            <button class="label" onclick={() => { current = t; confirmClose = null; show(); }}>
              {baseName(t.path)}
              {#if t.diff}<span class="kind">diff</span>{/if}
            </button>
            {#if dirty.has(t.path)}<span class="dirty" title="Não salvo"></span>{/if}
            <button
              class="x"
              class:warn={confirmClose === t.key}
              aria-label="Fechar {baseName(t.path)}"
              title={confirmClose === t.key ? "Clique de novo para descartar as mudanças" : "Fechar"}
              onclick={() => closeTab(t)}
            >×</button>
          </div>
        {/each}
      </div>
    {/if}
    {#if current}
      <div class="crumbs">
        <span class="path">{current.path.split("/").join(" / ")}</span>
        <span class="state">
          {#if status}{status}
          {:else if confirmClose}Mudanças não salvas: clique no × de novo para descartar
          {:else if lockedPaths.has(current.path)}apagado no disco
          {:else if dirty.has(current.path)}não salvo · Ctrl+S
          {:else if current.diff}HEAD ↔ disco
          {:else}salvo{/if}
        </span>
      </div>
    {/if}
    <div class="surface">
      <div class="host" class:hidden={!current || current.diff || !!currentNotice} bind:this={codeHost}></div>
      <div class="host" class:hidden={!current || !current.diff || !!currentNotice} bind:this={diffHost}></div>
      {#if !current}
        <div class="empty">Abra um arquivo na árvore<br />ou clique num arquivo alterado à direita.</div>
      {:else if currentNotice}
        <div class="empty">{currentNotice}</div>
      {/if}
    </div>
  </div>

  <GitChanges {changes} activeDiff={current?.diff ? current.path : null} onopen={(p) => void openFile(p, true)} />
</div>

<style>
  .editor {
    position: absolute;
    inset: 0;
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr) 280px;
  }
  .editor > :global(* + *) {
    border-left: 1px solid var(--border);
  }
  .hidden {
    display: none !important;
  }
  .center {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .ftabs {
    display: flex;
    height: 30px;
    flex-shrink: 0;
    border-bottom: 1px solid var(--border);
    overflow-x: auto;
    scrollbar-width: none;
  }
  .ftab {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 4px 0 0;
    border-right: 1px solid var(--border);
    white-space: nowrap;
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .ftab.on {
    color: var(--text);
    background: var(--surface);
  }
  .ftab button {
    background: none;
    border: 0;
    color: inherit;
    font: inherit;
    cursor: pointer;
  }
  .label {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 100%;
    padding: 0 4px 0 10px;
  }
  .kind {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--muted);
    border: 1px solid var(--border-strong);
    border-radius: 3px;
    padding: 0 3px;
  }
  .dirty {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text);
  }
  .x {
    width: 18px;
    height: 18px;
    border-radius: 4px;
    display: grid;
    place-items: center;
    color: var(--muted);
  }
  .x:hover {
    background: var(--selected);
    color: var(--text);
  }
  .x.warn {
    color: var(--red);
  }
  .crumbs {
    height: 26px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    padding: 0 var(--sp-3);
    font-size: 11px;
    color: var(--muted);
  }
  .crumbs .path {
    font-family: var(--mono);
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .crumbs .state {
    margin-left: auto;
    white-space: nowrap;
  }
  .surface {
    flex: 1;
    min-height: 0;
    position: relative;
  }
  .host {
    position: absolute;
    inset: 0;
  }
  .empty {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    padding: var(--sp-4);
    text-align: center;
    line-height: 1.8;
    color: var(--muted);
  }
</style>
