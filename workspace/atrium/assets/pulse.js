'use strict';
// Atrium test cards are independent of Workshop messages and task notes.
// All execution decisions are enforced by the server and its isolated adapter.
(() => {
  const find = id => document.getElementById(id);
  const csrf = document.querySelector('meta[name="playground-csrf"]').content;
  let state = null, pending = false;
  const text = (id, val) => { const e = find(id); if (e) e.textContent = String(val); };
  const el = (tag, cl, contents) => {
    const node = document.createElement(tag);
    if (cl) node.className = cl;
    if (contents !== undefined) node.textContent = String(contents);
    return node;
  };
  const add = (parent, tag, cl, contents) => { const node = el(tag, cl, contents); parent.append(node); return node; };
  async function request(action, cardId, ack) {
    if (pending) return;
    pending = true;
    try {
      const response = await fetch('/tests/action', {
        method: 'POST', cache: 'no-store',
        headers: {'Content-Type': 'application/json', 'X-Playground-CSRF': csrf},
        body: JSON.stringify({action, ...(cardId ? {card_id: cardId} : {}), ...(ack ? {ack} : {})})
      });
      const data = await response.json();
      if (!response.ok) throw Error(data.error || 'The test controller refused the request');
      state = data;
      render();
    } catch(e) {
      text('pulseState', 'NOT ACCEPTED: ' + e.message);
    } finally { pending = false; }
  }
  function btn(parent, label, action, card, disabled) {
    const b = add(parent, 'button', action === 'approve' ? 'secondary' : 'primary', label);
    b.type = 'button'; b.disabled = Boolean(disabled || pending);
    b.addEventListener('click', () => {
      if (action === 'approve') {
        if (!window.confirm('Approve only '+card.id+' at '+card.sha+' for the fixed '+card.operation+
             ' operation? This is NOT an Identity grant.')) return;
        return request('approve', card.id, 'APPROVE ' + card.id);
      }
      if (action === 'queue') {
        if (!window.confirm('Queue '+card.id+' once in the separately enabled isolated runner?')) return;
        return request('queue', card.id);
      }
      return request(action);
    });
    return b;
  }
  function cardView(card) {
    const box = el('article', 'pulse-card ' + (card.status === 'FAIL' ? 'pulse-fail' : ''));
    const head = add(box, 'div', 'pulse-card-head');
    add(head, 'strong', '', card.id + ' / ' + card.objective);
    add(head, 'span', 'pulse-status', card.status);
    const scope = add(box, 'p', 'pulse-scope', card.repository+' @ '+card.sha);
    scope.title = 'Exact source revision';
    add(box, 'p', '', 'Expected: ' + card.expected);
    const op = add(box, 'p', 'pulse-command', card.operation + ' · ' + card.argv.join(' '));
    op.title = 'Fixed allowlisted operation — not editable in Atrium';
    const meta = add(box, 'div', 'pulse-metadata');
    add(meta, 'span', '', card.job_id ? 'Job '+card.job_id : 'Not queued');
    add(meta, 'span', '', card.started_at ? 'Started '+card.started_at : 'No execution started');
    if (card.model_proposal) {
      add(box, 'p', 'pulse-caption', 'Cell selected by '+card.model_proposal.actor+
        ' · not an authorization grant');
      add(box, 'p', 'pulse-caption', 'Prediction: '+card.model_proposal.expected_result);
    } else if (card.id === 'NOTEBOOK-IDENTITY-LAB') {
      add(box, 'p', 'pulse-caption', 'AWAITING MODEL CELL SELECTION · real Nemotron not connected');
    }
    if (card.reason) add(box, 'p', 'pulse-reason', card.reason);
    const actions = add(box, 'div', 'pulse-buttons');
    if (card.status === 'PROPOSED') btn(actions, 'Approve this exact test', 'approve', card,
      card.id === 'NOTEBOOK-IDENTITY-LAB' && !card.model_proposal);
    if (card.status === 'APPROVED') btn(actions, 'Queue once', 'queue', card, !state.execution_enabled);
    if (card.status === 'QUEUED') add(actions, 'span', 'pulse-caption', 'Queued · awaiting next enabled pulse');
    if (card.status === 'WORKING') add(actions, 'span', 'pulse-caption', 'WORKING · later pulses will not start it twice');
    if (card.receipt) {
      const receipt = card.receipt;
      const proof = add(box, 'div', 'pulse-proof');
      add(proof, 'strong', '', 'EXECUTION RECEIPT · '+receipt.outcome+' · '+(receipt.executed ? 'EXECUTED' : 'NOT EXECUTED'));
      add(proof, 'p', '', 'Receipt '+receipt.receipt_id+' · exit '+(receipt.exit_code === null ? 'N/A' : receipt.exit_code));
      add(proof, 'p', '', 'Started '+receipt.started_at+' · finished '+receipt.finished_at);
      add(proof, 'p', '', 'SHA '+receipt.sha+' · output SHA-256 '+(receipt.output_sha256 || 'N/A'));
      if (receipt.cell_id) add(proof, 'p', '', 'Notebook cell '+receipt.cell_id+' · cell SHA-256 '+receipt.cell_sha256);
      if (receipt.notebook_blob_sha) add(proof, 'p', '', 'Notebook Git blob '+receipt.notebook_blob_sha);
      if (receipt.output_excerpt) {
        add(proof, 'span', 'pulse-caption', 'Bounded, redacted test output');
        add(proof, 'pre', 'pulse-output', receipt.output_excerpt);
      }
    }
    return box;
  }
  function render() {
    if (!state) return;
    text('pulseState', 'Pulse #'+state.tick_sequence+' · '+(state.paused ? 'PAUSED' : 'RUNNING')+
      ' · every '+state.interval_seconds+'s · next in '+state.next_tick_seconds+
      's · last '+(state.last_pulse_id || 'none'));
    text('pulseGate', state.execution_enabled
      ? 'LOCAL SANDBOX PILOT ENABLED · explicit approval still required · not a live Identity grant'
      : 'EXECUTION DISABLED · no test process can start. Approval records intent only; enable an isolated runner separately.');
    find('pulseGate').classList.toggle('enabled', state.execution_enabled);
    find('pulsePause').disabled = state.paused;
    find('pulseResume').disabled = !state.paused || !state.execution_enabled;
    find('pulseStep').disabled = !state.execution_enabled;
    find('pulseStop').disabled = !(state.cards || []).some(c => c.status === 'WORKING' || c.status === 'QUEUED');
    const host = find('pulseCardList'); host.replaceChildren();
    for (const card of (state.cards || [])) host.append(cardView(card));
  }
  async function refresh() {
    try {
      const response = await fetch('/tests/state', {cache:'no-store'});
      if (!response.ok) throw Error('Test-card service not responding (' + response.status + ')');
      state = await response.json();
      render();
    } catch (e) {
      text('pulseState', 'UNKNOWN: '+e.message);
      text('pulseGate', 'No scheduling or execution status verified.');
    }
  }
  find('pulsePause').addEventListener('click', () => request('pause'));
  find('pulseResume').addEventListener('click', () => request('resume'));
  find('pulseStep').addEventListener('click', () => request('step'));
  find('pulseStop').addEventListener('click', () => {
    if (window.confirm('Stop the active test process group and block queued work?')) request('stop');
  });
  refresh();
  setInterval(refresh, 3000);
})();