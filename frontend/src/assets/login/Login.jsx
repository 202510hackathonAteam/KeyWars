import React, { useState } from "react";
import "./Login.css";

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
  <div class="flex items-center justify-center min-h-screen bg-gradient-radial from-neutral-900 to-black text-white font-['Press_Start_2P'] relative overflow-hidden">
    <div class="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0"></div>

      <div class="text-center z-10">
        <div className="title">KEY WARS</div>

        <div className="gate"></div>

        <form onSubmit={enterArena} class="space-y-2">
          <input
            className="entry-form"
            type="text"
            placeholder="PLAYER NAME"
            maxLength={12}
            value={userName}
            onChange={(e) => setUserName(e.target.value)}
            required
            class="block mx-auto w-56 px-3 py-1.5 border-2 border-orange-500 bg-black text-white text-xs text-center rounded-md focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00] outline-none"
          />
          <input
            className="entry-form"
            type="password"
            placeholder="PASSWORD"
            maxLength={12}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            class="block mx-auto w-56 px-3 py-1.5 border-2 border-orange-500 bg-black text-white text-xs text-center rounded-md focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00] outline-none"
          />
          <div class="mt-1">
          <button type="submit" class="px-4 py-1.5 border-2 border-yellow-400 rounded-md bg-red-600 text-white text-xs hover:bg-orange-500 hover:shadow-[0_0_25px_#ffaa00] transition-transform hover:scale-110 mr-3">
            ENTER ARENA
          </button>
          <button type="button" class="px-4 py-1.5 border-2 border-yellow-400 rounded-md bg-blue-600 text-white text-xs hover:bg-cyan-400 hover:shadow-[0_0_25px_#00ffff] transition-transform hover:scale-110" onClick={signup}>
            NEW CHALLENGER
          </button>
          </div>
        </form>
        <footer class="absolute bottom-4 text-xs text-gray-500">
          © 2025 KEY WARS Tournament
        </footer>
      </div>
    </div>
  );
}
