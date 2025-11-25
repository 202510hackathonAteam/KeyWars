// src/assets/components/Signup/Signup.jsx
import React from "react";
// import "./Signup.css";

export default function Testcss() {
  return (
  <div class="flex items-center justify-center min-h-screen bg-gradient-radial from-neutral-900 to-black text-white font-['Press_Start_2P'] relative overflow-hidden">
  <div class="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,0,0,0.2),transparent_70%),radial-gradient(circle_at_bottom,rgba(255,165,0,0.15),transparent_80%),radial-gradient(circle_at_top,rgba(0,0,255,0.15),transparent_70%)] animate-pulse z-0"></div>

  <div class="text-center z-10">
    <h1 class="text-3xl text-red-500 mb-12 drop-shadow-[0_0_10px_#ff0000] animate-pulse">
      TYPING FIGHTER
    </h1>

    <div class="w-80 h-44 border-2 border-yellow-400 rounded-lg shadow-[0_0_25px_#ffaa00_inset] mx-auto mb-10 bg-gradient-to-b from-orange-950/70 to-black/90 relative">
      <span class="absolute -top-6 left-1/2 -translate-x-1/2 bg-black px-3 py-1 text-yellow-400 text-xs border border-yellow-400 rounded-md shadow-[0_0_10px_#ff8800]">
        ARENA ENTRANCE
      </span>
    </div>

    <form onsubmit="enterArena(event)" class="space-y-5">
      <input
        type="text"
        id="playerName"
        placeholder="PLAYER NAME"
        maxlength="12"
        required
        class="block mx-auto w-56 px-3 py-2 border-2 border-orange-500 bg-black text-white text-xs text-center rounded-md focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00] outline-none"
      />

      <input
        type="password"
        id="password"
        placeholder="PASSWORD"
        maxlength="20"
        required
        class="block mx-auto w-56 px-3 py-2 border-2 border-orange-500 bg-black text-white text-xs text-center rounded-md focus:border-yellow-400 focus:shadow-[0_0_10px_#ffaa00] outline-none"
      />

      <div class="mt-4">
        <button
          type="submit"
          class="px-4 py-2 border-2 border-yellow-400 rounded-md bg-red-600 text-white text-xs hover:bg-orange-500 hover:shadow-[0_0_25px_#ffaa00] transition-transform hover:scale-110 mr-3"
        >
          ENTER ARENA
        </button>
        <button
          type="button"
          onclick="signup()"
          class="px-4 py-2 border-2 border-yellow-400 rounded-md bg-blue-600 text-white text-xs hover:bg-cyan-400 hover:shadow-[0_0_25px_#00ffff] transition-transform hover:scale-110"
        >
          NEW CHALLENGER
        </button>
      </div>
    </form>

    <footer class="absolute bottom-4 text-xs text-gray-500">
      © 2025 Typing Fighter Tournament
    </footer>
  </div>
  </div>
  );
}
