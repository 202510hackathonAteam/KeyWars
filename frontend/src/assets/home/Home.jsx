// src/assets/components/Home/Home.jsx
import React, { useEffect, useRef, useState } from "react";
// import "./Home.css";


export default function Home() {
  // 現在接続中かどうかを保持するstate(connectによって再描画される。)
  const [connected, setConnected] = useState(false);
  // Websocketオブジェクト保持用のref(参照)＝再度レンダリングしても値はかわらない
  const wsRef = useRef(null);
  // Websocketの接続を担う関数 
  const connectWebSocket = () => {
    // 固定の user_idとroom を送る
    const user_id = "user_1234";
    const room = "match_10";
    const token = `dev:${user_id}:${room}`;

    // WebSocketオブジェクト生成し、接続
    const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);

    // 本番環境でhttpsが使えるなら、以下の方がいい
    // const ws = new WebSocket(`wss://localhost:8080/ws?user_id=${user_id}`);

    ws.onopen = () => {
      // 接続確立すると、以下のメッセージ
      console.log("✅ WebSocket 接続完了");
      // stateを更新
      setConnected(true);
      // ここでバックエンドに初期メッセージを送ってもOK
      ws.send(JSON.stringify({ type: "join", user_id, room}));
    };

    ws.onmessage = (event) => {
      console.log("📩 受信:", event.data);
    };

    ws.onclose = () => {
      console.log("❌ 接続が閉じられました");
      // stateを更新
      setConnected(false);
    };

    ws.onerror = (err) => {
      console.error("⚠️ WebSocket エラー:", err);
    };


    wsRef.current = ws;
  };
  

  // ✅ ページ離脱時（アンマウント時）に自動close
  useEffect(() => {
    // クリーンアップ関数
    return () => {
      if (wsRef.current) {
        console.log("🔌 ページ離脱によりWebSocketを閉じます");
        wsRef.current.close();
      }
    };
  }, []); // ← 空配列でマウント/アンマウント時のみ実行

  // WebSocket切断関数
  // ws.readyStateがOpenとかcloseの値を保持している
  const closeWebSocket = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log("🔌 WebSocket を手動で閉じます");
      wsRef.current.close();
    }
  };


  return (
    <div className="home-body">
      <div className="title">KEY WARS</div>

      <div className="neon-glow"></div>

      <div className="arena">
        <div className="player-info">PLAYER: GUEST</div>

        <div className="rule">
          🥊 ルール：<br />
          相手よりも速くタイピングできると相手を攻撃でき、<br />
          相手のLPが減ります。
        </div>

        <button className="btn"
         onClick={connectWebSocket}
         disabled={connected}
        >
          {connected ? "接続中..." : "対戦相手をさがす"}
        </button>
        <button className="btn"
         onClick={closeWebSocket}
         disabled={!connected}
        >
          切断する
        </button>
      </div>

      <footer>© 2025 Key Wars Tournament</footer>
    </div>
  );
}
