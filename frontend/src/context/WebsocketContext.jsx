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
      socket.send(JSON.stringify({ type: "hello" }));
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

      if (data.type === "match-found") {
        navigate("/battle");
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
      value={{ ws: wsRef.current, connected, connect, disconnect }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
