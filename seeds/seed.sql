-- ========== EXTENSIONS ==========
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========== SUBSCRIPTION PLANS (UUID STABIL) ==========
INSERT INTO public.subscription_plans 
(
    id, 
    name, 
    slug, 
    description, 
    price, 
    billing_period, 
    max_notes, 
    semantic_search_enabled, 
    ai_chat_enabled, 
    ai_daily_credit_limit
)
VALUES
(
    '05c035ac-2117-40ec-b38f-ab6b912fa114',
    'Free Plan',
    'free',
    'Basic note taking features',
    0,
    'monthly'::public.billing_period,
    50,
    false,
    false,
    0
),
(
    'ce8abb84-10e9-4f54-9f7c-4be693b18ea5',
    'Pro Plan',
    'pro',
    'Unlock AI Chat and Semantic Search',
    50000.00,
    'monthly'::public.billing_period,
    1000,
    true,
    true,
    50
),
(
    '6e45d34e-0664-46b3-bbf3-309995c9a723',
    'Enterprise Plan',
    'enterprise',
    'Unlimited power for power users',
    500000.00,
    'yearly'::public.billing_period,
    999999,
    true,
    true,
    1000
)
ON CONFLICT (slug) DO NOTHING;

-- ========== USER ZIKRI ==========
INSERT INTO public.users 
(
    email, 
    password_hash, 
    full_name, 
    role, 
    status, 
    email_verified, 
    email_verified_at,
    created_at, 
    updated_at, 
    ai_daily_usage,
    ai_daily_usage_last_reset
)
VALUES
(
    'blueleather11@gmail.com', 
    crypt('zikri234', gen_salt('bf')), 
    'Zikri', 
    'user'::public.user_role, 
    'active'::public.user_status, 
    true,
    NOW(), 
    NOW(), 
    NOW(), 
    0,
    NOW()
)
ON CONFLICT (email) DO NOTHING;

-- ========== USER → FREE PLAN ==========
WITH zikri_user AS (
    SELECT id 
    FROM public.users 
    WHERE email = 'blueleather11@gmail.com'
),
free_plan AS (
    SELECT id 
    FROM public.subscription_plans 
    WHERE slug = 'free'
)
INSERT INTO public.user_subscriptions
(
    user_id,
    plan_id,
    status,
    current_period_start,
    current_period_end,
    payment_status
)
SELECT 
    zikri_user.id,
    free_plan.id,
    'active'::public.subscription_status,
    NOW(),
    NOW() + INTERVAL '1 month',
    'success'::public.payment_status
FROM zikri_user, free_plan
WHERE EXISTS (SELECT 1 FROM zikri_user) 
  AND EXISTS (SELECT 1 FROM free_plan)
ON CONFLICT DO NOTHING;
