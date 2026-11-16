// Exercise the actual pure editor guard extracted from the served HTML.
// This uses a Node fixture only: no browser, owner data, or live TASS service.
const fs = require("node:fs");
const vm = require("node:vm");
const assert = require("node:assert/strict");

const html = fs.readFileSync("core/cmd/tass-core/dev.html", "utf8");
const start = html.indexOf("function edgeProblem(");
const end = html.indexOf("\nfunction kv(", start);
assert(start >= 0 && end > start, "editor guard and addEdge functions are present");
const script = html.slice(start, end);
assert(script.includes("function addEdge("), "addEdge uses the shared validation guard");
assert(
  html.includes("const bad = edgeProblem(e.from, e.to, rs.value, e.id);"),
  "editing an existing relation checks the guard before mutation",
);

const graph = {
  nodes: [
    { id: "a", block_id: "lesson:1" },
    { id: "b", block_id: "cell:1" },
    { id: "c", block_id: "skill:1" },
    { id: "d", block_id: "note:1" },
  ],
  edges: [],
};
const blocks = {
  "lesson:1": { kind: "lesson" },
  "cell:1": { kind: "code-cell" },
  "skill:1": { kind: "skill" },
  "note:1": { kind: "note" },
};
let snapshots = 0;
let error = "";
const context = vm.createContext({
  graph,
  relations: { supports: "support", uses: "skill use", next: "next", requires: "need", alternative: "or" },
  nodeById: id => graph.nodes.find(n => n.id === id),
  blockById: id => blocks[id],
  rid: () => "edge-new",
  snapshot: () => { snapshots += 1; },
  render: () => {},
  changed: () => {},
  msg: (message, bad) => { error = bad ? message : ""; },
});
vm.runInContext(script, context);

const check = (from, to, relation, ignoreId = "") => vm.runInContext(
  `edgeProblem(${JSON.stringify(from)},${JSON.stringify(to)},${JSON.stringify(relation)},${JSON.stringify(ignoreId)})`,
  context,
);
function add(from, to, relation) {
  error = "";
  vm.runInContext(`addEdge(${JSON.stringify(from)},${JSON.stringify(to)},${JSON.stringify(relation)},"reason")`, context);
}

assert.match(check("missing", "a", "supports"), /missing block/);
assert.match(check("a", "missing", "supports"), /missing block/);
assert.match(check("a", "a", "supports"), /itself/);
assert.match(check("a", "b", "executes"), /Unknown relation/);
assert.match(check("a", "b", "uses"), /skill/);
assert.equal(check("a", "c", "uses"), "");

for (const [from, to, relation] of [
  ["missing", "a", "supports"],
  ["a", "missing", "supports"],
  ["a", "b", "uses"],
]) {
  const edgesBefore = JSON.stringify(graph.edges);
  const snapshotsBefore = snapshots;
  add(from, to, relation);
  assert.equal(JSON.stringify(graph.edges), edgesBefore, "rejected addition must not mutate the graph");
  assert.equal(snapshots, snapshotsBefore, "rejected addition must not create an undo snapshot");
  assert.ok(error, "rejected addition reports a reason");
}

add("a", "c", "uses");
assert.equal(graph.edges.length, 1, "valid uses edge is added");
assert.equal(snapshots, 1, "valid addition creates one undo snapshot");
assert.match(check("a", "c", "uses"), /already exists/);

graph.edges = [
  { id: "next1", from: "a", to: "b", relation: "next" },
  { id: "next2", from: "b", to: "d", relation: "next" },
];
assert.match(check("d", "a", "next"), /circle/);
const cycleBefore = JSON.stringify(graph.edges);
const cycleSnapshots = snapshots;
add("d", "a", "next");
assert.equal(JSON.stringify(graph.edges), cycleBefore, "rejected NEXT cycle must not mutate the graph");
assert.equal(snapshots, cycleSnapshots, "rejected NEXT cycle must not create an undo snapshot");

graph.edges = [
  { id: "other", from: "b", to: "a", relation: "next" },
  { id: "editable", from: "a", to: "b", relation: "supports" },
];
assert.match(check("a", "b", "next", "editable"), /circle/, "supports-to-next edit must be refused");
assert.equal(check("a", "b", "supports", "editable"), "", "an unchanged edge does not duplicate itself");

console.log("PASS: editor edge guard rejects missing endpoints and invalid edges before graph mutation");
