import { branchName, columns, columnTone, initials, moveCard, slug, type Board, type BoardCard } from "../src/lib/board";

let failed = 0;
function eq<T>(label: string, got: T, want: T): void {
  if (JSON.stringify(got) !== JSON.stringify(want)) {
    failed++;
    console.error(`${label} = ${JSON.stringify(got)}; want ${JSON.stringify(want)}`);
  }
}

eq("slug", slug("Integração Apple Pay (Cakto & Pagarme)"), "integracao-apple-pay-cakto-pagarme");
eq("slug vazio", slug("  !!  "), "");
eq("branch", branchName("Task", 1290, "Estudo + Integração Catalogo META"), "task/1290-estudo-integracao-catalogo-meta");
eq("branch sem tipo", branchName("", 7, "Corrige login"), "feat/7-corrige-login");
eq("branch sem número", branchName("Bug", null, "Imagem do banner"), "bug/imagem-do-banner");
const long = branchName("US", 1177, "Integração com a Bling para sincronizar produtos e estoque entre as plataformas");
eq("branch longa cabe", long.length <= 60, true);
eq("branch longa corta em palavra", long, "us/1177-integracao-com-a-bling-para-sincronizar-produtos-e");
eq("tone blocked", columnTone("Blocked"), "blocked");
eq("tone done", columnTone("Done"), "done");
eq("tone neutro", columnTone("In Progress"), "");
eq("initials", initials("dalton02"), "DA");
eq("initials composto", initials("maria-souza"), "MS");

const card = (itemId: string, statusId: string, extra: Partial<BoardCard> = {}): BoardCard => ({
  itemId, kind: "issue", contentId: "c" + itemId, number: Number(itemId), title: "card " + itemId, url: "", state: "OPEN",
  repo: "acme/board", assignees: [], statusId, typeId: "", subTotal: 0, subDone: 0, ...extra
});
const board: Board = {
  projectId: "P", owner: "acme", number: 1, title: "B", url: "", public: false,
  status: { id: "S", name: "Status", options: [{ id: "todo", name: "Todo", description: "", color: "" }, { id: "prog", name: "In Progress", description: "", color: "" }] },
  type: null,
  cards: [card("1", "todo"), card("2", "prog", { assignees: ["Ana"], typeId: "bug" }), card("3", "")],
  repos: ["acme/board"], truncated: false
};
const all = { text: "", typeId: "", onlyMine: false, viewer: "ana" };
eq("colunas", columns(board, all).map((c) => [c.name, c.cards.length]), [["Sem status", 1], ["Todo", 1], ["In Progress", 1]]);
eq("só meus", columns(board, { ...all, onlyMine: true }).map((c) => c.cards.length), [0, 1]);
eq("por tipo", columns(board, { ...all, typeId: "bug" }).flatMap((c) => c.cards.map((x) => x.itemId)), ["2"]);
eq("busca por #", columns(board, { ...all, text: "#3" }).flatMap((c) => c.cards.map((x) => x.itemId)), ["3"]);
const moved = moveCard(board, "1", "prog");
eq("move", moved.cards[0].statusId, "prog");
eq("move não muta", board.cards[0].statusId, "todo");

if (failed) {
  console.error(`${failed} falha(s)`);
  process.exit(1);
}
console.log("board: ok");
