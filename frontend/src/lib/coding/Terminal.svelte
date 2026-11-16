<script lang="ts">
  import { onMount } from "svelte";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import "@xterm/xterm/css/xterm.css";
  import { ClipboardGetText, ClipboardSetText } from "../../../wailsjs/runtime/runtime";
  import { api, errorMessage, type TermChunk, type TermKind } from "../api";
  import { decodeBase64 } from "./coding";

  interface Props {
    id: string;
    dir: string;
    kind: TermKind;
    /** a aba está na frente e no modo terminal */
    visible: boolean;
    onexit: (exited: boolean) => void;
  }

  let { id, dir, kind, visible, onexit }: Props = $props();

  let host: HTMLDivElement;
  let term: Terminal | undefined;
  let fit: FitAddon | undefined;
  let exited = false;
  let lastSeq = -1; // -1: ainda esperando o histórico; os pedaços que chegarem antes ficam em pending
  let pending: TermChunk[] = [];
  let size = { cols: 0, rows: 0 };

  function write(c: TermChunk): void {
    if (c.seq <= lastSeq) return;
    lastSeq = c.seq;
    term?.write(decodeBase64(c.data));
  }

  async function open(restart: boolean): Promise<void> {
    if (!term) return;
    lastSeq = -1;
    try {
      const snap = await api.termOpen(id, dir, kind, term.cols, term.rows);
      /* ao reiniciar no mesmo xterm o histórico já está na tela; numa montagem nova, redesenha */
      if (!restart && snap.scrollback) term.write(decodeBase64(snap.scrollback));
      lastSeq = snap.seq;
      exited = false;
      onexit(false);
      const queued = pending;
      pending = [];
      queued.forEach(write);
      size = { cols: term.cols, rows: term.rows };
    } catch (e) {
      lastSeq = 0;
      exited = true;
      onexit(true);
      term.write(`\r\n\x1b[31m${errorMessage(e)}\x1b[0m\r\n\x1b[90mEnter para tentar de novo.\x1b[0m\r\n`);
    }
  }

  function refit(): void {
    if (!term || !fit || !visible || !host.offsetParent) return;
    fit.fit();
    if (lastSeq >= 0 && !exited && (term.cols !== size.cols || term.rows !== size.rows)) {
      size = { cols: term.cols, rows: term.rows };
      void api.termResize(id, term.cols, term.rows).catch(() => {});
    }
  }

  onMount(() => {
    const offData = api.onTermData(id, (c) => (lastSeq < 0 ? pending.push(c) : write(c)));
    const offExit = api.onTermExit(id, () => {
      exited = true;
      onexit(true);
      term?.write("\r\n\x1b[90mProcesso encerrado. Enter para reiniciar.\x1b[0m\r\n");
    });
    let disposed = false;
    let raf = 0;
    const ro = new ResizeObserver(() => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(refit);
    });

    void document.fonts.ready.then(() => {
      if (disposed) return;
      term = new Terminal({
        fontFamily: "'JetBrains Mono Variable', 'JetBrains Mono', 'Fira Code', Consolas, Menlo, monospace",
        fontSize: 13,
        lineHeight: 1.2,
        cursorBlink: true,
        allowProposedApi: true,
        scrollback: 10000,
        theme: {
          background: "#111111",
          foreground: "#c4c4c4",
          cursor: "#ededed",
          cursorAccent: "#111111",
          selectionBackground: "#3a3a3a"
        }
      });
      fit = new FitAddon();
      term.loadAddon(fit);
      term.open(host);
      /* Ctrl+Shift+C/V copiam e colam; Ctrl+C com seleção copia em vez de interromper */
      term.attachCustomKeyEventHandler((e) => {
        if (e.type !== "keydown" || !(e.ctrlKey || e.metaKey)) return true;
        const k = e.key.toLowerCase();
        if (k === "c" && (e.shiftKey || term!.hasSelection())) {
          void ClipboardSetText(term!.getSelection());
          term!.clearSelection();
          return false;
        }
        if (k === "v" && e.shiftKey) {
          void ClipboardGetText().then((t) => t && term!.paste(t));
          return false;
        }
        return true;
      });
      term.onData((d) => {
        if (exited) {
          if (d === "\r") void open(true);
          return;
        }
        if (lastSeq >= 0) void api.termWrite(id, d).catch(() => {});
      });
      fit.fit();
      ro.observe(host);
      void open(false);
      if (visible) term.focus();
    });

    return () => {
      disposed = true;
      cancelAnimationFrame(raf);
      ro.disconnect();
      offData();
      offExit();
      term?.dispose();
    };
  });

  $effect(() => {
    if (visible) {
      requestAnimationFrame(() => {
        refit();
        term?.focus();
      });
    }
  });
</script>

<div class="term" class:hidden={!visible} bind:this={host}></div>

<style>
  .term {
    position: absolute;
    inset: 0;
    padding: var(--sp-3) var(--sp-4);
    background: var(--bg);
  }
  .hidden {
    display: none;
  }
  .term :global(.xterm-viewport) {
    background: var(--bg) !important;
  }
</style>
