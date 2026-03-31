const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');

const compareModulePath = path.resolve(__dirname, 'artifact-compare.js');

const loadCompareModule = () => {
  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;

  global.document = { getElementById: () => null };
  global.window = {
    location: { search: '', pathname: '/board/artifacts/compare' },
    history: { replaceState() {} },
    addEventListener() {},
  };
  global.location = global.window.location;
  global.history = global.window.history;

  delete require.cache[compareModulePath];

  try {
    return require(compareModulePath);
  } finally {
    if (previousDocument === undefined) {
      delete global.document;
    } else {
      global.document = previousDocument;
    }

    if (previousWindow === undefined) {
      delete global.window;
    } else {
      global.window = previousWindow;
    }

    if (previousLocation === undefined) {
      delete global.location;
    } else {
      global.location = previousLocation;
    }

    if (previousHistory === undefined) {
      delete global.history;
    } else {
      global.history = previousHistory;
    }
  }
};

const { buildArtifactCompareURL, composeArtifactCompareMarkup } = loadCompareModule();

test('ready state renders unified diff content and navigation links', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'ready',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: {
      format: 'text/unified-diff',
      content: '--- previous:artifact-previous\n+++ current:artifact-current\n',
    },
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare ready',
      detail: 'Showing a unified diff between the current artifact and the previous artifact.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /Artifact Compare/);
  assert.match(markup, /previous:artifact-previous/);
  assert.match(markup, /current:artifact-current/);
  assert.match(markup, /Back to current artifact/);
  assert.match(markup, /Back to run workbench/);
});

test('no_previous state renders an empty-state compare panel', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'no_previous',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: null,
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'No previous artifact',
      detail: 'No earlier artifact with the same task and kind is available for compare.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /No previous artifact/);
  assert.doesNotMatch(markup, /artifact-previous/);
});

test('unsupported state keeps previous metadata but no diff', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'unsupported',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'screenshot',
      content_type: 'image\/png',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'screenshot',
      content_type: 'image\/png',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare unavailable',
      detail: 'Artifact compare currently supports text and structured-text content only.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /Compare unavailable/);
  assert.match(markup, /artifact-previous/);
  assert.doesNotMatch(markup, /text\/unified-diff/);
});

test('too_large state renders limit-aware messaging', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'too_large',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text\/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text\/plain; charset=utf-8',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare too large',
      detail: 'One or both artifacts exceed the 65536 byte compare limit.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /Compare too large/);
  assert.match(markup, /65536/);
});

test('refresh URLs stay keyed by the current artifact_id', () => {
  assert.equal(
    buildArtifactCompareURL('artifact-current'),
    '/board/artifacts/compare?artifact_id=artifact-current',
  );
});
