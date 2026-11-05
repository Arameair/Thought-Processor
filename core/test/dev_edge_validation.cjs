// Tests the *actual* pure editor guard extracted from the served HTML.
// CI Node runtime, no browser or live owner stack needed.
const fs = require("node:fs");
const vm = require("node:vm");
const assert = require("node:assert/strict");

const html=fs.readFileSync("core/cmd/tass-core/dev.html","utf8");
const start=html.indexOf("function edgeProblem("),end=html.indexOf("\nfunction kv(",start);
assert(start>=0 && end>start, "editor guard/addEdge functions present");
const script=html.slice(start,end);
assert(script.includes("function addEdge("), "addEdge uses new validation");
assert(html.includes("const problem = edgeProblem(e.from, e.to, rs.value, e.id);"),
       "editing an existing relation also checks before mutation");
const graph={nodes:[
 {id:"a",block_id:"lesson:1"},
 {id:"b",block_id:"cell:1"},
 {id:"c",block_id:"skill:1"},
 {id:"d",block_id:"note:1"},
],edges:[]};
const block={
 "lesson:1":{kind:"lesson"}, "cell:1":{kind:"code-cell"},
 "skill:1":{kind:"skill"}, "note:1":{kind:"note"}
};
let snapshots=0, error="",renders=0;
const context=vm.createContext({
 graph,relations:{supports:"support",uses:"skill use",next:"next",requires:"need",alternative:"or"},
 nodeById: id=>graph.nodes.find(n=>n.id===id),
 blockById: id=>block[id],
 rid:()=> "edge-new",
 snapshot:()=>{snapshots++},render:()=>{renders++},changed:()=>{},
 msg:(s,bad)=>{error=bad?s:""}
});
vm.runInContext(script,context);
const check=(from,to,relation,exceptId)=>vm.runInContext(
  `edgeProblem(${JSON.stringify(from)},${JSON.stringify(to)},${JSON.stringify(relation)},${JSON.stringify(exceptId||"")})`,context);
function add(from,to,relation){
 error="";
 vm.runInContext(`addEdge(${JSON.stringify(from)},${JSON.stringify(to)},${JSON.stringify(relation)},"reason")`,context);
}
assert.match(check("a","b","uses"),/skill/);
assert.equal(check("a","c","uses"),"");
assert.equal(check("a","b","supports"),"");
add("a","b","uses");
assert.equal(graph.edges.length,0, "invalid uses must not touch graph");
assert.equal(snapshots,0, "invalid uses must not create undo entry");
assert.match(error,/skill/);
add("a","c","uses");
assert.equal(graph.edges.length,1,"valid uses should be added");
assert.equal(snapshots,1);
assert.match(check("a","c","uses"),/already exists/);
graph.edges=[
 {id:"next1",from:"a",to:"b",relation:"next"},
 {id:"next2",from:"b",to:"d",relation:"next"}
];
assert.match(check("d","a","next"),/directed cycle/);
assert.equal(check("d","a","supports"),"");
const before=graph.edges.length;
add("d","a","next");
assert.equal(graph.edges.length,before, "NEXT cycle must be refused without mutation");
graph.edges=[{id:"other",from:"b",to:"a",relation:"next"},
             {id:"editable",from:"a",to:"b",relation:"supports"}];
assert.match(check("a","b","next","editable"),/directed cycle/,
             "edit changing supports→next must be refused");
assert.equal(check("a","b","supports","editable"),"",
             "an unchanged edge is not a duplicate of itself");
assert.match(check("x","a","supports"),/missing block/);
console.log("Editor edge guard PASS: uses, skill, cycles, edit, duplicate, missing node, no mutation");
