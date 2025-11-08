// src/routes.jsx
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./assets/home/Home";
import Login from "./assets/login/Login";
import Signup from "./assets/signup/Signup";
import Battle from "./assets/battle/Battle";
import Result from "./assets/result/Result";

export default function AppRoutes() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />
        <Route path="/battle" element={<Battle />} />
        <Route path="/result" element={<Result />} />
      </Routes>
    </BrowserRouter>
  );
}