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
const loginInput = document.getElementById("login");
const passwordInput = document.getElementById("password");
const youSymbolNode = document.getElementById("you-symbol");
const computerSymbolNode = document.getElementById("computer-symbol");
const sessionNode = document.getElementById("session-id");

const storageKeys = {
  accessToken: "tic-tac-toe.accessToken",
  refreshToken: "tic-tac-toe.refreshToken",
  gameID: "tic-tac-toe.gameID"
};

let gameID = "";
let authHeader = "";
let accessToken = "";
let refreshToken = "";
let userUUID = "";
let board = createEmptyBoard();
let currentGame = null;
let busy = false;
let finished = false;

function createEmptyBoard() {
  return Array.from({ length: BOARD_SIZE }, () => Array(BOARD_SIZE).fill(EMPTY));
}

function cloneBoard(source) {
  return source.map((row) => row.slice());
}

function symbolForCell(value) {
  if (value === USER) {
    return "X";
  }

  if (value === COMPUTER) {
    return "O";
  }

  return "";
}

function updateStatus(message) {
  statusNode.textContent = message;
}

function updateUser() {
  const user = playerLabel();
  const computer = user === "X" ? "O" : "X";
  youSymbolNode.textContent = user;
  computerSymbolNode.textContent = computer;
}

function updateSession() {
  sessionNode.textContent = gameID || "Нет активной игры";
}

