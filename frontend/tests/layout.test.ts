import { layoutMap, layoutSequence, wrap, NODE_H, NODE_W } from "../src/lib/architect/layout";

let failed = 0;
function check(name: string, ok: boolean, detail = ""): void {
  if (ok) {
    console.log(`ok    ${name}`);
  } else {
    failed++;
    console.log(`FALHA ${name} ${detail}`);
  }
}

const nodes = [
  { id: "db", kind: "store" },
  { id: "user", kind: "actor" },
  { id: "api", kind: "interface" },
  { id: "svc", kind: "service" },
  { id: "queue", kind: "queue" },
  { id: "worker", kind: "worker" },
  { id: "bank", kind: "external" },
  { id: "solto", kind: "service" }
];
const edges = [
  { from: "user", to: "api" },
  { from: "api", to: "svc" },
  { from: "svc", to: "db" },
  { from: "svc", to: "queue" },
  { from: "queue", to: "worker" },
  { from: "worker", to: "bank" },
  { from: "worker", to: "db" },
  { from: "worker", to: "svc" }, // ciclo
  { from: "svc", to: "fantasma" }
];

const map = layoutMap(nodes, edges);
const box = new Map(map.boxes.map((b) => [b.id, b]));

check("todo nó ganha uma caixa", map.boxes.length === nodes.length);
check(
  "nenhuma caixa se sobrepõe",
  map.boxes.every((a) => map.boxes.every((b) => a === b || Math.abs(a.x - b.x) >= NODE_W || Math.abs(a.y - b.y) >= NODE_H))
);
check(
  "todas as caixas cabem na área",
  map.boxes.every((b) => b.x >= 0 && b.y >= 0 && b.x + NODE_W <= map.width && b.y + NODE_H <= map.height)
);
check("o fluxo anda para a direita", box.get("user")!.x < box.get("api")!.x && box.get("api")!.x < box.get("svc")!.x);
check("o ciclo não derruba a ordem", box.get("svc")!.x < box.get("queue")!.x && box.get("queue")!.x < box.get("worker")!.x);
check(
  "banco e serviço externo ficam na ponta direita",
  box.get("db")!.rank === box.get("bank")!.rank && map.boxes.every((b) => b.rank <= box.get("db")!.rank)
);
check("aresta para nó inexistente é ignorada", map.routes.length === edges.length - 1 && map.routes.every((r) => r.index !== 8));
check("toda rota tem caminho e rótulo dentro da área", map.routes.every((r) => r.d.startsWith("M") && r.labelX > 0 && r.labelX < map.width && r.labelY > 0 && r.labelY < map.height));
{
  /* a -> b -> c na mesma linha, mais a ligação direta a -> c: ela não pode atravessar b */
  const line = layoutMap(
    [{ id: "a", kind: "actor" }, { id: "b", kind: "service" }, { id: "c", kind: "store" }],
    [{ from: "a", to: "b" }, { from: "b", to: "c" }, { from: "a", to: "c" }]
  );
  const b = line.boxes.find((x) => x.id === "b")!;
  const long = line.routes.find((r) => r.index === 2)!;
  check("ligação longa desvia da caixa do meio", long.d.includes(" L") && (long.labelY < b.y || long.labelY > b.y + NODE_H), long.d);
  check("o desvio cabe na área", long.labelY > 0 && long.labelY < line.height);
}
check("grafo vazio não quebra", layoutMap([], []).boxes.length === 0);
check("nó sozinho não quebra", layoutMap([{ id: "a", kind: "service" }], []).boxes.length === 1);

const seq = layoutSequence([
  { from: "user", to: "api" },
  { from: "api", to: "svc" },
  { from: "svc", to: "svc" },
  { from: "svc", to: "api" }
]);
check("linhas de vida na ordem de entrada", seq.lifelines.map((l) => l.id).join() === "user,api,svc");
check("um passo por linha, de cima para baixo", seq.arrows.every((a, i) => i === 0 || a.y > seq.arrows[i - 1].y));
check("passo interno vira laço", seq.arrows[2].self && !seq.arrows[1].self);
check("volta aponta para a esquerda", seq.arrows[3].x2 < seq.arrows[3].x1);
check("laço na última linha de vida reserva espaço", seq.width > seq.lifelines[2].x + 150);

check("wrap respeita o limite de linhas", wrap("um texto bem comprido que não cabe em duas linhas curtas", 14).length === 2);
check("wrap marca o corte", wrap("um texto bem comprido que não cabe em duas linhas curtas", 14)[1].endsWith("…"));
check("wrap não mexe em texto curto", wrap("Fila de saques", 18).join("|") === "Fila de saques");

if (failed) {
  console.log(`\n${failed} falha(s)`);
  process.exit(1);
}
