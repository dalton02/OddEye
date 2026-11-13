<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage } from "./api";

  interface Props {
    url: string;
    author: string;
    onclose: () => void;
  }

  let { url, author, onclose }: Props = $props();

  let html = $state("");
  let loading = $state(true);
  let error = $state("");

  const BLOCKED = "script, iframe, object, embed, style, link, meta, base, form";

  /* O GitHub já entrega o HTML sanitizado; isto é só uma segunda barreira antes do {@html}. */
  function sanitize(raw: string): string {
    const doc = new DOMParser().parseFromString(raw, "text/html");
    doc.querySelectorAll(BLOCKED).forEach((el) => el.remove());
    for (const el of doc.body.querySelectorAll("*")) {
      for (const attr of [...el.attributes]) {
        const name = attr.name.toLowerCase();
        const value = attr.value.trim().toLowerCase();
        if (name.startsWith("on")) el.removeAttribute(attr.name);
        else if ((name === "href" || name === "src" || name === "action") && value.startsWith("javascript:")) el.removeAttribute(attr.name);
      }
    }
    doc.querySelectorAll("video").forEach((v) => {
      v.setAttribute("controls", "");
      v.setAttribute("preload", "metadata");
      v.removeAttribute("autoplay");
    });
    doc.querySelectorAll("img").forEach((img) => img.setAttribute("loading", "lazy"));
    doc.querySelectorAll('input[type="checkbox"]').forEach((c) => c.setAttribute("disabled", ""));
    return doc.body.innerHTML;
  }

  onMount(() => {
    api
      .prDescription()
      .then((h) => (html = h ? sanitize(h) : ""))
      .catch((e) => (error = errorMessage(e)))
      .finally(() => (loading = false));
  });

  /* links abrem no navegador: o webview não pode sair do app */
  function onclick(e: MouseEvent): void {
    const a = (e.target as HTMLElement).closest("a");
    if (!a) return;
    e.preventDefault();
    const href = a.getAttribute("href") ?? "";
    if (href.startsWith("#")) {
      document.getElementById(href.slice(1))?.scrollIntoView();
      return;
    }
    if (/^https?:/i.test(a.href)) void api.openURL(a.href);
  }
</script>

<aside aria-label="Descrição da PR">
  <header>
    <h2>Descrição da PR</h2>
    <button class="close" onclick={onclose} aria-label="Fechar">
      <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M3 3l8 8M11 3l-8 8" /></svg>
    </button>
  </header>

  <div class="scroll">
    <p class="by">Escrita por {author}</p>
    {#if loading}
      <p class="muted">Carregando...</p>
    {:else if error}
      <p class="error-text">{error}</p>
    {:else if !html}
      <p class="muted">O autor não escreveu descrição.</p>
    {:else}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="md" {onclick}>{@html html}</div>
    {/if}
  </div>

  <footer>
    <span class="muted">Imagem ou vídeo não abriu?</span>
    <button class="link" onclick={() => api.openURL(url)}>Abrir no GitHub</button>
  </footer>
</aside>

<style>
  aside {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(720px, 100%);
    z-index: var(--z-drawer);
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--border-strong);
    background: var(--surface);
  }
  header {
    display: flex;
    align-items: center;
    height: 52px;
    padding: 0 var(--sp-2) 0 20px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  h2 {
    flex: 1;
    margin: 0;
    font-size: var(--fs-md);
    font-weight: 600;
  }
  .close {
    width: 36px;
    height: 36px;
    display: grid;
    place-items: center;
    border: none;
    border-radius: var(--r-sm);
    background: none;
    color: var(--text);
    transition: background var(--ease);
  }
  .close:hover {
    background: var(--selected);
  }
  svg {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
  }
  .scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--sp-4) 20px var(--sp-6);
  }
  .by {
    margin: 0 0 var(--sp-4);
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .muted {
    color: var(--muted);
  }
  footer {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    padding: var(--sp-3) 20px;
    border-top: 1px solid var(--border);
    flex-shrink: 0;
    font-size: var(--fs-xs);
  }

  /* Markdown renderizado pelo GitHub */
  .md {
    font-size: var(--fs-sm);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }
  .md :global(> :first-child) {
    margin-top: 0;
  }
  .md :global(h1),
  .md :global(h2),
  .md :global(h3),
  .md :global(h4) {
    margin: var(--sp-5) 0 var(--sp-2);
    font-weight: 600;
    line-height: 1.3;
  }
  .md :global(h1) {
    font-size: var(--fs-lg);
  }
  .md :global(h2) {
    font-size: var(--fs-md);
    padding-bottom: 4px;
    border-bottom: 1px solid var(--border);
  }
  .md :global(h3),
  .md :global(h4) {
    font-size: var(--fs-sm);
  }
  .md :global(p),
  .md :global(ul),
  .md :global(ol),
  .md :global(pre),
  .md :global(table),
  .md :global(blockquote),
  .md :global(details) {
    margin: 0 0 var(--sp-3);
  }
  .md :global(ul),
  .md :global(ol) {
    padding-left: 24px;
  }
  .md :global(li.task-list-item) {
    list-style: none;
  }
  .md :global(li.task-list-item input) {
    margin: 0 6px 0 -20px;
  }
  .md :global(a) {
    color: var(--text);
    text-decoration: underline;
    text-underline-offset: 3px;
    cursor: pointer;
  }
  .md :global(a:hover) {
    color: var(--primary);
  }
  .md :global(img),
  .md :global(video) {
    display: block;
    max-width: 100%;
    height: auto;
    border-radius: var(--r-sm);
  }
  .md :global(video) {
    width: 100%;
    max-height: 480px;
    background: #000;
  }
  .md :global(code) {
    font: var(--fs-xs) var(--mono);
    padding: 1px 4px;
    border-radius: 4px;
    background: var(--surface-2);
  }
  .md :global(pre) {
    padding: var(--sp-3);
    overflow: auto;
    border-radius: var(--r-sm);
    background: var(--surface-2);
  }
  .md :global(pre code) {
    padding: 0;
    background: none;
  }
  .md :global(blockquote) {
    padding-left: var(--sp-3);
    border-left: 3px solid var(--border-strong);
    color: var(--text-2);
  }
  .md :global(table) {
    display: block;
    overflow: auto;
    border-collapse: collapse;
  }
  .md :global(th),
  .md :global(td) {
    padding: 4px 10px;
    border: 1px solid var(--border);
  }
  .md :global(hr) {
    height: 1px;
    margin: var(--sp-4) 0;
    border: none;
    background: var(--border);
  }
  .md :global(summary) {
    cursor: pointer;
  }
</style>
