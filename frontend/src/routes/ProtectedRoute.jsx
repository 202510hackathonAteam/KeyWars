import { useLocation } from "react-router-dom";
import { useState, useEffect } from "react";

export default function ProtectedRoute({ children }) {
  const [isAuth, setIsAuth] = useState(null);
  const location = useLocation();

  // ログイン or サインアップページでは auth チェックしない
  if (location.pathname === "/login" || location.pathname === "/signup") {
    return children;
  }

  useEffect(() => {
    const check = async () => {
      try {
        const res = await fetch("/auth/check", {
          method: "POST",
          credentials: "include",
        });

        setIsAuth(res.ok);
      } catch {
        setIsAuth(false);
      }
    };

    check();
  }, []);

  if (isAuth === null) return <div>Loading...</div>;
  if (!isAuth) return <Navigate to="/login" replace />;

  return children;
}
