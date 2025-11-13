// src/routes/ProtectedRoute.jsx
import { Navigate } from "react-router-dom";
import { useEffect, useState } from "react";

export default function ProtectedRoute({ children }) {
  const [isAuth, setIsAuth] = useState(null);

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const res = await fetch("/auth/me", {
          method: "GET",
          credentials: "include", // Cookie が必須
        });

        if (res.ok) {
          setIsAuth(true);
        } else {
          setIsAuth(false);
        }
      } catch (err) {
        setIsAuth(false);
      }
    };

    checkAuth();
  }, []);

  // 認証状態が不明の間はローディング
  if (isAuth === null) return <div>Loading...</div>;

  // 未認証なら login に飛ばす
  if (!isAuth) return <Navigate to="/login" replace />;

  return children;
}