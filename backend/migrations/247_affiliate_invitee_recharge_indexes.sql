-- Invitee recharge totals are shown on the user affiliate page. Keep the
-- aggregation bounded to completed balance orders and make the lookup cheap.
CREATE INDEX IF NOT EXISTS idx_payment_orders_affiliate_recharge_lookup
    ON payment_orders (user_id, status, order_type, paid_at)
    WHERE status = 'COMPLETED' AND order_type = 'balance' AND paid_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_invitee_rebate_lookup
    ON user_affiliate_ledger (user_id, source_user_id, action)
    WHERE action = 'accrue';
