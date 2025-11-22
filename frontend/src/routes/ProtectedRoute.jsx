//protectedRoute.jsx
import { Navigate, useLocation } from "react-router-dom";
import { useContext } from "react";
import { AuthContext } from "../context/AuthContext";

export default function ProtectedRoute({ children }) {
  const { isAuth } = useContext(AuthContext);
  const location = useLocation();

  // const justLoggedInRef = useRef(localStorage.getItem("justLoggedIn") === "true");

  // ログイン or サインアップでは auth チェックしない
  if (location.pathname === "/login" || location.pathname === "/signup") {
    return children;
  }

  // useEffect(() => {
  //       // ログイン直後はチェックをスキップ
  //   if (justLoggedInRef.current) {
  //     setIsAuth(true);
  //     localStorage.removeItem("justLoggedIn");
  //     return;
  //   }

  if (isAuth === null) return <div>Loading...</div>;
  if (!isAuth) return <Navigate to="/login" replace />;

  return children;
}
