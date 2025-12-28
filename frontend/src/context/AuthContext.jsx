// AuthContext.jsx
import { createContext, useState, useEffect, useRef } from "react";
import { getCookie } from "../utils/cookieUtils";

export const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [isAuth, setIsAuth] = useState(null);
  const justLoggedInRef = useRef(localStorage.getItem("justLoggedIn") === "true");
  const API_URL = import.meta.env.VITE_API_URL

  useEffect(() => {

    if (justLoggedInRef.current) {
      setIsAuth(true);
      localStorage.removeItem("justLoggedIn");
      return;
    }

    const checkAuth = async () => {
      const csrf = getCookie("csrf_token");
      const res = await fetch(`${API_URL}/auth/check`, { 
        method: "POST", 
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
    });
      setIsAuth(res.ok);
    };
    checkAuth();
  }, []);

  return (
    <AuthContext.Provider value={{ isAuth, setIsAuth }}>
      {children}
    </AuthContext.Provider>
  );
}
