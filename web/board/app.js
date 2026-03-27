const root = document.getElementById("board-root");

if (root) {
  const cards = [
    { title: "Modules", body: "GET /board/modules?project_id=<id>" },
    { title: "Tasks", body: "GET /board/tasks?project_id=<id>" },
    { title: "Runs", body: "GET /board/runs/<id>" },
    { title: "Gateway", body: "POST /gateways/openclaw/command" },
  ];

  root.innerHTML = cards
    .map(
      (card) => `
        <article class="board-card">
          <h2>${card.title}</h2>
          <p>${card.body}</p>
        </article>
      `,
    )
    .join("");
}
