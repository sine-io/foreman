const modulesRoot = document.getElementById("modules-root");
const tasksRoot = document.getElementById("tasks-root");
const approvalsRoot = document.getElementById("approvals-root");
const projectInput = document.getElementById("project-id");
const refreshButton = document.getElementById("refresh-board");
const statusNode = document.getElementById("board-status");
const approvalWorkbenchLink = document.getElementById("approval-workbench-link");
const taskWorkbenchLink = document.getElementById("task-workbench-link");

if (modulesRoot && tasksRoot && approvalsRoot && projectInput && refreshButton && statusNode) {
  const updateWorkbenchLink = (projectId) => {
    if (!approvalWorkbenchLink) {
      return;
    }

    approvalWorkbenchLink.href = `/board/approvals/workbench?project_id=${encodeURIComponent(projectId)}`;

    if (taskWorkbenchLink) {
      taskWorkbenchLink.href = `/board/tasks/workbench?project_id=${encodeURIComponent(projectId)}`;
    }
  };

  const escapeHTML = (value) =>
    String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const renderColumns = (root, columns, options = {}) => {
    const { taskWorkbenchLinks = false } = options;
    const entries = Object.entries(columns || {});
    if (!entries.length) {
      root.innerHTML = '<p class="empty-state">No data yet.</p>';
      return;
    }

    root.innerHTML = entries
      .map(
        ([name, items]) => `
          <section class="board-column">
            <header class="column-header">
              <h3>${name}</h3>
              <span>${items.length}</span>
            </header>
            <div class="column-cards">
              ${items
                .map(
                  (item) => `
                    <article class="board-card">
                      <strong>${escapeHTML(item.name || item.summary || item.id)}</strong>
                      <p>${escapeHTML(item.state || item.reason || "")}</p>
                      ${
                        taskWorkbenchLinks && item.id
                          ? `<a class="artifact-link" href="/board/tasks/workbench?project_id=${encodeURIComponent(projectInput.value.trim() || "demo")}&task_id=${encodeURIComponent(item.id)}">Open task workbench</a>`
                          : ""
                      }
                    </article>
                  `,
                )
                .join("")}
            </div>
          </section>
        `,
      )
      .join("");
  };

  const renderApprovals = (items) => {
    if (!items || !items.length) {
      approvalsRoot.innerHTML = '<p class="empty-state">No pending approvals.</p>';
      return;
    }

    approvalsRoot.innerHTML = items
      .map(
        (item) => `
          <article class="approval-card">
            <p class="approval-summary">${escapeHTML(item.summary)}</p>
            <p class="approval-reason">${escapeHTML(item.reason)}</p>
            <p class="approval-meta">task=${escapeHTML(item.task_id)} approval=${escapeHTML(item.approval_id)}</p>
            <a class="artifact-link" href="/board/tasks/workbench?project_id=${encodeURIComponent(projectInput.value.trim() || "demo")}&task_id=${encodeURIComponent(item.task_id)}">Open task workbench</a>
          </article>
        `,
      )
      .join("");
  };

  const loadBoard = async () => {
    const projectId = projectInput.value.trim() || "demo";
    updateWorkbenchLink(projectId);
    statusNode.textContent = `Loading ${projectId}...`;

    try {
      const [modulesRes, tasksRes, approvalsRes] = await Promise.all([
        fetch(`/board/modules?project_id=${encodeURIComponent(projectId)}`),
        fetch(`/board/tasks?project_id=${encodeURIComponent(projectId)}`),
        fetch(`/board/approvals?project_id=${encodeURIComponent(projectId)}`),
      ]);

      const [modules, tasks, approvals] = await Promise.all([
        modulesRes.json(),
        tasksRes.json(),
        approvalsRes.json(),
      ]);

      renderColumns(modulesRoot, modules.columns);
      renderColumns(tasksRoot, tasks.columns, { taskWorkbenchLinks: true });
      renderApprovals(approvals.items);
      statusNode.textContent = `Loaded ${projectId}`;
    } catch (error) {
      console.error(error);
      statusNode.textContent = "Failed to load board data";
    }
  };

  refreshButton.addEventListener("click", loadBoard);
  projectInput.addEventListener("input", () => updateWorkbenchLink(projectInput.value.trim() || "demo"));
  updateWorkbenchLink(projectInput.value.trim() || "demo");
  loadBoard();
}
