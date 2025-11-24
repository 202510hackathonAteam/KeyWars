// WebSocketContext.jsx
import { createContext, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

export const WebSocketContext = createContext();

export function WebSocketProvider({ children }) {
  const wsRef = useRef(null);
  const [ws, setWs] = useState(null);
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef(null);
  const navigate = useNavigate();
  const [matchStartPayload, setMatchStartPayload] = useState(null);

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

      reconnectTimer.current = setTimeout(() => {
        console.log("WS: reconnecting...");
        connect();
      }, 3000);
    };

    socket.onerror = (e) => {
      console.log("WS ERROR", e);
    };

    socket.onmessage = (event) => {
      console.log("WS Message:", event.data);
      const data = JSON.parse(event.data);
      switch (data.type) {
          case "match.start":
            console.log("🔥 match.start received:", data);

            // context に保存（GamePage がこれを読む）
            setMatchStartPayload(data);
            break;

          case "match.found":
            // 試合発見 → battle 画面へ遷移
            navigate("/battle");
            break;

          case "match.end":
            console.log("試合終了:", data);
            break;

          default:
            console.log("WS message:", data);
        }
      };

    // setWs(socket);

  };

  const disconnect = () => {
    if (wsRef.current) wsRef.current.close();
    wsRef.current = null;
    setConnected(false);
    clearTimeout(reconnectTimer.current);
  };

  return (
    <WebSocketContext.Provider
      // value={{ ws: wsRef.current, connected, connect, disconnect, matchStartPayload, }}
      value={{
        wsRef,            // ← これが必要！
        connected,
        connect,
        disconnect,
        matchStartPayload,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
