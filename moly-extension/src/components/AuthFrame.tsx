import React from 'react';
import '@/sidebar/components/login-screen.css';
import '@/sidebar/components/about-me.css';

/** The frame of the screens that ask who is using Moly: the mark, the name and a card, centred. The same look as the development login. */
export const AuthFrame: React.FC<{ title: string; lead?: string; children: React.ReactNode }> = ({ title, lead, children }) => (
  <div className="login-screen">
    <div className="login-card">
      <div className="login-brand">
        <span className="brand-mark" aria-hidden="true" />
        <h1 lang="el">Μώλυ</h1>
      </div>
      <p className="login-tagline">Think before you send.</p>
      <h2 className="auth-title">{title}</h2>
      {lead && <p className="auth-lead">{lead}</p>}
      {children}
    </div>
  </div>
);

export default AuthFrame;
