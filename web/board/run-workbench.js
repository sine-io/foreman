const runInput = document.getElementById("run-workbench-run-id");
const refreshButton = document.getElementById("run-workbench-refresh");
const statusNode = document.getElementById("run-workbench-status");
const overviewRoot = document.getElementById("run-workbench-overview");
const metadataRoot = document.getElementById("run-workbench-metadata");

if (runInput && refreshButton && statusNode && overviewRoot && metadataRoot) {
  const state = {
    runId: "",
    detail: null,
    detailState: "idle",
    notice: "",
    noticeTone: "info",
  };

  const escapeHTML = (value) =>
    String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const readRunID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("run_id") || "";
  };

  const updateURLState = (runId) => {
    const searchParams = new URLSearchParams(window.location.search);
    if (runId) {
      searchParams.set("run_id", runId);
    } else {
      searchParams.delete("run_id");
    }

    const query = searchParams.toString();
    const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname;
    window.history.replaceState({}, "", nextURL);
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
  };

  const artifactTargetURL = (detail, artifact) =>
    (detail.artifact_target_urls && detail.artifact_target_urls[artifact.id]) || "";

  const artifactTargetID = (detail, artifact) => {
    const targetURL = artifactTargetURL(detail, artifact);
    return targetURL.startsWith("#") ? targetURL.slice(1) : "";
  };

  const renderArtifactTarget = (detail, artifact) => {
    const targetURL = artifactTargetURL(detail, artifact);
    return targetURL
      ? `<a class="artifact-link" href="${escapeHTML(targetURL)}">Open artifact target</a>`
      : '<span class="artifact-link artifact-link-muted">Artifact target unavailable</span>';
  };

  const renderArtifacts = (detail) => {
    if (!detail.artifacts || !detail.artifacts.length) {
      return '<p class="empty-state">No task-scoped artifacts available for this run yet.</p>';
    }

    return `
      <div class="artifact-list">
        ${detail.artifacts
          .map((artifact) => {
            const targetID = artifactTargetID(detail, artifact);
            return `
              <article class="artifact-card run-artifact-card"${targetID ? ` id="${escapeHTML(targetID)}"` : ""}>
                <p class="artifact-kind">${escapeHTML(artifact.kind || artifact.id)}</p>
                <strong>${escapeHTML(artifact.summary || artifact.path || artifact.id)}</strong>
                <p class="detail-copy">${escapeHTML(artifact.path || "Artifact path not recorded")}</p>
                ${renderArtifactTarget(detail, artifact)}
              </article>
            `;
          })
          .join("")}
      </div>
    `;
  };

  const renderOverview = () => {
    if (state.detailState === "idle") {
      overviewRoot.innerHTML =
        '<p class="empty-state">Enter a run_id to load the run workbench deep link.</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Run metadata will appear here.</p>';
      return;
    }

    if (state.detailState === "loading") {
      overviewRoot.innerHTML = '<p class="empty-state">Loading run detail...</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Loading run metadata...</p>';
      return;
    }

    if (state.detailState === "not_found") {
      overviewRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Run not found</p>
          <p class="detail-copy">No run with ID <code>${escapeHTML(state.runId)}</code> exists.</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Run metadata unavailable.</p>';
      return;
    }

    if (state.detailState === "error") {
      overviewRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Unable to load run detail</p>
          <p class="detail-copy">${escapeHTML(state.notice || "Refresh and try again.")}</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Run metadata unavailable.</p>';
      return;
    }

    const detail = state.detail;
    const taskWorkbenchLink = detail.task_workbench_url
      ? `<a class="board-link" href="${escapeHTML(detail.task_workbench_url)}">Open task workbench</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Task workbench unavailable</span>';
    const noticeMarkup = state.notice
      ? `<p class="detail-notice tone-${escapeHTML(state.noticeTone)}">${escapeHTML(state.notice)}</p>`
      : "";
    const primarySummary = detail.primary_summary || "No summary recorded.";

    overviewRoot.innerHTML = `
      <article class="approval-detail-card">
        <header class="approval-detail-header">
          <div>
            <p class="panel-kicker">Run ${escapeHTML(detail.run_id)}</p>
            <h3>${escapeHTML(primarySummary)}</h3>
          </div>
          <div class="approval-detail-badges">
            <span class="detail-pill detail-pill-state">${escapeHTML(detail.run_state || "unknown")}</span>
            <span class="detail-pill">${escapeHTML(detail.runner_kind || "runner unknown")}</span>
          </div>
        </header>

        ${noticeMarkup}

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Run State And Primary Conclusion</p>
          <p class="detail-copy">${escapeHTML(detail.run_state || "unknown")} • ${escapeHTML(primarySummary)}</p>
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Key Failure Or Result Summary</p>
          <p class="detail-copy">${escapeHTML(primarySummary)}</p>
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Artifact List</p>
          ${renderArtifacts(detail)}
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Related Task Context</p>
          <p class="detail-copy">${escapeHTML(detail.task_id || "Task not recorded")}</p>
          <p class="detail-copy">${escapeHTML(detail.task_summary || "No task summary recorded")}</p>
          ${taskWorkbenchLink}
        </section>
      </article>
    `;

    metadataRoot.innerHTML = `
      <article class="approval-detail-card">
        <section class="detail-grid detail-grid-secondary run-metadata-grid">
          <article class="detail-block">
            <p class="detail-label">Run ID</p>
            <p class="detail-copy">${escapeHTML(detail.run_id)}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Task ID</p>
            <p class="detail-copy">${escapeHTML(detail.task_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Project ID</p>
            <p class="detail-copy">${escapeHTML(detail.project_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Module ID</p>
            <p class="detail-copy">${escapeHTML(detail.module_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Runner</p>
            <p class="detail-copy">${escapeHTML(detail.runner_kind || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Run Created</p>
            <p class="detail-copy">${escapeHTML(detail.run_created_at || "Not recorded")}</p>
          </article>
        </section>
      </article>
    `;
  };

  const fetchJSON = async (url, options) => {
    const response = await fetch(url, options);
    if (response.status === 404) {
      return { notFound: true };
    }
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Request failed with status ${response.status}`);
    }
    return response.json();
  };

  const loadRunDetail = async () => {
    if (!state.runId) {
      state.detail = null;
      state.detailState = "idle";
      state.notice = "";
      renderOverview();
      setStatus("Enter a run_id to load run detail.");
      return;
    }

    state.detailState = "loading";
    state.notice = "";
    updateURLState(state.runId);
    renderOverview();
    setStatus(`Loading ${state.runId}...`);

    try {
      const detail = await fetchJSON(`/api/manager/runs/${encodeURIComponent(state.runId)}/workbench`, {
        method: "GET",
      });

      if (detail.notFound) {
        state.detail = null;
        state.detailState = "not_found";
        renderOverview();
        setStatus(`Run ${state.runId} not found.`, "danger");
        return;
      }

      state.detail = detail;
      state.detailState = "ready";
      renderOverview();
      setStatus(`Loaded ${state.runId}.`);
    } catch (error) {
      console.error(error);
      state.detail = null;
      state.detailState = "error";
      state.notice = error.message || "Failed to load run workbench.";
      state.noticeTone = "danger";
      renderOverview();
      setStatus(`Failed to load ${state.runId}.`, "danger");
    }
  };

  const refreshWorkbench = async () => {
    state.runId = runInput.value.trim();
    runInput.value = state.runId;
    state.noticeTone = "info";
    await loadRunDetail();
  };

  refreshButton.addEventListener("click", refreshWorkbench);
  runInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      refreshWorkbench();
    }
  });
  window.addEventListener("popstate", () => {
    runInput.value = readRunID();
    refreshWorkbench();
  });

  state.runId = readRunID();
  runInput.value = state.runId;
  refreshWorkbench();
}
