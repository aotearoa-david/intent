import {
  CSSProperties,
  ChangeEvent,
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  fetchGreeting,
  submitIntent,
  updateIntent,
  deleteIntent,
  listIntents,
  CreateIntentPayload,
  Pagination as PaginationInfo,
} from './api';
import type { IntentResponse } from './types';
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

type FiltersState = {
  q: string;
  collaborator: string;
  createdAfter: string;
  createdBefore: string;
  page: number;
  pageSize: number;
};

type BannerState =
  | { type: 'success' | 'error'; title: string; description?: string }
  | null;

type ListState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready' }
  | { status: 'error'; message: string };

const initialFormState: IntentFormState = {
  statement: '',
  context: '',
  expectedOutcome: '',
  collaborators: '',
};

const initialFilters: FiltersState = {
  q: '',
  collaborator: '',
  createdAfter: '',
  createdBefore: '',
  page: 1,
  pageSize: 10,
};

const formatDateInputValue = (value: string): string => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }

  const pad = (input: number) => input.toString().padStart(2, '0');
  const year = date.getFullYear();
  const month = pad(date.getMonth() + 1);
  const day = pad(date.getDate());
  const hours = pad(date.getHours());
  const minutes = pad(date.getMinutes());
  return `${year}-${month}-${day}T${hours}:${minutes}`;
};

const parseDateInputValue = (value: string): string => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString();
};

