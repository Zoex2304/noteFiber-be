-- seed_fixed.sql
BEGIN;

-- Clear existing data
DELETE FROM public.chat_message_raw;
DELETE FROM public.chat_message;
DELETE FROM public.note_embedding;
DELETE FROM public.note;
DELETE FROM public.notebook;
DELETE FROM public.chat_session;
DELETE FROM public.ai_credit_transactions;
DELETE FROM public.email_verification_tokens;
DELETE FROM public.password_reset_tokens;
DELETE FROM public.user_providers;
DELETE FROM public.user_subscriptions;
DELETE FROM public.users;
DELETE FROM public.subscription_plans;

-- Insert subscription plans (ID akan auto-generate)
INSERT INTO public.subscription_plans (name, slug, description, price, billing_period, max_notes, semantic_search_enabled, ai_chat_enabled, ai_daily_credit_limit) VALUES
('Free Plan', 'free', 'Basic plan with limited features', 0.00, 'monthly', 50, false, false, 5),
('Pro Plan', 'pro', 'Advanced features for power users', 9.99, 'monthly', 500, true, true, 50),
('Team Plan', 'team', 'Collaboration features for teams', 19.99, 'monthly', 5000, true, true, 200);

-- Insert users (ID akan auto-generate)
INSERT INTO public.users (email, password_hash, full_name, role, status, email_verified, email_verified_at, ai_daily_usage, ai_daily_usage_last_reset) VALUES
('admin@example.com', '$2a$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36WQoeG6Lruj3vjPGga31lW', 'System Admin', 'admin', 'active', true, NOW(), 0, NOW()),
('john.doe@example.com', '$2a$10$Trnkv9zZq5BwL7ZQmJkFyuGcKbB4p6H8NvC3D2E1F0G9H8J7K6L5M4N3', 'John Doe', 'user', 'active', true, NOW(), 2, NOW()),
('jane.smith@example.com', '$2a$10$AbCdEfGhIjKlMnOpQrStUvWxYz0123456789ABCDEFGHIJKLMNOPQ', 'Jane Smith', 'user', 'active', true, NOW(), 5, NOW()),
('bob.wilson@example.com', '$2a$10$ZyXwVuTsRqPoNiUlTmReViShInGgFrAnCe1234567890ABCDEFGH', 'Bob Wilson', 'user', 'pending', false, NULL, 0, NOW());

-- Insert user subscriptions (link ke user dan plan)
INSERT INTO public.user_subscriptions (user_id, plan_id, status, current_period_start, current_period_end, payment_status)
SELECT 
  u.id,
  p.id,
  CASE 
    WHEN u.email = 'john.doe@example.com' THEN 'active'::public.subscription_status
    WHEN u.email = 'jane.smith@example.com' THEN 'active'::public.subscription_status
    ELSE 'inactive'::public.subscription_status
  END,
  NOW() - INTERVAL '15 days',
  NOW() + INTERVAL '15 days',
  CASE 
    WHEN u.email = 'john.doe@example.com' THEN 'success'::public.payment_status
    WHEN u.email = 'jane.smith@example.com' THEN 'success'::public.payment_status
    ELSE 'pending'::public.payment_status
  END
FROM public.users u
CROSS JOIN public.subscription_plans p
WHERE (u.email = 'john.doe@example.com' AND p.slug = 'pro')
   OR (u.email = 'jane.smith@example.com' AND p.slug = 'team')
   OR (u.email = 'bob.wilson@example.com' AND p.slug = 'free');

-- Insert notebooks (parent_id cukup NULL, user_id refer ke users)
INSERT INTO public.notebook (name, parent_id, user_id)
SELECT 'Personal Notes', NULL::uuid, id FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT 'Meeting Notes', NULL::uuid, id FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT 'Work Projects', NULL::uuid, id FROM public.users WHERE email = 'jane.smith@example.com'
UNION ALL
SELECT 'Research', NULL::uuid, id FROM public.users WHERE email = 'jane.smith@example.com';

-- Insert notes
INSERT INTO public.note (title, content, notebook_id, user_id)
SELECT 
  'Weekly Goals',
  '1. Complete the database schema
2. Write unit tests
3. Review PRs from team members',
  n.id,
  n.user_id
FROM public.notebook n 
WHERE n.name = 'Personal Notes' 
  AND EXISTS (SELECT 1 FROM public.users u WHERE u.id = n.user_id AND u.email = 'john.doe@example.com')
UNION ALL
SELECT 
  'Meeting Summary',
  'Meeting with design team:
- Discussed UI improvements
- Agreed on new color scheme
- Next meeting scheduled for Friday',
  n.id,
  n.user_id
FROM public.notebook n 
WHERE n.name = 'Meeting Notes' 
  AND EXISTS (SELECT 1 FROM public.users u WHERE u.id = n.user_id AND u.email = 'john.doe@example.com');

-- Insert chat sessions
INSERT INTO public.chat_session (title, user_id)
SELECT 'Database Design Help', id FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT 'Code Review Questions', id FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT 'Project Planning', id FROM public.users WHERE email = 'jane.smith@example.com'
UNION ALL
SELECT 'Market Research', id FROM public.users WHERE email = 'jane.smith@example.com';

-- Insert AI credit transactions
INSERT INTO public.ai_credit_transactions (user_id, transaction_type, amount, service_used, notes)
SELECT id, 'grant'::public.ai_credit_transaction_type, 100, 'welcome_bonus', 'Initial credit for new user' 
FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT id, 'spend'::public.ai_credit_transaction_type, 5, 'chat_completion', 'Used for AI chat assistance' 
FROM public.users WHERE email = 'john.doe@example.com'
UNION ALL
SELECT id, 'grant'::public.ai_credit_transaction_type, 200, 'subscription_bonus', 'Credits from Team plan subscription' 
FROM public.users WHERE email = 'jane.smith@example.com';

-- Insert system logs
INSERT INTO public.system_logs (level, module, message, details)
VALUES 
('info', 'seeder', 'Initial data seeding started', '{"rows_affected": 15}'::jsonb),
('info', 'seeder', 'Subscription plans created', '{"count": 3}'::jsonb),
('info', 'seeder', 'Users created', '{"count": 4}'::jsonb),
('success', 'seeder', 'Data seeding completed successfully', '{"total_records": 25}'::jsonb);

-- Insert email verification token for pending user
INSERT INTO public.email_verification_tokens (user_id, token, expires_at)
SELECT id, 'ABC123', NOW() + INTERVAL '24 hours'
FROM public.users WHERE email = 'bob.wilson@example.com';

COMMIT;

-- Verification
DO $$
BEGIN
  RAISE NOTICE '=== SEEDING COMPLETED ===';
  RAISE NOTICE 'Users: %', (SELECT COUNT(*) FROM public.users);
  RAISE NOTICE 'Subscription Plans: %', (SELECT COUNT(*) FROM public.subscription_plans);
  RAISE NOTICE 'User Subscriptions: %', (SELECT COUNT(*) FROM public.user_subscriptions);
  RAISE NOTICE 'Notebooks: %', (SELECT COUNT(*) FROM public.notebook);
  RAISE NOTICE 'Notes: %', (SELECT COUNT(*) FROM public.note);
  RAISE NOTICE 'Chat Sessions: %', (SELECT COUNT(*) FROM public.chat_session);
  RAISE NOTICE 'AI Credit Transactions: %', (SELECT COUNT(*) FROM public.ai_credit_transactions);
  RAISE NOTICE 'System Logs: %', (SELECT COUNT(*) FROM public.system_logs);
END $$;