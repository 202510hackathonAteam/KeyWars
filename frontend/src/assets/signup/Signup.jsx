// src/assets/components/Signup/Signup.jsx
import React from "react";
import "./Signup.css";

export default function Signup() {
  return (
    <div className="signup-body">
      <div className="neon-glow"></div>

      <div className="container">
        <div className="title">KEY WARS</div>

        <div className="gate"></div>

        <form>
          <input type="text" placeholder="PLAYER NAME" maxLength="12" required />
          <br />
          <input type="text" placeholder="PASSWORD" maxLength="12" required />
          <br />
          <input
            type="text"
            placeholder="PASSWORD CONFIRM"
            maxLength="12"
            required
          />
          <br />
          <button type="button" className="signup-btn">
            OPEN WAR
          </button>
        </form>

        <footer>© 2025 KEY WARS Tournament</footer>
      </div>
    </div>
  );
}
