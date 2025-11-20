import { Navigate, useLocation } from "react-router-dom";
import { useState, useEffect } from "react";

function getCookie(name) {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop().split(";").shift();
  return null;
}

export default function ProtectedRoute({ children }) {
  const [isAuth, setIsAuth] = useState(null);
  const location = useLocation();

  // ログイン or サインアップでは auth チェックしない
  if (location.pathname === "/login" || location.pathname === "/signup") {
    return children;
  }

  useEffect(() => {
    const check = async () => {
      try {
        const csrfToken = getCookie("csrf_token");

        const res = await fetch("/auth/check", {
          method: "POST",
          credentials: "include",
          headers: {
            "Content-Type": "application/json",
            ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
          },
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
