// src/App.jsx
import AppRoutes from "./routes";
import { AuthProvider } from "./context/AuthContext";

console.log("ENV:", import.meta.env.VITE_API_URL);

function App() {
  return (
    <AuthProvider>
      <div>
        <AppRoutes />
      </div>
    </AuthProvider>
  );
}

export default App;