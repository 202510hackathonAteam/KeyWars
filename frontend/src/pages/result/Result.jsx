import React, { useContext, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import "./Result.css";
import { WebSocketContext } from "../../context/WebsocketContext";
import LogoutButton from "../../components/LogoutButton";

const Result = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // ★★★ WebSocket を切断/接続/リセットする関数を読み込む
  const { disconnect, connect, resetMatchState } = useContext(WebSocketContext);

  // ★★★ ページに入った瞬間に WebSocket を切断
  useEffect(() => {
    disconnect();
    resetMatchState();
  }, [disconnect, resetMatchState]);

  // Battleから受け取る
  const result = location.state?.result || "defeat"; // デフォルト値
  const isVictory = result === "victory";
  const isDefeat  = result === "defeat";
  const isDraw    = result === "draw";
  const totalRounds = location.state?.round || 20;
  console.log("🟢 Result received round:", totalRounds);
  const totalMiss = location.state?.totalMiss;
  console.log("🎯 自分の total miss:", totalMiss);


  useEffect(() => {
    document.body.classList.add(result);
    return () => {
      document.body.classList.remove(result);
    };
  }, [result]);

  // 紙吹雪を生成
  useEffect(() => {
    if (!isVictory) return;

    const confettiContainer = document.createElement("div");
    confettiContainer.classList.add("confetti-container");
    document.body.appendChild(confettiContainer);

 const confettiCount = 50; // ← 数も増やす

  for (let i = 0; i < confettiCount; i++) {
    const confetti = document.createElement("div");
    confetti.classList.add("confetti");

    // ランダム位置・色・サイズなど
    confetti.style.left = Math.random() * 100 + "vw"; // 横方向ランダム
    confetti.style.top = -50 - Math.random() * 200 + "px"; // ← 高さをランダムに！（これが重要）
    confetti.style.animationDelay = Math.random() * 6 + "s"; // ← 開始タイミングをもっとバラけさせる
    confetti.style.backgroundColor = getRandomColor();

    // ランダム速度と横揺れ周期
    confetti.style.setProperty("--fall-duration", 4 + Math.random() * 4 + "s");
    confetti.style.setProperty("--sway-duration", 1.5 + Math.random() * 2 + "s");

    // サイズを少しバラつかせる
    const size = 8 + Math.random() * 10;
    confetti.style.width = `${size}px`;
    confetti.style.height = `${size * 2.5}px`;

    // 初期回転
    confetti.style.transform = `rotate(${Math.random() * 360}deg)`;

    confettiContainer.appendChild(confetti);
  }

    // コンポーネント削除時に掃除
    return () => {
      confettiContainer.remove();
    };
  }, [isVictory]);

  const getRandomColor = () => {
    const colors = ["#7CFF6B", "#00FFB2", "#FFDD00", "#FF6B6B", "#00BFFF"];
    return colors[Math.floor(Math.random() * colors.length)];
  };


  const retry = () => {
    // 1. 新しい WebSocket を開いて queue.join を送る
    connect();

    // 2. ホームに戻る（そこで「接続中...」表示 → match.found で /battle に飛ぶ）
    navigate("/");
  };
  
  const goHome = () => navigate("/");

  return (
    <div className="min-h-screen flex flex-col items-center justify-center text-center font-['Press_Start_2P'] text-white overflow-hidden">
      <div className="absolute top-4 left-10 z-20">
        <LogoutButton />
      </div>

      {/* タイトル */}
      <h1 className="title">
        KEY WARS
      <span
        className={`block text-2xl mt-5 ${
          isVictory
            ? "text-[#7CFF6B] drop-shadow-[0_0_15px_#00ff88,0_0_40px_#00cc66] animate-blink"
            : isDraw
            ? "text-[#FFD966] drop-shadow-[0_0_15px_#ffcc00]"
            : "text-[#88FFB2] drop-shadow-[0_0_15px_#00ff99,0_0_40px_#00cc88] animate-blink"
        }`}
      >
        {isVictory && "YOU WIN!"}
        {isDraw && "DRAW"}
        {isDefeat && "YOU LOSE..."}
      </span>

      </h1>

      {/* 結果パネル */}
      <div className="
        relative w-[45vw] h-[25vh]
        border-4 border-[#ffcc00] rounded-[10px]
        shadow-[0_0_25px_#ffaa00,inset_0_0_20px_#ff3300]
        bg-[linear-gradient(180deg,rgba(50,10,0,0.7),rgba(0,0,0,0.9))]
        flex flex-col items-center
        p-6 mx-auto
        text-center
        space-y-4
      ">
      <div className="text-[1.5rem] text-[#ffcfcf] my-2 drop-shadow-[0_0_5px_#ff5900]">
          間違えた回数（通算{totalRounds}戦中）：<span className="text-white ml-2">{totalMiss}</span>
      </div>

        {/* ボタン群 */}
        <div className="mt-10 flex justify-center gap-6">
          <button
            onClick={retry}
            className="bg-[#ff3300] px-6 py-3 rounded-xl text-xs shadow-[0_0_15px_#ff6600] transition duration-300 hover:bg-[#ff6600] hover:shadow-[0_0_25px_#f15c51]"
          >
            再戦
          </button>
          <button
              onClick={() => {
                // （任意）履歴や WS の best effort クリーンアップ
                goHome?.();

                // Redis / WS / polling の残骸を完全に捨てる
                setTimeout(() => {
                  window.location.reload();
                }, 300);
              }}
            className="border-2 border-[#ff6600] px-6 py-3 rounded-xl text-xs shadow-[0_0_15px_#ff6600] transition duration-300 hover:bg-[#ff6600]/20"
          >
            ホームに戻る
          </button>
        </div>
      </div>
    </div>
  );
}

export default Result;
