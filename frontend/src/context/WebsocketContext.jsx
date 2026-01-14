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
  const pollingRef = useRef(null);
  const backoffRef = useRef(false);


  const getUserId = () => localStorage.getItem("user_name");

  const startPolling = () => {
    if (pollingRef.current) {
      console.log("🚫 polling already running");
      return
    };

    console.log("▶️ polling started");
    pollingRef.current = setInterval(async () => {
      await fetchFrontendState();
    }, 300);
  }

  const stopPolling = () => {
    if (!pollingRef.current) return;

    clearInterval(pollingRef.current);
    pollingRef.current = null;
    console.log("⏹ polling stopped");
  }

  const connect = () => {
    const userId = getUserId();
    if (!userId) {
      console.warn("UserID が無いため WS 接続をスキップ");
      return;
    }

    appliedRoundRef.current = null;
    appliedEndRef.current = false;
    matchStartedRef.current = false;

    if (wsRef.current) {
      console.log("WS: closing old socket...");
      wsRef.current.close();
    }

    const wsUrl = import.meta.env.VITE_WS_URL;  // MUST include /api/v1/ws
    const socket = new WebSocket(wsUrl);
    
    wsRef.current = socket;

    socket.onopen = () => {
      console.log("WS: connected");

      if (!matchStartedRef.current) {
        socket.send(JSON.stringify({ type: "queue.join" }));
      }
      setConnected(true);
      clearTimeout(reconnectTimer.current);
    };

    socket.onclose = () => {
      console.log("WS: disconnected");
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
            console.log("[push] 🔥 match.start received:", data);
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
            console.log("[push] 試合終了:", data);
            if (appliedEndRef.current) return;

            appliedEndRef.current = true;
            setMatchEndPayload(data);
            break;

          case "match.restore":
            console.log("再接続:", data);
            const restoredRound = data.state.round;

            // restore は「基準点をジャンプさせる」
            appliedRoundRef.current = restoredRound;
            appliedEndRef.current = false;

            // 試合再発見 → battle 画面へ遷移
            setMatchRestorePayload(data);
            setIsRestoring(true); 
            if (!matchStartedRef.current) {
              matchStartedRef.current = true;
              navigate("/battle");
              }
            break;
        }
      };

    // setWs(socket);

  };

  // HTTP ポーリングでフロントエンド再構築用の試合状態を取得し、
  // match.start / match.end を一度だけ適用するための関数
  const fetchFrontendState = async () => {
    const response = await fetch(`${API_URL}/api/v1/match/state`, {
      credentials: "include",
    });

    if (response.status === 429) {
      console.warn("🚫 429 received → backoff");

      stopPolling();
      backoffRef.current = true;

      setTimeout(() => {
        backoffRef.current = false;
        startPolling();
        console.log("🔁 polling resumed after backoff");
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

        console.log("[polling] 🔥 match.start received:", data);

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
        console.log("[polling] 試合終了:", data);

        appliedEndRef.current = true;
        setMatchEndPayload(data);
        break;
    }
  };

  // WebSocket 接続中のみポーリングを有効化し、
  // 試合終了（match.end）を検知したら自動で停止する
  useEffect(() => {
    if (!connected || appliedEndRef.current) {
      stopPolling();
      console.log("⏸ polling skipped (WS not connected)");
      return;
    }

    startPolling();
    console.log("▶️ polling started (WS connected)");

    return () => {
      stopPolling();
      console.log("⏹ polling stopped");
    };
  }, [connected]);

  // ===========================
  // 🔥 対戦をやめる（queue.left）
  // ===========================
  const leaveQueue = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn("WS not connected → queue.left を送信できません");
      return;
    }

    wsRef.current.send(JSON.stringify({ type: "queue.left" }));
    console.log("📤 Sent: queue.left");

    // 自動再接続も止めたい場合、disconnect する
    disconnect();

    // match 開始フラグをリセット
    matchStartedRef.current = false;
  };

  const disconnect = () => {
    if (wsRef.current) wsRef.current.close();
    wsRef.current = null;
    setConnected(false);
    clearTimeout(reconnectTimer.current);
    matchStartedRef.current = false;
  };

  const resetMatchState = () => {
    console.log("🔁 resetMatchState called");
    setMatchStartPayload(null);
    setMatchEndPayload(null);

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
        leaveQueue,
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
