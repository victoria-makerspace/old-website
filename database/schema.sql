
DROP SCHEMA IF EXISTS makerspace CASCADE;
CREATE SCHEMA makerspace;
ALTER DATABASE makerspace SET search_path TO makerspace, pg_catalog;

CREATE TABLE member (
	username text PRIMARY KEY,
	name text NOT NULL,
	password_key character(64) NOT NULL,
	password_salt character(64) NOT NULL UNIQUE,
	email text NOT NULL UNIQUE,
	email_validated boolean NOT NULL DEFAULT false,
	registered timestamp(0) NOT NULL DEFAULT now()
);
CREATE TYPE admin_privilege AS ENUM (
	'modify-member',
	'revoke-member',
	'do-transactions');
CREATE TABLE administrator (
	username text PRIMARY KEY REFERENCES member,
	privileges admin_privilege[]
);
CREATE TABLE session_http (
	token character(64) PRIMARY KEY,
	username text NOT NULL REFERENCES member,
	sign_in_time timestamp(0) NOT NULL DEFAULT now(),
	last_seen timestamp(0) NOT NULL DEFAULT now(),
	expires timestamp(0)
);
CREATE TYPE payment_profile_error AS ENUM (
	'no card');
CREATE TABLE payment_profile (
	username text PRIMARY KEY REFERENCES member,
	id text UNIQUE NOT NULL,
	-- null value implies profile is valid
	invalid_error payment_profile_error
);
CREATE TABLE student (
	username text PRIMARY KEY REFERENCES member,
	institution text,
	student_email text,
	graduation_date date
);
CREATE TYPE fee_category AS ENUM (
	'membership',
	'storage',
	'consumable');
CREATE TABLE fee (
	id serial PRIMARY KEY,
	category fee_category NOT NULL,
	identifier text NOT NULL,
	description text NOT NULL,
	amount real,
	-- Set to null for non-recurring values
	recurring interval DEFAULT '1 month',
	UNIQUE (category, identifier),
	-- Recurring fees require a fixed price
	CHECK (CASE WHEN recurring IS NOT NULL THEN amount IS NOT NULL END)
);
COPY fee (category, identifier, amount, description) FROM STDIN;
membership	regular	50.0	Membership dues
membership	student	30.0	Membership dues (student)
\.
CREATE TABLE invoice (
	id serial PRIMARY KEY,
	username text NOT NULL REFERENCES member,
	date date NOT NULL DEFAULT now(),
	profile text NOT NULL REFERENCES payment_profile,
	end_date date,
	description text,
	amount real,
	fee integer REFERENCES fee,
	CHECK (CASE WHEN amount IS NULL THEN fee IS NOT NULL END)
);
CREATE TABLE txn_scheduler_log (
	time timestamp(0) PRIMARY KEY DEFAULT now()
);
CREATE TABLE transaction (
	-- Beanstream value
	id integer PRIMARY KEY,
	approved boolean NOT NULL,
	time timestamp(0) NOT NULL DEFAULT now(),
	order_id text,
	name text,
	card character(4),
	ip_address text,
	amount real,
	invoice integer REFERENCES invoice,
	logged timestamp(0) REFERENCES txn_scheduler_log,
	-- XOR (amount, invoice) on null value
	CHECK ((amount IS NULL AND invoice IS NOT NULL) OR
		(amount IS NOT NULL AND invoice IS NULL))
);
CREATE TABLE missed_payment (
	date date NOT NULL DEFAULT now(),
	invoice integer NOT NULL REFERENCES invoice,
	transaction integer REFERENCES transaction,
	logged timestamp(0) REFERENCES txn_scheduler_log
);
