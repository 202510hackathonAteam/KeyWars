// src/assets/components/common/HomeButton.jsx
import React from "react";

export default function LoginbackButton() {
  return (
    <button
      onClick={() => (window.location.href = "/login")}
      className="
        absolute top-[clamp(2rem,15vh,4rem)] left-[clamp(0.5rem,15vw,5rem)]
        z-20 border-2 border-yellow-400 rounded-md
        bg-red-700 text-white
        text-[clamp(0.45rem,0.8vw,0.7rem)]
        px-[clamp(0.4rem,0.8vw,0.8rem)]
        py-[clamp(0.2rem,0.6vh,0.5rem)]
        hover:bg-orange-500 hover:shadow-[0_0_20px_#ffaa00]
        transition-all
      "
    >
      ← BACK TO LOGIN
    </button>
  );
}
