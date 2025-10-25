export type IntentResponse = {
  id: string;
  statement: string;
  context: string;
  expectedOutcome: string;
  collaborators: string[];
  createdAt: string;
};
