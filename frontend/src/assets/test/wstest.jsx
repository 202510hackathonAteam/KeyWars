// src/assets/components/Home/Home.jsx
import React, { useEffect, useRef, useState } from "react";

export default function Home() {
  const [connected, setConnected] = useState(false);
  const [userId, setUserId] = useState("");
  const [room, setRoom] = useState("match_10");
  const [message, setMessage] = useState(""); // ← 送信メッセージ用
  const [logs, setLogs] = useState([]); // ← 受信ログ表示用
  const wsRef = useRef(null);

  // WebSocket接続
  const connectWebSocket = () => {
    if (!userId) {
      alert("ユーザーIDを入力してください");
      return;
    }

    const token = `dev:${userId}:${room}`;
    const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);

    ws.onopen = () => {
      console.log("✅ WebSocket 接続完了");
      setConnected(true);
      ws.send(JSON.stringify({ type: "join", user_id: userId, room }));
      setLogs((prev) => [...prev, "✅ 接続しました"]);
    };

    ws.onmessage = (event) => {
      console.log("📩 受信:", event.data);
      setLogs((prev) => [...prev, "📩 " + event.data]);
    };

    ws.onclose = () => {
      console.log("❌ 接続が閉じられました");
      setConnected(false);
      setLogs((prev) => [...prev, "❌ 接続が閉じられました"]);
    };

    ws.onerror = (err) => {
      console.error("⚠️ WebSocket エラー:", err);
      setLogs((prev) => [...prev, "⚠️ エラー: " + err.message]);
    };

    wsRef.current = ws;
  };

  // メッセージ送信
  const sendMessage = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const payload = {
        type: "message",
        user_id: userId,
        room,
        body: message,
      };
      wsRef.current.send(JSON.stringify(payload));
      setLogs((prev) => [...prev, "📤 送信: " + message]);
      setMessage(""); // 入力欄リセット
    } else {
      alert("WebSocketが接続されていません。");
    }
  };

  // 手動切断
  const closeWebSocket = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log("🔌 WebSocket を手動で閉じます");
      wsRef.current.close();
    }
  };

  // ページ離脱時に接続を閉じる
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        console.log("🔌 ページ離脱によりWebSocketを閉じます");
        wsRef.current.close();
      }
    };
  }, []);

  return (
    <div className="home-body">
      <div className="title">KEY WARS (WebSocketテスト)</div>

      <div className="arena">
        <div className="player-info">PLAYER: {userId || "GUEST"}</div>

        {/* 接続設定 */}
        <div className="input-group">
          <label>
            ユーザーID：
            <input
              type="text"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="例: user_1234"
              disabled={connected}
            />
          </label>
        </div>

        <div className="input-group">
          <label>
            ルーム名：
            <input
              type="text"
              value={room}
              onChange={(e) => setRoom(e.target.value)}
              placeholder="例: match_10"
              disabled={connected}
            />
          </label>
        </div>

        <button className="btn" onClick={connectWebSocket} disabled={connected}>
          {connected ? "接続中..." : "接続する"}
        </button>

        <button className="btn" onClick={closeWebSocket} disabled={!connected}>
          切断する
        </button>

        {/* ✅ メッセージ送信欄 */}
        <div className="input-group" style={{ marginTop: "20px" }}>
          <label>
            メッセージ送信：
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="送信したいメッセージ"
              disabled={!connected}
            />
          </label>
          <button className="btn" onClick={sendMessage} disabled={!connected || !message}>
            送信
          </button>
        </div>

        {/* ✅ ログ表示 */}
        <div className="log-box" style={{ marginTop: "20px", background: "#222", color: "#0f0", padding: "10px", borderRadius: "8px", height: "200px", overflowY: "auto" }}>
          <div>📜 通信ログ:</div>
          {logs.map((log, idx) => (
            <div key={idx}>{log}</div>
          ))}
        </div>
      </div>

      <footer>© 2025 Key Wars Tournament</footer>
    </div>
  );
}
