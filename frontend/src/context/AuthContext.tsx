import React, { createContext, useContext, useEffect, useState } from 'react';
import { api } from '../lib/api';

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (password: string) => Promise<void>;
  logout: () => Promise<void>;
  openLoginModal: () => void;
  closeLoginModal: () => void;
  isLoginModalOpen: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoginModalOpen, setIsLoginModalOpen] = useState(false);

  useEffect(() => {
    api
      .checkAuth()
      .then((res) => setIsAuthenticated(res.authenticated))
      .catch(() => setIsAuthenticated(false))
      .finally(() => setIsLoading(false));

    const handleAuthReq = () => {
      setIsAuthenticated(false);
      setIsLoginModalOpen(true);
    };

    window.addEventListener('auth-required', handleAuthReq);
    return () => window.removeEventListener('auth-required', handleAuthReq);
  }, []);

  const login = async (password: string) => {
    const res = await api.login(password);
    if (res.authenticated) {
      setIsAuthenticated(true);
      setIsLoginModalOpen(false);
    }
  };

  const logout = async () => {
    await api.logout();
    setIsAuthenticated(false);
    setIsLoginModalOpen(true);
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        login,
        logout,
        openLoginModal: () => setIsLoginModalOpen(true),
        closeLoginModal: () => setIsLoginModalOpen(false),
        isLoginModalOpen,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
};
