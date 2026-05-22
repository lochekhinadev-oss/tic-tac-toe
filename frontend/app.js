const EMPTY = 0;
const USER = 1;
const COMPUTER = 2;
const BOARD_SIZE = 3;

const boardNode = document.getElementById("board");
const statusNode = document.getElementById("status");
const newGameButton = document.getElementById("new-game");
const newPlayerGameButton = document.getElementById("new-player-game");
const gamesNode = document.getElementById("games");
const leaderboardNode = document.getElementById("leaderboard");
const joinForm = document.getElementById("join-form");
const joinGameInput = document.getElementById("join-game-id");
const authForm = document.getElementById("auth-form");
const signinButton = document.getElementById("signin");
const signupButton = document.getElementById("signup");
const logoutButton = document.getElementById("logout");
const deleteAccountButton = document.getElementById("delete-account");
const deleteConfirmModal = document.getElementById("delete-confirm-modal");
const deleteConfirmYesButton = document.getElementById("delete-confirm-yes");
const deleteConfirmCancelButtons = document.querySelectorAll("[data-delete-cancel]");
const loginInput = document.getElementById("login");
const passwordInput = document.getElementById("password");
const youSymbolNode = document.getElementById("you-symbol");
const computerSymbolNode = document.getElementById("computer-symbol");
const sessionNode = document.getElementById("session-id");

const storageKeys = {
  gameID: "tic-tac-toe.gameID"
};

let gameID = "";
let userUUID = "";
let board = createEmptyBoard();
let currentGame = null;
let busy = false;
let finished = false;
let deleteConfirmResolver = null;
let deleteConfirmPreviousFocus = null;

function createEmptyBoard() {
  return Array.from({ length: BOARD_SIZE }, () => Array(BOARD_SIZE).fill(EMPTY));
}

function cloneBoard(source) {
  return source.map((row) => row.slice());
}

function symbolForCell(value) {
  if (value === USER) {
    return "🐾";
  }

  if (value === COMPUTER) {
    return "○";
  }

  return "";
}

function winningCellsForBoard(sourceBoard) {
  const lines = [
    [[0, 0], [0, 1], [0, 2]],
    [[1, 0], [1, 1], [1, 2]],
    [[2, 0], [2, 1], [2, 2]],
    [[0, 0], [1, 0], [2, 0]],
    [[0, 1], [1, 1], [2, 1]],
    [[0, 2], [1, 2], [2, 2]],
    [[0, 0], [1, 1], [2, 2]],
    [[0, 2], [1, 1], [2, 0]]
  ];

  for (const line of lines) {
    const [[r1, c1], [r2, c2], [r3, c3]] = line;
    const value = sourceBoard[r1][c1];
    if (value !== EMPTY && value === sourceBoard[r2][c2] && value === sourceBoard[r3][c3]) {
      return line;
    }
  }

  return [];
}

function updateStatus(message) {
  const text = String(message || "").trim();
  statusNode.textContent = text.startsWith(">") ? text : `> ${text}`;
}

function updateUser() {
  youSymbolNode.textContent = "🐾";
  computerSymbolNode.textContent = "○";
}

function updateSession() {
  sessionNode.textContent = gameID || "Нет активной игры";
}

function updateAuthControls() {
  const loggedIn = Boolean(userUUID);
  logoutButton.disabled = !loggedIn;
  deleteAccountButton.disabled = !loggedIn;
}

function renderLeaderboardEmpty(message) {
  leaderboardNode.innerHTML = "";
  const empty = document.createElement("div");
  empty.className = "leaderboard-empty";
  empty.textContent = message;
  leaderboardNode.appendChild(empty);
}

