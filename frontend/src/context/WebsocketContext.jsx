// WebSocketContext.jsx
import { createContext, useContext, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

export const WebSocketContext = createContext();

export function WebSocketProvider({ children }) {
  const [ws, setWs] = useState(null);
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef(null);
  const userId = localStorage.getItem("user_name");
  const navigate = useNavigate();

  const connect = () => {
    if (!userId) {
      console.warn("UserID が無いため WS 接続をスキップ");
      return;
    }

    const wsUrl = import.meta.env.VITE_WS_URL;
    const socket = new WebSocket(`${wsUrl}?user_id=${userId}`);

    socket.onopen = () => {
      console.log("WS: connected");
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

    socket.onmessage = (event) => {
      console.log("WS Message:", event.data);

      const data = JSON.parse(event.data);

      // 🥊 マッチング成功 → battle へ遷移
      if (data.type === "match-found") {
        navigate("/battle");
      }

      // Battle 内の処理などもここに追加可能
    };

    setWs(socket);
  };

  const disconnect = () => {
    if (ws) {
      ws.close();
    }
    setConnected(false);
    clearTimeout(reconnectTimer.current);
  };

  return (
    <WebSocketContext.Provider
      value={{
        ws,
        connected,
        connect,
        disconnect,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
