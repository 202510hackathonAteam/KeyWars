// src/pages/home/Home.jsx
import React, { useEffect, useRef, useState, useContext, Component } from "react";
import { WebSocketContext } from "../../context/WebsocketContext";
import LogoutButton from "../../components/LogoutButton";
// import "./Home.css";

export default function Home() {
  // 現在接続中かどうかを保持するstate(connectによって再描画される。)
  const { connect, connected, leaveQueue } = useContext(WebSocketContext);

  const storedUserName = localStorage.getItem("user_name") || "GUEST";
  // Websocketの接続を担う関数 
  // const connectWebSocket = () => {
  //   // 固定の user_id を送る
  //   const user_id = "user_1234";
  //   const wsUrl = import.meta.env.VITE_WS_URL;

  //   // WebSocketオブジェクト生成し、接続
  //   const ws = new WebSocket(`${wsUrl}?user_id=${user_id}`);
  //   // 本番環境でhttpsが使えるなら、以下の方がいい
  //   // const ws = new WebSocket(`wss://localhost:8080/api/v1/ws`);

  //   ws.onopen = () => {
  //     // 接続確立すると、以下のメッセージ
  //     console.log("✅ WebSocket 接続完了");
  //     // stateを更新
  //     setConnected(true);
  //     // ここでバックエンドに初期メッセージを送ってもOK
  //     ws.send(JSON.stringify({ type: "join", user_id }));
  //   };

  //   ws.onmessage = (event) => {
  //     console.log("📩 受信:", event.data);
  //   };

  //   ws.onclose = () => {
  //     console.log("❌ 接続が閉じられました");
  //     // stateを更新
  //     setConnected(false);
  //   };

  //   ws.onerror = (err) => {
  //     console.error("⚠️ WebSocket エラー:", err);
  //   };


  //   wsRef.current = ws;
  // };
  

  // // ✅ ページ離脱時（アンマウント時）に自動close
  // useEffect(() => {
  //   // クリーンアップ関数
  //   return () => {
  //     if (wsRef.current) {
  //       console.log("🔌 ページ離脱によりWebSocketを閉じます");
  //       wsRef.current.close();
  //     }
  //   };
  // }, []); // ← 空配列でマウント/アンマウント時のみ実行

  // // WebSocket切断関数
  // // ws.readyStateがOpenとかcloseの値を保持している
  // const closeWebSocket = () => {
  //   if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
  //     console.log("🔌 WebSocket を手動で閉じます");
  //     wsRef.current.close();
  //   }
  // };


  return (
    <div className="flex items-center justify-center min-h-screen bg-gradient-radial from-neutral-900 to-black text-white font-['Press_Start_2P'] relative overflow-hidden">
      <div className="absolute top-4 left-10 z-20">
        <LogoutButton />
      </div>
      {/* 背景グラデーション */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0 pointer-events-none" ></div>
              <div className="home-body z-10 text-center">
        {/* タイトル */}
        <div className="text-[55px] text-[#ff4444] mb-[60px] drop-shadow-[0_0_10px_#ff0000] animate-[flicker_2s_infinite]">
          KEY WARS
        </div>

        {/* アリーナ枠 */}
        <div className="
          relative w-[45vw] h-[55vh]
          border-4 border-[#ffcc00] rounded-[10px]
          shadow-[0_0_25px_#ffaa00,inset_0_0_20px_#ff3300]
          bg-[linear-gradient(180deg,rgba(50,10,0,0.7),rgba(0,0,0,0.9))]
          flex flex-col items-center
          p-6 mx-auto
          text-center
          space-y-4
        ">
          {/* 枠のタイトル */}
          <div className="absolute -top-7 left-1/2 -translate-x-1/2 bg-black px-3 py-1 text-[#ffcc00] text-[10px] border border-[#ffcc00] rounded-md shadow-[0_0_10px_#ff8800]">
            FIGHTER'S LOUNGE
          </div>

          {/* プレイヤー情報 */}
          <div className="text-[4vh] text-[#00ff99] mb-2 mt-[5vh] drop-shadow-[0_0_5px_#00ff99]">
            PLAYER:  {(storedUserName || "GUEST").toUpperCase()}
          </div>

          {/* ルール説明 */}
          <div className="text-[2vh] text-[#ffcc00] bg-[rgba(0,0,0,0.6)] border border-[#ffaa00] rounded-lg p-3 my-[22px] mb-[6vh] w-4/5 mx-auto leading-relaxed shadow-[0_0_10px_#ff8800]">
            🥊 ルール：<br />
            相手よりも速くタイピングできると相手を攻撃でき、<br />
            相手のLPが減ります。
          </div>

          {/* ボタン群 */}
          <div className="flex flex-col items-center gap-5">
            <button
              className="px-6 py-3 bg-[#ff0000] border-2 border-[#ffcc00] rounded-lg text-white text-[3vh] uppercase shadow-[0_0_15px_#ff0000] transition-transform hover:bg-[#ff8800] hover:shadow-[0_0_25px_#ffaa00] hover:scale-110"
              onClick={connect}
              disabled={connected}
            >
              {connected ? "接続中..." : "対戦相手をさがす"}
            </button>
            {connected && (
              <button
                onClick={leaveQueue}
                className="
                  px-6 py-3
                  rounded-lg
                  text-white
                  text-[2vh]           /* ← 🔥 ボタン文字を大きく */
                  uppercase
                  shadow-[0_0_10px_#ff4444]
                  transition-transform
                  hover:bg-[#550000]
                  hover:shadow-[0_0_20px_#ff0000]
                  hover:scale-110
                "
              >
                対戦をやめる
              </button>
            )}

            {/* <button
              className="px-6 py-3 bg-[#ff0000] border-2 border-[#ffcc00] rounded-lg text-white text-[12px] uppercase shadow-[0_0_15px_#ff0000] transition-transform hover:bg-[#ff8800] hover:shadow-[0_0_25px_#ffaa00] hover:scale-110 disabled:opacity-50 disabled:cursor-not-allowed"
              onClick={closeWebSocket}
              disabled={!connected}
            >
              切断する
            </button> */}
          </div>
        </div>

        <footer className="absolute bottom-1 left-0 w-full text-center text-[clamp(0.45rem,0.8vw,0.65rem)] text-gray-500">© 2025 Key Wars Tournament</footer>
      </div>
    </div>
  );
}
