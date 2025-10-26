import {
  ChangeEvent,
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  createGoal,
  updateGoal,
  deleteGoal,
  listGoals,
  GoalPayload,
  Pagination as PaginationInfo,
} from './api';
import type { GoalResponse } from './types';
import { formatDateInputValue, formatTimestamp, parseDateInputValue } from './date';

const initialFormState = {
  title: '',
  clarityStatement: '',
  guardrails: '',
  decisionRights: '',
  constraints: '',
  successCriteria: '',
};

const initialFilters = {
  q: '',
  createdAfter: '',
  createdBefore: '',
  page: 1,
  pageSize: 10,
};

type BannerState =
  | { type: 'success' | 'error'; title: string; description?: string }
  | null;

type ListState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready' }
  | { status: 'error'; message: string };

type GoalFormState = typeof initialFormState;

type GoalFiltersState = typeof initialFilters;

const parseListValues = (value: string): string[] =>
  value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);

const formatListValues = (values: string[]): string => values.join('\n');

const GoalsManager = () => {
  const [formState, setFormState] = useState<GoalFormState>(initialFormState);
  const [formMode, setFormMode] = useState<'create' | 'update'>('create');
  const [editingGoalId, setEditingGoalId] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [banner, setBanner] = useState<BannerState>(null);
  const [filters, setFilters] = useState<GoalFiltersState>(initialFilters);
  const [goals, setGoals] = useState<GoalResponse[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [listState, setListState] = useState<ListState>({ status: 'idle' });

  const loadGoals = useCallback(async () => {
    setListState({ status: 'loading' });
    try {
      const response = await listGoals({
        page: filters.page,
        pageSize: filters.pageSize,
        q: filters.q.trim() || undefined,
        createdAfter: filters.createdAfter || undefined,
        createdBefore: filters.createdBefore || undefined,
      });
      setGoals(response.items);
      setPagination(response.pagination);
      setListState({ status: 'ready' });
    } catch (error) {
      console.error('Failed to load goals', error);
      const message = error instanceof Error ? error.message : 'Unable to load goals';
      setListState({ status: 'error', message });
    }
  }, [filters]);

  useEffect(() => {
    void loadGoals();
  }, [loadGoals]);

  const updateField = (
    field: keyof GoalFormState,
  ) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const value = event.target.value;
    setFormState((previous) => ({ ...previous, [field]: value }));
  };

  const resetForm = () => {
    setFormState(initialFormState);
    setFormMode('create');
    setEditingGoalId(null);
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSubmitting(true);
    setBanner(null);

    try {
      const payload: GoalPayload = {
        title: formState.title.trim(),
        clarityStatement: formState.clarityStatement.trim(),
        guardrails: parseListValues(formState.guardrails),
        decisionRights: parseListValues(formState.decisionRights),
        constraints: parseListValues(formState.constraints),
        successCriteria: parseListValues(formState.successCriteria),
      };

      let response: GoalResponse;

      if (formMode === 'update' && editingGoalId) {
        response = await updateGoal(editingGoalId, payload);
        setFormState({
          title: response.title,
          clarityStatement: response.clarityStatement,
          guardrails: formatListValues(response.guardrails),
          decisionRights: formatListValues(response.decisionRights),
          constraints: formatListValues(response.constraints),
          successCriteria: formatListValues(response.successCriteria),
        });
        setBanner({
          type: 'success',
          title: 'Goal updated',
          description: `Saved changes to “${response.title}”.`,
        });
      } else {
        response = await createGoal(payload);
        setBanner({
          type: 'success',
          title: 'Goal created',
          description: `Defined “${response.title}” at ${formatTimestamp(response.createdAt)}.`,
        });
        resetForm();
      }

      await loadGoals();
    } catch (error) {
      console.error('Failed to submit goal', error);
      const message = error instanceof Error ? error.message : 'Unable to save goal';
      setBanner({ type: 'error', title: 'Unable to save goal', description: message });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleEdit = (goal: GoalResponse) => {
    setFormState({
      title: goal.title,
      clarityStatement: goal.clarityStatement,
      guardrails: formatListValues(goal.guardrails),
      decisionRights: formatListValues(goal.decisionRights),
      constraints: formatListValues(goal.constraints),
      successCriteria: formatListValues(goal.successCriteria),
    });
    setFormMode('update');
    setEditingGoalId(goal.id);
    setBanner(null);
  };

  const handleDelete = async (goal: GoalResponse) => {
    if (!window.confirm(`Delete goal “${goal.title}”? This cannot be undone.`)) {
      return;
    }

    try {
      await deleteGoal(goal.id);
      if (editingGoalId === goal.id) {
        resetForm();
      }
      setBanner({
        type: 'success',
        title: 'Goal deleted',
        description: `Removed “${goal.title}”.`,
      });
      await loadGoals();
    } catch (error) {
      console.error('Failed to delete goal', error);
      const message = error instanceof Error ? error.message : 'Unable to delete goal';
      setBanner({ type: 'error', title: 'Unable to delete goal', description: message });
    }
  };

  const updateFilterField = (field: 'q') => (event: ChangeEvent<HTMLInputElement>) => {
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
    () => filters.q.trim() !== '' || filters.createdAfter !== '' || filters.createdBefore !== '',
    [filters],
  );

  const currentPage = pagination?.page ?? filters.page;
  const totalPages = pagination?.totalPages ?? 0;
  const totalItems = pagination?.totalItems ?? 0;
  const pageSize = pagination?.pageSize ?? filters.pageSize;
  const startItem = totalItems === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const endItem = totalItems === 0 ? 0 : Math.min(currentPage * pageSize, totalItems);

  return (
    <section className="intent-goals">
      <section className="intent-card">
        <header>
          <h2>{formMode === 'update' ? 'Update goal' : 'Define a chapter goal'}</h2>
          <p>
            Capture the clarity statement, guardrails, decision rights, constraints, and success criteria that guide intents for
            upcoming chapter sessions.
          </p>
        </header>

        <form className="intent-form" onSubmit={handleSubmit}>
          <label>
            <span>Goal title</span>
            <input
              value={formState.title}
              onChange={updateField('title')}
              placeholder="Increase onboarding throughput"
              required
            />
          </label>

          <label>
            <span>Clarity statement</span>
            <textarea
              value={formState.clarityStatement}
              onChange={updateField('clarityStatement')}
              placeholder="Why this goal matters, the scope, and key outcomes"
              required
            />
          </label>

          <label>
            <span>Guardrails (one per line or comma separated)</span>
            <textarea
              value={formState.guardrails}
              onChange={updateField('guardrails')}
              placeholder="Protect member focus time\nShare updates async"
            />
          </label>

          <label>
            <span>Decision rights (one per line or comma separated)</span>
            <textarea
              value={formState.decisionRights}
              onChange={updateField('decisionRights')}
              placeholder="Teams may ship with feature flags\nEngineers can pick tooling"
            />
          </label>

          <label>
            <span>Constraints (one per line or comma separated)</span>
            <textarea
              value={formState.constraints}
              onChange={updateField('constraints')}
              placeholder="Guardrail A\nGuardrail B"
            />
          </label>

          <label>
            <span>Success criteria (one per line or comma separated)</span>
            <textarea
              value={formState.successCriteria}
              onChange={updateField('successCriteria')}
              placeholder="Measure that signals the goal is achieved"
            />
          </label>

          <div className="intent-form-actions">
            <button type="submit" className="button-primary" disabled={isSubmitting}>
              {formMode === 'update' ? 'Save goal changes' : 'Create goal'}
            </button>
            {formMode === 'update' && (
              <button
                type="button"
                className="button-secondary"
                onClick={resetForm}
                disabled={isSubmitting}
              >
                Cancel edit
              </button>
            )}
            {isSubmitting && <span>Saving…</span>}
          </div>
        </form>

        {banner && (
          <div className={`intent-alert ${banner.type === 'success' ? 'intent-alert-success' : 'intent-alert-error'}`}>
            <div className="intent-banner">
              <strong>{banner.title}</strong>
              {banner.description && <p>{banner.description}</p>}
            </div>
          </div>
        )}
      </section>

      <section className="intent-card intent-list-card">
        <header>
          <h2>Goals catalog</h2>
          <p>Review, filter, and align on the chapter goals currently guiding intents.</p>
        </header>

        <div className="intent-filters">
          <div className="intent-filters-grid">
            <label>
              <span>Search</span>
              <input
                type="search"
                value={filters.q}
                onChange={updateFilterField('q')}
                placeholder="Search titles and clarity statements"
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
            <label>
              <span>Page size</span>
              <select value={filters.pageSize} onChange={handlePageSizeChange}>
                {[5, 10, 20, 50].map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="intent-filters-actions">
            <div>
              {hasActiveFilters ? 'Filters active' : 'No filters applied'}
            </div>
            <button type="button" className="button-secondary" onClick={clearFilters} disabled={!hasActiveFilters}>
              Clear filters
            </button>
          </div>
        </div>

        {listState.status === 'error' && (
          <div className="intent-alert intent-alert-error">
            <div className="intent-banner">
              <strong>Unable to load goals.</strong>
              <p>{listState.message}</p>
              <button type="button" className="button-secondary" onClick={() => { void loadGoals(); }}>
                Try again
              </button>
            </div>
          </div>
        )}

        {listState.status === 'loading' && goals.length === 0 && (
          <p className="intent-list-placeholder">Loading goals…</p>
        )}

        {listState.status === 'ready' && goals.length === 0 && (
          <p className="intent-list-placeholder">No goals found. Adjust filters or define a new goal above.</p>
        )}

        {goals.length > 0 && (
          <div className="intent-table-wrapper">
            <table className="intent-table">
              <thead>
                <tr>
                  <th>Title</th>
                  <th>Clarity statement</th>
                  <th>Guardrails</th>
                  <th>Decision rights</th>
                  <th>Constraints</th>
                  <th>Success criteria</th>
                  <th>Created</th>
                  <th>Updated</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {goals.map((goal) => (
                  <tr key={goal.id} className={editingGoalId === goal.id ? 'intent-table-row--active' : undefined}>
                    <td>
                      <strong>{goal.title}</strong>
                    </td>
                    <td>{goal.clarityStatement}</td>
                    <td>
                      {goal.guardrails.length === 0 ? (
                        <span className="intent-table-empty">—</span>
                      ) : (
                        <div className="intent-table-collaborators">
                          {goal.guardrails.map((guardrail) => (
                            <span key={guardrail} className="intent-collaborator-badge">
                              {guardrail}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>
                      {goal.decisionRights.length === 0 ? (
                        <span className="intent-table-empty">—</span>
                      ) : (
                        <div className="intent-table-collaborators">
                          {goal.decisionRights.map((decisionRight) => (
                            <span key={decisionRight} className="intent-collaborator-badge">
                              {decisionRight}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>
                      {goal.constraints.length === 0 ? (
                        <span className="intent-table-empty">—</span>
                      ) : (
                        <div className="intent-table-collaborators">
                          {goal.constraints.map((constraint) => (
                            <span key={constraint} className="intent-collaborator-badge">
                              {constraint}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>
                      {goal.successCriteria.length === 0 ? (
                        <span className="intent-table-empty">—</span>
                      ) : (
                        <div className="intent-table-collaborators">
                          {goal.successCriteria.map((criterion) => (
                            <span key={criterion} className="intent-collaborator-badge">
                              {criterion}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>{formatTimestamp(goal.createdAt)}</td>
                    <td>{formatTimestamp(goal.updatedAt)}</td>
                    <td>
                      <div className="intent-table-actions">
                        <button type="button" onClick={() => handleEdit(goal)}>
                          Edit
                        </button>
                        <button type="button" className="danger" onClick={() => handleDelete(goal)}>
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
              {totalItems === 0 ? 'No goals' : `Showing ${startItem}–${endItem} of ${totalItems} goals`}
            </div>
            <div className="intent-pagination__controls">
              <button type="button" onClick={() => goToPage(Math.max(1, currentPage - 1))} disabled={currentPage <= 1}>
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

        {listState.status === 'loading' && goals.length > 0 && <p className="intent-list-loading-inline">Refreshing…</p>}
      </section>
    </section>
  );
};

export default GoalsManager;
