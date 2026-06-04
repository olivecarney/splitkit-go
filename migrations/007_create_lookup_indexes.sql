CREATE INDEX idx_group_members_group_id_created_at
    ON group_members (group_id, created_at);

CREATE INDEX idx_expenses_group_id_created_at
    ON expenses (group_id, created_at DESC);

CREATE INDEX idx_expense_splits_user_id
    ON expense_splits (user_id);

CREATE INDEX idx_settlements_group_id_created_at
    ON settlements (group_id, created_at DESC);
