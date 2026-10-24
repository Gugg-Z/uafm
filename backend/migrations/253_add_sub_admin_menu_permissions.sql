-- Add menu-level read-only permissions for delegated administrators (sub_admin).
-- Ported from 58token migration 153.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS admin_menu_permissions TEXT NOT NULL DEFAULT '[]';
