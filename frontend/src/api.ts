export async function fetchGreeting(): Promise<string> {
  const response = await fetch('/api/hello');
  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  const body: { message: string } = await response.json();
  return body.message;
}
