// src/components/LogoutButton.jsx
import { getCookie } from "../utils/cookieUtils";
import { useNavigate } from "react-router-dom";

export default function LogoutButton() {
  const navigate = useNavigate();

  const handleLogout = async () => {
    const csrf = getCookie("csrf_token");

    try {
      const res = await fetch("/auth/signout", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
      });

      // どの状態でもとりあえずログインへ
      navigate("/login");
    } catch (err) {
      console.error("Logout error:", err);
      navigate("/login");
    }
  };

  return (
    <button
      onClick={handleLogout}
      className="
        absolute top-5 left-5 
        px-4 py-2 
        bg-red-600 text-white text-xs 
        border border-yellow-400 rounded-lg
        shadow-[0_0_10px_#ff0000]
        hover:bg-red-400 hover:shadow-[0_0_15px_#ffaa00]
        transition
        z-50
      "
    >
      LOGOUT
    </button>
  );
}
