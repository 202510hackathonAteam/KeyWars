import React, { useState } from "react";
// import "./Login.css";

export default function Login() {
  const [userName, setUserName] = useState("");
  const [password, setPassword] = useState("");

  const enterArena = async (e) => {
    e.preventDefault();

    try {
      const response = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_name: userName, password }),
      });

      if (!response.ok) throw new Error("Login failed");

      const data = await response.json();

      // JWTトークンなどを保存
      localStorage.setItem("token", data.token);
      localStorage.setItem("user_name", userName.trim());

      window.location.href = "/home";
    } catch (error) {
      alert("ログインに失敗しました");
      console.error(error);
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
          <input
            className="entry-form"
            type="text"
            placeholder="PLAYER NAME"
            maxLength={12}
            value={userName}
            onChange={(e) => setUserName(e.target.value)}
            required
          />
          <br />
          <input
            className="entry-form"
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
