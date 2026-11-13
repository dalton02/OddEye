<script lang="ts">
  import { onMount } from "svelte";
  import Galaxy from "./lib/Galaxy.svelte";
  /* versão clara da logo (silhueta branca, fundo transparente), gerada a partir de logo.png para o tema escuro */
  import logo from "./assets/logo-light.png";
  import RichText from "./lib/RichText.svelte";
  import { createResolver } from "./lib/paths";
  import CodeView from "./lib/CodeView.svelte";
  import FileDrawer from "./lib/FileDrawer.svelte";
  import ReviewModal from "./lib/ReviewModal.svelte";
  import MusicPlayer from "./lib/MusicPlayer.svelte";
  import Settings from "./lib/Settings.svelte";
  import UpdateModal from "./lib/UpdateModal.svelte";
  import Inbox from "./lib/Inbox.svelte";
  import Setup from "./lib/Setup.svelte";
  import ArchitectView from "./lib/architect/ArchitectView.svelte";
  import { chaptersOf, narrate, ownerOf } from "./lib/architect/chapters";
  import { player } from "./lib/player.svelte";
  import {
    api,
    errorMessage,
    modeLabel,
    severityLabel,
    type Config,
    type Deps,
    type Draft,
    type DraftInput,
    type OpenPR,
    type WatcherSnapshot,
    type FileView,
    type Mode,
    type Progress,
    type ReviewEvent,
    type Session,
    type UpdateInfo
  } from "./lib/api";

  type Phase = "home" | "loading" | "tour";

  let phase = $state<Phase>("home");
  let link = $state("");
  let deps = $state<Deps | null>(null);
  let watcherSnap = $state<WatcherSnapshot | null>(null);
  let settingsOpen = $state(false);
  let updateInfo = $state<UpdateInfo | null>(null);
  let error = $state("");
  let progress = $state<Progress | null>(null);
  let progressLog = $state<string[]>([]);
  let session = $state<Session | null>(null);
  let index = $state(-1);
  let activePath = $state<string | null>(null);
  let file = $state<FileView | null>(null);
  let fileLoading = $state(false);
  let fileError = $state("");
  let drafts = $state<Draft[]>([]);
  let visited = $state(new Set<string>());
  let drawerOpen = $state(false);
  let reviewOpen = $state(false);
  let menuOpen = $state(false);
  let keysOpen = $state(false);
  let celebrate = $state("");
  let recent = $state<string[]>([]);
  let codeView: CodeView | undefined = $state();
  let codeFullscreen = $state(false);
  let showAttention = $state(false);
  /* de onde o usuário veio ao seguir um link de arquivo; undefined = não veio por link (null = visão geral) */
  let backTo = $state<string | null | undefined>(undefined);

  /* modo de revisor da próxima análise; no tour acompanha o modo da sessão aberta */
  let mode = $state<Mode>("code");
  /* modo Arquiteto: capítulo atual, passo do fluxo (-1 = fluxo inteiro), componente clicado no mapa e
     perguntas marcadas para o review */
  let chapterIdx = $state(0);
  let flowStep = $state(-1);
  let selectedNode = $state<string | null>(null);
  let picked = $state<string[]>([]);

  const modes: { value: Mode; short: string; hint: string }[] = [
    {
      value: "code",
      short: "Os arquivos críticos, com o código na tela.",
      hint: "O Claude separa os arquivos críticos e te guia pelo código, arquivo por arquivo."
    },
    {
      value: "architect",
      short: "A abordagem, os trade-offs e os diagramas, sem código.",
      hint: "O Claude explica a abordagem, os trade-offs e a arquitetura com diagramas, sem falar de código."
    }
  ];

  const RECENT_KEY = "oddeye:recent";

  /* primeira execução: avisa que o app vai abrir junto com o sistema (a opção já vem ligada) */
  let autostartNotice = $state<Config | null>(null);

  async function dismissAutostartNotice(keep: boolean): Promise<void> {
    const cfg = autostartNotice;
    if (!cfg) return;
    autostartNotice = null;
    try {
      await api.saveSettings({ ...cfg, launchAtLogin: keep, autostartNoticeSeen: true });
    } catch (e) {
      error = String(e);
    }
  }

  onMount(() => {
    api.checkDeps().then((d) => (deps = d));
    api.pendingUpdate().then((u) => {
      if (u) updateInfo = u;
    });
    api.getSettings().then((v) => {
      if (v.config.launchAtLogin && !v.config.autostartNoticeSeen) autostartNotice = v.config;
      if (phase === "home") mode = v.config.defaultMode;
    });
    void player.init();
    try {
      recent = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
    } catch {
      recent = [];
    }
    api.watcherState().then((s) => (watcherSnap = s));
    const offs = [
      api.onProgress((p) => {
        progress = p;
        if (p.stage !== "thinking" && p.message && progressLog[progressLog.length - 1] !== p.message) {
          progressLog = [...progressLog.slice(-5), p.message];
        }
      }),
      api.onWatcher((s) => (watcherSnap = s)),
      api.onShowSettings(() => (settingsOpen = true)),
      api.onUpdateAvailable((u) => (updateInfo = u)),
      /* clique na notificação ou no menu da bandeja: abre a PR direto no tour (rascunhos ficam em disco) */
      api.onOpenPR((ref) => {
        if (phase === "loading") return;
        link = ref;
        /* PR aberta de fora da janela usa o modo padrão das configurações, que pode ter mudado */
        api
          .getSettings()
          .then((v) => (mode = v.config.defaultMode))
          .catch(() => {})
          .finally(() => void start());
      })
    ];
    return () => offs.forEach((off) => off());
  });

  async function recheckDeps(): Promise<void> {
    deps = await api.checkDeps();
  }

  function openPending(pr: OpenPR): void {
    link = pr.url;
    void start();
  }

  const depsOk = $derived(!!deps && !deps.ghError && !deps.claudeError);
  const tourFiles = $derived(session?.files ?? []);
  const inTour = $derived(activePath !== null && tourFiles.some((f) => f.path === activePath));
  const entry = $derived(tourFiles.find((f) => f.path === activePath));
  const attention = $derived(entry?.attention ?? file?.attention ?? []);
  const blueprint = $derived(session?.mode === "architect" ? session.blueprint : null);
  const chapters = $derived(blueprint ? chaptersOf(blueprint) : []);
  const chapter = $derived(chapters[Math.min(chapterIdx, chapters.length - 1)]);
  const flowSteps = $derived(blueprint && chapter?.kind === "flow" ? blueprint.flows[chapter.flow].steps.length : 0);
  /* cada pergunta escolhida vira um parágrafo, sem marcador de lista */
  const reviewBody = $derived(picked.join("\n\n"));
  /* a narração de arquitetura não cita arquivo, então nada nela vira link */
  const noLinks = (): null => null;

  const skippedReason = $derived(session?.skipped.find((s) => s.path === activePath)?.reason ?? "");

  const resolve = $derived(
    createResolver([
      ...new Set([
        ...(session?.files.map((f) => f.path) ?? []),
        ...(session?.others.map((f) => f.path) ?? []),
        ...(session?.skipped.map((f) => f.path) ?? []),
        ...(session?.repoFiles ?? [])
      ])
    ])
  );

  const references = $derived(
    (entry?.references ?? []).map((r) => ({ ...r, resolved: resolve(r.path) }))
  );

  /* quem citou o arquivo aberto como contexto, para o mascote explicar por que ele importa */
  const citedBy = $derived.by(() => {
    if (!activePath || entry) return null;
    for (const f of tourFiles) {
      const ref = f.references?.find((r) => resolve(r.path) === activePath);
      if (ref) return { from: f.path, why: ref.why };
    }
    return null;
  });

  const speech = $derived.by((): string => {
    if (celebrate) return celebrate;
    if (phase === "loading") {
      if (progress?.stage === "thinking") return "Pensando...";
      return progress?.message ?? "Carregando...";
    }
    if (!session) return "";
    if (blueprint && chapter && activePath === null) {
      return narrate(blueprint, chapter, { step: flowStep, selectedNode, chapters: chapters.length });
    }
    if (activePath === null) {
      if (session.files.length === 0) return session.summary;
      const n = session.files.length;
      return `${session.summary}\n\nDos ${session.totalFiles} arquivos, separei ${n} que merecem sua atenção. Use as setas (ou h/l) para começar o tour.`;
    }
    if (entry?.explanation) return entry.explanation;
    if (citedBy) return `Esse arquivo não foi alterado nesta PR. Eu citei ele em \`${citedBy.from}\`: ${citedBy.why}`;
    if (fileLoading || !file) return "Abrindo o arquivo...";
    if (!file.inPR) return "Esse arquivo não foi alterado nesta PR: está aqui só como contexto. Comentários nele vão no corpo do review.";
    if (skippedReason) return `Esse aqui eu descartei logo de cara (${skippedReason}). Mas fica à vontade para conferir.`;
    if (blueprint) {
      const owner = ownerOf(blueprint, file.path);
      if (owner) return `Esse arquivo faz parte de ${owner.label}. ${owner.role}`;
      return "Esse arquivo foi alterado na PR, mas não pertence a nenhum componente do mapa.";
    }
    return "Esse arquivo não entrou na minha seleção: a mudança dele me pareceu de baixo risco. Mas fica à vontade para revisar.";
  });

  /* reuse: recarrega a PR já aberta (troca de modo) em vez de ler o campo de link */
  async function start(force = false, reuse = false): Promise<void> {
    const url = (session?.url && (force || reuse) ? session.url : link).trim();
    if (!url) return;
    error = "";
    progress = null;
    progressLog = [];
    phase = "loading";
    try {
      const s = await api.loadPR(url, mode, force);
      session = s;
      mode = s.mode;
      chapterIdx = 0;
      flowStep = -1;
      selectedNode = null;
      picked = [];
      drafts = s.drafts;
      visited = new Set();
      index = -1;
      activePath = null;
      file = null;
      phase = "tour";
      recent = [s.url, ...recent.filter((r) => r !== s.url)].slice(0, 5);
      try {
        localStorage.setItem(RECENT_KEY, JSON.stringify(recent));
      } catch {
        /* armazenamento indisponível: a lista de recentes é só conveniência */
      }
    } catch (e) {
      error = errorMessage(e);
      phase = session && (force || reuse) ? "tour" : "home";
      if (session && phase === "tour") mode = session.mode;
    }
  }

  function switchMode(m: Mode): void {
    if (m === mode) return;
    mode = m;
    if (phase === "tour") void start(false, true);
  }

  /* Modo Arquiteto: as setas andam pelos passos de um fluxo antes de trocar de capítulo. */
  function goChapter(delta: number): void {
    if (!blueprint) return;
    celebrate = "";
    if (activePath !== null) {
      /* estava num arquivo aberto a partir do mapa: volta para o capítulo */
      activePath = null;
      index = -1;
      file = null;
      backTo = undefined;
      return;
    }
    if (chapter?.kind === "flow") {
      const next = flowStep + delta;
      if (next >= -1 && next < flowSteps) {
        flowStep = next;
        return;
      }
    }
    jumpChapter(chapterIdx + delta, delta < 0);
  }

  function jumpChapter(target: number, fromEnd = false): void {
    if (!blueprint || target < 0) return;
    if (target >= chapters.length) {
      reviewOpen = true;
      return;
    }
    celebrate = "";
    chapterIdx = target;
    const c = chapters[target];
    /* voltando para um fluxo, cai no último passo: a seta para trás desfaz exatamente a seta para a frente */
    flowStep = fromEnd && c.kind === "flow" ? blueprint.flows[c.flow].steps.length - 1 : -1;
  }

  function followLink(path: string): void {
    if (path === activePath) return;
    backTo = activePath;
    openPath(path, true);
  }

  function goBack(): void {
    const target = backTo;
    backTo = undefined;
    if (target === null || target === undefined) go(-index - 1);
    else openPath(target);
  }

  async function openPath(path: string, viaLink = false): Promise<void> {
    if (!viaLink) backTo = undefined;
    activePath = path;
    index = tourFiles.findIndex((f) => f.path === path);
    drawerOpen = false;
    fileLoading = true;
    fileError = "";
    celebrate = "";
    try {
      const f = await api.openFile(path);
      if (activePath !== path) return;
      file = f;
      visited = new Set([...visited, path]);
    } catch (e) {
      fileError = errorMessage(e);
    } finally {
      fileLoading = false;
    }
  }

  function go(delta: number): void {
    if (!session || reviewOpen) return;
    if (blueprint) {
      goChapter(delta);
      return;
    }
    const base = inTour ? index : -1;
    const target = base + delta;
    if (target < 0) {
      activePath = null;
      index = -1;
      file = null;
      backTo = undefined;
      return;
    }
    if (target >= tourFiles.length) {
      reviewOpen = true;
      return;
    }
    openPath(tourFiles[target].path);
  }

  async function saveDraft(d: DraftInput): Promise<void> {
    drafts = await api.saveDraft(d);
  }

  async function removeDraft(id: string): Promise<void> {
    drafts = await api.removeDraft(id);
  }

  async function submitReview(event: ReviewEvent, body: string): Promise<void> {
    const url = await api.submitReview(event, body);
    drafts = [];
    picked = [];
    reviewOpen = false;
    celebrate = "Review publicado no GitHub! Obrigado por revisar com atenção. Clique no link no topo para ver a PR.";
    lastReviewUrl = url;
  }

  let lastReviewUrl = $state("");

  function jumpToDraft(d: Draft): void {
    reviewOpen = false;
    openPath(d.path).then(() => setTimeout(() => codeView?.revealLine(d.line), 150));
  }

  function newPR(): void {
    phase = "home";
    session = null;
    file = null;
    activePath = null;
    celebrate = "";
    error = "";
    link = "";
  }

  const TOUR_STEP: Record<string, -1 | 1> = { ArrowLeft: -1, h: -1, k: -1, ArrowRight: 1, l: 1, j: 1 };

  $effect(() => {
    if (!codeView) codeFullscreen = false;
  });

  function onKey(e: KeyboardEvent): void {
    if (phase !== "tour" || reviewOpen || settingsOpen || updateInfo) return;
    const el = document.activeElement as HTMLElement | null;
    const editorFocused = !!el?.classList.contains("inputarea") && !!el.closest(".monaco-editor");
    if (!editorFocused && (el?.tagName === "INPUT" || el?.tagName === "TEXTAREA")) return;
    const inCode = editorFocused || codeFullscreen;
    const step = TOUR_STEP[e.key];
    if (step && !e.ctrlKey && !e.metaKey && (e.altKey || !inCode)) {
      e.preventDefault();
      go(step);
    } else if (e.key === "Enter" && codeView && (!el || el === document.body || editorFocused)) {
      e.preventDefault();
      codeFullscreen = true;
      codeView.focus();
    } else if ((e.key === "c" || e.key === "C") && file && !e.ctrlKey && !e.metaKey) {
      e.preventDefault();
      codeView?.commentSelectionOrCursor();
    } else if (e.key === "Escape") {
      if (menuOpen || keysOpen) closePopovers();
      else if (drawerOpen) drawerOpen = false;
      else if (inCode) {
        codeFullscreen = false;
        el?.blur();
      }
    }
  }

  function closePopovers(): void {
    menuOpen = false;
    keysOpen = false;
  }

  function short(path: string): string {
    return path.split("/").pop() ?? path;
  }
