export function ide(workspaceId: string): string {
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>Workspace ${workspaceId} — Vega Ephemeral Dev</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.css" />
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body, html { margin: 0; height: 100%; background: #0b0f17; color: #e6edf3; font: 13px/1.4 ui-sans-serif, system-ui, sans-serif; }
  header { display: flex; align-items: center; gap: 12px; padding: 8px 12px; background: #111827; border-bottom: 1px solid #1f2937; }
  header b { font-weight: 600; }
  header .id { background: #1f2937; padding: 2px 8px; border-radius: 4px; font-family: ui-monospace, monospace; font-size: 12px; }
  header .status { margin-left: auto; opacity: .7; font-size: 12px; }
  .layout { display: grid; grid-template-columns: 220px 1fr 1fr; grid-template-rows: 1fr 240px; height: calc(100vh - 41px); }
  .tree    { grid-column: 1; grid-row: 1 / span 2; background: #0f172a; border-right: 1px solid #1f2937; overflow: auto; padding: 8px 4px; }
  .editor  { grid-column: 2; grid-row: 1; background: #1e1e1e; overflow: hidden; }
  .term    { grid-column: 2; grid-row: 2; }
  .preview { grid-column: 3; grid-row: 1 / span 2; background: white; border-left: 1px solid #1f2937; display: flex; flex-direction: column; }
  .preview iframe { flex: 1; border: 0; }
  .preview .bar { padding: 6px 10px; background: #111827; color: #e6edf3; font-size: 12px; display: flex; align-items: center; gap: 8px; border-bottom: 1px solid #1f2937; }
  .preview .bar .dot { width: 8px; height: 8px; border-radius: 50%; background: #10b981; }
  .term { background: #000; border-top: 1px solid #1f2937; padding: 6px; }
  .term .bar { color: #94a3b8; font-size: 12px; padding-bottom: 4px; }
  .tree .row { padding: 2px 6px; cursor: pointer; border-radius: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tree .row:hover { background: #1f2937; }
  .tree .row.active { background: #1d4ed8; color: white; }
  .tree .row.dir { color: #93c5fd; }
  .dirty::after { content: " ●"; color: #fbbf24; }
</style>
</head>
<body>
<header>
  <b>Ephemeral Dev Environment</b>
  <span class="id">${workspaceId}</span>
  <span class="status" id="status">connecting…</span>
</header>
<div class="layout">
  <div class="tree" id="tree"></div>
  <div class="editor" id="editor"></div>
  <div class="preview">
    <div class="bar"><span class="dot"></span><span>preview · /preview/</span><span id="reloads" style="margin-left:auto;opacity:.6">0 reloads</span></div>
    <iframe id="preview" src="/ws/${workspaceId}/preview/"></iframe>
  </div>
  <div class="term">
    <div class="bar">terminal · bash</div>
    <div id="term" style="height: calc(100% - 22px);"></div>
  </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.js"></script>
<script src="https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/min/vs/loader.js"></script>
<script>
  const WS_ID = ${JSON.stringify(workspaceId)};
  const base = '/ws/' + WS_ID;
  const status = document.getElementById('status');
  const setStatus = (s) => { status.textContent = s; };

  // ---------- Monaco ----------
  let editor;
  let currentPath = null;
  let dirty = false;
  const pendingByPath = new Map();

  require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/min/vs' } });
  require(['vs/editor/editor.main'], () => {
    editor = monaco.editor.create(document.getElementById('editor'), {
      value: '// Pick a file from the tree to start editing.',
      language: 'plaintext',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
    });
    editor.onDidChangeModelContent(() => { if (currentPath) markDirty(true); });
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, saveCurrent);
    loadTree();
  });

  function markDirty(d) {
    dirty = d;
    const row = document.querySelector('.tree .row.active');
    if (row) row.classList.toggle('dirty', d);
  }

  function langFor(path) {
    if (path.endsWith('.html')) return 'html';
    if (path.endsWith('.css')) return 'css';
    if (path.endsWith('.js')) return 'javascript';
    if (path.endsWith('.ts')) return 'typescript';
    if (path.endsWith('.json')) return 'json';
    if (path.endsWith('.go')) return 'go';
    if (path.endsWith('.md')) return 'markdown';
    return 'plaintext';
  }

  // ---------- File tree ----------
  async function loadTree() {
    setStatus('loading tree…');
    const r = await fetch(base + '/api/tree');
    const tree = await r.json();
    const root = document.getElementById('tree');
    root.innerHTML = '';
    renderTree(tree, root, 0);
    setStatus('ready');
  }

  function renderTree(node, parent, depth) {
    if (depth > 0) {
      const row = document.createElement('div');
      row.className = 'row ' + node.type;
      row.style.paddingLeft = (depth * 12) + 'px';
      row.textContent = (node.type === 'dir' ? '▸ ' : '  ') + node.name;
      row.dataset.path = node.path;
      if (node.type === 'file') row.onclick = () => openFile(node.path, row);
      parent.appendChild(row);
    }
    if (node.children) for (const c of node.children) renderTree(c, parent, depth + 1);
  }

  async function openFile(path, row) {
    if (dirty && !confirm('Discard unsaved changes?')) return;
    document.querySelectorAll('.tree .row.active').forEach(el => el.classList.remove('active'));
    row.classList.add('active');
    setStatus('loading ' + path + '…');
    const r = await fetch(base + '/api/file?path=' + encodeURIComponent(path));
    const text = await r.text();
    const model = monaco.editor.createModel(text, langFor(path));
    editor.setModel(model);
    currentPath = path;
    markDirty(false);
    setStatus('editing ' + path);
  }

  async function saveCurrent() {
    if (!currentPath) return;
    const text = editor.getValue();
    pendingByPath.set(currentPath, Date.now());
    setStatus('saving ' + currentPath + '…');
    const r = await fetch(base + '/api/file?path=' + encodeURIComponent(currentPath), {
      method: 'PUT', body: text,
    });
    if (r.ok) { markDirty(false); setStatus('saved ' + currentPath); }
    else { setStatus('save failed: ' + r.status); }
  }

  // ---------- Terminal ----------
  const term = new Terminal({ cursorBlink: true, fontSize: 13, theme: { background: '#000000' } });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(document.getElementById('term'));
  requestAnimationFrame(() => fit.fit());
  window.addEventListener('resize', () => fit.fit());

  const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const termWs = new WebSocket(wsProto + '//' + location.host + base + '/api/terminal');
  termWs.binaryType = 'arraybuffer';
  termWs.onopen = () => {
    const { cols, rows } = term;
    termWs.send(JSON.stringify({ type: 'resize', cols, rows }));
  };
  termWs.onmessage = (ev) => {
    if (typeof ev.data === 'string') term.write(ev.data);
    else term.write(new Uint8Array(ev.data));
  };
  termWs.onclose = () => term.write('\\r\\n\\x1b[31m[terminal disconnected]\\x1b[0m\\r\\n');
  term.onData((d) => { if (termWs.readyState === 1) termWs.send(d); });
  term.onResize(({ cols, rows }) => {
    if (termWs.readyState === 1) termWs.send(JSON.stringify({ type: 'resize', cols, rows }));
  });

  // ---------- File watcher / hot reload ----------
  let reloads = 0;
  const evWs = new WebSocket(wsProto + '//' + location.host + base + '/api/events');
  evWs.onmessage = (ev) => {
    try {
      const e = JSON.parse(ev.data);
      if (!e.path) return;
      if (e.path.startsWith('public/') || e.path.startsWith('public\\\\')) {
        reloads++;
        document.getElementById('reloads').textContent = reloads + ' reloads';
        const iframe = document.getElementById('preview');
        iframe.src = iframe.src.split('?')[0] + '?t=' + Date.now();
      }
    } catch {}
  };
</script>
</body>
</html>`;
}
