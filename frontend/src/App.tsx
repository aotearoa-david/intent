import { CSSProperties, useEffect, useState } from 'react';
import { fetchGreeting } from './api';
import './app.css';

type GreetingState = {
  status: 'loading' | 'success' | 'error';
  message?: string;
};

const App = () => {
  const [greeting, setGreeting] = useState<GreetingState>({ status: 'loading' });

  useEffect(() => {
    fetchGreeting()
      .then((message) => setGreeting({ status: 'success', message }))
      .catch((error) => {
        console.error('Failed to load greeting', error);
        setGreeting({ status: 'error' });
      });
  }, []);

  return (
    <main style={styles.main}>
      <section style={styles.card}>
        <h1>Intent Platform</h1>
        <p>Your hello-world workspace for future EPICs, FEATUREs, and USER STORIES.</p>
        {greeting.status === 'loading' && <p>Loading greeting…</p>}
        {greeting.status === 'success' && <p data-testid="greeting">{greeting.message}</p>}
        {greeting.status === 'error' && (
          <p role="alert">We could not reach the API. Check the backend logs.</p>
        )}
      </section>
    </main>
  );
};

const styles: Record<string, CSSProperties> = {
  main: {
    fontFamily: 'system-ui, sans-serif',
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background:
      'radial-gradient(circle at top left, rgba(79, 70, 229, 0.2), transparent 55%), radial-gradient(circle at bottom right, rgba(16, 185, 129, 0.2), transparent 55%)',
  },
  card: {
    maxWidth: '480px',
    padding: '2.5rem',
    borderRadius: '1rem',
    boxShadow: '0 25px 50px -12px rgba(30, 64, 175, 0.25)',
    backgroundColor: 'rgba(255, 255, 255, 0.9)',
    textAlign: 'center',
  },
};

export default App;
