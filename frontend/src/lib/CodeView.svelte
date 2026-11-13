<script lang="ts">
  import { mount, onMount, unmount, untrack } from "svelte";
  import { monaco, languageFor } from "./monaco";
  import ZoneCard from "./ZoneCard.svelte";
  import type { Draft, DraftInput, FileView } from "./api";

  interface Props {
    file: FileView;
    drafts: Draft[];
    fullscreen?: boolean;
    onsavedraft: (d: DraftInput) => Promise<void>;
    onremovedraft: (id: string) => Promise<void>;
  }

  let { file, drafts, fullscreen = false, onsavedraft, onremovedraft }: Props = $props();

  let container: HTMLDivElement;
  let editor: monaco.editor.IStandaloneCodeEditor | undefined = $state();
  let selection = $state<{ start: number; end: number } | null>(null);
  let cursorLine = $state(1);

  interface Zone {
    zoneId: string;
    widget: monaco.editor.IOverlayWidget;
    node: HTMLDivElement;
    component: ReturnType<typeof mount>;
    observer: ResizeObserver;
  }

  let zones: Zone[] = [];
  let composer: { startLine: number; line: number; id: string; body: string; refocus: boolean } | null = null;
  let baseDecorations: monaco.editor.IEditorDecorationsCollection | undefined;
  let hoverDecoration: monaco.editor.IEditorDecorationsCollection | undefined;
  let flashDecoration: monaco.editor.IEditorDecorationsCollection | undefined;
  let commentGlyphs: monaco.editor.IEditorDecorationsCollection | undefined;
  let commentable = new Set<number>();
  let zoneSeq = 0;

  onMount(() => {
    editor = monaco.editor.create(container, {
      theme: "oddeye",
      readOnly: true,
      domReadOnly: true,
      automaticLayout: true,
      fontFamily: "'JetBrains Mono Variable', 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'SF Mono', Consolas, Menlo, monospace",
      fontSize: 13,
      fontLigatures: true,
      lineHeight: 22,
      glyphMargin: true,
      lineNumbersMinChars: 4,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      smoothScrolling: false,
      cursorBlinking: "blink",
      renderLineHighlight: "line",
      stickyScroll: { enabled: true },
      bracketPairColorization: { enabled: true },
      guides: { bracketPairs: false, indentation: true },
      contextmenu: false,
      padding: { top: 12, bottom: 12 },
      scrollbar: { verticalScrollbarSize: 12, horizontalScrollbarSize: 12 },
      fixedOverflowWidgets: true
    });
    baseDecorations = editor.createDecorationsCollection();
    hoverDecoration = editor.createDecorationsCollection();
    flashDecoration = editor.createDecorationsCollection();

    const ed = editor;
    /* a fonte do editor é empacotada com o app e pode terminar de carregar depois da primeira medição */
    void document.fonts.ready.then(() => monaco.editor.remeasureFonts());
    ed.onMouseMove((e) => {
      const t = e.target;
      const inGutter =
        t.type === monaco.editor.MouseTargetType.GUTTER_GLYPH_MARGIN ||
        t.type === monaco.editor.MouseTargetType.GUTTER_LINE_NUMBERS ||
        t.type === monaco.editor.MouseTargetType.GUTTER_LINE_DECORATIONS;
      const line = t.position?.lineNumber;
      if (!inGutter || !line || file.readOnly) {
        hoverDecoration?.clear();
        return;
      }
      hoverDecoration?.set([
        {
          range: new monaco.Range(line, 1, line, 1),
          options: {
            glyphMarginClassName: "pg-glyph-add" + (commentable.has(line) ? "" : " outside"),
            glyphMarginHoverMessage: {
              value: commentable.has(line) ? "Comentar nesta linha" : "Linha fora do diff: o comentário vai no corpo do review"
            }
          }
        }
      ]);
    });
    ed.onMouseLeave(() => hoverDecoration?.clear());
    ed.onMouseDown((e) => {
      const t = e.target;
      if (
        (t.type === monaco.editor.MouseTargetType.GUTTER_GLYPH_MARGIN ||
          t.type === monaco.editor.MouseTargetType.GUTTER_LINE_NUMBERS) &&
        t.position &&
        !file.readOnly
      ) {
        e.event.preventDefault();
        openComposer(t.position.lineNumber, t.position.lineNumber);
      }
    });
    ed.onDidChangeCursorSelection((e) => {
      const s = e.selection;
      cursorLine = s.positionLineNumber;
      selection = s.isEmpty() ? null : { start: s.startLineNumber, end: s.endColumn === 1 && s.endLineNumber > s.startLineNumber ? s.endLineNumber - 1 : s.endLineNumber };
    });
    ed.onDidLayoutChange(() => zones.forEach((z) => placeHorizontally(z.node)));

    return () => {
      clearZones();
      ed.getModel()?.dispose();
      ed.dispose();
    };
  });

  /* Troca de arquivo: novo model, decorações do diff e da atenção, e scroll até o trecho relevante. */
  $effect(() => {
    const f = file;
    const ed = editor;
    if (!ed) return;
    composer = null;
    const old = ed.getModel();
    const model = monaco.editor.createModel(f.content, languageFor(f.path));
    ed.setModel(model);
    old?.dispose();
    commentable = new Set(f.commentable);

    const decos: monaco.editor.IModelDeltaDecoration[] = [];
    for (const l of f.added) {
      decos.push({
        range: new monaco.Range(l, 1, l, 1),
        options: { isWholeLine: true, className: "pg-line-added", linesDecorationsClassName: "pg-gutter-added" }
      });
    }
    for (const a of f.attention) {
      decos.push({
        range: new monaco.Range(a.line, 1, a.line, 1),
        options: {
          isWholeLine: true,
          className: "pg-line-attention",
          glyphMarginClassName: "pg-glyph-attention",
          glyphMarginHoverMessage: { value: "**Claude:** " + a.note },
          hoverMessage: { value: "**Claude:** " + a.note },
          overviewRuler: { color: "#e3b341", position: monaco.editor.OverviewRulerLane.Right }
        }
      });
    }
    baseDecorations?.set(decos);

    const target = f.attention[0]?.line ?? f.added[0] ?? 1;
    ed.setPosition({ lineNumber: target, column: 1 });
    ed.revealLineInCenter(target, monaco.editor.ScrollType.Immediate);
    untrack(() => renderZones(drafts));
  });

  $effect(() => {
    const d = drafts;
    if (editor) untrack(() => renderZones(d));
  });

  export function focus(): void {
    editor?.focus();
  }

  export function revealLine(line: number): void {
    if (!editor) return;
    editor.revealLineInCenter(line, monaco.editor.ScrollType.Immediate);
    editor.setPosition({ lineNumber: line, column: 1 });
    flashDecoration?.set([{ range: new monaco.Range(line, 1, line, 1), options: { isWholeLine: true, className: "pg-line-flash" } }]);
    setTimeout(() => flashDecoration?.clear(), 1300);
  }

  export function commentSelectionOrCursor(): void {
    if (file.readOnly) return;
    if (selection) openComposer(selection.start, selection.end);
    else openComposer(cursorLine, cursorLine);
  }

  function openComposer(startLine: number, line: number, id = "", body = ""): void {
    composer = { startLine, line, id, body, refocus: !!editor?.hasTextFocus() };
    renderZones(drafts);
  }

  function isInDiff(start: number, end: number): boolean {
    for (let l = start; l <= end; l++) if (!commentable.has(l)) return false;
    return true;
  }

  function placeHorizontally(node: HTMLElement): void {
    if (!editor) return;
    const info = editor.getLayoutInfo();
    node.style.left = info.contentLeft + "px";
    node.style.width = Math.max(200, info.contentWidth - info.verticalScrollbarWidth - 24) + "px";
  }

  function clearZones(): void {
    if (!editor) return;
    const ed = editor;
    ed.changeViewZones((acc) => zones.forEach((z) => acc.removeZone(z.zoneId)));
    for (const z of zones) {
      z.observer.disconnect();
      unmount(z.component);
      ed.removeOverlayWidget(z.widget);
    }
    zones = [];
  }

  /* Cada caixa de comentário é um view zone (abre espaço entre as linhas) com um overlay widget
     posicionado em cima dele: o overlay recebe foco e teclado, coisa que o view zone sozinho não deixa. */
  function addZone(afterLine: number, props: Record<string, unknown>): void {
    if (!editor) return;
    const ed = editor;
    const node = document.createElement("div");
    node.className = "pg-zone";
    node.style.position = "absolute";
    node.style.zIndex = "5";
    node.style.padding = "6px 0 8px";
    placeHorizontally(node);
    for (const ev of ["keydown", "keyup", "keypress", "mousedown", "contextmenu"]) {
      node.addEventListener(ev, (e) => e.stopPropagation());
    }
    node.addEventListener(
      "wheel",
      (e) => {
        const el = e.target as HTMLElement;
        if (el.tagName === "TEXTAREA" && el.scrollHeight > el.clientHeight) return;
        e.preventDefault();
        ed.setScrollTop(ed.getScrollTop() + e.deltaY);
      },
      { passive: false }
    );

    const widget: monaco.editor.IOverlayWidget = {
      getId: () => "pg-zone-" + ++zoneSeq,
      getDomNode: () => node,
      getPosition: () => null
    };
    const id = widget.getId();
    widget.getId = () => id;
    ed.addOverlayWidget(widget);
    const component = mount(ZoneCard, { target: node, props: props as never });

    const spacer = document.createElement("div");
    const zone: monaco.editor.IViewZone = {
      afterLineNumber: afterLine,
      heightInPx: 120,
      domNode: spacer,
      suppressMouseDown: true,
      onDomNodeTop: (top) => {
        node.style.top = top + "px";
      }
    };
    let zoneId = "";
    ed.changeViewZones((acc) => {
      zoneId = acc.addZone(zone);
    });
    const observer = new ResizeObserver(() => {
      const h = Math.ceil(node.getBoundingClientRect().height);
      if (h && h !== zone.heightInPx) {
        zone.heightInPx = h;
        ed.changeViewZones((acc) => acc.layoutZone(zoneId));
      }
    });
    observer.observe(node);
    zones.push({ zoneId, widget, node, component, observer });
  }

  function renderZones(all: Draft[]): void {
    if (!editor) return;
    clearZones();
    const mine = all.filter((d) => d.path === file.path);
    const glyphs: monaco.editor.IModelDeltaDecoration[] = [];
    for (const d of mine) {
      glyphs.push({
        range: new monaco.Range(d.line, 1, d.line, 1),
        options: { glyphMarginClassName: "pg-glyph-comment", glyphMarginHoverMessage: { value: d.body } }
      });
      if (composer?.id === d.id) continue;
      addZone(d.line, {
        mode: "view",
        startLine: d.startLine,
        line: d.line,
        body: d.body,
        inDiff: d.inDiff,
        onedit: () => openComposer(d.startLine, d.line, d.id, d.body),
        onremove: () => onremovedraft(d.id)
      });
    }
    commentGlyphs?.clear();
    commentGlyphs = editor.createDecorationsCollection(glyphs);

    if (composer) {
      const c = composer;
      addZone(c.line, {
        mode: "edit",
        startLine: c.startLine,
        line: c.line,
        body: c.body,
        inDiff: isInDiff(c.startLine, c.line),
        oncancel: () => {
          composer = null;
          renderZones(drafts);
          if (c.refocus) editor?.focus();
        },
        onsave: async (body: string) => {
          composer = null;
          if (c.refocus) editor?.focus();
          await onsavedraft({ id: c.id, path: file.path, startLine: c.startLine, line: c.line, body });
        }
      });
    }
  }
