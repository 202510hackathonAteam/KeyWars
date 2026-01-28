// WebSocketContext.jsx
import { createContext, useRef, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";

export const WebSocketContext = createContext();

export function WebSocketProvider({ children }) {
  const API_URL = import.meta.env.VITE_API_URL
  const wsRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef(null);
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

  const newSession = () => {
    sessionIdRef.current += 1;
    stopPolling();

    if (backoffTimerRef.current) {
      clearTimeout(backoffTimerRef.current);
      backoffTimerRef.current = null;
    }
  }

  const startPolling = () => {
    if (pollingRef.current) return;

    const sessionId = sessionIdRef.current;

    pollingRef.current = setInterval(async () => {
      await fetchFrontendState(sessionId);
    }, 300);
  }

  const stopPolling = () => {
    if (!pollingRef.current) return;

    clearInterval(pollingRef.current);
    pollingRef.current = null;
  }

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
      clearTimeout(reconnectTimer.current);
    };

    socket.onclose = () => {
      setConnected(false);

      // reconnectTimer.current = setTimeout(() => {
      //   console.log("WS: reconnecting...");
      //   connect();
      // }, 3000);
    };

    socket.onerror = (e) => {
      console.log("WS ERROR", e);
    };

    socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      switch (data.type) {
          case "welcome":
            localStorage.setItem("user_id", data.user_id);
            break;
          case "match.start":
            const round = data.state.round;
            // すでにこのラウンドを適用済みなら無視
            if (appliedRoundRef.current === round) {
              return;
            }

            appliedRoundRef.current = round;
            // context に保存（GamePage がこれを読む）
            setMatchStartPayload(data);

            // round=1 のときだけ battle へ遷移
            if (
              data.state?.round === 1 &&
              !matchStartedRef.current
            ) {
              matchStartedRef.current = true; // ← 一度だけ遷移
              navigate("/battle");
            }
            break;

          case "match.end":
            if (appliedEndRef.current) return;

            endLockRef.current = true;
            appliedEndRef.current = true;
            setMatchEndPayload(data);
            setTimeout(() => {
              endLockRef.current = false;
            }, 5000);
            break;

          case "match.restore":

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
            break;
        }
      };

    // setWs(socket);

  };

  useEffect(() => {
    if (!isRestoring) return;
    if (!matchRestorePayload) return;
    if (!matchStartPayload) return;

    setIsRestoring(false);
  }, [isRestoring, matchStartPayload]);

  // HTTP ポーリングでフロントエンド再構築用の試合状態を取得し、
  // match.start / match.end を一度だけ適用するための関数
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

      backoffTimerRef.current = setTimeout(() => {
        if (sessionIdRef.current !== sessionId) return;
        startPolling();
      }, 600); // ← バックオフ時間

      return;
    }

    if (response.status === 204) return;
    if (!response.ok) return;

    const data = await response.json();
    if (!data) return;

    switch (data.type) {
      case "match.start":
        const nextRound = data?.state?.round;
        if (
          appliedRoundRef.current !== null &&
          nextRound <= appliedRoundRef.current
        ) {
          return;
        }

        appliedRoundRef.current = nextRound;

        // context に保存（GamePage がこれを読む）
        setMatchStartPayload(data);

        // round=1 のときだけ battle へ遷移
        if (nextRound === 1 && !matchStartedRef.current) {
          matchStartedRef.current = true; // ← 一度だけ遷移
          navigate("/battle");
        }
        break;

      case "match.end":
        if (appliedEndRef.current) return;

        endLockRef.current = true;
        appliedEndRef.current = true;
        setMatchEndPayload(data);
        setTimeout(() => {
          endLockRef.current = false;
        }, 5000);
        break;
    }
  };

  // WebSocket 接続中のみポーリングを有効化し、
  // 試合終了（match.end）を検知したら自動で停止する
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

  const disconnect = () => {
    newSession();
    stopPolling();
    setConnected(false);
    matchStartedRef.current = false;
    clearTimeout(reconnectTimer.current);
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  };

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
      // value={{ ws: wsRef.current, connected, connect, disconnect, matchStartPayload, }}
      value={{
        wsRef,            // ← これが必要！
        connected,
        connect,
        fetchFrontendState,
        disconnect,
        leaveQueueFromHome,
        leaveQueueFromBattle,
        matchStartPayload,
        matchEndPayload,
        resetMatchState,
        matchRestorePayload,
        isRestoring,      // ← これ追加！
        setIsRestoring,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
