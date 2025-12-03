--
-- PostgreSQL database dump
--

\restrict EZWaoUsDfCBsZYtCDfefSmmCzP5349VB5gN3yCKDyN4zpAKDFhg8cRYQq2TObVq

-- Dumped from database version 17.6
-- Dumped by pg_dump version 17.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO postgres;

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA public IS '';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: ai_credit_transaction_type; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.ai_credit_transaction_type AS ENUM (
    'grant',
    'spend',
    'refund',
    'adjustment'
);


ALTER TYPE public.ai_credit_transaction_type OWNER TO postgres;

--
-- Name: billing_period; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.billing_period AS ENUM (
    'monthly',
    'yearly'
);


ALTER TYPE public.billing_period OWNER TO postgres;

--
-- Name: payment_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.payment_status AS ENUM (
    'pending',
    'success',
    'failed',
    'refunded'
);


ALTER TYPE public.payment_status OWNER TO postgres;

--
-- Name: subscription_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.subscription_status AS ENUM (
    'active',
    'inactive',
    'canceled',
    'trial'
);


ALTER TYPE public.subscription_status OWNER TO postgres;

--
-- Name: user_role; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.user_role AS ENUM (
    'user',
    'admin'
);


ALTER TYPE public.user_role OWNER TO postgres;

--
-- Name: user_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.user_status AS ENUM (
    'pending',
    'active',
    'suspended',
    'deleted'
);


ALTER TYPE public.user_status OWNER TO postgres;

