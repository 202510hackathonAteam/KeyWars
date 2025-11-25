// src/App.jsx
import AppRoutes from "./routes";
import { AuthProvider } from "./context/AuthContext";
import { WebSocketProvider } from "./context/WebsocketContext";

console.log("ENV:", import.meta.env.VITE_API_URL);

function App() {
  return (
    <AuthProvider>
        <AppRoutes />
    </AuthProvider>
  );
}

export default App;