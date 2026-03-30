const projectInput = document.getElementById("task-workbench-project-id");
const taskInput = document.getElementById("task-workbench-task-id");
const refreshButton = document.getElementById("task-workbench-refresh");
const statusNode = document.getElementById("task-workbench-status");
const overviewRoot = document.getElementById("task-workbench-overview");
const metadataRoot = document.getElementById("task-workbench-metadata");

if (projectInput && taskInput && refreshButton && statusNode && overviewRoot && metadataRoot) {
  const state = {
    projectId: "demo",
    taskId: "",
    detail: null,
    detailState: "idle",
    busyAction: "",
    notice: "",
    noticeTone: "info",
  };

  const actionLabels = {
    dispatch: "Dispatch",
    cancel: "Cancel",
    reprioritize: "Reprioritize",
    retry: "Retry",
    open_latest_run: "Open Latest Run",
    open_approval_workbench: "Open Approval Workbench",
  };

  const escapeHTML = (value) =>
    String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const readProjectID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("project_id") || "demo";
  };

  const readTaskID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("task_id") || "";
  };

  const updateURLState = (projectId, taskId) => {
    const searchParams = new URLSearchParams(window.location.search);
    searchParams.set("project_id", projectId || "demo");
    if (taskId) {
      searchParams.set("task_id", taskId);
    } else {
      searchParams.delete("task_id");
    }

    const query = searchParams.toString();
    const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname;
    window.history.replaceState({}, "", nextURL);
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
  };

  const findAction = (detail, actionID) =>
    (detail.available_actions || []).find((action) => action.action_id === actionID) || null;

  const actionDisabledReason = (detail, action) => {
    if (!action) {
      return "";
    }

    return action.disabled_reason || (detail.disabled_reasons && detail.disabled_reasons[action.action_id]) || "";
  };

  const renderPrimaryAction = (detail, actionID) => {
    const action = findAction(detail, actionID) || { action_id: actionID, enabled: false };
    const disabledReason = actionDisabledReason(detail, action);
    const label = actionLabels[actionID] || actionID;
    const isBusy = state.busyAction === actionID;
    const disabled = !action.enabled || Boolean(state.busyAction);

    return `
      <article class="task-action-card${disabled ? " is-disabled" : ""}">
        <div class="task-action-copy">
          <strong>${escapeHTML(label)}</strong>
          <p>${escapeHTML(disabledReason || "Available now")}</p>
        </div>
        ${
          actionID === "reprioritize"
            ? `
              <label class="task-action-inline-field">
                <span>Priority</span>
                <input
                  id="task-workbench-priority"
                  class="board-input task-priority-input"
                  type="number"
                  min="1"
                  value="${escapeHTML(action.current_value || detail.priority || 1)}"
                  ${disabled ? "disabled" : ""}
                />
              </label>
              <button class="board-button board-button-secondary" type="button" data-task-action="reprioritize" ${disabled ? "disabled" : ""}>${escapeHTML(
                isBusy ? "Saving..." : label,
              )}</button>
            `
            : `
              <button class="board-button${actionID === "cancel" ? " board-button-secondary" : ""}" type="button" data-task-action="${escapeHTML(
                actionID,
              )}" ${disabled ? "disabled" : ""}>${escapeHTML(isBusy ? "Working..." : label)}</button>
            `
        }
      </article>
    `;
  };

  const renderLinkAction = (detail, actionID, href, fallbackReason) => {
    const action = findAction(detail, actionID) || { action_id: actionID, enabled: false };
    const disabledReason = actionDisabledReason(detail, action) || fallbackReason;
    const label = actionLabels[actionID] || actionID;

    return `
      <article class="task-action-card${action.enabled && href ? "" : " is-disabled"}">
        <div class="task-action-copy">
          <strong>${escapeHTML(label)}</strong>
          <p>${escapeHTML(action.enabled && href ? "Open detail view" : disabledReason)}</p>
        </div>
        ${
          action.enabled && href
            ? `<a class="board-link" href="${escapeHTML(href)}">${escapeHTML(label)}</a>`
            : `<span class="board-link board-link-disabled" aria-disabled="true">${escapeHTML(label)}</span>`
        }
      </article>
    `;
  };

  const renderArtifacts = (detail) => {
    if (!detail.artifacts || !detail.artifacts.length) {
      return '<p class="empty-state">No task artifacts recorded yet.</p>';
    }

    return `
      <div class="artifact-list">
        ${detail.artifacts
          .map(
            (artifact) => `
              <article class="artifact-card">
                <p class="artifact-kind">${escapeHTML(artifact.kind || artifact.id)}</p>
                <strong>${escapeHTML(artifact.summary || artifact.path || artifact.id)}</strong>
                <p class="detail-copy">${escapeHTML(artifact.path || "Artifact path not recorded")}</p>
              </article>
            `,
          )
          .join("")}
      </div>
    `;
  };

  const renderOverview = () => {
    if (state.detailState === "idle") {
      overviewRoot.innerHTML =
        '<p class="empty-state">Enter a task_id to load a task workbench deep link.</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Task metadata will appear here.</p>';
      return;
    }

    if (state.detailState === "loading") {
      overviewRoot.innerHTML = '<p class="empty-state">Loading task detail...</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Loading task metadata...</p>';
      return;
    }

    if (state.detailState === "not_found") {
      overviewRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Task not found</p>
          <p class="detail-copy">
            No task with ID <code>${escapeHTML(state.taskId)}</code> exists for project
            <code>${escapeHTML(state.projectId)}</code>.
          </p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Task metadata unavailable.</p>';
      return;
    }

    if (state.detailState === "error") {
      overviewRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Unable to load task detail</p>
          <p class="detail-copy">${escapeHTML(state.notice || "Refresh and try again.")}</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Task metadata unavailable.</p>';
      return;
    }

    const detail = state.detail;
    const latestRunSummary = detail.latest_run_id
      ? `
          <section class="detail-block detail-block-wide">
            <p class="detail-label">Latest run</p>
            <p class="detail-copy">${escapeHTML(detail.latest_run_id)} • ${escapeHTML(detail.latest_run_state || "unknown")}</p>
            <p class="detail-copy">${escapeHTML(detail.latest_run_summary || "No summary recorded")}</p>
            ${
              detail.run_detail_url
                ? `<a class="artifact-link" href="${escapeHTML(detail.run_detail_url)}">Open run detail</a>`
                : '<span class="artifact-link artifact-link-muted">Run detail unavailable</span>'
            }
          </section>
        `
      : `
          <section class="detail-block detail-block-wide">
            <p class="detail-label">Latest run</p>
            <p class="detail-copy">No latest run yet.</p>
          </section>
        `;
    const approvalHref = detail.latest_approval_id ? detail.approval_workbench_url : "";
    const approvalSummary = `
      <section class="detail-block detail-block-wide">
        <p class="detail-label">Approval summary</p>
        <p class="detail-copy">${escapeHTML(detail.latest_approval_id || "No approval history")}</p>
        <p class="detail-copy">${escapeHTML(detail.latest_approval_reason || (detail.latest_approval_id ? "No approval reason recorded" : "No approval history"))}</p>
        ${
          detail.latest_approval_id
            ? `<a class="artifact-link" href="${escapeHTML(approvalHref)}">Open approval workbench</a>`
            : '<span class="artifact-link artifact-link-muted">No approval history</span>'
        }
      </section>
    `;
    const noticeMarkup = state.notice
      ? `<p class="detail-notice tone-${escapeHTML(state.noticeTone)}">${escapeHTML(state.notice)}</p>`
      : "";

    overviewRoot.innerHTML = `
      <article class="approval-detail-card">
        <header class="approval-detail-header">
          <div>
            <p class="panel-kicker">Task ${escapeHTML(detail.task_id)}</p>
            <h3>${escapeHTML(detail.summary || detail.task_id)}</h3>
          </div>
          <div class="approval-detail-badges">
            <span class="detail-pill detail-pill-state">${escapeHTML(detail.task_state || "unknown")}</span>
            <span class="detail-pill">Priority ${escapeHTML(detail.priority)}</span>
          </div>
        </header>

        ${noticeMarkup}

        <section class="detail-grid">
          <article class="detail-block">
            <p class="detail-label">Project</p>
            <p class="detail-copy">${escapeHTML(detail.project_id)}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Module</p>
            <p class="detail-copy">${escapeHTML(detail.module_id || "Not recorded")}</p>
          </article>
        </section>

        <section class="task-action-grid">
          ${renderPrimaryAction(detail, "dispatch")}
          ${renderPrimaryAction(detail, "retry")}
          ${renderPrimaryAction(detail, "cancel")}
          ${renderPrimaryAction(detail, "reprioritize")}
          ${renderLinkAction(detail, "open_latest_run", detail.run_detail_url, "No latest run")}
          ${renderLinkAction(detail, "open_approval_workbench", approvalHref, "No approval history")}
        </section>

        ${latestRunSummary}
        ${approvalSummary}

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Artifacts</p>
          ${renderArtifacts(detail)}
        </section>
      </article>
    `;

    metadataRoot.innerHTML = `
      <article class="approval-detail-card">
        <section class="detail-block detail-block-wide">
          <p class="detail-label">Write scope</p>
          <p class="detail-copy">${escapeHTML(detail.write_scope || "Not recorded")}</p>
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Task type</p>
          <p class="detail-copy">${escapeHTML(detail.task_type || "Not recorded")}</p>
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Acceptance</p>
          <p class="detail-copy">${escapeHTML(detail.acceptance || "Not recorded")}</p>
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

  const loadTaskDetail = async () => {
    if (!state.taskId) {
      state.detail = null;
      state.detailState = "idle";
      state.notice = "";
      renderOverview();
      setStatus("Enter a task_id to load task detail.");
      return;
    }

    state.detailState = "loading";
    state.notice = "";
    updateURLState(state.projectId, state.taskId);
    renderOverview();
    setStatus(`Loading ${state.taskId}...`);

    try {
      const detail = await fetchJSON(
        `/api/manager/tasks/${encodeURIComponent(state.taskId)}/workbench?project_id=${encodeURIComponent(state.projectId)}`,
        { method: "GET" },
      );

      if (detail.notFound) {
        state.detail = null;
        state.detailState = "not_found";
        renderOverview();
        setStatus(`Task ${state.taskId} not found.`, "danger");
        return;
      }

      state.detail = detail;
      state.detailState = "ready";
      renderOverview();
      setStatus(`Loaded ${state.taskId}.`);
    } catch (error) {
      console.error(error);
      state.detail = null;
      state.detailState = "error";
      state.notice = error.message || "Failed to load task workbench.";
      state.noticeTone = "danger";
      renderOverview();
      setStatus(`Failed to load ${state.taskId}.`, "danger");
    }
  };

  const refreshWorkbench = async () => {
    state.projectId = projectInput.value.trim() || "demo";
    state.taskId = taskInput.value.trim();
    projectInput.value = state.projectId;
    taskInput.value = state.taskId;
    state.noticeTone = "info";
    await loadTaskDetail();
  };

  const submitAction = async (actionID) => {
    if (!state.detail || !state.taskId || state.busyAction) {
      return;
    }

    let body;
    let endpoint = `/api/manager/tasks/${encodeURIComponent(state.taskId)}/${actionID}?project_id=${encodeURIComponent(state.projectId)}`;
    if (actionID === "reprioritize") {
      const priorityNode = document.getElementById("task-workbench-priority");
      const priorityValue = Number(priorityNode ? priorityNode.value : 0);
      if (!Number.isInteger(priorityValue) || priorityValue < 1) {
        state.notice = "Priority must be an integer >= 1.";
        state.noticeTone = "warning";
        renderOverview();
        return;
      }

      body = JSON.stringify({ priority: priorityValue });
    }

    state.busyAction = actionID;
    state.notice = "";
    state.noticeTone = "info";
    renderOverview();

    try {
      const response = await fetchJSON(endpoint, {
        method: "POST",
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body,
      });

      state.notice = response.message || `${actionLabels[actionID] || actionID} completed.`;
      state.noticeTone = "info";
      await loadTaskDetail();
      setStatus(`Updated ${state.taskId}.`);
    } catch (error) {
      console.error(error);
      state.notice = error.message || "Task action failed.";
      state.noticeTone = "danger";
      renderOverview();
      setStatus(`Failed to update ${state.taskId}.`, "danger");
    } finally {
      state.busyAction = "";
      renderOverview();
    }
  };

  overviewRoot.addEventListener("click", async (event) => {
    const button = event.target.closest("[data-task-action]");
    if (!button) {
      return;
    }

    await submitAction(button.dataset.taskAction);
  });

  refreshButton.addEventListener("click", refreshWorkbench);
  projectInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      refreshWorkbench();
    }
  });
  taskInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      refreshWorkbench();
    }
  });
  window.addEventListener("popstate", () => {
    projectInput.value = readProjectID();
    taskInput.value = readTaskID();
    refreshWorkbench();
  });

  state.projectId = readProjectID();
  state.taskId = readTaskID();
  projectInput.value = state.projectId;
  taskInput.value = state.taskId;
  refreshWorkbench();
}
