import React, { useEffect, useRef, useState, useContext } from "react";
// import "./Battle.css";
import { useNavigate } from "react-router-dom";
import { checkTyping } from "../../utils/typingLogic"; 
import { WebSocketContext } from "../../context/WebsocketContext";

// === ターゲットローマ字のハイライト表示 ===
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
                ? "text-[5vh] text-[#3ce27a] font-bold drop-shadow-[0_0_5px_#3ce27a]"
                : "text-[5vh] text-white/40"
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
  const inputRef = useRef(null);

  // 入力状態
  const [input, setInput] = useState("");
  const [progress, setProgress] = useState(0);
  const [missCount, setMissCount] = useState(0);
  const missCountRef = useRef(0);
  const [targetRomaji, setTargetRomaji] = useState("");
  const [promptText, setPromptText] = useState("");
  const ignoreTypingRef = useRef(false);

  // 時間 & HP
  const [timeLeft, setTimeLeft] = useState(60);
  const [playerHp, setPlayerHp] = useState(150);
  const [enemyHp, setEnemyHp] = useState(150);
  const MAX_HP = 150;

  // カウントダウン
  const [countdown, setCountdown] = useState(null);

  // WebSocket
  const { wsRef, matchStartPayload, matchRestorePayload, isRestoring, setIsRestoring } = useContext(WebSocketContext);

  // タイマー管理
  const timerRef = useRef(null);
  const countdownRef = useRef(null);
  const finishSentRef = useRef(false);

  // === ラウンド単位の一意IDを保持（タイマー二重起動防止）===
  const roundIdRef = useRef(null);
  
  // サーバーが次の問題を送ってくる間に表示する。
  const [isWaiting, setIsWaiting] = useState(false);

  // callsign
  const [myCallsign, setMyCallsign] = useState("");
  const [enemyCallsign, setEnemyCallsign] = useState("");

  // websocket閉じる
  const { leaveQueueFromBattle } = useContext(WebSocketContext);

  // カウントダウン時のインプット不可
  const isInputDisabled = countdown !== null || isRestoring;

  // 同じ round を二重に適用しないためのガード用
  const appliedRoundRef = useRef(null);


  // ============================================
  // answer.finish / answer.timeout を1回だけ送る
  // ============================================
  const doFinishOnce = (type) => {
    if (finishSentRef.current) return;

    // 二重送信を防ぐ
    finishSentRef.current = true;

    // 自分側のタイマーは即終了
    clearInterval(timerRef.current);

    // 送信データを作る
    const msg = {
      type,
      body: {
        match_id: matchStartPayload.match_id,
        miss_count: missCountRef.current,
      },
    };

    // サーバーへ通知
    wsRef.current?.send(JSON.stringify(msg));

    // 相手待ち UI へ
    setIsWaiting(true);
  };

  const sendAnswerFinish = () => {
    doFinishOnce("answer.finish");
  }
  const sendAnswerTimeout = () => {
    doFinishOnce("answer.timeout");
  }

  // ============================================
  // 🕒 ラウンドタイマー（1秒間隔）
  // ============================================
  const startBattleTimer = (roundId) => {
    clearInterval(timerRef.current);

    timerRef.current = setInterval(() => {
      // もし別ラウンドになっていたらこのタイマーは古いので終了
      if (roundIdRef.current !== roundId) {
        clearInterval(timerRef.current);
        return;
      }

      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          sendAnswerTimeout();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  // ============================================
  // 🔥 match.start を受け取ったときのメイン処理
  // ============================================
  useEffect(() => {
    if (!matchStartPayload) return;

    const round = matchStartPayload.state.round;

    // --- 重複ガード ---
    if (appliedRoundRef.current === round) return;

    // 初めて見るラウンドなので適用
    appliedRoundRef.current = round;

    setIsRestoring(false);

    // ★ 相手待ちモード解除（次のラウンドが始まったので）
    setIsWaiting(false);

    // 🔥 ラウンドID を設定（これで二重ラウンドを防ぐ）
    roundIdRef.current = matchStartPayload.state.round;

    // 🔥 finish フラグをリセット
    finishSentRef.current = false;

    const myId = localStorage.getItem("user_id");

      // --- ★ プレイヤーが自分なのか判定 ---
    const isPlayer1 = matchStartPayload.player1 === myId;

    const p1Hp = matchStartPayload.state.player1_lifepoint;
    const p2Hp = matchStartPayload.state.player2_lifepoint;

     // --- ★ callsign / user_name の設定 ---
    if (isPlayer1) {
      setMyCallsign(matchStartPayload.player1_name);
      setEnemyCallsign(matchStartPayload.player2_name);
    } else {
      setMyCallsign(matchStartPayload.player2_name);
      setEnemyCallsign(matchStartPayload.player1_name);
    }

    // // 🔥 古いタイマーを完全停止
    // clearInterval(timerRef.current);
    // clearInterval(countdownRef.current);

      // --- ★ HP を反映 ---
    if (isPlayer1) {
      setPlayerHp(p1Hp);
      setEnemyHp(p2Hp);
    } else {
      setPlayerHp(p2Hp);
      setEnemyHp(p1Hp);
    }

    // 状態リセット
    setInput("");
    setProgress(0);
    setMissCount(0);
    missCountRef.current = 0;
    setTargetRomaji(matchStartPayload.prompt.target_romaji);
    setPromptText(matchStartPayload.prompt.prompt_text_ja);

    const sec = Math.floor(matchStartPayload.prompt.limit_ms / 1000);
    setTimeLeft(sec);

    // === 3秒カウントダウン開始 ===
    let count = 3;
    setCountdown(count);

    countdownRef.current = setInterval(() => {
      count -= 1;

      if (roundIdRef.current !== matchStartPayload.state.round) {
        clearInterval(countdownRef.current);
        return;
      }

      if (count > 0) {
        setCountdown(count);
      } else {
        clearInterval(countdownRef.current);
        setCountdown(null);

        // 入力欄フォーカス
        inputRef.current?.focus();

        // バトルタイマー開始
        startBattleTimer(roundIdRef.current);
      }
    }, 1000);

    // return () => {
    //   clearInterval(timerRef.current);
    //   clearInterval(countdownRef.current);
    // };
  }, [matchStartPayload]);

  // ============================================
  // 📝 再接続処理
  // ============================================

  useEffect(() => {
  if (!matchRestorePayload) return;

  const restore = matchRestorePayload;

  const myId = localStorage.getItem("user_id");
  const isPlayer1 = matchStartPayload?.player1 === myId;

  const p1Hp = restore.state.player1_lifepoint;
  const p2Hp = restore.state.player2_lifepoint;

  // HP 復元
  if (isPlayer1) {
    setPlayerHp(p1Hp);
    setEnemyHp(p2Hp);
  } else {
    setPlayerHp(p2Hp);
    setEnemyHp(p1Hp);
  }

  // ラウンド番号復元
  roundIdRef.current = restore.state.round;

  // 入力停止
  ignoreTypingRef.current = true;

  // Countdown / Timer 停止
  clearInterval(timerRef.current);
  clearInterval(countdownRef.current);

  // 本当の次ラウンドは「次の match.start」で再開される
}, [matchRestorePayload]);

  // ============================================
  // 📝 入力処理
  // ============================================
  
  const handleTyping = (e) => {
    const val = e.target.value;
    if (ignoreTypingRef.current) {
      setInput(val); // 入力欄だけ更新
      return;        // ← ロジック実行しない
    }

    // ★ 通常入力
    setInput(val);

    const { newProgress, isCorrect, isFinish, miss } =
      checkTyping(val, targetRomaji, progress);

    if (!isCorrect) {
      setMissCount((prev) => prev + miss);
      missCountRef.current += miss;
      return;
    }

    setProgress(newProgress);

    if (isFinish) {
      sendAnswerFinish();
    }
  };

  // ============================================
  // 🔚 バトル終了
  // ============================================
  const { matchEndPayload, resetMatchState } = useContext(WebSocketContext);

  useEffect(() => {
  if (!matchEndPayload) return;

  const myId = localStorage.getItem("user_id"); // ← 自分のユーザーID

  // ★ 自分の total miss count を判定
  const isPlayer1 = matchEndPayload.player1 === myId;
  const myTotalMiss = isPlayer1
    ? matchEndPayload.player1_total_miss_count
    : matchEndPayload.player2_total_miss_count;

  let result;

  if (matchEndPayload.result === "draw") {
    result = "draw";
  } else if (matchEndPayload.result === "win") {
    result = matchEndPayload.winner === myId
      ? "victory"
      : "defeat";
  } else {
    console.warn("Unknown match result:", matchEndPayload.result);
    result = "defeat";
  }

  const payloadForResult = matchEndPayload;
  resetMatchState();

  // 結果画面へ
  navigate("/result", {
    state: {
      result: result,
      totalMiss: myTotalMiss,              
      matchEnd: payloadForResult, // ← 必要ならデータ丸ごと送れる
      round: matchStartPayload?.state?.round,  // ← 追加！！
    },
  });

}, [matchEndPayload, navigate, resetMatchState]);

  useEffect(() => {
    if (countdown === null) {
      setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    }
  }, [countdown]);


  return (
    
    <div className="
    mt-[5vh]
    max-w-[90vw] mx-auto my-6 
    p-5 rounded-xl 
    bg-gradient-to-b from-white/5 to-black/5
    h-[85vh]
    from-white/5 
    to-black/5 
    shadow-[0_8px_30px_rgba(252,2,2,0.6)]
    gap-y-6"   /* ← 内部要素間に余白を取る */
    >
      {/* リストア時の入力禁止 */}
      {isRestoring && (
      <div className="fixed inset-0 bg-black/80 flex flex-col items-center justify-center text-white z-50">
        <div className="text-[clamp(2rem,5vw,4rem)] font-bold animate-pulse">
          データロード中...
        </div>
        <div className="mt-4 text-[clamp(1rem,2vw,2rem)] opacity-80">
          試合状態を復元しています
        </div>
      </div>
      )}
      {/* 相手の入力待ち文 */}
      {isWaiting && (
        <div className="fixed inset-0 bg-black/70 flex flex-col items-center justify-center z-50">
          <div className="text-white text-[clamp(2rem,4vw,3.5rem)] mb-4 animate-pulse">
            ただいま相手の回答を待っています...
          </div>
          <div className="text-white text-[clamp(1rem,3vw,3.5rem)] mb-4 animate-pulse">
            相手が回答を送ると結果が確定します
          </div>
        </div>
      )}
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
              <div
                className="
                  text-[clamp(0.8rem,1vw,2rem)]
                  min-w-[25vw]
                  max-w-[25vw]
                  whitespace-nowrap
                  overflow-hidden
                  text-ellipsis
                "
              >
                CALLSIGN: {myCallsign}
              </div>
              <div className="bg-white/5 p-2 rounded-md mt-2">
                <div className="h-[16px] bg-neutral-800 rounded-md overflow-hidden">
                  <div
                    className="h-full w-full bg-gradient-to-r from-[#3ce27a] to-[#afffb0] transition-all duration-300"
                    style={{ width: `${(playerHp / MAX_HP) * 100}%` }}
                  ></div>
                </div>
                <div className="text-[10px] mt-1 opacity-90">
                  <span>{playerHp}</span> / 150
                </div>
              </div>
            </div>
          </div>

          {/* プレイヤー2 */}
          <div className="flex items-center gap-2 w-[48%] justify-end">
            <div>
              <div
                className="
                  text-[clamp(0.8rem,1vw,2rem)]
                  min-w-[25vw]
                  max-w-[25vw]
                  whitespace-nowrap
                  overflow-hidden
                  text-ellipsis
                "
              >
                CALLSIGN: {enemyCallsign}
              </div>
              <div className="bg-white/5 p-2 rounded-md mt-2">
                <div className="h-[16px] bg-neutral-800 rounded-md overflow-hidden">
                  <div
                    className="h-full w-full bg-gradient-to-r from-[#3ce27a] to-[#afffb0]"
                    style={{ width: `${(enemyHp / MAX_HP) * 100}%`  }}
                  ></div>
                </div>
                <div className="text-[10px] mt-1 opacity-90 text-center">
                  <span>{enemyHp}</span> / 150
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
          <div className="text-[1rem] opacity-90">ROUND {matchStartPayload?.state?.round}</div>
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

        <div className="bg-gradient-to-b from-[#260707] to-[#0c0404] p-3 rounded-md border border-white/5 flex items-center justify-center min-h-[60px] text-[clamp(1.2rem,2vw,7rem)]">
          <div>
            {countdown !== null && (
            <div className="absolute inset-0 flex items-center justify-center text-[clamp(3rem,6vw,10rem)] 
              font-bold text-white drop-shadow-[0_0_20px_#ffaa00]">
              {countdown}
            </div>
          )}

            {/* 現在の問題番号 */}
            <div className="text-white text-lg font-bold tracking-wide mb-1 text-[3vh]" >
              第 {matchStartPayload?.state?.round} 問目
            </div>

              {/* 日本語のお題 */}
            <div className="text-white text-lg font-bold tracking-wide text-[3vh]">
              {promptText}
            </div>


           {/* Type + Romaji（2行目） */}
            <div className="flex items-center text-white text-base text-[5vh]">
              <span>Type "</span>
              {renderHighlightedRomaji(targetRomaji, progress)}
              <span>"</span>
            </div>
          </div>
        </div>

        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={handleTyping}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              return;
            }

            // ★ Backspace と Delete はゲームロジック無視
            if (e.key === "Backspace" || e.key === "Delete") {
              ignoreTypingRef.current = true;
            } else {
              ignoreTypingRef.current = false;
            }
          }}
          disabled={isInputDisabled}
          placeholder="ここにタイプ"
          className="
            w-full p-[3vh] rounded-lg border-2 border-white/5 bg-transparent text-white 
            font-['Press_Start_2P'] text-[] 
            focus:outline-none focus:border-[#ffcc00] 
            focus:shadow-[0_0_10px_#ffaa00] 
            disabled:opacity-30 
            disabled:cursor-not-allowed"
        />

        <div className="grid gap-2 mt-2">
          <div className="bg-[#24140f] p-3 rounded-lg text-center">
            <div className="text-[2vh] opacity-80">TypeMiss（通算）</div>
            <div className="text-[2vh]">{missCount}</div>
          </div>
        </div>

        <div className="flex justify-center gap-3 mt-2">
          <button
              className="px-3 py-2 rounded-lg border-2 border-white/5 text-white hover:bg-red-900/40 text-[2vh] "
              onClick={() => {
                // 1. queue.left を送信して接続解除
                leaveQueueFromBattle();

                // 2. 結果画面へ defeat として遷移
                navigate("/result", {
                  state: {
                    result: "defeat",
                    matchEnd: null,     // ← 試合データはない
                    forceQuit: true,    // ← 任意（Result 側で判定可能）
                  },
                });
              }}
            >
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
