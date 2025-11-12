// src/App.jsx
import AppRoutes from "./routes";

console.log("ENV:", import.meta.env.VITE_API_URL);

function App() {

  return (
    
    <div>
      <AppRoutes />
    </div>
  );
}

export default App;