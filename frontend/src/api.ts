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

export type IntentResponse = {
  id: string;
  statement: string;
  context: string;
  expectedOutcome: string;
  collaborators: string[];
  createdAt: string;
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
