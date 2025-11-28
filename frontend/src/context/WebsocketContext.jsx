// WebSocketContext.jsx
import { createContext, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

export const WebSocketContext = createContext();

export function WebSocketProvider({ children }) {
  const wsRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef(null);
  const navigate = useNavigate();
  const [matchStartPayload, setMatchStartPayload] = useState(null);
  const [matchEndPayload, setMatchEndPayload] = useState(null);
  const matchStartedRef = useRef(false);

  const getUserId = () => localStorage.getItem("user_name");

  const connect = () => {
    const userId = getUserId();
    if (!userId) {
      console.warn("UserID が無いため WS 接続をスキップ");
      return;
    }


    if (wsRef.current) {
      console.log("WS: closing old socket...");
      wsRef.current.close();
    }

    const wsUrl = import.meta.env.VITE_WS_URL;  // MUST include /api/v1/ws
    const socket = new WebSocket(wsUrl);
    
    wsRef.current = socket;

    socket.onopen = () => {
      console.log("WS: connected");
      socket.send(JSON.stringify({ type: "queue.join" }));
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
      console.log("WS Message:", event.data);
      const data = JSON.parse(event.data);
      switch (data.type) {
          case "welcome":
            localStorage.setItem("user_id", data.uid);
            break;
          case "match.start":
            console.log("🔥 match.start received:", data);

            // context に保存（GamePage がこれを読む）
            setMatchStartPayload(data);
            break;

          case "match.found":
            // 試合発見 → battle 画面へ遷移
            if (!matchStartedRef.current) {
            matchStartedRef.current = true; // ← 一度だけ遷移
            navigate("/battle");
            }
            break;

          case "match.end":
            console.log("試合終了:", data);
            setMatchEndPayload(data);  
            matchStartedRef.current = false;
            break;

          default:
            console.log("WS message:", data);
        }
      };

    // setWs(socket);

  };


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

  return (
    <WebSocketContext.Provider
      // value={{ ws: wsRef.current, connected, connect, disconnect, matchStartPayload, }}
      value={{
        wsRef,            // ← これが必要！
        connected,
        connect,
        disconnect,
        leaveQueue,
        matchStartPayload,
        matchEndPayload,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