function renderLeaderboard(players) {
  leaderboardNode.innerHTML = "";

  if (!players || players.length === 0) {
    renderLeaderboardEmpty("Пока нет данных для таблицы лидеров.");
    return;
  }

  players.forEach((player, index) => {
    const row = document.createElement("div");
    row.className = "leaderboard-item";

    const rank = document.createElement("div");
    rank.className = "leaderboard-rank";
    rank.textContent = String(index + 1);

    const login = document.createElement("div");
    login.className = "leaderboard-login";
    login.textContent = player.login || player.uuid || "unknown";

    const ratio = document.createElement("div");
    ratio.className = "leaderboard-ratio";
    ratio.textContent = `Win ratio: ${Number(player.winRatio || 0).toFixed(2)}`;

    row.appendChild(rank);
    row.appendChild(login);
    row.appendChild(ratio);
    leaderboardNode.appendChild(row);
  });
}

function clearGameState() {
  gameID = "";
  currentGame = null;
  board = createEmptyBoard();
  finished = false;
  localStorage.removeItem(storageKeys.gameID);
  updateSession();
  renderBoard();
}

function clearAuthState(message) {
  userUUID = "";
  clearGameState();
  gamesNode.innerHTML = "";
  renderLeaderboardEmpty("sign in to unlock the leaderboard.");
  updateUser();
  updateAuthControls();
  updateStatus(message || "awaiting login...");
}

function openDeleteConfirm() {
  if (!deleteConfirmModal) {
    return Promise.resolve(false);
  }

  deleteConfirmPreviousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  deleteConfirmModal.hidden = false;
  document.body.classList.add("modal-open");

  if (deleteConfirmYesButton) {
    deleteConfirmYesButton.focus();
  }

  return new Promise((resolve) => {
    deleteConfirmResolver = resolve;
  });
}

function closeDeleteConfirm(result) {
  if (!deleteConfirmModal || !deleteConfirmResolver) {
    return;
  }

  const resolve = deleteConfirmResolver;
  deleteConfirmResolver = null;
  deleteConfirmModal.hidden = true;
  document.body.classList.remove("modal-open");
  resolve(result);

  if (deleteConfirmPreviousFocus instanceof HTMLElement) {
    deleteConfirmPreviousFocus.focus();
  }

  deleteConfirmPreviousFocus = null;
}

function cancelDeleteConfirm() {
  closeDeleteConfirm(false);
  updateStatus("deletion cancelled.");
}

async function readResponsePayload(response) {
  return response.json().catch(() => ({}));
}

