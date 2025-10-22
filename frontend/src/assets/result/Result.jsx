import React, { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import "./Result.css";

const Result = ({ result = "victory" }) => {
  const navigate = useNavigate();

  useEffect(() => {
    document.body.className = result; // victory or defeat
    return () => {
      document.body.className = ""; // クリーンアップ
    };
  }, [result]);

  const retry = () => {
    alert("再戦スタート！");
    // navigate("/battle"); などに変更可能
  };

  const goHome = () => {
    alert("ホームに戻ります");
    navigate("/");
  };

  return (
    <div className="result-container">
      <h1>KEY WARS</h1>

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
