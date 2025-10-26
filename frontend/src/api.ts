import type { GoalResponse, IntentResponse } from './types';

export async function fetchGreeting(): Promise<string> {
  const response = await fetch('/api/hello');
  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  const body: { message: string } = await response.json();
  return body.message;
}

export type CreateIntentPayload = {
  statement: string;
  context: string;
  expectedOutcome: string;
  collaborators: string[];
};

export async function submitIntent(payload: CreateIntentPayload): Promise<IntentResponse> {
  const response = await fetch('/api/intents', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as IntentResponse;
}

export async function updateIntent(id: string, payload: CreateIntentPayload): Promise<IntentResponse> {
  const response = await fetch(`/api/intents/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as IntentResponse;
}

export async function deleteIntent(id: string): Promise<void> {
  const response = await fetch(`/api/intents/${id}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    try {
      const body = await response.json();
      if (typeof body?.error === 'string') {
        message = body.error;
      }
    } catch (error) {
      // Ignore JSON parsing issues and keep the default error message.
    }
    throw new Error(message);
  }
}

export type ListIntentParams = {
  page?: number;
  pageSize?: number;
  q?: string;
  collaborator?: string;
  createdAfter?: string;
  createdBefore?: string;
};

export type Pagination = {
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
};

export type ListIntentsResponse = {
  items: IntentResponse[];
  pagination: Pagination;
};

function buildQuery(params: ListIntentParams): string {
  const searchParams = new URLSearchParams();

  if (params.page && params.page > 0) {
    searchParams.set('page', params.page.toString());
  }
  if (params.pageSize && params.pageSize > 0) {
    searchParams.set('pageSize', params.pageSize.toString());
  }
  if (params.q) {
    searchParams.set('q', params.q);
  }
  if (params.collaborator) {
    searchParams.set('collaborator', params.collaborator);
  }
  if (params.createdAfter) {
    searchParams.set('createdAfter', params.createdAfter);
  }
  if (params.createdBefore) {
    searchParams.set('createdBefore', params.createdBefore);
  }

  const query = searchParams.toString();
  return query ? `?${query}` : '';
}

export async function listIntents(params: ListIntentParams = {}): Promise<ListIntentsResponse> {
  const response = await fetch(`/api/intents${buildQuery(params)}`);
  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as ListIntentsResponse;
}

export type GoalPayload = {
  title: string;
  clarityStatement: string;
  guardrails: string[];
  decisionRights: string[];
  constraints: string[];
  successCriteria: string[];
};

export type ListGoalsParams = {
  page?: number;
  pageSize?: number;
  q?: string;
  createdAfter?: string;
  createdBefore?: string;
};

export type ListGoalsResponse = {
  items: GoalResponse[];
  pagination: Pagination;
};

function buildGoalQuery(params: ListGoalsParams): string {
  const searchParams = new URLSearchParams();

  if (params.page && params.page > 0) {
    searchParams.set('page', params.page.toString());
  }
  if (params.pageSize && params.pageSize > 0) {
    searchParams.set('pageSize', params.pageSize.toString());
  }
  if (params.q) {
    searchParams.set('q', params.q);
  }
  if (params.createdAfter) {
    searchParams.set('createdAfter', params.createdAfter);
  }
  if (params.createdBefore) {
    searchParams.set('createdBefore', params.createdBefore);
  }

  const query = searchParams.toString();
  return query ? `?${query}` : '';
}

export async function createGoal(payload: GoalPayload): Promise<GoalResponse> {
  const response = await fetch('/api/goals', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as GoalResponse;
}

export async function updateGoal(id: string, payload: GoalPayload): Promise<GoalResponse> {
  const response = await fetch(`/api/goals/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as GoalResponse;
}

export async function deleteGoal(id: string): Promise<void> {
  const response = await fetch(`/api/goals/${id}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    try {
      const body = await response.json();
      if (typeof body?.error === 'string') {
        message = body.error;
      }
    } catch (error) {
      // Ignore JSON parsing issues and keep the default error message.
    }
    throw new Error(message);
  }
}

export async function listGoals(params: ListGoalsParams = {}): Promise<ListGoalsResponse> {
  const response = await fetch(`/api/goals${buildGoalQuery(params)}`);
  const body = await response.json();

  if (!response.ok) {
    const message = typeof body?.error === 'string' ? body.error : `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return body as ListGoalsResponse;
}

export type { IntentResponse, GoalResponse } from './types';
