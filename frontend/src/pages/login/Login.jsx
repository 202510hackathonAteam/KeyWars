// Login.jsx
import React, { useState, useEffect } from "react";
import "./Login.css";
import { getCookie } from "../../utils/cookieUtils.jsx";


// loginファンクション
export default function Login() {
  // UserNameとPasswordの状態管理
  const [userName, setUserName] = useState("");
  const [password, setPassword] = useState("");
  const [csrfToken, setCsrfToken] = useState("");

  const API_URL = import.meta.env.VITE_API_URL
  // ★ ① 初回レンダリング時に /healthz を叩いて CSRF Cookie を取得
  useEffect(() => {
    const fetchCsrf = async () => {
      try {
        await fetch(`${API_URL}/healthz`, {
          method: "GET",
          credentials: "include",  // ← Cookie を受け取るために必要
        });

        // /healthz のレスポンスで Set-Cookie された csrf_token を取得
        const token = getCookie("csrf_token");
        console.log("Retrieved CSRF Token from cookie:", token);
        setCsrfToken(token);
      } catch (error) {
        console.error("failed to fetch CSRF:", error);
      }
    };

    fetchCsrf();
  }, []);

  const enterArena = async (e) => {
    console.log("ENTER ARENA CALLED");
    e.preventDefault();

    try {
      const response = await fetch(`${API_URL}/auth/signin`, {
        method: "POST",
        headers: { "Content-Type": "application/json",
          ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
        },
        body: JSON.stringify({ user_name: userName, password }),
        credentials: "include", 
      });
      if (!response.ok) throw new Error("Login failed");

     
      localStorage.setItem("user_name", userName.trim());
      localStorage.setItem("justLoggedIn", "true");
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
    <div className="
  flex flex-col items-center justify-center
  min-h-screen
  bg-gradient-radial from-neutral-900 to-black text-white
  font-['Press_Start_2P']
  relative overflow-hidden
  py-[clamp(2rem,8vh,5rem)]
    ">
      
      {/* 背景グラデーション */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0 pointer-events-none" ></div>

      {/* メインコンテンツ */}
      <div className="flex flex-col items-center gap-[clamp(1.8rem,6vh,4rem)] z-50">
        <div
          className="title"
        >
          KEY WARS
        </div>

        <div className="gate mb-[clamp(0.3rem,1vh,0.5rem)]"></div>

        <form
          onSubmit={enterArena}
          onChange={() => console.log("FORM CHANGED")}
          className="flex flex-col space-y-[clamp(0.5rem,2vh,1.5rem)] w-full"
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
            w-[clamp(200px,40vw,360px)]
            h-[clamp(32px,6vh,48px)]
            text-[clamp(1rem,1.2vw,1.2rem)]
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
            w-[clamp(200px,40vw,360px)]
            h-[clamp(32px,6vh,48px)]
            text-[clamp(1rem,1.2vw,1.2rem)]
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
                text-[clamp(1rem,1.2vw,1.2rem)]
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
                text-[clamp(1rem,1.2vw,1.2rem)]
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
