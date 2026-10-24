ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allow_balance_subscription_purchase BOOLEAN NOT NULL DEFAULT FALSE;
