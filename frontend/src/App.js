import React, { useState } from 'react';
import MenuUploader from './MenuUploader';
import MenuDisplay from './MenuDisplay';

function App() {
  const [menu, setMenu] = useState(null);
  const [status, setStatus] = useState('');
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const handleMenuProcessed = (processedMenu) => {
    setMenu(processedMenu);
    setStatus('Menu processed successfully!');
  };

  const handleStatusUpdate = (newStatus) => {
    setStatus(newStatus);
  };

  const handleLogin = () => {
    // In Choreo deployment, this would redirect to /auth/login
    window.location.href = '/auth/login';
  };

  const handleLogout = () => {
    // In Choreo deployment, this would redirect to /auth/logout with session hint
    const sessionHint = document.cookie
      .split('; ')
      .find(row => row.startsWith('session_hint='))
      ?.split('=')[1];
    
    window.location.href = `/auth/logout${sessionHint ? `?session_hint=${sessionHint}` : ''}`;
  };

  return (
    <div className="app">
      <header className="header">
        <h1>🍽️ Menu Visualizer</h1>
        <p>Transform your restaurant menu photos into interactive digital experiences</p>
        
        {/* Authentication buttons for Choreo managed auth */}
        <div className="auth-container">
          {isAuthenticated ? (
            <button className="auth-button logout" onClick={handleLogout}>
              Logout
            </button>
          ) : (
            <button className="auth-button" onClick={handleLogin}>
              Login
            </button>
          )}
        </div>
      </header>

      {status && (
        <div className={`status ${status.includes('Error') ? 'error' : status.includes('processing') ? 'processing' : 'success'}`}>
          {status}
        </div>
      )}

      <MenuUploader 
        onMenuProcessed={handleMenuProcessed}
        onStatusUpdate={handleStatusUpdate}
      />

      {menu && <MenuDisplay menu={menu} />}
    </div>
  );
}

export default App;
