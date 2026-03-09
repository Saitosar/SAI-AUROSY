CREATE TABLE IF NOT EXISTS workflow_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_run_tasks (
    workflow_run_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    step_index INTEGER NOT NULL,
    PRIMARY KEY (workflow_run_id, task_id),
    FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id)
);
