import React, { useState } from "react";
import "./Login.css";

// loginファンクション
export default function Login() {
  // UserNameとPasswordの状態管理
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
      localStorage.setItem("token", data.token);
      localStorage.setItem("user_name", userName.trim());
      window.location.href = "/";
    } catch (error) {
      alert("ログインに失敗しました");
      console.error(error);
    }
  };

  const signup = () => {
    window.location.href = "/signup";
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gradient-radial from-neutral-900 to-black text-white font-['Press_Start_2P'] relative overflow-hidden">
      {/* 背景グラデーション */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0 pointer-events-none" ></div>

      {/* メインコンテンツ */}
      <div>
        <div
          className="title"
        >
          KEY WARS
        </div>

        <div className="gate mb-[clamp(0.3rem,1vh,0.5rem)]"></div>

        <form
          onSubmit={enterArena}
          className="flex flex-col space-y-[clamp(0.3rem,1vh,0.8rem)] w-full"
        >
          <input
            type="text"
            placeholder="PLAYER NAME"
            maxLength={12}
            value={userName}
            onChange={(e) => setUserName(e.target.value)}
            required
            className="
              block mx-auto text-center
              border-2 border-orange-500 bg-black text-white
              rounded-md outline-none
              px-[clamp(0.4rem,1vw,0.5rem)]
              py-[clamp(0.2rem,0.6vh,0.7rem)]
              text-[clamp(0.6rem,0.9vw,0.8rem)]
              focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00]
              transition-all
            "
          />

          <input
            type="password"
            placeholder="PASSWORD"
            maxLength={12}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="
              block mx-auto text-center
              border-2 border-orange-500 bg-black text-white
              rounded-md outline-none
              px-[clamp(0.4rem,1vw,0.5rem)]
              py-[clamp(0.2rem,0.6vh,0.7rem)]
              text-[clamp(0.6rem,0.9vw,0.8rem)]
              focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00]
              transition-all
            "
          />

          <div className="mt-[clamp(0.4rem,1vh,1rem)] flex flex-wrap justify-center gap-4">
            <button
              type="submit"
              className="
                px-[clamp(0.4rem,1vw,0.5rem)]
                py-[clamp(0.2rem,0.6vh,0.7rem)]
                border-2 border-yellow-400 rounded-md
                bg-red-600 text-white
                text-[clamp(0.6rem,0.9vw,0.8rem)]
                hover:bg-orange-500
                hover:shadow-[0_0_25px_#ffaa00]
                transition-transform hover:scale-105
              "
            >
              ENTER ARENA
            </button>

            <button
              type="button"
              onClick={signup}
              className="
                px-[clamp(0.4rem,1vw,0.5rem)]
                py-[clamp(0.2rem,0.6vh,0.7rem)]
                border-2 border-yellow-400 rounded-md
                bg-blue-600 text-white
                text-[clamp(0.6rem,0.9vw,0.8rem)]
                hover:bg-cyan-400
                hover:shadow-[0_0_25px_#00ffff]
                transition-transform hover:scale-105
              "
            >
              NEW CHALLENGER
            </button>
          </div>
        </form>

        <footer className="absolute bottom-1 left-0 w-full text-center text-[clamp(0.45rem,0.8vw,0.65rem)] text-gray-500">
          © 2025 KEY WARS Tournament
        </footer>
      </div>
    </div>
  );
}
