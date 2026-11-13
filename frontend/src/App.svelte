<script lang="ts">
  import { onMount } from "svelte";
  import Galaxy from "./lib/Galaxy.svelte";
  import logo from "./assets/logo.png";
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
  import { chaptersOf, narrate, ownerOf, prKindLabel } from "./lib/architect/chapters";
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
  let typing = $state(false);
  let revisit = $state(false);
  let celebrate = $state("");
  let recent = $state<string[]>([]);
  let codeView: CodeView | undefined = $state();
  let codeFullscreen = $state(false);
  let richText: RichText | undefined = $state();
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

  const modes: { value: Mode; hint: string }[] = [
    { value: "code", hint: "O Claude separa os arquivos críticos e te guia pelo código, arquivo por arquivo." },
    { value: "architect", hint: "O Claude explica a abordagem, os trade-offs e a arquitetura com diagramas, sem falar de código." }
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
      if (progress?.stage === "thinking") return progress.message || "Pensando...";
      return progress?.message ?? "Preparando a nave...";
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
    revisit = visited.has(path);
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
      if (drawerOpen) drawerOpen = false;
      else if (inCode) {
        codeFullscreen = false;
        el?.blur();
      }
    }
  }

  function short(path: string): string {
    return path.split("/").pop() ?? path;
  }
</script>

<svelte:window onkeydown={onKey} />

<Galaxy dim={phase === "tour"} />

