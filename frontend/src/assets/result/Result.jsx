import React, { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
// import "./Result.css";

const Result = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // Battleから受け取る
  const result = location.state?.result || "victory"; // デフォルト値

  useEffect(() => {
    document.body.classList.add(result);
    return () => {
      document.body.classList.remove(result);
    };
  }, [result]);

  const retry = () => navigate("/battle");
  const goHome = () => navigate("/");

  return (
    <div className="result-container">
      <h1 className="title">KEY WARS</h1>

      <div className="result-panel">
        <div className="stat">
          間違えた回数（通算10戦）：<span id="mistakes">7</span>
        </div>

        <div className="buttons">
          <button onClick={retry}>再戦</button>
          <button onClick={goHome}>ホームに戻る</button>
        </div>
      </div>
    </div>
  );
};

export default Result;
