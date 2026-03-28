const projectInput = document.getElementById("approval-workbench-project-id");
const refreshButton = document.getElementById("approval-workbench-refresh");
const statusNode = document.getElementById("approval-workbench-status");
const queueRoot = document.getElementById("approval-workbench-queue");
const detailRoot = document.getElementById("approval-workbench-detail");

if (projectInput && refreshButton && statusNode && queueRoot && detailRoot) {
  const state = {
    projectId: "demo",
    queue: [],
    selectedApprovalID: "",
    detail: null,
    detailState: "idle",
    queueState: "idle",
    busy: false,
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

  const readProjectID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("project_id") || "demo";
  };

  const readApprovalID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("approval_id") || "";
  };

  const updateURLState = (projectId, approvalId) => {
    const searchParams = new URLSearchParams(window.location.search);
    searchParams.set("project_id", projectId || "demo");
    if (approvalId) {
      searchParams.set("approval_id", approvalId);
    } else {
      searchParams.delete("approval_id");
    }

    const query = searchParams.toString();
    const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname;
    window.history.replaceState({}, "", nextURL);
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
  };

  const clearWorkbenchState = () => {
    state.queue = [];
    state.detail = null;
    state.selectedApprovalID = "";
    state.detailState = "idle";
    state.queueState = "loading";
  };

  const renderQueue = () => {
    if (state.queueState === "loading" && !state.queue.length) {
      queueRoot.innerHTML = '<p class="empty-state">Loading queue...</p>';
      return;
    }

    if (!state.queue.length) {
      queueRoot.innerHTML =
        '<p class="empty-state">No pending approvals for this project right now.</p>';
      return;
    }

    queueRoot.innerHTML = state.queue
      .map((item) => {
        const selectedClass = item.approval_id === state.selectedApprovalID ? " is-selected" : "";
        return `
          <button
            type="button"
            class="approval-queue-item${selectedClass}"
            data-approval-id="${escapeHTML(item.approval_id)}"
          >
            <span class="approval-queue-risk risk-${escapeHTML(item.risk_level || "unknown")}">${escapeHTML(
              item.risk_level || "unknown",
            )}</span>
            <strong>${escapeHTML(item.summary || item.approval_id)}</strong>
            <p>task ${escapeHTML(item.task_id)} • priority ${escapeHTML(item.priority)}</p>
          </button>
        `;
      })
      .join("");
  };

  const renderArtifacts = (detail) => {
    if (!detail.artifacts || !detail.artifacts.length) {
      return '<p class="empty-state">No recorded artifacts.</p>';
    }

    return `
      <div class="artifact-list">
        ${detail.artifacts
          .map((artifact) => {
            const runLink = detail.run_detail_url
              ? `<a class="artifact-link" href="${escapeHTML(detail.run_detail_url)}">Open run view</a>`
              : '<span class="artifact-link artifact-link-muted">Run unavailable</span>';
            return `
              <article class="artifact-card">
                <p class="artifact-kind">${escapeHTML(artifact.kind || artifact.id)}</p>
                <strong>${escapeHTML(artifact.summary || artifact.path || artifact.id)}</strong>
                ${runLink}
              </article>
            `;
          })
          .join("")}
      </div>
    `;
  };

  const renderDetail = () => {
    if (state.detailState === "loading") {
      detailRoot.innerHTML = '<p class="empty-state">Loading approval detail...</p>';
      return;
    }

    if (state.detailState === "not_found") {
      detailRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Approval not found</p>
          <p class="detail-copy">
            No approval with ID <code>${escapeHTML(state.selectedApprovalID)}</code> exists for this
            environment.
          </p>
        </article>
      `;
      return;
    }

    if (state.detailState === "error") {
      detailRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Unable to load approval detail</p>
          <p class="detail-copy">${escapeHTML(state.notice || "Refresh and try again.")}</p>
        </article>
      `;
      return;
    }

    if (!state.detail) {
      detailRoot.innerHTML =
        '<p class="empty-state">Choose an approval from the queue or load one with an approval_id deep link.</p>';
      return;
    }

    const detail = state.detail;
    const isPending = detail.approval_state === "pending";
    const canRetryDispatch = detail.task_state === "approved_pending_dispatch";
    const rejectionValue = state.noticeTone === "warning" ? state.notice : detail.rejection_reason || "";
    const noticeMarkup = state.notice
      ? `<p class="detail-notice tone-${escapeHTML(state.noticeTone)}">${escapeHTML(state.notice)}</p>`
      : "";
    const actions = `
      <section class="approval-action-bar">
        ${
          isPending
            ? `
              <button class="board-button" type="button" data-action="approve" data-approval-id="${escapeHTML(detail.approval_id)}"${
                state.busy ? " disabled" : ""
              }>Approve</button>
              <button class="board-button board-button-secondary" type="button" data-action="reject" data-approval-id="${escapeHTML(detail.approval_id)}"${
                state.busy ? " disabled" : ""
              }>Reject</button>
            `
            : ""
        }
        ${
          canRetryDispatch
            ? `
              <button class="board-button board-button-secondary" type="button" data-action="retry-dispatch" data-approval-id="${escapeHTML(detail.approval_id)}"${
                state.busy ? " disabled" : ""
              }>Retry Dispatch</button>
            `
            : ""
        }
      </section>
    `;
    const rejectionInput = isPending
      ? `
        <label class="detail-field">
          <span>Rejection note</span>
          <textarea id="approval-rejection-reason" class="detail-textarea" rows="3" placeholder="Explain why this should be rejected.">${escapeHTML(
            rejectionValue,
          )}</textarea>
        </label>
      `
      : "";
    const runSummary = detail.run_detail_url
      ? `<a class="artifact-link" href="${escapeHTML(detail.run_detail_url)}">Open run view</a>`
      : '<span class="artifact-link artifact-link-muted">Run detail unavailable</span>';

    detailRoot.innerHTML = `
      <article class="approval-detail-card">
        <header class="approval-detail-header">
          <div>
            <p class="panel-kicker">Approval ${escapeHTML(detail.approval_id)}</p>
            <h3>${escapeHTML(detail.summary || detail.task_id)}</h3>
          </div>
          <div class="approval-detail-badges">
            <span class="approval-queue-risk risk-${escapeHTML(detail.risk_level || "unknown")}">${escapeHTML(
              detail.risk_level || "unknown",
            )}</span>
            <span class="detail-pill detail-pill-state">${escapeHTML(detail.approval_state)}</span>
          </div>
        </header>

        ${noticeMarkup}

        <section class="detail-grid">
          <article class="detail-block">
            <p class="detail-label">Risk</p>
            <p class="detail-copy">${escapeHTML(detail.risk_level || "unknown")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Approval reason</p>
            <p class="detail-copy">${escapeHTML(detail.reason || "No reason supplied.")}</p>
          </article>
        </section>

        ${actions}
        ${rejectionInput}

        <section class="detail-grid detail-grid-secondary">
          <article class="detail-block">
            <p class="detail-label">Task</p>
            <p class="detail-copy">${escapeHTML(detail.task_id)} • ${escapeHTML(detail.task_state || "unknown")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Policy rule</p>
            <p class="detail-copy">${escapeHTML(detail.policy_rule || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Created</p>
            <p class="detail-copy">${escapeHTML(detail.created_at || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Run</p>
            <p class="detail-copy">${escapeHTML(detail.run_id || "No run yet")} • ${escapeHTML(
              detail.run_state || "n/a",
            )}</p>
            ${runSummary}
          </article>
        </section>

        ${
          detail.rejection_reason
            ? `
              <section class="detail-block detail-block-wide">
                <p class="detail-label">Rejection reason</p>
                <p class="detail-copy">${escapeHTML(detail.rejection_reason)}</p>
              </section>
            `
            : ""
        }

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Assistant summary preview</p>
          <p class="detail-copy">${escapeHTML(detail.assistant_summary_preview || "No summary preview.")}</p>
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Artifacts</p>
          ${renderArtifacts(detail)}
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

  const loadQueue = async () => {
    state.queueState = "loading";
    renderQueue();

    const projectId = state.projectId;
    const data = await fetchJSON(
      `/api/manager/projects/${encodeURIComponent(projectId)}/approvals`,
      { method: "GET" },
    );

    state.queue = data.items || [];
    state.queueState = "ready";
    renderQueue();
  };

  const loadApproval = async (approvalId) => {
    state.selectedApprovalID = approvalId;
    state.detailState = "loading";
    state.notice = "";
    updateURLState(state.projectId, approvalId);
    renderQueue();
    renderDetail();

    const data = await fetchJSON(`/api/manager/approvals/${encodeURIComponent(approvalId)}`, {
      method: "GET",
    });

    if (data.notFound) {
      state.detail = null;
      state.detailState = "not_found";
      renderQueue();
      renderDetail();
      return;
    }

    state.detail = data;
    state.detailState = "ready";
    renderQueue();
    renderDetail();
  };

  const selectApproval = async (approvalId) => {
    await loadApproval(approvalId);
  };

  const refreshWorkbench = async () => {
    const previousProjectId = state.projectId;
    const nextProjectId = projectInput.value.trim() || "demo";
    const projectChanged = nextProjectId !== previousProjectId;
    const requestedApprovalID = projectChanged ? "" : readApprovalID();

    state.projectId = nextProjectId;
    projectInput.value = state.projectId;
    state.notice = "";
    state.noticeTone = "info";
    clearWorkbenchState();
    updateURLState(state.projectId, requestedApprovalID);
    renderQueue();
    renderDetail();
    setStatus(`Loading ${state.projectId} approvals...`);

    try {
      await loadQueue();
      const queuedApproval = requestedApprovalID
        ? state.queue.find((item) => item.approval_id === requestedApprovalID)
        : null;

      if (queuedApproval) {
        await selectApproval(requestedApprovalID);
      } else if (requestedApprovalID && !projectChanged) {
        await loadApproval(requestedApprovalID);
      } else if (state.queue.length) {
        await selectApproval(state.queue[0].approval_id);
      } else {
        state.selectedApprovalID = "";
        state.detail = null;
        state.detailState = "idle";
        updateURLState(state.projectId, "");
        renderQueue();
        renderDetail();
      }

      setStatus(`Loaded ${state.projectId} approvals.`);
    } catch (error) {
      console.error(error);
      state.queueState = "error";
      state.detail = null;
      state.detailState = "error";
      state.notice = error.message || "Failed to load approval workbench.";
      state.noticeTone = "danger";
      renderQueue();
      renderDetail();
      setStatus(`Failed to load ${state.projectId} approvals.`, "danger");
    }
  };

  const submitAction = async (actionName, approvalId) => {
    if (state.busy) {
      return;
    }

    let body;
    let endpoint = `/api/manager/approvals/${encodeURIComponent(approvalId)}/${actionName}`;
    if (actionName === "retry-dispatch") {
      endpoint = `/api/manager/approvals/${encodeURIComponent(approvalId)}/retry-dispatch`;
    }

    if (actionName === "reject") {
      const rejectionReasonNode = document.getElementById("approval-rejection-reason");
      const rejectionReason = rejectionReasonNode ? rejectionReasonNode.value.trim() : "";
      if (!rejectionReason) {
        state.notice = "Rejection note is required before rejecting an approval.";
        state.noticeTone = "warning";
        renderDetail();
        return;
      }

      body = JSON.stringify({ rejection_reason: rejectionReason });
    }

    state.busy = true;
    state.notice = "";
    state.noticeTone = "info";
    renderDetail();

    try {
      await fetchJSON(endpoint, {
        method: "POST",
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body,
      });

      const currentIndex = state.queue.findIndex((item) => item.approval_id === approvalId);
      const nextItem = state.queue[currentIndex + 1] || state.queue[currentIndex - 1] || null;
      state.queue = state.queue.filter((item) => item.approval_id !== approvalId);
      renderQueue();

      if (nextItem) {
        await selectApproval(nextItem.approval_id);
      } else {
        await loadApproval(approvalId);
      }

      state.notice = `${actionName} completed for ${approvalId}.`;
      state.noticeTone = "info";
      setStatus(`Updated ${approvalId}.`);
    } catch (error) {
      console.error(error);
      state.notice = error.message || "Approval action failed.";
      state.noticeTone = "danger";
      renderDetail();
      setStatus(`Failed to update ${approvalId}.`, "danger");
    } finally {
      state.busy = false;
      renderDetail();
    }
  };

  queueRoot.addEventListener("click", async (event) => {
    const button = event.target.closest("[data-approval-id]");
    if (!button) {
      return;
    }

    await selectApproval(button.dataset.approvalId);
  });

  detailRoot.addEventListener("click", async (event) => {
    const button = event.target.closest("[data-action][data-approval-id]");
    if (!button) {
      return;
    }

    await submitAction(button.dataset.action, button.dataset.approvalId);
  });

  refreshButton.addEventListener("click", refreshWorkbench);
  projectInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      refreshWorkbench();
    }
  });
  window.addEventListener("popstate", () => {
    projectInput.value = readProjectID();
    refreshWorkbench();
  });

  state.projectId = readProjectID();
  projectInput.value = state.projectId;
  refreshWorkbench();
}
