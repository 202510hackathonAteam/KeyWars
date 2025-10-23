import React, { useState } from "react";
// import "./Login.css";

// Loginという名前でコンポーネント化
export default function Login() {
  // playerNameをstateとして、セット
  const [playerName, setPlayerName] = useState("");
  const [password, setPassword] = useState("");

  const enterArena = (e) => {
    e.preventDefault();
    if (playerName.trim()) {
      localStorage.setItem("playerName", playerName.trim());
      window.location.href = "/home"; // React Router を使う場合は navigate("/home")
    }
  };

  const signup = () => {
    window.location.href = "/signup";
  };

  return (
    <div className="login-body">
      <div className="neon-glow"></div>

      <div className="container">
        <div className="title">KEY WARS</div>

        <div className="gate"></div>

        <form onSubmit={enterArena}>
          <input className="entry-form"
            type="text"
            placeholder="PLAYER NAME"
            maxLength={12}
            value={playerName}
            onChange={(e) => setPlayerName(e.target.value)}
            required
          />
          <br />
          <input className="entry-form"
            type="password"
            placeholder="PASSWORD"
            maxLength={12}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <br />
          <button type="submit" className="login-btn">
            ENTER ARENA
          </button>
          <button type="button" className="signup-btn" onClick={signup}>
            NEW CHALLENGER
          </button>
        </form>

        <footer>© 2025 KEY WARS Tournament</footer>
      </div>
    </div>
  );
}
