import React, { useEffect, useRef, useState, useContext } from "react";
// import "./Battle.css";
import { useNavigate } from "react-router-dom";
import { checkTyping } from "../../utils/typingLogic"; 
import { WebSocketContext } from "../../context/WebsocketContext";

function renderHighlightedRomaji(target, progress) {
  return (
    <div className="flex gap-1 text-xl">
      {target.split("").map((char, index) => {
        const isTyped = index < progress;

        return (
          <span
            key={index}
            className={
              isTyped
                ? "text-[#3ce27a] font-bold drop-shadow-[0_0_5px_#3ce27a]"
                : "text-white/40"
            }
          >
            {char}
          </span>
        );
      })}
    </div>
  );
}

export default function GamePage() {
  const navigate = useNavigate();
  const inputRef = useRef(null); // ← 入力欄参照を作成

  // typing用 state
  const [input, setInput] = useState("");
  const [progress, setProgress] = useState(0);
  const [missCount, setMissCount] = useState(0);
  const [targetRomaji, setTargetRomaji] = useState("");
  const [promptText, setPromptText] = useState(""); 
  const [limitMs, setLimitMs] = useState(60000);
  const [timeLeft, setTimeLeft] = useState(60);

  const [playerHp, setPlayerHp] = useState(100);
  const [enemyHp, setEnemyHp] = useState(100);


  const { wsRef, matchStartPayload } = useContext(WebSocketContext);

  useEffect(() => {
  if (!matchStartPayload) return;

  console.log("🎯 match.start in GamePage:", matchStartPayload);

  setTargetRomaji(matchStartPayload.prompt.target_romaji);
  setPromptText(matchStartPayload.prompt.prompt_text_ja);
  setLimitMs(matchStartPayload.prompt.limit_ms);

  setTimeLeft(Math.floor(matchStartPayload.prompt.limit_ms / 1000));
  inputRef.current?.focus();
}, [matchStartPayload]);


  const sendAnswerFinish = () => {
  if (!wsRef.current) {
    console.warn("WS not connected");
    return;
  }
  if (!matchStartPayload) {
    console.warn("match.start payload missing");
    return;
  }

  const msg = {
    type: "answer.finish",
    match_id: matchStartPayload.match_id,
    miss_count: missCount,
  };

  console.log("🔥 SEND answer.finish:", msg);
  wsRef.current.send(JSON.stringify(msg));
};

const sendAnswerTimeout = () => {
  if (!wsRef.current) {
    console.warn("WS not connected");
    return;
  }
  if (!matchStartPayload) {
    console.warn("match.start payload missing");
    return;
  }

  const msg = {
    type: "answer.timeout",
    match_id: matchStartPayload.match_id,
    miss_count: missCount,
  };

  console.log("🔥 SEND answer.timeout:", msg);
  wsRef.current.send(JSON.stringify(msg));
};


    // タイピング判定ロジック
  const handleTyping = (e) => {
    const val = e.target.value;
    setInput(val);

    const { newProgress, isCorrect, isFinish, miss } =
      checkTyping(val, targetRomaji, progress);

    if (!isCorrect) {
      setMissCount((prev) => prev + miss);
      return;
    }

    setProgress(newProgress);

    if (isFinish) {
      sendAnswerFinish(); // WebSocket送信（後で作る）
    }
  };

  const finishBattle = (didWin) => {
    const result = didWin ? "victory" : "defeat";
    navigate("/result", { state: { result } });
  };

    // ページ表示時にフォーカスを当てる
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
  const timer = setInterval(() => {
    setTimeLeft((prev) => {
      if (prev <= 1) {
        clearInterval(timer);
        sendAnswerTimeout();  // ← 追加！
        finishBattle(false); // 0になったら終了処理へ
        return 0;
      }
      return prev - 1;
    });
  }, 1000); // 1秒ごと

  return () => clearInterval(timer); // コンポーネントが消えたらタイマー停止
}, []);


  return (
    <div className="
    mt-[5vh]
    max-w-[90vw] mx-auto my-6 
    p-5 rounded-xl 
    bg-gradient-to-b from-white/5 to-black/5
    h-[85vh]
    from-white/5 
    to-black/5 
    shadow-[0_8px_30px_rgba(252,2,2,0.6)]"
    gap-y-6   /* ← 内部要素間に余白を取る */
    >
      {/* 戦闘エリア */}
      <div className="relative flex flex-col justify-between rounded-lg bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.02),transparent_40%)] p-5 overflow-hidden">
        {/* プレイヤー行 */}
        <div className="flex justify-between items-center">
          {/* プレイヤー1 */}
          <div className="flex items-center gap-2 w-[48%]">
            <div className="relative w-[17vh] h-[17vh] rounded-md bg-gradient-to-br from-neutral-700 to-neutral-900 border-2 border-white/5 flex items-center justify-center">
              <img
                className="absolute w-[17vh]"
                src="../../../public/img/ai_right.jpeg"
                alt="P1"
              />
            </div>
            <div>
              <div className="text-[clamp(1.2rem,2vw,2rem)]">CALLSIGN: KYO</div>
              <div className="bg-white/5 p-2 rounded-md mt-2">
                <div className="h-[16px] bg-neutral-800 rounded-md overflow-hidden">
                  <div
                    className="h-full w-full bg-gradient-to-r from-[#3ce27a] to-[#afffb0] transition-all duration-300"
                    style={{ width: `${playerHp}%` }}
                  ></div>
                </div>
                <div className="text-[10px] mt-1 opacity-90">
                  <span>100</span> / 100
                </div>
              </div>
            </div>
          </div>

          {/* プレイヤー2 */}
          <div className="flex items-center gap-2 w-[48%] justify-end text-right">
            <div>
              <div className="text-[clamp(1.2rem,2vw,2rem)]">CALLSIGN: RYU</div>
              <div className="bg-white/5 p-2 rounded-md mt-2">
                <div className="h-[16px] bg-neutral-800 rounded-md overflow-hidden">
                  <div
                    className="h-full w-full bg-gradient-to-r from-[#3ce27a] to-[#afffb0]"
                    style={{ width: "100%" }}
                  ></div>
                </div>
                <div className="text-[10px] mt-1 opacity-90 text-center">
                  <span>100</span> / 100
                </div>
              </div>
            </div>
            <div className="relative w-[17vh] h-[17vh] rounded-md bg-gradient-to-br from-neutral-700 to-neutral-900 border-2 border-white/5 flex items-center justify-center">
              <img
                className="absolute w-[17vw]"
                src="../../../public/img/azuna_left.jpeg"
                alt="P2"
              />
            </div>
          </div>
        </div>

        {/* VSバッジ */}
        <div className="absolute left-1/2 top-5 -translate-x-1/2 text-center">
          <div className="text-[1rem] opacity-90">ROUND 1</div>
          <div className="text-[clamp(1.2rem,3vw,7rem)] text-[#ff0d00] drop-shadow-[0_2px_8px_rgba(255,200,0,0.1)]">
            VS
          </div>
        </div>

        {/* 中央下部 - 説明とタイマー */}
        <div className="flex flex-col items-center justify-center">
          <div className="text-[14px] opacity-85">
            Type to attack! 正確にタイプすると相手のHPが減る
          </div>
          <div className="mt-3 px-4 py-2 rounded-lg bg-white/5 text-center w-[120px]">
            <div className="text-[12px] opacity-80">時間</div>
            <div className="text-[clamp(1.2rem,2vw,7rem)] mt-1">{timeLeft}</div>
            <div className="text-[12px] opacity-80">秒</div>
          </div>
        </div>

        <div className="absolute right-[-60px] bottom-[-60px] text-[220px] rotate-[-20deg] opacity-[0.08] select-none pointer-events-none">
          FIGHT
        </div>
      </div>

      {/* サイドパネル */}
      <div className="rounded-lg bg-gradient-to-b from-white/5 to-black/5 p-5 flex flex-col gap-3">
        <div className="text-[12px] text-[#ff0d00] tracking-wider">KEY WARS</div>

        <div className="bg-gradient-to-b from-[#260707] to-[#0c0404] p-3 rounded-md border border-white/5 flex items-center justify-center min-h-[60px] text-[clamp(1.2rem,2vw,7rem)]">
          <div>
            <span>Type "</span>
            {renderHighlightedRomaji(targetRomaji, progress)}
            {/* <span className="font-bold text-[#3ce27a]">{promptText}</span> */}
            <span>"</span>
          </div>
        </div>

        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={handleTyping}
          placeholder="Enterで攻撃開始　ここにタイプ"
          className="w-full p-[3vh] rounded-lg border-2 border-white/5 bg-transparent text-white font-['Press_Start_2P'] text-[] focus:outline-none focus:border-[#ffcc00] focus:shadow-[0_0_10px_#ffaa00]"
        />

        <div className="grid gap-2 mt-2">
          <div className="bg-[#24140f] p-3 rounded-lg text-center">
            <div className="text-[10px] opacity-80">TypeMiss（通算）</div>
            <div className="text-[18px]">{missCount}</div>
          </div>
        </div>

        <div className="flex justify-center gap-3 mt-2">
          <button className="px-3 py-2 rounded-lg border-2 border-white/5 text-white hover:bg-red-900/40">
            やめる
          </button>
        </div>

        {/* デバッグ用ボタン */}
        {/* <div className="flex justify-center gap-3 mt-3">
          <button
            onClick={() => finishBattle(true)}
            className="px-4 py-2 bg-green-700 rounded-lg hover:bg-green-600 text-[12px]"
          >
            勝利にする
          </button>
          <button
            onClick={() => finishBattle(false)}
            className="px-4 py-2 bg-red-700 rounded-lg hover:bg-red-600 text-[12px]"
          >
            敗北にする
          </button>
        </div> */}
      </div>
    </div>
  );
}
