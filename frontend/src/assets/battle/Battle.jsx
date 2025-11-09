import React from "react";
import "./Battle.css";

export default function GamePage() {
  return (
    <div className="stage">
      <div className="battle">
        <div className="ui-row">
          <div className="player-box">
            <div className="nameplate">
              <div className="avatar">
                <img className="imgp1" src="../../../public/img/ai_right.jpeg" alt="P1" />
              </div>
              <div>
                <div style={{ fontSize: "12px" }}>CALLSIGN: KYO</div>
                <div className="hp-wrap">
                  <div className="hp-bar">
                    <div className="hp-fill" style={{ width: "100%" }}></div>
                  </div>
                  <div className="hp-num">
                    <span>100</span> / 100
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="player-box" style={{ textAlign: "right" }}>
            <div className="nameplate" style={{ justifyContent: "flex-end" }}>
              <div>
                <div style={{ fontSize: "12px" }}>CALLSIGN: RYU</div>
                <div className="hp-wrap">
                  <div className="hp-bar">
                    <div className="hp-fill" style={{ width: "100%" }}></div>
                  </div>
                  <div className="hp-num">
                    <span>100</span> / 100
                  </div>
                </div>
              </div>
              <div className="avatar">
                <img className="imgp2" src="../../../public/img/azuna_left.jpeg" alt="P2" />
              </div>
            </div>
          </div>
        </div>

        <div className="vs-badge">
          <div className="round">ROUND 1</div>
          <div className="vs">VS</div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            flexDirection: "column",
          }}
        >
          <div style={{ fontSize: "14px", opacity: 0.85 }}>
            Type to attack! 正確にタイプすると相手のHPが減る
          </div>
          <div style={{ padding: "14px", borderRadius: "10px", background: "rgba(255,255,255,0.02)", width: "120px", textAlign: "center", marginTop: "12px" }}>
            <div style={{ fontSize: "12px", opacity: 0.8 }}>時間</div>
            <div style={{ fontSize: "24px", marginTop: "6px" }}>60</div>
            <div style={{ fontSize: "12px", opacity: 0.8 }}>秒</div>
          </div>
        </div>

        <div className="bg-decor">FIGHT</div>
      </div>

      <div className="side">
        <div className="panel-title">KEY WARS</div>
        <div className="word-target">
          <div>
            <span>Type "</span>
            <span style={{ fontWeight: 700 }}>example</span>
            <span>"</span>
          </div>
        </div>

        <div className="kbd-area">
          <input
            type="text"
            placeholder="Enterで攻撃開始　ここにタイプ"
            autoComplete="off"
          />
        </div>

        <div className="stats">
          <div className="stat">
            <div style={{ fontSize: "10px", opacity: 0.8 }}>TypeMiss（通算）</div>
            <div className="num">0</div>
          </div>
        </div>

        <div className="btns">
          <button className="btn">やめる</button>
        </div>

        <div style={{ fontSize: "10px", opacity: 0.7, marginTop: "12px" }}>
          デザインは『格闘ゲーム風』の雰囲気を参考にしたオリジナルUIです。
        </div>
      </div>
    </div>
  );
}
