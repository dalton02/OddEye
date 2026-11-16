<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type CodingTab, type TermKind } from "../api";
  import { folderName, nextActive, nextIndex, shortcutFor, shortPath } from "./coding";
  import Terminal from "./Terminal.svelte";
  import Editor from "./Editor.svelte";

  interface Props {
    /** um modal do app está aberto: os atalhos ficam com ele */
    paused?: boolean;
  }

  let { paused = false }: Props = $props();

  interface Tab extends CodingTab {
    kind: TermKind;
    mode: "term" | "editor";
    /** terminal e editor só montam na primeira vez que a aba aparece, para não subir N claudes de uma vez */
    termMounted: boolean;
    editorMounted: boolean;
    exited: boolean;
  }

  let tabs = $state<Tab[]>([]);
  let activeId = $state("");
  let loaded = $state(false);
  let error = $state("");
  const editors: Record<string, Editor | undefined> = $state({});

  const active = $derived(tabs.find((t) => t.id === activeId) ?? null);

  function toTab(t: CodingTab): Tab {
    return { ...t, kind: t.kind === "shell" ? "shell" : "claude", mode: "term", termMounted: false, editorMounted: false, exited: false };
  }

  function persist(): void {
    void api.saveCodingPrefs({ tabs: tabs.map(({ id, dir, kind }) => (kind === "shell" ? { id, dir, kind } : { id, dir })), active: activeId }).catch(() => {});
  }

  function select(t: Tab): void {
    activeId = t.id;
    if (t.mode === "term") t.termMounted = true;
    else t.editorMounted = true;
    persist();
  }

  async function addTab(): Promise<void> {
    error = "";
    try {
      const dir = await api.pickFolder();
      if (!dir) return;
      const t = toTab({ id: crypto.randomUUID(), dir });
      tabs.push(t);
      select(tabs[tabs.length - 1]);
    } catch (e) {
      error = errorMessage(e);
    }
  }

  /** Ctrl+Shift+T: o shell do usuário na pasta da aba ativa (ou na pasta pessoal), sem seletor */
  async function addShell(): Promise<void> {
    error = "";
    try {
      const dir = active?.dir || (await api.homeDir());
      tabs.push(toTab({ id: crypto.randomUUID(), dir, kind: "shell" }));
      select(tabs[tabs.length - 1]);
    } catch (e) {
      error = errorMessage(e);
    }
  }

  function closeTab(t: Tab): void {
    const i = tabs.indexOf(t);
    if (i < 0) return;
    void api.termClose(t.id).catch(() => {});
    const next = nextActive(tabs, i);
    tabs.splice(i, 1);
    delete editors[t.id];
    if (activeId === t.id) {
      if (next) select(next);
      else activeId = "";
    }
    persist();
  }

  function toggleMode(): void {
    if (!active) return;
    active.mode = active.mode === "term" ? "editor" : "term";
    if (active.mode === "editor") active.editorMounted = true;
    else active.termMounted = true;
  }

  function onKey(e: KeyboardEvent): void {
    if (paused) return;
    const sc = shortcutFor(e);
    if (!sc) return;
    if (sc === "home") return; // Ctrl+Shift+Tab: tratado pelo App, que troca a aba da home
    if (sc === "save") {
      if (active?.mode !== "editor") return; // no terminal, o Ctrl+S segue para o claude
      e.preventDefault();
      e.stopPropagation();
      void editors[active.id]?.save();
      return;
    }
    e.preventDefault();
    e.stopPropagation();
    if (sc === "editor") toggleMode();
    else if (sc === "new") void addTab();
    else if (sc === "shell") void addShell();
    else if (sc === "next") {
      if (tabs.length) select(tabs[nextIndex(tabs.length, tabs.findIndex((t) => t.id === activeId))]);
    }
    else if (sc === "close") {
      if (active) closeTab(active);
    } else if (tabs[sc.tab]) select(tabs[sc.tab]);
  }

  onMount(() => {
    /* captura: o atalho é tratado antes do xterm e do Monaco verem a tecla */
    window.addEventListener("keydown", onKey, true);
    api
      .codingPrefs()
      .then((p) => {
        tabs = p.tabs.map(toTab);
        const first = tabs.find((t) => t.id === p.active) ?? tabs[0];
        if (first) select(first);
      })
      .catch((e) => (error = errorMessage(e)))
      .finally(() => (loaded = true));
    return () => window.removeEventListener("keydown", onKey, true);
  });
</script>

