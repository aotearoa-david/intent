export type IntentResponse = {
  id: string;
  statement: string;
  context: string;
  expectedOutcome: string;
  collaborators: string[];
  createdAt: string;
};

export type GoalResponse = {
  id: string;
  title: string;
  clarityStatement: string;
  constraints: string[];
  successCriteria: string[];
  createdAt: string;
  updatedAt: string;
};
