-- Seed a "Board" agent for each company that doesn't have one yet.
-- The Board represents the human operator and appears in all agent selectors.
INSERT INTO agents (company_id, role, title, system_prompt, manager_id, runtime, monthly_budget)
SELECT id, 'board', 'Board', 'You are the Board. You represent the human operator.', NULL, 'claude_code', 2147483647
FROM companies
WHERE id NOT IN (
    SELECT company_id FROM agents WHERE role = 'board'
);
