// WebSocketContext.jsx
import { createContext, useRef, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";

export const WebSocketContext = createContext();

export function WebSocketProvider({ children }) {
  const API_URL = import.meta.env.VITE_API_URL
  const wsRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const navigate = useNavigate();
  const [matchStartPayload, setMatchStartPayload] = useState(null);
  const [matchEndPayload, setMatchEndPayload] = useState(null);
  const [matchRestorePayload, setMatchRestorePayload] = useState(null);
  const matchStartedRef = useRef(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const appliedRoundRef = useRef(null);
  const appliedEndRef = useRef(false);
  const sessionIdRef = useRef(0);
  const backoffTimerRef = useRef(null);
  const pollingRef = useRef(null);
  const endLockRef = useRef(false);


  const getUserId = () => localStorage.getItem("user_name");

  // 新しいセッションを開始し、以降は旧セッション由来の処理を無視する
  const newSession = () => {
    sessionIdRef.current += 1;
    stopPolling();

    if (backoffTimerRef.current) {
      clearTimeout(backoffTimerRef.current);
      backoffTimerRef.current = null;
    }
  }

  // セッション単位でフロント状態を定期取得する polling を開始
  const startPolling = () => {
    if (pollingRef.current) return;

    const sessionId = sessionIdRef.current;
    const pollingIntervalMs = 300;

    pollingRef.current = setInterval(async () => {
      await fetchFrontendState(sessionId);
    }, pollingIntervalMs);
  }

  // 実行中の polling を停止する
  const stopPolling = () => {
    if (!pollingRef.current) return;

    clearInterval(pollingRef.current);
    pollingRef.current = null;
  }

  // 接続時に通知された user_id を永続化する
  const handleWelcome = (data) => {
    localStorage.setItem("user_id", data.user_id);
  };

  // match.start によるラウンド状態遷移を適用する
  const handleMatchStart = (data) => {
    const nextRound = data?.state?.round;
    if (
      appliedRoundRef.current !== null &&
      nextRound <= appliedRoundRef.current
    ) return;

    appliedRoundRef.current = nextRound;

    // context に保存（GamePage がこれを読む）
    setMatchStartPayload(data);

    // round=1 のときだけ battle へ遷移
    if (nextRound === 1 && !matchStartedRef.current) {
      matchStartedRef.current = true; // ← 一度だけ遷移
      navigate("/battle");
    }
  };

  // match.end による試合終了状態を確定させる
  const handleMatchEnd = (data) => {
    if (appliedEndRef.current) return;

    endLockRef.current = true;
    appliedEndRef.current = true;
    setMatchEndPayload(data);

    const endLockReleaseDelayMs = 5000;
    setTimeout(() => {
      endLockRef.current = false;
    }, endLockReleaseDelayMs);
  };

  // match.restore による状態復帰を行い、試合進行フェーズへ遷移する
  const handleMatchRestore = (data) => {
    if (endLockRef.current) return;
    setIsRestoring(true);

    newSession();

    appliedEndRef.current = false;

    // restore は「基準点をジャンプさせる」
    const restoredRound = data.state.round;
    appliedRoundRef.current = restoredRound;

    // 試合再発見 → battle 画面へ遷移
    setMatchRestorePayload(data);
    if (!matchStartedRef.current) {
      matchStartedRef.current = true;
      navigate("/battle");
    }
  };

  // WebSocket push メッセージ用の handler 一覧
  const pushMessageHandlers = {
    "welcome": handleWelcome,
    "match.start": handleMatchStart,
    "match.end": handleMatchEnd,
    "match.restore": handleMatchRestore,
  };

  // HTTP polling で適用可能なメッセージ用の handler 一覧
  const pollingMessageHandlers = {
    "match.start": handleMatchStart,
    "match.end": handleMatchEnd,
  };

  // WebSocket 接続を開始する
  const connect = () => {
    const userId = getUserId();
    if (!userId) {
      console.warn("UserID が無いため WS 接続をスキップ");
      return;
    }

    // 試合終了クールダウン中は新規接続しない
    if (endLockRef.current) return;

    // 既存 WS があれば必ず閉じる
    if (
      wsRef.current &&
      wsRef.current.readyState === WebSocket.OPEN
    ) {
      wsRef.current.close();
      wsRef.current = null;
    }

    const wsUrl = import.meta.env.VITE_WS_URL;  // MUST include /api/v1/ws
    const socket = new WebSocket(wsUrl);
    
    wsRef.current = socket;

    socket.onopen = () => {
      if (!matchStartedRef.current) {
        socket.send(JSON.stringify({ type: "queue.join" }));
      }
      setConnected(true);
    };

    socket.onclose = () => {
      setConnected(false);
    };

    socket.onerror = (e) => {
      console.log("WS ERROR", e);
    };

    socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      pushMessageHandlers[data.type]?.(data);
    };
  };

  // 復帰処理完了を検知し、通常進行フェーズへ遷移させる
  useEffect(() => {
    if (!isRestoring) return;
    if (!matchRestorePayload) return;
    if (!matchStartPayload) return;

    setIsRestoring(false);
  }, [isRestoring, matchStartPayload]);

  // WebSocket 補助として、HTTP polling により状態再同期を行う
  const fetchFrontendState = async (sessionId) => {
    if (sessionIdRef.current !== sessionId) return;

    const response = await fetch(`${API_URL}/api/v1/match/state`, {
      credentials: "include",
    });

    if (response.status === 429) {
      console.warn("🚫 429 received → backoff");

      stopPolling();

      if (backoffTimerRef.current) {
        clearTimeout(backoffTimerRef.current);
      }

      const sessionId = sessionIdRef.current;
      const backoffDelayMs = 600;

      backoffTimerRef.current = setTimeout(() => {
        if (sessionIdRef.current !== sessionId) return;
        startPolling();
      }, backoffDelayMs);

      return;
    }

    if (response.status === 204) return;
    if (!response.ok) return;

    const data = await response.json();
    if (!data) return;

    pollingMessageHandlers[data.type]?.(data);
  };

  // 接続フェーズに応じて polling の有効／無効を制御する
  useEffect(() => {
    if (!connected && !isRestoring) {
      stopPolling();
      return;
    }

    if (!appliedEndRef.current) {
      startPolling();
    }
  }, [connected]);

  // ===========================
  // 🔥 対戦をやめる（queue.left）
  // ===========================
  const leaveQueueFromHome = () => {
    wsRef.current?.send(JSON.stringify({ type: "queue.left" }));
    window.location.reload();
  };

  const leaveQueueFromBattle = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn("WS not connected → queue.left を送信できません");
      return;
    }

    wsRef.current.send(JSON.stringify({ type: "queue.left" }));
  };

  // 現在の対戦セッションを終了し、通信・状態を完全にリセットする
  const disconnect = () => {
    newSession();
    stopPolling();
    setConnected(false);
    matchStartedRef.current = false;
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  };

  // 次回の対戦に備えて、試合関連の状態を初期化する
  const resetMatchState = () => {
    setMatchStartPayload(null);
    setMatchEndPayload(null);
    setMatchRestorePayload(null);

    appliedRoundRef.current = null;
    appliedEndRef.current = false;
    matchStartedRef.current = false;
  };

  return (
    <WebSocketContext.Provider
      value={{
        wsRef,
        connected,
        connect,
        disconnect,
        leaveQueueFromHome,
        leaveQueueFromBattle,
        matchStartPayload,
        matchEndPayload,
        resetMatchState,
        matchRestorePayload,
        isRestoring,
        setIsRestoring,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