</script>

<div class="window" class:fullscreen>
  <div class="titlebar">
    <span class="path" title={file.path}>{file.path}</span>
    {#if file.readOnly}
      <span class="tag del">removido (versão da base)</span>
    {:else if !file.inPR}
      <span class="tag">fora da PR (contexto)</span>
    {:else}
      <span class="tag add">+{file.added.length} linhas novas</span>
    {/if}
    {#if fullscreen}
      <span class="keys">Alt + ← → troca de arquivo · C comenta · Esc sai</span>
    {:else if !file.readOnly}
      <button class="btn small" onclick={commentSelectionOrCursor} title="Atalho: C">
        {selection && selection.end > selection.start ? `Comentar L${selection.start}–L${selection.end}` : `Comentar L${selection?.start ?? cursorLine}`}
      </button>
    {/if}
  </div>
  <div class="editor" bind:this={container}></div>
</div>

<style>
  .window {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: var(--bg);
  }
  .window.fullscreen {
    position: fixed;
    inset: 0;
    z-index: var(--z-fullscreen);
  }
  .titlebar {
    display: flex;
    align-items: center;
    gap: var(--sp-4);
    height: 40px;
    padding: 0 var(--sp-5);
    border-bottom: 1px solid var(--border);
    font-size: var(--fs-xs);
    flex-shrink: 0;
  }
  .path {
    flex: 1;
    min-width: 0;
    font-family: var(--mono);
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  .fullscreen .path {
    color: var(--text);
  }
  .tag {
    color: var(--muted);
    white-space: nowrap;
  }
  .keys {
    color: var(--muted);
    white-space: nowrap;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
</style>