<div class="app">
  {#if phase !== "tour"}
    <div class="home-player"><MusicPlayer /></div>
  {/if}
  {#if phase === "home"}
    <main class="home">
      <img class="logo" src={logo} width="150" height="150" alt="" />
      <h1>Pull<span>sar</span></h1>
      <p class="tagline">Cole o link de uma pull request. O Claude separa só o que importa e te guia arquivo por arquivo.</p>

      <form
        class="search"
        onsubmit={(e) => {
          e.preventDefault();
          start();
        }}
      >
        <input bind:value={link} placeholder="https://github.com/org/repo/pull/123" spellcheck="false" autocomplete="off" />
        <button class="btn primary" type="submit" disabled={!link.trim() || !depsOk}>Decolar</button>
      </form>

      <div class="mode-pick">
        <div class="modes" role="group" aria-label="Modo de revisor">
          {#each modes as m (m.value)}
            <button class:on={mode === m.value} aria-pressed={mode === m.value} onclick={() => switchMode(m.value)}>{modeLabel[m.value]}</button>
          {/each}
        </div>
        <span>{modes.find((m) => m.value === mode)?.hint}</span>
      </div>

      {#if error}<p class="error">{error}</p>{/if}

      {#if autostartNotice}
        <div class="notice" role="status">
          <p>O Oddeye vai abrir junto com o sistema, direto na bandeja, para avisar quando chegar PR nova. Dá para mudar depois em Configurações.</p>
          <div class="notice-actions">
            <button class="btn" onclick={() => dismissAutostartNotice(false)}>Não abrir com o sistema</button>
            <button class="btn primary" onclick={() => dismissAutostartNotice(true)}>Entendi</button>
          </div>
        </div>
      {/if}

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
      <div class="loading-row">
        <img class="logo" src={logo} width="140" height="140" alt="" />
        <div class="bubble big">
          <p class="speech">{speech}</p>
          {#if progress?.files && mode === "code"}
            <p class="count">{progress.files} arquivo{progress.files === 1 ? "" : "s"} explicado{progress.files === 1 ? "" : "s"}</p>
          {/if}
        </div>
      </div>
      <ul class="log">
        {#each progressLog as l, i (l + i)}
          <li>{l}</li>
        {/each}
      </ul>
      <button class="btn" onclick={() => api.cancel()}>Cancelar</button>
    </main>
  {:else if session}
    <header class="topbar">
      <button class="btn icon" onclick={() => (drawerOpen = !drawerOpen)} title="Lista de arquivos">☰ Arquivos</button>
      <div class="pr">
        <button class="pr-title" onclick={() => api.openURL(lastReviewUrl || session!.url)} title="Abrir no GitHub">
          {session.title}
        </button>
        <span class="pr-meta">
          {session.ref.owner}/{session.ref.repo}#{session.ref.number} · {session.author} · {session.headBranch} → {session.baseBranch}
          · <b class="add">+{session.additions}</b> <b class="del">-{session.deletions}</b>
        </span>
      </div>
      <div class="modes" role="group" aria-label="Modo de revisor">
        {#each modes as m (m.value)}
          <button class:on={mode === m.value} aria-pressed={mode === m.value} title={m.hint} onclick={() => switchMode(m.value)}>
            {modeLabel[m.value]}
          </button>
        {/each}
      </div>
      {#if session.fromCache}
        <button class="btn" onclick={() => start(true)} title="A análise veio do cache para este commit">Reanalisar</button>
      {/if}
      <button class="btn" onclick={newPR}>Nova PR</button>
    </header>

    <section class="claude">
      <img class="logo" src={logo} width="104" height="104" alt="" />
      <div class="bubble">
        <div class="bubble-head">
          {#if blueprint && chapter && activePath === null}
            <span class="sev none">{chapterIdx === 0 ? (prKindLabel[blueprint.kind] ?? "Arquitetura") : `Capítulo ${chapterIdx}`}</span>
            <span class="fname plain">{chapter.kind === "flow" ? "Fluxo: " : ""}{chapter.title}</span>
            {#if chapter.kind === "flow" && flowStep >= 0}<span class="why">passo {flowStep + 1} de {flowSteps}</span>{/if}
          {:else if activePath}
            {@const severity = entry?.severity ?? ""}
            {#if !blueprint}<span class="sev {severity || 'none'}">{severityLabel[severity]}</span>{/if}
            <span class="fname" title={activePath}>{entry ? short(activePath) : activePath}</span>
            {#if entry?.why}<span class="why">{entry.why}</span>{/if}
          {:else if activePath === null}
            <span class="fname">Visão geral da PR</span>
          {/if}
          {#if backTo !== undefined}
            <button class="back" onclick={goBack}>
              ← voltar para {backTo === null ? (blueprint ? "o " + (chapter?.title ?? "tour").toLowerCase() : "a visão geral") : short(backTo)}
            </button>
          {/if}
        </div>
        <div class="speech-box" role="presentation" onclick={() => richText?.finish()}>
          <p class="speech">
            <RichText
              bind:this={richText}
              text={speech}
              resolve={blueprint && activePath === null ? noLinks : resolve}
              onlink={followLink}
              instant={revisit && !blueprint}
              ontyping={(t) => (typing = t)}
            />
          </p>
        </div>
        {#if activePath && (attention.length || references.length)}
          <div class="extras">
            {#if attention.length}
              <button class="toggle" class:open={showAttention} onclick={() => (showAttention = !showAttention)}>
                <span class="warn-dot">!</span>
                {attention.length} ponto{attention.length === 1 ? "" : "s"} de atenção
                <span class="caret">{showAttention ? "▾" : "▸"}</span>
              </button>
            {/if}
            {#each references as r (r.path)}
              {#if r.resolved}
                <button class="ref" title={r.why} onclick={() => followLink(r.resolved!)}>
                  <span class="ref-label">citado</span>{short(r.resolved)}
                </button>
              {:else}
                <span class="ref missing" title="Não encontrei este arquivo no repositório. {r.why}">
                  <span class="ref-label">citado</span>{r.path}
                </span>
              {/if}
            {/each}
          </div>
          {#if showAttention && attention.length}
            <div class="chips">
              {#each attention as a}
                <div class="chip">
                  <button class="line" onclick={() => codeView?.revealLine(a.line)} title="Ir para a linha {a.line}">L{a.line}</button>
                  <span><RichText text={a.note} {resolve} onlink={followLink} instant /></span>
                </div>
              {/each}
            </div>
          {/if}
        {/if}
      </div>
    </section>

    <section class="stage">
      <button
        class="arrow"
        onclick={() => go(-1)}
        disabled={activePath === null && (!blueprint || (chapterIdx === 0 && flowStep < 0))}
        aria-label="Anterior"
      >
        <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7" /></svg>
      </button>

      <div class="center">
        {#if blueprint && chapter && activePath === null}
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
        {:else if activePath === null}
          <div class="overview">
            {#if session.files.length === 0}
              <p class="empty">Nada crítico nesta PR. Abra a lista de arquivos se quiser revisar mesmo assim.</p>
            {/if}
            {#each session.files as f, i (f.path)}
              <button class="card" onclick={() => openPath(f.path)} style="animation-delay: {i * 50}ms">
                <span class="idx">{i + 1}</span>
                <span class="card-main">
                  <span class="card-top">
                    <span class="sev {f.severity}">{severityLabel[f.severity]}</span>
                    <span class="card-path">{f.path}</span>
                  </span>
                  <span class="card-why">{f.why}</span>
                </span>
                <span class="diff"><b>+{f.additions}</b> <i>-{f.deletions}</i></span>
              </button>
            {/each}
            <p class="foot">
              {session.skipped.length} descartados (testes, docs, gerados) · {session.others.length} alterados sem risco relevante
            </p>
          </div>
        {:else if fileError}
          <div class="overview"><p class="error">{fileError}</p></div>
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
          <div class="overview"><p class="empty">Carregando...</p></div>
        {/if}
      </div>

      <button class="arrow" onclick={() => go(1)} aria-label="Próximo">
        <svg viewBox="0 0 24 24"><path d="M9 5l7 7-7 7" /></svg>
      </button>
    </section>

    <footer class="bottombar">
      <div class="dots">
        {#if blueprint}
          {#each chapters as c, i (i)}
            <button
              class="dot"
              class:overview-dot={i === 0}
              class:seen={i < chapterIdx}
              class:current={activePath === null && i === chapterIdx}
              onclick={() => {
                if (activePath !== null) goChapter(0);
                jumpChapter(i);
              }}
              title="{i === 0 ? '' : i + '. '}{c.title}"
            ></button>
          {/each}
        {:else}
          <button class="dot overview-dot" class:current={activePath === null} onclick={() => go(-index - 1)} title="Visão geral"></button>
        {/if}
        {#each tourFiles as f, i (f.path)}
          <button
            class="dot {f.severity}"
            class:seen={visited.has(f.path)}
            class:current={activePath === f.path}
            onclick={() => openPath(f.path)}
            title="{i + 1}. {f.path}"
          ></button>
        {/each}
      </div>
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
      <span class="hint">
        {blueprint && activePath === null ? "← → ou hjkl andam pelos capítulos e pelos passos do fluxo" : "← → ou hjkl navegam · Enter abre o código em tela cheia, Esc sai · C comenta"}
      </span>
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
        {error}
        <button onclick={() => (error = "")}>×</button>
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

  .home-player {
    position: fixed;
    top: 16px;
    right: 18px;
    z-index: 10;
  }

  /* Seletor de modo de revisor (home e topo do tour) */
  .modes {
    display: inline-flex;
    flex-shrink: 0;
    padding: 3px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 11px;
  }
  .modes button {
    border: none;
    background: none;
    color: var(--muted);
    padding: 5px 13px;
    border-radius: 8px;
    font-size: 13px;
  }
  .modes button:hover {
    color: var(--text);
  }
  .modes button.on {
    background: rgba(217, 119, 87, 0.22);
    color: var(--text);
    font-weight: 700;
  }
  .mode-pick {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 12px;
    width: 100%;
    text-align: left;
    font-size: 13px;
    color: var(--muted);
  }
  .fname.plain {
    font-family: var(--ui);
    font-size: 14.5px;
  }

  /* Home */
  .home {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 24px;
    width: min(720px, 100%);
  }
  h1 {
    margin: 18px 0 6px;
    font-size: 54px;
    letter-spacing: -0.02em;
    font-weight: 800;
  }
  h1 span {
    background: linear-gradient(90deg, #f0a283, #b69cff, #8ab4ff);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  .tagline {
    color: var(--muted);
    margin: 0 0 28px;
    font-size: 16px;
  }
  .search {
    display: flex;
    gap: 10px;
    width: 100%;
    padding: 8px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 16px;
    box-shadow: 0 0 60px rgba(120, 140, 255, 0.12);
  }
  .search input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--text);
    font: 15px var(--mono);
    padding: 10px 12px;
  }
  .search .btn {
    padding: 10px 22px;
    font-size: 14px;
  }
  .error {
    color: #ff9b94;
    margin-top: 16px;
    white-space: pre-wrap;
  }
  .notice {
    margin-top: 18px;
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 14px 16px;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--panel);
    text-align: left;
    font-size: 14px;
  }
  .notice p {
    margin: 0;
  }
  .notice-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    flex-wrap: wrap;
  }
  .recent {
    margin-top: 22px;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
    align-items: center;
    font-size: 12px;
    color: var(--muted);
  }
  .recent button {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 999px;
    padding: 4px 10px;
    font: 12px var(--mono);
  }

  /* Loading */
  .loading {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
    width: min(760px, 100%);
    padding: 24px;
  }
  .loading-row {
    display: flex;
    align-items: flex-start;
    gap: 22px;
    width: 100%;
  }
  .log {
    list-style: none;
    padding: 0;
    margin: 0;
    width: 100%;
    font: 12.5px var(--mono);
    color: var(--muted);
  }
  .log li::before {
    content: "› ";
    color: var(--claude);
  }
  .log li:last-child {
    color: var(--text);
  }
  .count {
    margin: 8px 0 0;
    font-size: 12px;
    color: var(--claude-soft);
  }

  /* Bubble */
  .bubble {
    position: relative;
    flex: 1;
    min-width: 0;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 12px 16px;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.35);
  }
  .bubble::before {
    content: "";
    position: absolute;
    left: -9px;
    top: 34px;
    width: 16px;
    height: 16px;
    background: inherit;
    border-left: 1px solid var(--border);
    border-bottom: 1px solid var(--border);
    transform: rotate(45deg);
  }
  .bubble.big {
    padding: 18px 20px;
  }
  .bubble-head {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 6px;
    min-width: 0;
  }
  .fname {
    font-family: var(--mono);
    font-weight: 700;
    font-size: 13.5px;
    white-space: nowrap;
  }
  .why {
    color: var(--muted);
    font-size: 12.5px;
    font-style: italic;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .speech-box {
    display: block;
    width: 100%;
  }
  .back {
    margin-left: auto;
    background: none;
    border: 1px solid var(--border);
    color: #9cc0f5;
    border-radius: 999px;
    padding: 2px 10px;
    font-size: 12px;
    white-space: nowrap;
  }
  .back:hover {
    border-color: rgba(156, 192, 245, 0.5);
  }
  .extras {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-top: 10px;
  }
  .toggle {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: rgba(227, 179, 65, 0.07);
    border: 1px solid rgba(227, 179, 65, 0.28);
    color: #e9d49b;
    border-radius: 999px;
    padding: 3px 10px 3px 4px;
    font-size: 12px;
  }
  .toggle:hover,
  .toggle.open {
    background: rgba(227, 179, 65, 0.14);
  }
  .warn-dot {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #c99a35;
    color: #16171b;
    display: grid;
    place-items: center;
    font-weight: 800;
    font-size: 11px;
  }
  .toggle .caret {
    color: var(--muted);
  }
  .ref {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: rgba(91, 140, 214, 0.1);
    border: 1px solid rgba(91, 140, 214, 0.3);
    color: #b9d0f3;
    border-radius: 999px;
    padding: 3px 10px 3px 4px;
    font: 12px var(--mono);
  }
  .ref:hover {
    background: rgba(91, 140, 214, 0.2);
  }
  .ref.missing {
    opacity: 0.6;
    border-style: dashed;
    cursor: help;
  }
  .ref-label {
    font: 700 10px var(--ui);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    background: rgba(91, 140, 214, 0.25);
    border-radius: 999px;
    padding: 1px 6px;
  }
  .speech {
    margin: 0;
    font-size: 14.5px;
    line-height: 1.55;
    white-space: pre-wrap;
    max-height: 132px;
    overflow: auto;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 10px;
  }
  .chip {
    display: flex;
    gap: 8px;
    align-items: baseline;
    background: rgba(227, 179, 65, 0.07);
    border: 1px solid rgba(227, 179, 65, 0.25);
    color: #e9d9ad;
    border-radius: 8px;
    padding: 4px 9px;
    font-size: 12px;
    max-width: 100%;
  }
  .chip .line {
    flex-shrink: 0;
    background: none;
    border: none;
    padding: 0;
    font: 700 12px var(--mono);
    color: var(--yellow);
    text-decoration: underline;
    text-decoration-color: rgba(227, 179, 65, 0.4);
    text-underline-offset: 2px;
  }

  /* Tour */
  .topbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 20px 0;
  }
  .pr {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .pr-title {
    background: none;
    border: none;
    padding: 0;
    color: var(--text);
    font-size: 15px;
    font-weight: 700;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pr-title:hover {
    color: var(--accent);
  }
  .pr-meta {
    font-size: 12px;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .add {
    color: var(--green);
    font-weight: 400;
  }
  .del {
    color: var(--red);
    font-weight: 400;
  }
  .logo {
    flex-shrink: 0;
    display: block;
  }
  .claude {
    display: flex;
    /* topo: a logo fica parada enquanto o texto do balão cresce */
    align-items: flex-start;
    gap: 18px;
    padding: 12px 84px 10px 40px;
  }
  .stage {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: stretch;
    gap: 14px;
    padding: 0 14px;
  }
  .center {
    flex: 1;
    min-width: 0;
    min-height: 0;
  }
  .arrow {
    align-self: center;
    width: 54px;
    height: 54px;
    flex-shrink: 0;
    border-radius: 50%;
    border: 1px solid var(--border);
    background: var(--panel);
    display: grid;
    place-items: center;
    transition: transform 0.15s, background 0.15s, border-color 0.15s;
  }
  .arrow svg {
    width: 26px;
    height: 26px;
    fill: none;
    stroke: var(--text);
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .arrow:hover:not(:disabled) {
    transform: scale(1.08);
    background: rgba(217, 119, 87, 0.2);
    border-color: rgba(217, 119, 87, 0.6);
  }
  .overview {
    height: 100%;
    overflow: auto;
    padding: 8px 4px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .card {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px 16px;
    color: var(--text);
    text-align: left;
    animation: rise 0.35s ease-out both;
    transition: border-color 0.15s, transform 0.15s;
  }
  .card:hover {
    border-color: rgba(217, 119, 87, 0.55);
    transform: translateX(4px);
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
  }
  .idx {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    background: rgba(217, 119, 87, 0.15);
    color: var(--claude-soft);
    font-weight: 800;
    flex-shrink: 0;
  }
  .card-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .card-top {
    display: flex;
    gap: 10px;
    align-items: center;
    min-width: 0;
  }
  .card-path {
    font-family: var(--mono);
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card-why {
    font-size: 13px;
    color: var(--muted);
  }
  .diff {
    font-family: var(--mono);
    font-size: 12px;
    white-space: nowrap;
  }
  .diff b {
    color: var(--green);
    font-weight: 400;
  }
  .diff i {
    color: var(--red);
    font-style: normal;
  }
  .foot,
  .empty {
    color: var(--muted);
    font-size: 13px;
    text-align: center;
  }
  .bottombar {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 20px 14px;
  }
  .dots {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
  }
  .dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    border: 1.5px solid rgba(255, 255, 255, 0.25);
    background: transparent;
    padding: 0;
    transition: transform 0.15s;
  }
  .dot.critical {
    border-color: var(--red);
  }
  .dot.high {
    border-color: var(--yellow);
  }
  .dot.medium {
    border-color: var(--accent);
  }
  .dot.seen.critical {
    background: var(--red);
  }
  .dot.seen.high {
    background: var(--yellow);
  }
  .dot.seen.medium {
    background: var(--accent);
  }
  .dot.current {
    transform: scale(1.5);
    box-shadow: 0 0 10px rgba(255, 255, 255, 0.5);
  }
  .overview-dot {
    border-radius: 3px;
    border-color: var(--claude);
  }
  .overview-dot.current {
    background: var(--claude);
  }
  .pos {
    font-size: 13px;
    font-weight: 700;
    white-space: nowrap;
  }
  .hint {
    flex: 1;
    font-size: 12px;
    color: var(--muted);
    text-align: right;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .toast {
    position: fixed;
    bottom: 70px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 60;
    background: rgba(80, 20, 20, 0.95);
    border: 1px solid var(--red);
    border-radius: 10px;
    padding: 10px 14px;
    font-size: 13px;
    display: flex;
    gap: 12px;
    align-items: center;
    max-width: 80vw;
  }
  .toast button {
    background: none;
    border: none;
    color: var(--text);
    font-size: 18px;
  }
</style>