</script>

<svelte:window onkeydown={onKey} onclick={closePopovers} />

{#if phase !== "tour"}<Galaxy />{/if}

<div class="app">
  {#if phase !== "tour"}
    <div class="home-player">
      <MusicPlayer />
      <button class="btn square" onclick={() => (settingsOpen = true)} title="Configurações" aria-label="Configurações">
        <svg class="gear" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>
      </button>
    </div>
  {/if}
  {#if phase === "home"}
    <main class="home">
      <div class="brand">
        <img class="logo" src={logo} width="44" height="44" alt="" />
        <h1>Oddeye</h1>
      </div>

      <div class="start">
        <form
          class="search"
          onsubmit={(e) => {
            e.preventDefault();
            start();
          }}
        >
          <input
            class="field mono"
            bind:value={link}
            aria-label="Link da pull request"
            placeholder="https://github.com/org/repo/pull/123"
            spellcheck="false"
            autocomplete="off"
          />
          <button class="btn primary" type="submit" disabled={!link.trim() || !depsOk}>Analisar</button>
        </form>
        <div class="mode-pick">
          <div class="seg" role="group" aria-label="Modo de revisor">
            {#each modes as m (m.value)}
              <button class:on={mode === m.value} aria-pressed={mode === m.value} title={m.hint} onclick={() => switchMode(m.value)}>
                {modeLabel[m.value]}
              </button>
            {/each}
          </div>
          <span>{modes.find((m) => m.value === mode)?.short}</span>
        </div>
        {#if error}<p class="error-text">{error}</p>{/if}
      </div>

      {#if deps && (!depsOk || deps.missingScopes.length > 0)}
        <Setup {deps} onrecheck={recheckDeps} />
      {/if}

      {#if depsOk}
        <Inbox
          snapshot={watcherSnap}
          onopen={openPending}
          oncheck={() => api.checkNow()}
          onsettings={() => (settingsOpen = true)}
        />
      {/if}

      {#if autostartNotice}
        <div class="notice" role="status">
          <span>O Oddeye abre junto com o sistema, direto na bandeja, para avisar de PR nova. Dá para mudar depois em Configurações.</span>
          <button class="link" onclick={() => dismissAutostartNotice(false)}>Não abrir com o sistema</button>
          <button class="link" onclick={() => dismissAutostartNotice(true)}>Entendi</button>
        </div>
      {/if}

      {#if recent.length}
        <div class="recent">
          <span>Recentes</span>
          {#each recent as r}
            <button onclick={() => (link = r)}>{r.replace("https://github.com/", "")}</button>
          {/each}
        </div>
      {/if}
    </main>
  {:else if phase === "loading"}
    <main class="loading">
      <div class="status">
        <span class="mono label">{link.replace("https://github.com/", "")}</span>
        <h1>{speech}</h1>
        {#if progress?.stage === "thinking" && progress.message}
          <p class="thought">{progress.message}</p>
        {/if}
        {#if progress?.files && mode === "code"}
          <span class="count">{progress.files} arquivo{progress.files === 1 ? "" : "s"} explicado{progress.files === 1 ? "" : "s"}</span>
        {/if}
      </div>
      {#if progressLog.length}
        <ul class="log">
          {#each progressLog as l, i (l + i)}
            <li>{l}</li>
          {/each}
        </ul>
      {/if}
      <div><button class="btn" onclick={() => api.cancel()}>Cancelar</button></div>
    </main>
  {:else if session}
    <header class="topbar">
      <button class="btn" onclick={() => (drawerOpen = !drawerOpen)} title="Lista de arquivos">
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M2 3.5h10M2 7h10M2 10.5h10" /></svg>
        Arquivos
      </button>
      <div class="pr">
        <button class="pr-title" onclick={() => api.openURL(lastReviewUrl || session!.url)} title="Abrir no GitHub">
          {session.title}
        </button>
        <span class="pr-ref">{session.ref.owner}/{session.ref.repo}#{session.ref.number}</span>
      </div>
      <span class="mode-name">Modo {modeLabel[mode]}</span>
      <div class="pop-anchor">
        <button
          class="btn square"
          class:on={menuOpen}
          aria-label="Mais ações"
          aria-expanded={menuOpen}
          onclick={(e) => {
            e.stopPropagation();
            keysOpen = false;
            menuOpen = !menuOpen;
          }}
        >
          <svg class="dots-icon" viewBox="0 0 14 14" aria-hidden="true"><circle cx="2.5" cy="7" r="1.2" /><circle cx="7" cy="7" r="1.2" /><circle cx="11.5" cy="7" r="1.2" /></svg>
        </button>
        {#if menuOpen}
          <div class="menu" role="menu">
            <span class="label">Modo do revisor</span>
            {#each modes as m (m.value)}
              <button role="menuitemradio" aria-checked={mode === m.value} class:on={mode === m.value} title={m.hint} onclick={() => switchMode(m.value)}>
                <svg viewBox="0 0 14 14" aria-hidden="true">{#if mode === m.value}<path d="M2.5 7.5l3 3 6-7" />{/if}</svg>
                {modeLabel[m.value]}
              </button>
            {/each}
            <hr />
            {#if session.fromCache}
              <button role="menuitem" class="plain" onclick={() => start(true)} title="A análise veio do cache para este commit">Reanalisar</button>
            {/if}
            <button role="menuitem" class="plain" onclick={() => api.openURL(lastReviewUrl || session!.url)}>Abrir no GitHub</button>
            <button role="menuitem" class="plain" onclick={newPR}>Nova PR</button>
            <hr />
            <button role="menuitem" onclick={() => (settingsOpen = true)}>
              <svg class="gear" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>
              Configurações
            </button>
          </div>
        {/if}
      </div>
    </header>

    <section class="brief">
      <!-- na visão geral do Arquiteto o título fica na própria tela, então a faixa só traz o texto -->
      {#if !(blueprint && activePath === null && chapterIdx === 0 && backTo === undefined)}
      <div class="brief-head">
        {#if blueprint && chapter && activePath === null}
          <h2>{chapter.kind === "flow" ? "Fluxo: " : ""}{chapter.title}</h2>
        {:else if activePath}
          {@const severity = entry?.severity ?? ""}
          {#if !blueprint}<span class="sev {severity || 'none'}">{severityLabel[severity]}</span>{/if}
          <span class="fname" title={activePath}>{activePath}</span>
          {#if entry?.why}<span class="why">{entry.why}</span>{/if}
        {:else}
          <span class="label">
            {session.author} · {session.headBranch} → {session.baseBranch} ·
            <span class="add">+{session.additions}</span> <span class="del">−{session.deletions}</span>
          </span>
        {/if}
        {#if backTo !== undefined}
          <button class="link back" onclick={goBack}>
            Voltar para {backTo === null ? (blueprint ? "o " + (chapter?.title ?? "tour").toLowerCase() : "a visão geral") : short(backTo)}
          </button>
        {/if}
      </div>
      {/if}
      <p class="speech">
        <RichText text={speech} resolve={blueprint && activePath === null ? noLinks : resolve} onlink={followLink} />
      </p>
      {#if activePath && (attention.length || references.length)}
        <div class="extras">
          {#if attention.length}
            <button class="link attention" aria-expanded={showAttention} onclick={() => (showAttention = !showAttention)}>
              {attention.length} ponto{attention.length === 1 ? "" : "s"} de atenção
            </button>
          {/if}
          {#if references.length}
            <span class="refs">
              Citado:
              {#each references as r (r.path)}
                {#if r.resolved}
                  <button class="link mono" title={r.why} onclick={() => followLink(r.resolved!)}>{short(r.resolved)}</button>
                {:else}
                  <span class="mono missing" title="Não encontrei este arquivo no repositório. {r.why}">{r.path}</span>
                {/if}
              {/each}
            </span>
          {/if}
        </div>
        {#if showAttention && attention.length}
          <ul class="notes">
            {#each attention as a}
              <li>
                <button class="link mono" onclick={() => codeView?.revealLine(a.line)} title="Ir para a linha {a.line}">L{a.line}</button>
                <span><RichText text={a.note} {resolve} onlink={followLink} /></span>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </section>

    <section class="stage">
      {#if blueprint && chapter && activePath === null}
        <div class="pane arch">
          <ArchitectView
            {blueprint}
            {session}
            {chapters}
            {chapter}
            bind:step={flowStep}
            bind:selectedNode
            bind:picked
            onjump={jumpChapter}
            onopenfile={followLink}
            onreview={() => (reviewOpen = true)}
          />
        </div>
      {:else if activePath === null}
        <div class="pane overview">
          {#if session.files.length === 0}
            <p class="empty">Nada crítico nesta PR. Abra a lista de arquivos se quiser revisar mesmo assim.</p>
          {:else}
            <ul class="files">
              {#each session.files as f (f.path)}
                <li>
                  <button onclick={() => openPath(f.path)}>
                    <span class="sev {f.severity}">{severityLabel[f.severity]}</span>
                    <span class="file-main">
                      <span class="mono file-path">{f.path}</span>
                      <span class="file-why">{f.why}</span>
                    </span>
                    <span class="mono diff"><span class="add">+{f.additions}</span> <span class="del">−{f.deletions}</span></span>
                  </button>
                </li>
              {/each}
            </ul>
          {/if}
          <p class="foot">
            {session.skipped.length} descartados (testes, docs, gerados) · {session.others.length} alterados sem risco relevante
          </p>
        </div>
      {:else if fileError}
        <div class="pane overview"><p class="error-text">{fileError}</p></div>
      {:else if file}
        <CodeView
          bind:this={codeView}
          {file}
          {drafts}
          fullscreen={codeFullscreen}
          onsavedraft={saveDraft}
          onremovedraft={removeDraft}
        />
      {:else}
        <div class="pane overview"><p class="empty">Carregando...</p></div>
      {/if}
    </section>

    <footer class="bottombar">
      <button
        class="btn square"
        onclick={() => go(-1)}
        disabled={activePath === null && (!blueprint || (chapterIdx === 0 && flowStep < 0))}
        aria-label="Anterior"
        title="Anterior"
      >
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M8.5 3L4.5 7l4 4" /></svg>
      </button>
      <button class="btn square" onclick={() => go(1)} aria-label="Próximo" title="Próximo">
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M5.5 3l4 4-4 4" /></svg>
      </button>
      <span class="pos">
        {#if blueprint && activePath === null}
          {chapterIdx === 0 ? "Visão geral" : `Capítulo ${chapterIdx} de ${chapters.length - 1}`}
        {:else if blueprint}
          Código
        {:else if activePath === null}
          Visão geral
        {:else if inTour}
          Arquivo {index + 1} de {tourFiles.length}
        {:else}
          Fora do tour
        {/if}
      </span>
      <div class="steps">
        {#if blueprint}
          {#each chapters as c, i (i)}
            <button
              class="step"
              class:seen={i < chapterIdx}
              class:current={activePath === null && i === chapterIdx}
              onclick={() => {
                if (activePath !== null) goChapter(0);
                jumpChapter(i);
              }}
              title="{i === 0 ? '' : i + '. '}{c.title}"
              aria-label="{i === 0 ? '' : i + '. '}{c.title}"
            ></button>
          {/each}
        {:else}
          <button
            class="step"
            class:seen={activePath !== null}
            class:current={activePath === null}
            onclick={() => go(-index - 1)}
            title="Visão geral"
            aria-label="Visão geral"
          ></button>
        {/if}
        {#each tourFiles as f, i (f.path)}
          <button
            class="step"
            class:seen={visited.has(f.path)}
            class:current={activePath === f.path}
            onclick={() => openPath(f.path)}
            title="{i + 1}. {f.path}"
            aria-label="{i + 1}. {f.path}"
          ></button>
        {/each}
      </div>
      <span class="grow"></span>
      <div class="pop-anchor">
        <button
          class="btn square"
          class:on={keysOpen}
          aria-label="Atalhos de teclado"
          aria-expanded={keysOpen}
          title="Atalhos de teclado"
          onclick={(e) => {
            e.stopPropagation();
            menuOpen = false;
            keysOpen = !keysOpen;
          }}>?</button
        >
        {#if keysOpen}
          <dl class="keys">
            <div><dt>← → ou h j k l</dt><dd>Anterior e próximo</dd></div>
            <div><dt>Enter</dt><dd>Código em tela cheia</dd></div>
            <div><dt>Esc</dt><dd>Sair da tela cheia</dd></div>
            <div><dt>C</dt><dd>Comentar a linha ou seleção</dd></div>
            <div><dt>Alt + ← →</dt><dd>Trocar de arquivo dentro do código</dd></div>
          </dl>
        {/if}
      </div>
      <MusicPlayer />
      <button class="btn primary" onclick={() => (reviewOpen = true)}>
        Enviar review{drafts.length ? ` (${drafts.length})` : ""}
      </button>
    </footer>

    {#if drawerOpen}
      <FileDrawer {session} {activePath} {visited} {drafts} onopen={openPath} onclose={() => (drawerOpen = false)} />
    {/if}

    {#if reviewOpen}
      <ReviewModal
        {drafts}
        prLabel="{session.ref.owner}/{session.ref.repo}#{session.ref.number}: {session.title}"
        onsubmit={submitReview}
        onclose={() => (reviewOpen = false)}
        onjump={jumpToDraft}
        initialBody={reviewBody}
      />
    {/if}

    {#if error}
      <div class="toast" role="alert">
        <span>{error}</span>
        <button onclick={() => (error = "")} aria-label="Fechar aviso">
          <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M3 3l8 8M11 3l-8 8" /></svg>
        </button>
      </div>
    {/if}
  {/if}
  {#if settingsOpen}
    <Settings onclose={() => (settingsOpen = false)} />
  {/if}
  {#if updateInfo}
    <UpdateModal info={updateInfo} onclose={() => (updateInfo = null)} />
  {/if}
</div>

<style>
  .app {
    position: relative;
    z-index: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .mono {
    font-family: var(--mono);
  }
  .grow {
    flex: 1;
  }

  .home-player {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    position: fixed;
    top: var(--sp-4);
    right: var(--sp-5);
    z-index: 10;
  }

  /* Home e análise: uma coluna estreita, alinhada à esquerda */
  .home,
  .loading {
    width: min(640px, 100%);
    margin: 0 auto;
    padding: 108px var(--sp-5) var(--sp-6);
    display: flex;
    flex-direction: column;
    gap: var(--sp-6);
    overflow: auto;
    /* fundo sólido na coluna: as estrelas passam pelas laterais, nunca por trás do texto */
    background: var(--bg);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }
  .logo {
    display: block;
    flex-shrink: 0;
  }
  h1 {
    margin: 0;
    font-size: var(--fs-xl);
    font-weight: 600;
    line-height: 1.2;
  }
  .start {
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
  }
  .search {
    display: flex;
    gap: var(--sp-2);
  }
  .search .field,
  .search .btn {
    height: 44px;
  }
  .search .field {
    flex: 1;
    min-width: 0;
    background: var(--surface);
  }
  .search .btn {
    padding: 0 20px;
  }
  .mode-pick {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--sp-3);
    color: var(--muted);
  }
  .error-text {
    margin: 0;
  }
  .notice {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-2) var(--sp-4);
    color: var(--muted);
  }
  .notice span {
    flex: 1 1 320px;
  }
  .recent {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-2) var(--sp-4);
    color: var(--muted);
  }
  .recent button {
    padding: 0;
    border: none;
    background: none;
    color: var(--muted);
    font: var(--fs-xs) var(--mono);
    transition: color var(--ease);
  }
  .recent button:hover {
    color: var(--text);
  }

  /* Análise */
  .loading {
    padding-top: 212px;
    gap: var(--sp-5);
  }
  .status {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .status h1 {
    font-size: var(--fs-lg);
    line-height: 1.4;
  }
  .thought {
    margin: 0;
    color: var(--text-2);
  }
  .count {
    color: var(--muted);
  }
  .log {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font: var(--fs-xs) var(--mono);
    color: var(--muted);
  }
  .log li:last-child {
    color: var(--text);
  }

  /* Tour: barra do topo */
  .topbar {
    display: flex;
    align-items: center;
    gap: var(--sp-4);
    height: 52px;
    padding: 0 var(--sp-4);
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .pr {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
  }
  .pr-title {
    min-width: 0;
    padding: 0;
    border: none;
    background: none;
    color: var(--text);
    font-size: var(--fs-sm);
    font-weight: 600;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pr-title:hover {
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .pr-ref {
    font: var(--fs-xs) var(--mono);
    color: var(--muted);
    white-space: nowrap;
  }
  .mode-name {
    color: var(--muted);
    white-space: nowrap;
  }
  .btn.on {
    background: var(--selected);
    border-color: var(--primary);
  }
  /* o ícone vem numa grade de 24: traço mais grosso para igualar o peso dos ícones de 14 */
  svg.gear {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .dots-icon {
    fill: currentColor;
    stroke: none;
  }

  /* Menus suspensos */
  .pop-anchor {
    position: relative;
    display: flex;
  }
  .menu,
  .keys {
    position: absolute;
    right: 0;
    z-index: var(--z-popover);
    margin: 0;
    border: 1px solid var(--border-strong);
    border-radius: var(--r-md);
    background: var(--surface);
    box-shadow: var(--shadow);
  }
  .menu {
    top: calc(100% + 8px);
    width: 240px;
    padding: 6px;
    display: flex;
    flex-direction: column;
  }
  .menu .label {
    padding: var(--sp-2) 10px var(--sp-1);
  }
  .menu button {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    height: 36px;
    padding: 0 10px;
    border: none;
    border-radius: var(--r-sm);
    background: none;
    color: var(--text);
    font-size: var(--fs-sm);
    text-align: left;
    transition: background var(--ease);
  }
  .menu button:hover,
  .menu button.on {
    background: var(--selected);
  }
  .menu svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .menu hr {
    width: 100%;
    height: 1px;
    margin: 6px 0;
    border: none;
    background: var(--border);
  }
  .keys {
    bottom: calc(100% + 8px);
    width: 340px;
    padding: var(--sp-2) var(--sp-4);
  }
  .keys div {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
    padding: var(--sp-2) 0;
  }
  .keys div + div {
    border-top: 1px solid var(--border);
  }
  .keys dt {
    width: 120px;
    flex-shrink: 0;
    font: var(--fs-xs) var(--mono);
  }
  .keys dd {
    margin: 0;
    color: var(--text-2);
  }

  /* Tour: explicação do revisor */
  .brief {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    padding: 20px var(--sp-5);
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
    /* o texto aparece inteiro; só um resumo muito longo rola, e aí a barra fica na borda da janela */
    max-height: 40vh;
    overflow-y: auto;
  }
  .brief-head {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    min-width: 0;
  }
  .brief h2 {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 600;
    line-height: 1.4;
  }
  .fname {
    font-family: var(--mono);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    direction: rtl;
    text-align: left;
    flex-shrink: 1;
    min-width: 80px;
  }
  .why {
    flex: 1;
    min-width: 0;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .back {
    margin-left: auto;
    white-space: nowrap;
  }
  .speech {
    margin: 0;
    max-width: 880px;
    font-size: var(--fs-md);
    white-space: pre-wrap;
  }
  .extras {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-2) var(--sp-5);
    color: var(--muted);
  }
  .attention::before {
    content: "";
    display: inline-block;
    width: 8px;
    height: 8px;
    margin-right: 6px;
    border-radius: 50%;
    background: var(--yellow);
  }
  .refs {
    display: inline-flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-1) var(--sp-3);
  }
  .refs .mono {
    font-size: var(--fs-xs);
  }
  .missing {
    text-decoration: line-through;
    cursor: help;
  }
  .notes {
    list-style: none;
    margin: 0;
    padding: 0;
    max-width: 880px;
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .notes li {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    color: var(--text-2);
  }
  .notes .link {
    flex-shrink: 0;
    font-size: var(--fs-xs);
    color: var(--yellow);
  }

  /* Tour: conteúdo */
  .stage {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .pane {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--sp-6) var(--sp-5);
  }
  /* as telas do Arquiteto têm rolagem própria: aqui só entra a margem */
  .pane.arch {
    overflow: hidden;
    padding-bottom: 0;
  }
  .overview {
    display: flex;
    flex-direction: column;
    gap: var(--sp-5);
    width: min(880px + 2 * var(--sp-5), 100%);
    margin: 0 auto;
  }
  .files {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .files li {
    border-top: 1px solid var(--border);
  }
  .files li:last-child {
    border-bottom: 1px solid var(--border);
  }
  .files button {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
    width: 100%;
    padding: 14px var(--sp-2);
    border: none;
    background: none;
    color: var(--text);
    font-size: var(--fs-sm);
    text-align: left;
    transition: background var(--ease);
  }
  .files button:hover {
    background: var(--surface);
  }
  .files .sev {
    width: 72px;
    flex-shrink: 0;
  }
  .file-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .file-path {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .file-why {
    color: var(--muted);
  }
  .diff {
    font-size: var(--fs-xs);
    white-space: nowrap;
  }
  .foot,
  .empty {
    margin: 0;
    color: var(--muted);
  }

  /* Tour: rodapé */
  .bottombar {
    display: flex;
    align-items: center;
    gap: var(--sp-4);
    height: 56px;
    padding: 0 var(--sp-4);
    border-top: 1px solid var(--border);
    flex-shrink: 0;
  }
  .bottombar > .btn.square + .btn.square {
    margin-left: calc(var(--sp-2) - var(--sp-4));
  }
  .pos {
    font-weight: 600;
    white-space: nowrap;
  }
  .steps {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    min-width: 0;
  }
  /* o botão é mais alto que a barra desenhada, para o alvo do clique não ter 4 px */
  .step {
    width: 24px;
    min-width: 6px;
    flex-shrink: 1;
    height: 20px;
    padding: 0;
    border: none;
    background: none;
  }
  .step::before {
    content: "";
    display: block;
    height: 4px;
    border-radius: 2px;
    background: var(--border-strong);
    transition: background var(--ease);
  }
  .step.seen::before {
    background: var(--muted);
  }
  .step.current::before,
  .step:hover::before {
    background: var(--primary);
  }

  .toast {
    position: fixed;
    bottom: 72px;
    left: 50%;
    translate: -50% 0;
    z-index: var(--z-toast);
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    max-width: min(560px, 80vw);
    padding: 10px var(--sp-2) 10px var(--sp-4);
    border: 1px solid var(--red);
    border-radius: var(--r-sm);
    background: var(--surface);
  }
  .toast button {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    display: grid;
    place-items: center;
    border: none;
    border-radius: 4px;
    background: none;
    color: var(--text);
  }
  .toast svg {
    width: 12px;
    height: 12px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
  }
</style>