const formatTimestamp = (value: string): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const App = () => {
  const [greeting, setGreeting] = useState<GreetingState>({ status: 'loading' });
  const [intentForm, setIntentForm] = useState<IntentFormState>(initialFormState);
  const [formMode, setFormMode] = useState<'create' | 'update'>('create');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [editingIntentId, setEditingIntentId] = useState<string | null>(null);
  const [banner, setBanner] = useState<BannerState>(null);
  const [filters, setFilters] = useState<FiltersState>(initialFilters);
  const [intents, setIntents] = useState<IntentResponse[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [listState, setListState] = useState<ListState>({ status: 'idle' });

  useEffect(() => {
    fetchGreeting()
      .then((message) => setGreeting({ status: 'success', message }))
      .catch((error) => {
        console.error('Failed to load greeting', error);
        setGreeting({ status: 'error' });
      });
  }, []);

  const loadIntents = useCallback(async (): Promise<void> => {
    setListState({ status: 'loading' });
    try {
      const response = await listIntents({
        page: filters.page,
        pageSize: filters.pageSize,
        q: filters.q.trim() || undefined,
        collaborator: filters.collaborator.trim() || undefined,
        createdAfter: filters.createdAfter || undefined,
        createdBefore: filters.createdBefore || undefined,
      });
      setIntents(response.items);
      setPagination(response.pagination);
      setListState({ status: 'ready' });
    } catch (error) {
      console.error('Failed to load intents', error);
      const message = error instanceof Error ? error.message : 'Unable to load intents';
      setListState({ status: 'error', message });
    }
  }, [filters]);

  useEffect(() => {
    void loadIntents();
  }, [loadIntents]);

  const updateField = (
    field: keyof IntentFormState,
  ) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const value = event.target.value;
    setIntentForm((previous) => ({ ...previous, [field]: value }));
  };

  const resetForm = () => {
    setIntentForm(initialFormState);
    setFormMode('create');
    setEditingIntentId(null);
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSubmitting(true);
    setBanner(null);

    try {
      const collaborators = intentForm.collaborators
        .split(',')
        .map((value) => value.trim())
        .filter(Boolean);

      const payload: CreateIntentPayload = {
        statement: intentForm.statement,
        context: intentForm.context,
        expectedOutcome: intentForm.expectedOutcome,
        collaborators,
      };

      let response: IntentResponse;

      if (formMode === 'update' && editingIntentId) {
        response = await updateIntent(editingIntentId, payload);
        setBanner({
          type: 'success',
          title: 'Intent updated',
          description: `Saved changes to “${response.statement}”.`,
        });
      } else {
        response = await submitIntent(payload);
        setBanner({
          type: 'success',
          title: 'Intent captured',
          description: `Recorded “${response.statement}” at ${formatTimestamp(response.createdAt)}.`,
        });
      }

      resetForm();
      await loadIntents();
    } catch (error) {
      console.error('Failed to submit intent', error);
      const message = error instanceof Error ? error.message : 'Unable to submit intent';
      setBanner({ type: 'error', title: 'Unable to save intent', description: message });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleEdit = (intent: IntentResponse) => {
    setEditingIntentId(intent.id);
    setFormMode('update');
    setIntentForm({
      statement: intent.statement,
      context: intent.context,
      expectedOutcome: intent.expectedOutcome,
      collaborators: intent.collaborators.join(', '),
    });
    setBanner(null);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleCancelEdit = () => {
    resetForm();
    setBanner(null);
  };

  const handleDelete = async (intent: IntentResponse) => {
    const confirmed = window.confirm(`Delete the intent “${intent.statement}”?`);
    if (!confirmed) {
      return;
    }

    setBanner(null);
    try {
      await deleteIntent(intent.id);
      if (editingIntentId === intent.id) {
        resetForm();
      }
      setBanner({
        type: 'success',
        title: 'Intent deleted',
        description: `Removed “${intent.statement}”.`,
      });
      await loadIntents();
    } catch (error) {
      console.error('Failed to delete intent', error);
      const message = error instanceof Error ? error.message : 'Unable to delete intent';
      setBanner({ type: 'error', title: 'Unable to delete intent', description: message });
    }
  };

  const updateFilterField = (field: 'q' | 'collaborator') =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.target.value;
      setFilters((previous) => ({ ...previous, [field]: value, page: 1 }));
    };

  const updateDateFilter = (field: 'createdAfter' | 'createdBefore') =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const isoValue = parseDateInputValue(event.target.value);
      setFilters((previous) => ({ ...previous, [field]: isoValue, page: 1 }));
    };

  const handlePageSizeChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const value = Number(event.target.value);
    setFilters((previous) => ({ ...previous, pageSize: Number.isNaN(value) ? previous.pageSize : value, page: 1 }));
  };

  const goToPage = (page: number) => {
    if (page < 1) {
      return;
    }
    setFilters((previous) => ({ ...previous, page }));
  };

  const clearFilters = () => {
    setFilters((previous) => ({ ...initialFilters, pageSize: previous.pageSize }));
  };

  const hasActiveFilters = useMemo(
    () =>
      filters.q.trim() !== '' ||
      filters.collaborator.trim() !== '' ||
      filters.createdAfter !== '' ||
      filters.createdBefore !== '',
    [filters],
  );

  const currentPage = pagination?.page ?? filters.page;
  const totalPages = pagination?.totalPages ?? 0;
  const totalItems = pagination?.totalItems ?? 0;
  const pageSize = pagination?.pageSize ?? filters.pageSize;
  const startItem = totalItems === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const endItem = totalItems === 0 ? 0 : Math.min(currentPage * pageSize, totalItems);

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
          <h2>{formMode === 'update' ? 'Update intent' : 'Declare your intent'}</h2>
          <p>
            Capture the statement, context, expected outcome, and who you need so your Chapter can inspect and align
            quickly.
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

          <div className="intent-form-actions">
            <button type="submit" className="button-primary" disabled={isSubmitting}>
              {isSubmitting ? 'Saving…' : formMode === 'update' ? 'Save changes' : 'Submit intent'}
            </button>
            {formMode === 'update' && (
              <button type="button" className="button-secondary" onClick={handleCancelEdit} disabled={isSubmitting}>
                Cancel
              </button>
            )}
          </div>
        </form>

        {banner && (
          <div
            role="status"
            className={`intent-alert ${banner.type === 'success' ? 'intent-alert-success' : 'intent-alert-error'} intent-banner`}
          >
            <strong>{banner.title}</strong>
            {banner.description && <p>{banner.description}</p>}
          </div>
        )}
      </section>

      <section className="intent-card intent-list-card">
        <header>
          <h2>Intents</h2>
          <p>Inspect, filter, and manage intents captured by your team.</p>
        </header>

        <div className="intent-filters">
          <div className="intent-filters-grid">
            <label>
              <span>Search</span>
              <input
                type="search"
                value={filters.q}
                onChange={updateFilterField('q')}
                placeholder="Search statement, context, or outcome"
              />
            </label>

            <label>
              <span>Collaborator</span>
              <input
                type="search"
                value={filters.collaborator}
                onChange={updateFilterField('collaborator')}
                placeholder="Filter by collaborator"
              />
            </label>

            <label>
              <span>Created after</span>
              <input
                type="datetime-local"
                value={formatDateInputValue(filters.createdAfter)}
                onChange={updateDateFilter('createdAfter')}
              />
            </label>

            <label>
              <span>Created before</span>
              <input
                type="datetime-local"
                value={formatDateInputValue(filters.createdBefore)}
                onChange={updateDateFilter('createdBefore')}
              />
            </label>
          </div>

          <div className="intent-filters-actions">
            <label>
              <span>Page size</span>
              <select value={filters.pageSize} onChange={handlePageSizeChange}>
                {[5, 10, 20, 50].map((size) => (
                  <option key={size} value={size}>
                    {size} per page
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className="button-secondary"
              onClick={clearFilters}
              disabled={!hasActiveFilters}
            >
              Clear filters
            </button>
          </div>
        </div>

        {listState.status === 'error' && (
          <div className="intent-alert intent-alert-error intent-banner">
            <strong>Unable to load intents.</strong>
            <p>{listState.message}</p>
            <button type="button" className="button-secondary" onClick={() => { void loadIntents(); }}>
              Try again
            </button>
          </div>
        )}

        {listState.status === 'loading' && intents.length === 0 && (
          <p className="intent-list-placeholder">Loading intents…</p>
        )}

        {listState.status === 'ready' && intents.length === 0 && (
          <p className="intent-list-placeholder">No intents found. Adjust filters or create a new intent above.</p>
        )}

        {intents.length > 0 && (
          <div className="intent-table-wrapper">
            <table className="intent-table">
              <thead>
                <tr>
                  <th>Statement</th>
                  <th>Context</th>
                  <th>Expected outcome</th>
                  <th>Collaborators</th>
                  <th>Created</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {intents.map((intent) => (
                  <tr key={intent.id} className={editingIntentId === intent.id ? 'intent-table-row--active' : undefined}>
                    <td>
                      <strong>{intent.statement}</strong>
                    </td>
                    <td>{intent.context}</td>
                    <td>{intent.expectedOutcome}</td>
                    <td>
                      {intent.collaborators.length === 0 ? (
                        <span className="intent-table-empty">—</span>
                      ) : (
                        <div className="intent-table-collaborators">
                          {intent.collaborators.map((collaborator) => (
                            <span key={collaborator} className="intent-collaborator-badge">
                              {collaborator}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>{formatTimestamp(intent.createdAt)}</td>
                    <td>
                      <div className="intent-table-actions">
                        <button type="button" onClick={() => handleEdit(intent)}>
                          Edit
                        </button>
                        <button type="button" className="danger" onClick={() => handleDelete(intent)}>
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {pagination && pagination.totalPages > 0 && (
          <div className="intent-pagination">
            <div className="intent-pagination__summary">
              {totalItems === 0 ? 'No intents' : `Showing ${startItem}–${endItem} of ${totalItems} intents`}
            </div>
            <div className="intent-pagination__controls">
              <button
                type="button"
                onClick={() => goToPage(Math.max(1, currentPage - 1))}
                disabled={currentPage <= 1}
              >
                Previous
              </button>
              <span>
                Page {currentPage} of {totalPages || 1}
              </span>
              <button
                type="button"
                onClick={() => goToPage(totalPages === 0 ? 1 : Math.min(totalPages, currentPage + 1))}
                disabled={totalPages === 0 || currentPage >= totalPages}
              >
                Next
              </button>
            </div>
          </div>
        )}

        {listState.status === 'loading' && intents.length > 0 && <p className="intent-list-loading-inline">Refreshing…</p>}
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
