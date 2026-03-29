-- Conductor initial schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    goal TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE agent_runtime AS ENUM ('claude_code', 'openclaw');
CREATE TYPE agent_status AS ENUM ('idle', 'working', 'paused', 'blocked', 'failed', 'done', 'degraded');

CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    title TEXT NOT NULL,
    system_prompt TEXT NOT NULL DEFAULT '',
    manager_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    runtime agent_runtime NOT NULL DEFAULT 'claude_code',
    status agent_status NOT NULL DEFAULT 'idle',
    monthly_budget INTEGER NOT NULL DEFAULT 100000, -- tokens
    token_spend INTEGER NOT NULL DEFAULT 0,
    chat_token_spend INTEGER NOT NULL DEFAULT 0,
    pid INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE hire_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE pending_hires (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    requested_by_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    role_title TEXT NOT NULL,
    reporting_to_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    system_prompt TEXT NOT NULL DEFAULT '',
    runtime agent_runtime NOT NULL DEFAULT 'claude_code',
    budget_allocation INTEGER NOT NULL DEFAULT 50000,
    initial_task TEXT,
    status hire_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE issue_status AS ENUM ('pending', 'in_progress', 'blocked', 'done', 'failed');

CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    assignee_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    parent_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    status issue_status NOT NULL DEFAULT 'pending',
    output_path TEXT,
    created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_failure_reason TEXT,
    escalation_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE issue_dependencies (
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    depends_on_issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, depends_on_issue_id)
);

CREATE TYPE escalation_trigger AS ENUM ('explicit_failure', 'attempt_limit', 'budget_exhausted', 'dependency_timeout');
CREATE TYPE escalation_status AS ENUM ('open', 'resolved', 'escalated_to_human');

CREATE TABLE escalations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    current_assignee_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    escalation_chain JSONB NOT NULL DEFAULT '[]',
    trigger escalation_trigger NOT NULL,
    status escalation_status NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE issues ADD CONSTRAINT fk_issue_escalation
    FOREIGN KEY (escalation_id) REFERENCES escalations(id) ON DELETE SET NULL;

CREATE TABLE heartbeats (
    agent_id UUID PRIMARY KEY REFERENCES agents(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consecutive_misses INTEGER NOT NULL DEFAULT 0
);

CREATE TYPE permission_level AS ENUM ('read', 'write', 'admin');

CREATE TABLE fs_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    permission_level permission_level NOT NULL DEFAULT 'read',
    granted_by UUID REFERENCES agents(id) ON DELETE SET NULL,
    UNIQUE(agent_id, path)
);

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    actor_id UUID, -- NULL = system
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    issue_id UUID REFERENCES issues(id) ON DELETE SET NULL,
    messages JSONB NOT NULL DEFAULT '[]',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resumed_at TIMESTAMPTZ
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    escalation_id UUID REFERENCES escalations(id) ON DELETE CASCADE,
    payload JSONB NOT NULL DEFAULT '{}',
    dismissed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_companies (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'admin',
    PRIMARY KEY (user_id, company_id)
);

-- Indexes
CREATE INDEX idx_agents_company ON agents(company_id);
CREATE INDEX idx_agents_manager ON agents(manager_id);
CREATE INDEX idx_issues_company ON issues(company_id);
CREATE INDEX idx_issues_assignee ON issues(assignee_id);
CREATE INDEX idx_issues_parent ON issues(parent_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_audit_log_company ON audit_log(company_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX idx_notifications_company ON notifications(company_id);
CREATE INDEX idx_fs_permissions_agent ON fs_permissions(agent_id);