--
-- Name: get_current_user_id(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_current_user_id() RETURNS uuid
    LANGUAGE plpgsql STABLE
    AS $$
BEGIN
    -- TODO: Ganti dengan logic membaca JWT/Session di production
    -- Untuk testing, return ID admin
    RETURN '00000000-0000-0000-0000-000000000001'::uuid;
END;
$$;


ALTER FUNCTION public.get_current_user_id() OWNER TO postgres;

--
-- Name: get_current_user_role(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_current_user_role() RETURNS public.user_role
    LANGUAGE plpgsql STABLE
    AS $$
BEGIN
    -- TODO: Ganti dengan logic membaca JWT/Session di production
    RETURN 'admin'::public.user_role;
END;
$$;


ALTER FUNCTION public.get_current_user_role() OWNER TO postgres;

--
-- Name: set_current_timestamp_updated_at(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.set_current_timestamp_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
  _new_value TIMESTAMP WITH TIME ZONE;
BEGIN
  _new_value := now();
  IF NEW.updated_at IS DISTINCT FROM _new_value THEN
    NEW.updated_at = _new_value;
  END IF;
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.set_current_timestamp_updated_at() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: subscription_plans; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.subscription_plans (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    description text,
    price numeric(10,2) NOT NULL,
    billing_period public.billing_period,
    max_notes integer,
    semantic_search_enabled boolean DEFAULT false NOT NULL,
    ai_chat_enabled boolean DEFAULT false NOT NULL,
    ai_daily_credit_limit integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.subscription_plans OWNER TO postgres;

--
-- Name: user_subscriptions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    status public.subscription_status DEFAULT 'inactive'::public.subscription_status NOT NULL,
    current_period_start timestamp with time zone NOT NULL,
    current_period_end timestamp with time zone NOT NULL,
    payment_status public.payment_status DEFAULT 'pending'::public.payment_status NOT NULL,
    midtrans_transaction_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_subscriptions OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email text NOT NULL,
    password_hash text,
    full_name text NOT NULL,
    role public.user_role DEFAULT 'user'::public.user_role NOT NULL,
    status public.user_status DEFAULT 'pending'::public.user_status NOT NULL,
    email_verified boolean DEFAULT false NOT NULL,
    email_verified_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ai_daily_usage integer DEFAULT 0 NOT NULL,
    ai_daily_usage_last_reset timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: admin_payment_audit_view; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.admin_payment_audit_view AS
 SELECT us.id AS subscription_id,
    us.midtrans_transaction_id,
    u.full_name AS user_name,
    u.email AS user_email,
    sp.name AS plan_name,
    us.payment_status,
    sp.price,
    us.current_period_start,
    us.current_period_end,
    us.created_at AS transaction_date
   FROM ((public.user_subscriptions us
     JOIN public.users u ON ((us.user_id = u.id)))
     JOIN public.subscription_plans sp ON ((us.plan_id = sp.id)))
  ORDER BY us.created_at DESC
  WITH NO DATA;


ALTER MATERIALIZED VIEW public.admin_payment_audit_view OWNER TO postgres;

--
-- Name: chat_session; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chat_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL,
    user_id uuid DEFAULT '00000000-0000-0000-0000-000000000001'::uuid NOT NULL
);


ALTER TABLE public.chat_session OWNER TO postgres;

--
-- Name: note; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.note (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title character varying NOT NULL,
    content character varying NOT NULL,
    notebook_id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL,
    user_id uuid DEFAULT '00000000-0000-0000-0000-000000000001'::uuid NOT NULL
);


ALTER TABLE public.note OWNER TO postgres;

--
-- Name: admin_user_performance_summary; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.admin_user_performance_summary AS
 SELECT u.id AS user_id,
    u.email,
    u.full_name,
    u.status,
    u.created_at AS join_date,
    ( SELECT count(n.id) AS count
           FROM public.note n
          WHERE (n.user_id = u.id)) AS total_notes,
    ( SELECT count(cs.id) AS count
           FROM public.chat_session cs
          WHERE (cs.user_id = u.id)) AS total_chats,
    u.ai_daily_usage,
    COALESCE(sub.status, 'inactive'::public.subscription_status) AS subscription_status,
    COALESCE(sp.name, 'N/A'::text) AS current_plan_name
   FROM ((public.users u
     LEFT JOIN public.user_subscriptions sub ON (((u.id = sub.user_id) AND (sub.status = 'active'::public.subscription_status))))
     LEFT JOIN public.subscription_plans sp ON ((sub.plan_id = sp.id)))
  WHERE (u.role = 'user'::public.user_role)
  ORDER BY u.created_at DESC
  WITH NO DATA;


ALTER MATERIALIZED VIEW public.admin_user_performance_summary OWNER TO postgres;

--
-- Name: ai_credit_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ai_credit_transactions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    transaction_type public.ai_credit_transaction_type NOT NULL,
    amount integer NOT NULL,
    service_used text,
    related_id uuid,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.ai_credit_transactions OWNER TO postgres;

--
-- Name: chat_message; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chat_message (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role character varying NOT NULL,
    chat character varying NOT NULL,
    chat_session_id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL
);


ALTER TABLE public.chat_message OWNER TO postgres;

--
-- Name: chat_message_raw; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chat_message_raw (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role character varying NOT NULL,
    chat character varying NOT NULL,
    chat_session_id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL
);


ALTER TABLE public.chat_message_raw OWNER TO postgres;

--
-- Name: note_embedding; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.note_embedding (
    id uuid NOT NULL,
    document character varying NOT NULL,
    embedding_value public.vector(3072) NOT NULL,
    note_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false
);


ALTER TABLE public.note_embedding OWNER TO postgres;

--
-- Name: notebook; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.notebook (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying NOT NULL,
    parent_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL,
    user_id uuid DEFAULT '00000000-0000-0000-0000-000000000001'::uuid NOT NULL
);


ALTER TABLE public.notebook OWNER TO postgres;

--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.password_reset_tokens (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.password_reset_tokens OWNER TO postgres;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: semantic_searchable_notes; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.semantic_searchable_notes AS
 SELECT n.id AS note_id,
    n.title,
    n.content,
    ne.embedding_value AS embedding,
    n.user_id
   FROM (public.note n
     JOIN public.note_embedding ne ON ((n.id = ne.note_id)))
  WHERE (n.is_deleted = false);


ALTER VIEW public.semantic_searchable_notes OWNER TO postgres;

--
-- Name: user_payment_history; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.user_payment_history AS
 SELECT us.user_id,
    u.full_name,
    sp.name AS plan_name,
    sp.price,
    us.payment_status,
    us.midtrans_transaction_id,
    us.created_at AS payment_date
   FROM ((public.user_subscriptions us
     JOIN public.users u ON ((us.user_id = u.id)))
     JOIN public.subscription_plans sp ON ((us.plan_id = sp.id)))
  ORDER BY us.created_at DESC;


ALTER VIEW public.user_payment_history OWNER TO postgres;

--
-- Name: user_providers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    provider_name text NOT NULL,
    provider_user_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_providers OWNER TO postgres;

--
-- Name: ai_credit_transactions ai_credit_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ai_credit_transactions
    ADD CONSTRAINT ai_credit_transactions_pkey PRIMARY KEY (id);


--
-- Name: chat_message chat_message_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_message
    ADD CONSTRAINT chat_message_pkey PRIMARY KEY (id);


--
-- Name: chat_message_raw chat_message_raw_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_message_raw
    ADD CONSTRAINT chat_message_raw_pkey PRIMARY KEY (id);


--
-- Name: chat_session chat_session_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_pkey PRIMARY KEY (id);


--
-- Name: note_embedding note_embedding_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.note_embedding
    ADD CONSTRAINT note_embedding_pkey PRIMARY KEY (id);


--
-- Name: note note_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.note
    ADD CONSTRAINT note_pkey PRIMARY KEY (id);


--
-- Name: notebook notebook_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notebook
    ADD CONSTRAINT notebook_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: subscription_plans subscription_plans_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_pkey PRIMARY KEY (id);


--
-- Name: subscription_plans subscription_plans_slug_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_slug_key UNIQUE (slug);


--
-- Name: user_providers user_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_providers
    ADD CONSTRAINT user_providers_pkey PRIMARY KEY (id);


--
-- Name: user_providers user_providers_unique_provider; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_providers
    ADD CONSTRAINT user_providers_unique_provider UNIQUE (provider_name, provider_user_id);


--
-- Name: user_subscriptions user_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_subscriptions
    ADD CONSTRAINT user_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: admin_payment_audit_view_midtrans_transaction_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX admin_payment_audit_view_midtrans_transaction_id_idx ON public.admin_payment_audit_view USING btree (midtrans_transaction_id);


--
-- Name: admin_payment_audit_view_payment_status_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX admin_payment_audit_view_payment_status_idx ON public.admin_payment_audit_view USING btree (payment_status);


--
-- Name: admin_payment_audit_view_subscription_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX admin_payment_audit_view_subscription_id_idx ON public.admin_payment_audit_view USING btree (subscription_id);


--
-- Name: admin_user_performance_summary_full_name_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX admin_user_performance_summary_full_name_idx ON public.admin_user_performance_summary USING btree (full_name);


--
-- Name: admin_user_performance_summary_subscription_status_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX admin_user_performance_summary_subscription_status_idx ON public.admin_user_performance_summary USING btree (subscription_status);


--
-- Name: admin_user_performance_summary_user_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX admin_user_performance_summary_user_id_idx ON public.admin_user_performance_summary USING btree (user_id);


--
-- Name: ai_credit_transactions_service_used_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ai_credit_transactions_service_used_idx ON public.ai_credit_transactions USING btree (service_used);


--
-- Name: ai_credit_transactions_user_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ai_credit_transactions_user_id_idx ON public.ai_credit_transactions USING btree (user_id);


--
-- Name: chat_session_user_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chat_session_user_id_idx ON public.chat_session USING btree (user_id);


--
-- Name: note_user_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX note_user_id_idx ON public.note USING btree (user_id);


--
-- Name: notebook_user_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX notebook_user_id_idx ON public.notebook USING btree (user_id);


--
-- Name: chat_session set_public_chat_session_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER set_public_chat_session_updated_at BEFORE UPDATE ON public.chat_session FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();


--
-- Name: note set_public_note_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER set_public_note_updated_at BEFORE UPDATE ON public.note FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();


--
-- Name: notebook set_public_notebook_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER set_public_notebook_updated_at BEFORE UPDATE ON public.notebook FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();


--
-- Name: user_subscriptions set_public_user_subscriptions_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER set_public_user_subscriptions_updated_at BEFORE UPDATE ON public.user_subscriptions FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();


--
-- Name: users set_public_users_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER set_public_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();


--
-- Name: ai_credit_transactions ai_credit_transactions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ai_credit_transactions
    ADD CONSTRAINT ai_credit_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: chat_message chat_message_chat_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_message
    ADD CONSTRAINT chat_message_chat_session_id_fkey FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id);


--
-- Name: chat_message_raw chat_message_raw_chat_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_message_raw
    ADD CONSTRAINT chat_message_raw_chat_session_id_fkey FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id);


--
-- Name: chat_session chat_session_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: note_embedding note_embedding_note_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.note_embedding
    ADD CONSTRAINT note_embedding_note_id_fkey FOREIGN KEY (note_id) REFERENCES public.note(id);


--
-- Name: note note_notebook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.note
    ADD CONSTRAINT note_notebook_id_fkey FOREIGN KEY (notebook_id) REFERENCES public.notebook(id);


--
-- Name: note note_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.note
    ADD CONSTRAINT note_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: notebook notebook_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notebook
    ADD CONSTRAINT notebook_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.notebook(id);


--
-- Name: notebook notebook_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notebook
    ADD CONSTRAINT notebook_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: password_reset_tokens password_reset_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_providers user_providers_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_providers
    ADD CONSTRAINT user_providers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_subscriptions user_subscriptions_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_subscriptions
    ADD CONSTRAINT user_subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: user_subscriptions user_subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_subscriptions
    ADD CONSTRAINT user_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: chat_session Users can manage their chat sessions; Type: POLICY; Schema: public; Owner: postgres
--

CREATE POLICY "Users can manage their chat sessions" ON public.chat_session USING ((user_id = public.get_current_user_id())) WITH CHECK ((user_id = public.get_current_user_id()));


--
-- Name: note_embedding Users can manage their note embeddings; Type: POLICY; Schema: public; Owner: postgres
--

CREATE POLICY "Users can manage their note embeddings" ON public.note_embedding USING ((EXISTS ( SELECT 1
   FROM public.note n
  WHERE ((n.id = note_embedding.note_id) AND (n.user_id = public.get_current_user_id())))));


--
-- Name: notebook Users can manage their notebooks; Type: POLICY; Schema: public; Owner: postgres
--

CREATE POLICY "Users can manage their notebooks" ON public.notebook USING ((user_id = public.get_current_user_id())) WITH CHECK ((user_id = public.get_current_user_id()));


--
-- Name: note Users can manage their notes; Type: POLICY; Schema: public; Owner: postgres
--

CREATE POLICY "Users can manage their notes" ON public.note USING ((user_id = public.get_current_user_id())) WITH CHECK ((user_id = public.get_current_user_id()));


--
-- Name: chat_message Users can read/write their chat messages; Type: POLICY; Schema: public; Owner: postgres
--

CREATE POLICY "Users can read/write their chat messages" ON public.chat_message USING ((EXISTS ( SELECT 1
   FROM public.chat_session cs
  WHERE ((cs.id = chat_message.chat_session_id) AND (cs.user_id = public.get_current_user_id()))))) WITH CHECK ((EXISTS ( SELECT 1
   FROM public.chat_session cs
  WHERE ((cs.id = chat_message.chat_session_id) AND (cs.user_id = public.get_current_user_id())))));


--
-- Name: chat_message; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.chat_message ENABLE ROW LEVEL SECURITY;

--
-- Name: chat_message_raw; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.chat_message_raw ENABLE ROW LEVEL SECURITY;

--
-- Name: chat_session; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.chat_session ENABLE ROW LEVEL SECURITY;

--
-- Name: note; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.note ENABLE ROW LEVEL SECURITY;

--
-- Name: note_embedding; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.note_embedding ENABLE ROW LEVEL SECURITY;

--
-- Name: notebook; Type: ROW SECURITY; Schema: public; Owner: postgres
--

ALTER TABLE public.notebook ENABLE ROW LEVEL SECURITY;

--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;


--
-- PostgreSQL database dump complete
--

\unrestrict EZWaoUsDfCBsZYtCDfefSmmCzP5349VB5gN3yCKDyN4zpAKDFhg8cRYQq2TObVq

