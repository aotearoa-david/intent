import { CSSProperties, ChangeEvent, FormEvent, useEffect, useState } from 'react';
import { fetchGreeting, submitIntent, IntentResponse } from './api';
import './app.css';

type GreetingState = {
  status: 'loading' | 'success' | 'error';
  message?: string;
};

type IntentFormState = {
  statement: string;
  context: string;
  expectedOutcome: string;
  collaborators: string;
};

type IntentSubmissionState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'success'; response: IntentResponse }
  | { status: 'error'; message: string };

const App = () => {
  const [greeting, setGreeting] = useState<GreetingState>({ status: 'loading' });
  const [intentForm, setIntentForm] = useState<IntentFormState>({
    statement: '',
    context: '',
    expectedOutcome: '',
    collaborators: '',
  });
  const [intentState, setIntentState] = useState<IntentSubmissionState>({ status: 'idle' });

  useEffect(() => {
    fetchGreeting()
      .then((message) => setGreeting({ status: 'success', message }))
      .catch((error) => {
        console.error('Failed to load greeting', error);
        setGreeting({ status: 'error' });
      });
  }, []);

  const updateField = (field: keyof IntentFormState) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const value = event.target.value;
    setIntentForm((previous) => ({ ...previous, [field]: value }));
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setIntentState({ status: 'submitting' });

    try {
      const collaborators = intentForm.collaborators
        .split(',')
        .map((value) => value.trim())
        .filter(Boolean);

      const response = await submitIntent({
        statement: intentForm.statement,
        context: intentForm.context,
        expectedOutcome: intentForm.expectedOutcome,
        collaborators,
      });

      setIntentState({ status: 'success', response });
      setIntentForm({ statement: '', context: '', expectedOutcome: '', collaborators: '' });
    } catch (error) {
      console.error('Failed to submit intent', error);
      const message = error instanceof Error ? error.message : 'Unable to submit intent';
      setIntentState({ status: 'error', message });
    }
  };

  return (
    <main style={styles.main}>
      <section style={styles.heroCard}>
        <h1>Intent Platform</h1>
        <p>Your hello-world workspace for future EPICs, FEATUREs, and USER STORIES.</p>
        {greeting.status === 'loading' && <p>Loading greeting…</p>}
        {greeting.status === 'success' && <p data-testid="greeting">{greeting.message}</p>}
        {greeting.status === 'error' && (
          <p role="alert">We could not reach the API. Check the backend logs.</p>
        )}
      </section>

      <section className="intent-card">
        <header>
          <h2>Declare your intent</h2>
          <p>
            Capture the statement, context, expected outcome, and who you need so your Chapter can inspect and align quickly.
          </p>
        </header>

        <form className="intent-form" onSubmit={handleSubmit}>
          <label>
            <span>“I intend to…”</span>
            <textarea
              value={intentForm.statement}
              onChange={updateField('statement')}
              placeholder="I intend to…"
              required
            />
          </label>

          <label>
            <span>Context</span>
            <textarea
              value={intentForm.context}
              onChange={updateField('context')}
              placeholder="What did you observe, learn, or decide that triggered this intent?"
              required
            />
          </label>

          <label>
            <span>Expected outcome</span>
            <textarea
              value={intentForm.expectedOutcome}
              onChange={updateField('expectedOutcome')}
              placeholder="What will be true when you are done?"
              required
            />
          </label>

          <label>
            <span>Needed collaborators (comma separated)</span>
            <input
              type="text"
              value={intentForm.collaborators}
              onChange={updateField('collaborators')}
              placeholder="Observability lead, Mobile guild, …"
            />
          </label>

          <button type="submit" disabled={intentState.status === 'submitting'}>
            {intentState.status === 'submitting' ? 'Submitting…' : 'Submit intent'}
          </button>
        </form>

        {intentState.status === 'error' && (
          <div role="alert" className="intent-alert intent-alert-error">
            <strong>We could not capture your intent.</strong>
            <p>{intentState.message}</p>
          </div>
        )}

        {intentState.status === 'success' && (
          <div className="intent-alert intent-alert-success" data-testid="intent-success">
            <strong>Intent captured!</strong>
            <p>
              Recorded <em>{intentState.response.statement}</em> at{' '}
              {new Date(intentState.response.createdAt).toLocaleString()}.
            </p>
            {intentState.response.collaborators.length > 0 && (
              <p>Collaborators: {intentState.response.collaborators.join(', ')}</p>
            )}
          </div>
        )}
      </section>
    </main>
  );
};

const styles: Record<string, CSSProperties> = {
  main: {
    fontFamily: 'system-ui, sans-serif',
    minHeight: '100vh',
    display: 'grid',
    gap: '2rem',
    padding: '4rem 1.5rem',
    justifyItems: 'center',
    background:
      'radial-gradient(circle at top left, rgba(79, 70, 229, 0.2), transparent 55%), radial-gradient(circle at bottom right, rgba(16, 185, 129, 0.2), transparent 55%)',
  },
  heroCard: {
    maxWidth: '640px',
    padding: '2.5rem',
    borderRadius: '1rem',
    boxShadow: '0 25px 50px -12px rgba(30, 64, 175, 0.25)',
    backgroundColor: 'rgba(255, 255, 255, 0.9)',
    textAlign: 'center',
  },
};

export default App;