async function api(path, options = {}) {
  const headers = Object.assign({}, options.headers || {});
  if (options.body && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(path, Object.assign({}, options, {
    headers,
    credentials: "include"
  }));
  const payload = await readResponsePayload(response);
  if (!response.ok) {
    if (response.status === 401 && userUUID) {
      clearAuthState("session expired. sign in again.");
    }
    throw new Error(payload.message || "request failed");
  }
  return payload;
}

async function loadLeaderboard() {
  if (!userUUID) {
    renderLeaderboardEmpty("sign in to unlock the leaderboard.");
    return;
  }

  const payload = await api("/games/leaderboard?n=10", { method: "GET" });
  renderLeaderboard(payload.players || []);
}

function credentials() {
  const login = loginInput.value.trim();
  const password = passwordInput.value;
  if (!login || !password) {
    throw new Error("Введите login и password.");
  }
  return { login, password };
}

async function signUp() {
  const data = credentials();
  await api("/users", {
    method: "POST",
    body: JSON.stringify(data)
  });
  updateStatus("account created. please sign in.");
}

async function signIn(event = null) {
  if (event) {
    event.preventDefault();
  }

  const data = credentials();
  const payload = await api("/auth/sessions", {
    method: "POST",
    body: JSON.stringify(data)
  });

  userUUID = payload.uuid || "";
  updateUser();
  updateAuthControls();
  updateStatus("session created.");
  await startNewGame();
}

function syncState(game) {
  currentGame = game;
  board = game.field;
  gameID = game.uuid;
  localStorage.setItem(storageKeys.gameID, gameID);
  updateUser();
  updateSession();

  if (game.state === "player_wins") {
    finished = true;
    updateStatus(game.winner_uuid === userUUID ? "victory detected ✨" : "opponent won the match.");
    return;
  }

  if (game.state === "draw") {
    finished = true;
    updateStatus("draw detected. paws at peace.");
    return;
  }

  if (game.state === "waiting_players") {
    finished = false;
    updateStatus("waiting for second player...");
    return;
  }

  finished = false;
  updateStatus(game.next_player_uuid === userUUID ? "your turn" : "opponent turn");
}

function renderBoard() {
  boardNode.innerHTML = "";
  const winningCells = currentGame && currentGame.state === "player_wins" ? winningCellsForBoard(board) : [];

  for (let row = 0; row < BOARD_SIZE; row += 1) {
    for (let col = 0; col < BOARD_SIZE; col += 1) {
      const value = board[row][col];
      const button = document.createElement("button");
      button.type = "button";
      button.className = "cell";
      button.textContent = symbolForCell(value);

      if (value === USER) {
        button.classList.add("player");
      }

      if (value === COMPUTER) {
        button.classList.add("computer");
      }

      if (value !== EMPTY) {
        button.classList.add("cell--filled");
      }

      if (winningCells.some(([winRow, winCol]) => winRow === row && winCol === col)) {
        button.classList.add("cell--win");
      }

      button.disabled = busy || finished || !userUUID || value !== EMPTY;
      button.setAttribute("aria-label", `Клетка ${row + 1}-${col + 1}${value ? `: ${symbolForCell(value)}` : ", пустая"}`);
      button.addEventListener("click", () => {
        void makeMove(row, col);
      });
      boardNode.appendChild(button);
    }
  }
}

async function makeMove(row, col) {
  if (
    busy ||
    finished ||
    !userUUID ||
    !currentGame ||
    currentGame.next_player_uuid !== userUUID ||
    board[row][col] !== EMPTY
  ) {
    return;
  }

  const previousBoard = cloneBoard(board);
  const nextBoard = cloneBoard(board);
  nextBoard[row][col] = playerSymbol();

  busy = true;
  board = nextBoard;
  updateStatus(currentGame.mode === "computer" ? "computer thinking..." : "sending move...");
  renderBoard();

  try {
    const game = await api("/games/" + encodeURIComponent(gameID) + "/move", {
      method: "POST",
      body: JSON.stringify({
        uuid: gameID,
        field: board
      })
    });
    syncState(game);
    if (game.state === "player_wins" || game.state === "draw") {
      await loadLeaderboard();
    }
  } catch (error) {
    board = previousBoard;
    updateStatus(error.message || "move failed.");
  } finally {
    busy = false;
    renderBoard();
  }
}

function playerSymbol() {
  if (currentGame && currentGame.player_o_uuid === userUUID) {
    return COMPUTER;
  }

  return USER;
}

async function startNewGame() {
  await createGame("computer");
}

async function hostPlayerGame() {
  await createGame("player");
  await loadGames();
}

async function createGame(mode) {
  if (!userUUID) {
    board = createEmptyBoard();
    currentGame = null;
    finished = false;
    updateStatus("sign in to start a match.");
    renderBoard();
    return;
  }

  busy = true;
  renderBoard();
  try {
    const game = await api("/games", {
      method: "POST",
      body: JSON.stringify({ mode })
    });
    syncState(game);
    await loadGames();
    await loadLeaderboard();
  } catch (error) {
    board = createEmptyBoard();
    currentGame = null;
    finished = false;
    updateStatus(error.message || "failed to create match.");
  } finally {
    busy = false;
    renderBoard();
  }
}

async function loadGames() {
  if (!userUUID) {
    gamesNode.innerHTML = "";
    return;
  }

  const payload = await api("/games", { method: "GET" });
  gamesNode.innerHTML = "";
  if (payload.games.length === 0) {
    const empty = document.createElement("div");
    empty.className = "game-row";
    empty.textContent = "Нет доступных PvP-игр.";
    gamesNode.appendChild(empty);
    return;
  }

  payload.games.forEach((game) => {
    const row = document.createElement("div");
    row.className = "game-row";

    const label = document.createElement("span");
    label.textContent = game.uuid;

    const button = document.createElement("button");
    button.className = "secondary";
    button.type = "button";
    button.textContent = "Войти";
    button.addEventListener("click", () => {
      void joinGame(game.uuid);
    });

    row.appendChild(label);
    row.appendChild(button);
    gamesNode.appendChild(row);
  });
}

async function refreshCurrentGame() {
  if (!userUUID || !gameID) {
    return;
  }

  try {
    const game = await api("/games/" + encodeURIComponent(gameID), { method: "GET" });
    syncState(game);
    if (game.state === "player_wins" || game.state === "draw") {
      await loadLeaderboard();
    }
  } catch (error) {
    updateStatus(error.message || "failed to refresh match.");
  } finally {
    renderBoard();
  }
}

async function restoreSession() {
  try {
    const user = await api("/users/me", { method: "GET" });
    userUUID = user.uuid;
    updateUser();
    updateAuthControls();

    clearGameState();
    updateStatus("session restored.");

    await loadGames();
    await loadLeaderboard();
    await startNewGame();
    return true;
  } catch (error) {
    clearAuthState("awaiting login...");
    return false;
  }
}

async function signOut() {
  try {
    await api("/auth/sessions/current", {
      method: "DELETE"
    });
  } catch (error) {
    // Clear local state even if the server-side revoke failed or the session is already invalid.
  } finally {
    clearAuthState("signed out.");
  }
}

async function deleteAccount() {
  if (!userUUID) {
    clearAuthState("sign in to delete the account.");
    return;
  }

  const confirmed = await openDeleteConfirm();
  if (!confirmed) {
    return;
  }

  let deleted = false;
  try {
    await api("/users/me", {
      method: "DELETE"
    });
    deleted = true;
  } catch (error) {
    if ((error.message || "").toLowerCase().includes("not found")) {
      deleted = true;
      return;
    }
    throw error;
  } finally {
    if (deleted) {
      clearAuthState("account deleted.");
    }
  }
}

async function initializeApp() {
  renderBoard();
  updateAuthControls();
  renderLeaderboardEmpty("sign in to unlock the leaderboard.");
  await restoreSession();
}

async function joinGame(uuid) {
  if (!uuid) {
    updateStatus("enter a session id.");
    return;
  }

  busy = true;
  renderBoard();
  try {
    const game = await api("/games/" + encodeURIComponent(uuid) + "/join", { method: "POST" });
    syncState(game);
    joinGameInput.value = "";
    await loadGames();
    await loadLeaderboard();
  } catch (error) {
    updateStatus(error.message || "failed to join match.");
  } finally {
    busy = false;
    renderBoard();
  }
}

signupButton.addEventListener("click", () => {
  void signUp().catch((error) => updateStatus(error.message || "registration failed."));
});
signinButton.addEventListener("click", () => {
  void signIn().catch((error) => updateStatus(error.message || "sign in failed."));
});
authForm.addEventListener("submit", (event) => {
  void signIn(event).catch((error) => updateStatus(error.message || "sign in failed."));
});
logoutButton.addEventListener("click", () => {
  void signOut();
});
deleteAccountButton.addEventListener("click", () => {
  void deleteAccount().catch((error) => updateStatus(error.message || "account deletion failed."));
});
deleteConfirmCancelButtons.forEach((button) => {
  button.addEventListener("click", cancelDeleteConfirm);
});
if (deleteConfirmYesButton) {
  deleteConfirmYesButton.addEventListener("click", () => {
    closeDeleteConfirm(true);
  });
}
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && deleteConfirmResolver) {
    cancelDeleteConfirm();
  }
});
newGameButton.addEventListener("click", () => {
  void startNewGame();
});
newPlayerGameButton.addEventListener("click", () => {
  void hostPlayerGame().catch((error) => updateStatus(error.message || "failed to create match."));
});
joinForm.addEventListener("submit", (event) => {
  event.preventDefault();
  void joinGame(joinGameInput.value.trim());
});

updateUser();
updateSession();
updateAuthControls();
void initializeApp();
setInterval(() => {
  if (userUUID && gameID && currentGame && currentGame.mode === "player" && !busy && !finished) {
    void refreshCurrentGame();
  }
}, 3000);
