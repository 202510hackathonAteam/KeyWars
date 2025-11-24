// src/assets/components/Signup/Signup.jsx
import React, { useState, useEffect } from "react";
import "./Signup.css";
import LoginbackButton from "../../components/LoginbackButton.jsx";
import { getCookie } from "../../utils/cookieUtils.jsx";


export default function Signup() {
  const [userName, setUserName] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [error, setError] = useState("");
  const [csrfToken, setCsrfToken] =useState("")

  useEffect(() => {
      const fetchCsrf = async () => {
        try {
          await fetch("/healthz", {
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

  const handleSignup = async (e) => {
    console.log("Signup CALLED");
    e.preventDefault();

    // 入力チェック
    if (!userName || !password || !passwordConfirm) {
      setError("全ての項目を入力してください。");
      return;
    }
    if (password !== passwordConfirm) {
      setError("パスワードが一致しません。");
      return;
    }

    try {
      const signupresponse = await fetch("/auth/signup",{
        method: "POST",
        headers: { "Content-Type": "application/json",
          ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
        },
        body: JSON.stringify({
        user_name: userName,
        password: password,
        confirm_password: passwordConfirm,   // ← これが必須！！
      }),
        credentials: "include", 
      });

      // ここが重要：エラー時も JSON を読む
      const data = await signupresponse.json().catch(() => null);

      if (!signupresponse.ok) {
        setError(data?.message || "サーバーエラーが発生しました。");
        return;
      }

      // Cookie に token がセットされるので React 側で token を保存しない
      localStorage.setItem("user_name", userName.trim());
      localStorage.setItem("justLoggedIn", "true");
      window.location.href = "/";

    } catch (err) {
      setError("通信エラーが発生しました。");
      console.error(err);
    }

    
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
    <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0 pointer-events-none" ></div>
    < LoginbackButton/>
    {/* <div className="signup-body"> */}
      <div className="flex flex-col items-center gap-[clamp(1.5rem,5vh,3rem)] z-10">
        <div className="title">KEY WARS</div>

        <div className="gate"></div>

        <form
          onSubmit={handleSignup} 
          className="flex flex-col items-center space-y-[clamp(0.5rem,2vh,1.5rem)] w-full"
        >
          <input 
          type="text" 
          placeholder="PLAYER NAME" 
          maxLength="12" 
          required
          value={userName}
          onChange={(e) => setUserName(e.target.value)} 
          className="
            block mx-auto text-center
            border-2 border-orange-500 bg-black text-white
            rounded-md outline-none
            w-[clamp(200px,40vw,360px)]
            h-[clamp(32px,5vh,48px)]
            text-[clamp(0.6rem,0.9vw,0.8rem)]
            focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00]
            transition-all
            "
          />
          <input 
          type="password"
          placeholder="PASSWORD" 
          maxLength="12" 
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="
            block mx-auto text-center
            border-2 border-orange-500 bg-black text-white
            rounded-md outline-none
            w-[clamp(200px,40vw,360px)]
            h-[clamp(32px,5vh,48px)]
            text-[clamp(0.6rem,0.9vw,0.8rem)]
            focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00]
            transition-all
            "
           />
          <input
            type="password"
            placeholder="PASSWORD CONFIRM"
            maxLength="12"
            required
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
            className="
            block mx-auto text-center
            border-2 border-orange-500 bg-black text-white
            rounded-md outline-none
            w-[clamp(200px,40vw,360px)]
            h-[clamp(32px,5vh,48px)]
            text-[clamp(0.6rem,0.9vw,0.8rem)]
            focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00]
            transition-all
            "
          />
          {error && (
              <p className="text-red-500 text-[clamp(0.45rem,0.8vw,0.7rem)] text-center">
                {error}
              </p>
            )}
          <button 
          type="submit"
          className="
                min-w-[6rem] max-w-[10rem]
                px-[clamp(0.4rem,0.5vw,0.5rem)]
                py-[clamp(0.2rem,0.6vh,0.7rem)]
                border-2 border-yellow-400 rounded-md
                bg-blue-600 text-white
                text-[clamp(0.6rem,0.9vw,0.8rem)]
                hover:bg-cyan-400
                hover:shadow-[0_0_25px_#00ffff]
                transition-transform hover:scale-105
              "
              >
            OPEN WAR
          </button>
        </form>

        <footer className="absolute bottom-1 left-0 w-full text-center text-[clamp(0.45rem,0.8vw,0.65rem)] text-gray-500">© 2025 KEY WARS Tournament</footer>
        </div>
    </div>
  // </div>
  );
}
