<script lang="ts">
  import type { Blueprint, Session } from "../api";
  import type { Chapter } from "./chapters";
  import Decisions from "./Decisions.svelte";
  import FlowDiagram from "./FlowDiagram.svelte";
  import ImpactChart from "./ImpactChart.svelte";
  import Overview from "./Overview.svelte";
  import Risks from "./Risks.svelte";
  import SystemMap, { type MapView } from "./SystemMap.svelte";

  interface Props {
    blueprint: Blueprint;
    session: Session;
    chapters: Chapter[];
    chapter: Chapter;
    /** passo do fluxo em destaque (-1 = fluxo inteiro) */
    step: number;
    selectedNode: string | null;
    picked: string[];
    onjump: (index: number) => void;
    onopenfile: (path: string) => void;
    onreview: () => void;
  }

  let { blueprint, session, chapters, chapter, step = $bindable(), selectedNode = $bindable(), picked = $bindable(), onjump, onopenfile, onreview }: Props =
    $props();

  let mapView = $state<MapView>("diff");

  /* o que foi alterado em arquivos que o Claude não atribuiu a nenhum componente */
  const loose = $derived.by(() => {
    const claimed = new Set(blueprint.nodes.flatMap((n) => n.files));
    const rest = session.others.filter((f) => !claimed.has(f.path));
    return { additions: rest.reduce((n, f) => n + f.additions, 0), deletions: rest.reduce((n, f) => n + f.deletions, 0) };
  });

  function showOnMap(id: string): void {
    selectedNode = id;
    mapView = "diff";
    onjump(chapters.findIndex((c) => c.kind === "map"));
  }
</script>

{#if chapter.kind === "overview"}
  <Overview {blueprint} {session} {chapters} {onjump} />
{:else if chapter.kind === "map"}
  <SystemMap nodes={blueprint.nodes} edges={blueprint.edges} bind:selected={selectedNode} bind:view={mapView} {onopenfile} />
{:else if chapter.kind === "flow"}
  {#key chapter.flow}
    <FlowDiagram flow={blueprint.flows[chapter.flow]} nodes={blueprint.nodes} bind:step />
  {/key}
{:else if chapter.kind === "decisions"}
  <Decisions decisions={blueprint.decisions} />
{:else if chapter.kind === "impact"}
  <ImpactChart impact={blueprint.impact} nodes={blueprint.nodes} {loose} />
{:else}
  <Risks risks={blueprint.risks} questions={blueprint.questions} nodes={blueprint.nodes} bind:picked onnode={showOnMap} {onreview} />
{/if}
