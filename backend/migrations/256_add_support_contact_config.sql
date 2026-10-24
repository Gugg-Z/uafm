-- Seed support configuration without changing an installation's menu choices.
INSERT INTO settings (key, value, updated_at)
VALUES ('support_contact_config', '{"enabled":true,"title":"","description":"","contacts":[]}', NOW())
ON CONFLICT (key) DO NOTHING;