function updateAuthControls() {
  const loggedIn = Boolean(authHeader);
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

function canRetryAuth(path, options) {
  if (options.skipAuth === true || options.retryAuth === false || !refreshToken) {
    return false;
  }

  if (path === "/auth" || path === "/signup" || path.startsWith("/auth/")) {
    return false;
  }

  return true;
}

function persistSession(tokens) {
  accessToken = tokens.accessToken || "";
  refreshToken = tokens.refreshToken || "";
  authHeader = accessToken ? `${tokens.type || "Bearer"} ${accessToken}` : "";

  if (accessToken) {
    localStorage.setItem(storageKeys.accessToken, accessToken);
  } else {
    localStorage.removeItem(storageKeys.accessToken);
  }

  if (refreshToken) {
    localStorage.setItem(storageKeys.refreshToken, refreshToken);
  } else {
    localStorage.removeItem(storageKeys.refreshToken);
  }

  updateAuthControls();
}

function clearStoredSession() {
  localStorage.removeItem(storageKeys.accessToken);
  localStorage.removeItem(storageKeys.refreshToken);
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
  accessToken = "";
  refreshToken = "";
  authHeader = "";
  userUUID = "";
  clearStoredSession();
  clearGameState();
  gamesNode.innerHTML = "";
  renderLeaderboardEmpty("Войди в систему, чтобы увидеть таблицу лидеров.");
  updateUser();
  updateAuthControls();
  updateStatus(message || "Войди, чтобы продолжить.");
}

async function readResponsePayload(response) {
  return response.json().catch(() => ({}));
}

function playerSymbol() {
  if (currentGame && currentGame.player_o_uuid === userUUID) {
    return COMPUTER;
  }

  return USER;
}

function playerLabel() {
  return playerSymbol() === COMPUTER ? "O" : "X";
}

async function api(path, options = {}) {
  const headers = Object.assign({}, options.headers || {});
  if (options.skipAuth !== true && authHeader) {
    headers.Authorization = authHeader;
  }
  if (options.body && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(path, Object.assign({}, options, { headers }));
  const payload = await readResponsePayload(response);
  if (response.status === 401 && canRetryAuth(path, options)) {
    try {
      await refreshAccessToken();
      return api(path, Object.assign({}, options, { retryAuth: false }));
    } catch (error) {
      clearAuthState(error.message || "Сессия истекла. Войди снова.");
      throw new Error(error.message || "Сессия истекла. Войди снова.");
    }
  }
  if (!response.ok) {
    throw new Error(payload.message || "request failed");
  }
  return payload;
}

async function refreshAccessToken() {
  if (!refreshToken) {
    throw new Error("Сессия истекла. Войди снова.");
  }

  const tokens = await api("/auth/refresh/access", {
    method: "POST",
    body: JSON.stringify({ refreshToken }),
    skipAuth: true,
    retryAuth: false
  });

  persistSession(tokens);
  return tokens;
}

async function loadLeaderboard() {
  if (!authHeader) {
    renderLeaderboardEmpty("Войди в систему, чтобы увидеть таблицу лидеров.");
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
  await api("/signup", {
    method: "POST",
    body: JSON.stringify(data)
  });
  updateStatus("Пользователь зарегистрирован. Теперь войди.");
}

async function signIn(event = null) {
  if (event) {
    event.preventDefault();
  }

  const data = credentials();
  const tokens = await api("/auth", {
    method: "POST",
    body: JSON.stringify(data)
  });

  persistSession(tokens);
  const user = await api("/users/me");
  userUUID = user.uuid;
  updateUser();

  const storedGameID = localStorage.getItem(storageKeys.gameID);
  if (storedGameID) {
    await restoreGame(storedGameID);
    await loadLeaderboard();
    return;
  } else {
    await startNewGame();
  }
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
    updateStatus(game.winner_uuid === userUUID ? "Ты победила." : "Победил соперник.");
    return;
  }

  if (game.state === "draw") {
    finished = true;
    updateStatus("Ничья.");
    return;
  }

  if (game.state === "waiting_players") {
    finished = false;
    updateStatus("Ожидание второго игрока.");
    return;
  }

  finished = false;
  updateStatus(game.next_player_uuid === userUUID ? "Твой ход." : "Ход соперника.");
}

function renderBoard() {
  boardNode.innerHTML = "";

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

      const isUserTurn = currentGame && currentGame.next_player_uuid === userUUID;
      button.disabled = busy || finished || !authHeader || !isUserTurn || value !== EMPTY;
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
    !authHeader ||
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
  updateStatus(currentGame.mode === "computer" ? "Компьютер думает..." : "Отправляю ход...");
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
    updateStatus(error.message || "Не удалось выполнить ход.");
  } finally {
    busy = false;
    renderBoard();
  }
}

async function startNewGame() {
  await createGame("computer");
}

async function hostPlayerGame() {
  await createGame("player");
  await loadGames();
}

async function createGame(mode) {
  if (!authHeader) {
    board = createEmptyBoard();
    currentGame = null;
    finished = false;
    updateStatus("Зарегистрируйся или войди, чтобы начать игру.");
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
    updateStatus(error.message || "Не удалось создать игру.");
  } finally {
    busy = false;
    renderBoard();
  }
}

async function loadGames() {
  if (!authHeader) {
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

async function restoreGame(uuid) {
  if (!uuid) {
    return;
  }

  try {
    const game = await api("/games/" + encodeURIComponent(uuid), { method: "GET" });
    syncState(game);
  } catch (error) {
    localStorage.removeItem(storageKeys.gameID);
    updateStatus(error.message || "Не удалось восстановить игру.");
  }
}

async function refreshCurrentGame() {
  if (!authHeader || !gameID) {
    return;
  }

  try {
    const game = await api("/games/" + encodeURIComponent(gameID), { method: "GET" });
    syncState(game);
    if (game.state === "player_wins" || game.state === "draw") {
      await loadLeaderboard();
    }
  } catch (error) {
    updateStatus(error.message || "Не удалось обновить игру.");
  } finally {
    renderBoard();
  }
}

async function restoreSession() {
  const storedRefreshToken = localStorage.getItem(storageKeys.refreshToken);
  if (!storedRefreshToken) {
    clearAuthState("Войди, чтобы продолжить.");
    return false;
  }

  refreshToken = storedRefreshToken;

  try {
    const tokens = await api("/auth/refresh/access", {
      method: "POST",
      body: JSON.stringify({ refreshToken: storedRefreshToken }),
      skipAuth: true,
      retryAuth: false
    });
    persistSession(tokens);

    const user = await api("/users/me");
    userUUID = user.uuid;
    updateUser();

    const storedGameID = localStorage.getItem(storageKeys.gameID);
    if (storedGameID) {
      await restoreGame(storedGameID);
    } else {
      clearGameState();
      updateStatus("Сессия восстановлена.");
    }

    await loadGames();
    await loadLeaderboard();
    return true;
  } catch (error) {
    clearAuthState("Сессия истекла. Войди снова.");
    return false;
  }
}

async function signOut() {
  if (!refreshToken) {
    clearAuthState("Вышли из системы.");
    return;
  }

  try {
    await api("/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refreshToken }),
      skipAuth: true,
      retryAuth: false
    });
  } catch (error) {
    // Clear local state even if the server-side revoke failed or the token is already invalid.
  } finally {
    clearAuthState("Вышли из системы.");
  }
}

async function deleteAccount() {
  if (!authHeader) {
    clearAuthState("Войди, чтобы удалить аккаунт.");
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
      clearAuthState("Аккаунт удалён.");
    }
  }
}

async function initializeApp() {
  renderBoard();
  updateAuthControls();
  renderLeaderboardEmpty("Войди в систему, чтобы увидеть таблицу лидеров.");
  await restoreSession();
}

async function joinGame(uuid) {
  if (!uuid) {
    updateStatus("Укажи ID игровой сессии.");
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
    updateStatus(error.message || "Не удалось присоединиться.");
  } finally {
    busy = false;
    renderBoard();
  }
}

signupButton.addEventListener("click", () => {
  void signUp().catch((error) => updateStatus(error.message || "Не удалось зарегистрироваться."));
});
signinButton.addEventListener("click", () => {
  void signIn().catch((error) => updateStatus(error.message || "Не удалось войти."));
});
authForm.addEventListener("submit", (event) => {
  void signIn(event).catch((error) => updateStatus(error.message || "Не удалось войти."));
});
logoutButton.addEventListener("click", () => {
  void signOut();
});
deleteAccountButton.addEventListener("click", () => {
  void deleteAccount().catch((error) => updateStatus(error.message || "Не удалось удалить аккаунт."));
});
newGameButton.addEventListener("click", () => {
  void startNewGame();
});
newPlayerGameButton.addEventListener("click", () => {
  void hostPlayerGame().catch((error) => updateStatus(error.message || "Не удалось создать игру."));
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
  if (authHeader && gameID && currentGame && currentGame.mode === "player" && !busy && !finished) {
    void refreshCurrentGame();
  }
}, 3000);
