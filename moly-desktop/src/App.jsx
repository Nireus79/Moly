import React, { useState, useEffect } from 'react';
import SetupWizard from './components/SetupWizard';
import MainApp from './components/MainApp';
import './App.css';

export default function App() {
  const [isSetup, setIsSetup] = useState(false);
  const [setupComplete, setSetupComplete] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkSetupStatus();
  }, []);

  async function checkSetupStatus() {
    try {
      const nativeHostExists = await window.moly.checkNativeHost();
      const proxyRunning = await window.moly.getProxyStatus();

      if (nativeHostExists && proxyRunning) {
        setSetupComplete(true);
        setIsSetup(true);
      } else {
        setIsSetup(false);
      }
    } catch (error) {
      console.error('Setup check failed:', error);
      setIsSetup(false);
    } finally {
      setLoading(false);
    }
  }

  async function handleSetupComplete() {
    setSetupComplete(true);
    setIsSetup(true);
  }

  if (loading) {
    return (
      <div className="app-container loading">
        <div className="spinner"></div>
        <p>Initializing Moly...</p>
      </div>
    );
  }

  if (!setupComplete) {
    return <SetupWizard onComplete={handleSetupComplete} />;
  }

  return <MainApp />;
}
