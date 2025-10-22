// src/assets/components/Home/Home.jsx
import React from "react";
import "./Home.css";

export default function Home() {
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

        <button className="btn">対戦相手をさがす</button>
      </div>

      <footer>© 2025 Key Wars Tournament</footer>
    </div>
  );
}