<div class="win">
  <div class="tabbar" role="tablist" aria-label="Sessões do Claude">
    {#each tabs as t, i (t.id)}
      <div class="tab" class:on={t.id === activeId} role="tab" aria-selected={t.id === activeId} title={t.dir}>
        <button class="label" onclick={() => select(t)}>
          <span class="dot" class:dead={t.exited}></span>
          {#if t.kind === "shell"}<span class="sh" title="Terminal">&gt;_</span>{/if}
          <span>{folderName(t.dir)}</span>
          <span class="path">{shortPath(t.dir)}</span>
          {#if i < 9}<span class="num">{i + 1}</span>{/if}
        </button>
        <button class="x" aria-label="Fechar {folderName(t.dir)}" title="Fechar (Ctrl+W)" onclick={() => closeTab(t)}>×</button>
      </div>
    {/each}
    <button class="add" onclick={() => void addTab()} title="Nova aba do Claude (Ctrl+T) · terminal (Ctrl+Shift+T)" aria-label="Nova aba">+</button>
    {#if active}
      <button class="mode" onclick={toggleMode}>
        {active.mode === "term" ? "Abrir editor" : active.kind === "shell" ? "Voltar ao terminal" : "Voltar ao Claude"} <kbd>Ctrl E</kbd>
      </button>
    {/if}
  </div>

  <div class="body">
    {#each tabs as t (t.id)}
      {#if t.termMounted}
        <Terminal id={t.id} dir={t.dir} kind={t.kind} visible={t.id === activeId && t.mode === "term"} onexit={(x) => (t.exited = x)} />
      {/if}
      {#if t.editorMounted}
        <Editor bind:this={editors[t.id]} root={t.dir} visible={t.id === activeId && t.mode === "editor"} />
      {/if}
    {/each}

    {#if loaded && !tabs.length}
      <div class="empty">
        <p>Cada aba roda um Claude Code numa pasta do seu computador.</p>
        <button class="btn primary" onclick={() => void addTab()}>Abrir pasta</button>
        <p class="keys">
          <span><kbd>Ctrl T</kbd> nova aba</span>
          <span><kbd>Ctrl Shift T</kbd> terminal</span>
          <span><kbd>Ctrl Tab</kbd> próxima aba</span>
          <span><kbd>Ctrl W</kbd> fechar</span>
          <span><kbd>Ctrl 1–9</kbd> trocar</span>
          <span><kbd>Ctrl E</kbd> editor</span>
          <span><kbd>Ctrl S</kbd> salvar</span>
        </p>
      </div>
    {/if}
    {#if error}<p class="error-text err">{error}</p>{/if}
  </div>
</div>

<style>
  .win {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--bg);
    overflow: hidden;
  }
  .tabbar {
    display: flex;
    align-items: stretch;
    flex-shrink: 0;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabbar button {
    background: none;
    border: 0;
    color: inherit;
    font: inherit;
    cursor: pointer;
  }
  .tab {
    display: flex;
    align-items: center;
    height: 34px;
    padding-right: 6px;
    flex-shrink: 0;
    border-right: 1px solid var(--border);
    color: var(--muted);
    white-space: nowrap;
    transition: color var(--ease), background var(--ease);
  }
  .tab.on {
    background: var(--bg);
    color: var(--text);
  }
  .label {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    height: 100%;
    padding: 0 var(--sp-2) 0 var(--sp-3);
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--muted);
  }
  .tab.on .dot {
    background: var(--text);
  }
  .dot.dead,
  .tab.on .dot.dead {
    background: var(--red);
  }
  .sh {
    font-family: var(--mono);
    font-size: 11px;
    font-weight: 600;
  }
  .path,
  .num {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--muted);
  }
  .num {
    font-size: 10px;
  }
  .x {
    width: 18px;
    height: 18px;
    border-radius: 4px;
    display: grid;
    place-items: center;
    color: var(--muted);
    opacity: 0;
  }
  .tab:hover .x,
  .tab.on .x,
  .x:focus-visible {
    opacity: 1;
  }
  .x:hover {
    background: var(--selected);
    color: var(--text);
  }
  .add {
    width: 34px;
    flex-shrink: 0;
    color: var(--muted);
    font-size: 16px !important;
  }
  .add:hover {
    color: var(--text);
    background: var(--surface-2);
  }
  .mode {
    margin-left: auto;
    padding: 0 var(--sp-3);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    border-left: 1px solid var(--border) !important;
    color: var(--muted);
    font-size: var(--fs-xs) !important;
  }
  .mode:hover {
    color: var(--text);
  }
  kbd {
    font-family: var(--mono);
    font-size: 11px;
    border: 1px solid var(--border-strong);
    border-radius: 4px;
    padding: 0 4px;
    color: var(--text-2);
  }
  .body {
    flex: 1;
    min-height: 0;
    position: relative;
  }
  .empty {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--sp-4);
    color: var(--muted);
    text-align: center;
    padding: var(--sp-4);
  }
  .empty p {
    margin: 0;
  }
  .keys {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--sp-4);
    font-size: var(--fs-xs);
  }
  .err {
    position: absolute;
    left: var(--sp-4);
    bottom: var(--sp-3);
    margin: 0;
  }
</style>
