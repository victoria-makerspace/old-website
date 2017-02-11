
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
-- CREATE TYPE privelege AS ENUM ('');
CREATE TABLE administrator (
	username text PRIMARY KEY REFERENCES member
	--  privileges privilege[]
);
CREATE TABLE session_http (
	token character(64) PRIMARY KEY,
	username text NOT NULL REFERENCES member,
	sign_in_time timestamp(0) NOT NULL DEFAULT now(),
	last_seen timestamp(0) NOT NULL DEFAULT now(),
	expires timestamp(0)
);
CREATE TABLE payment_profile (
	username text PRIMARY KEY REFERENCES member,
	id text UNIQUE NOT NULL,
	error bool NOT NULL DEFAULT false,
	error_message text,
	CHECK (error = false AND error_message IS NULL OR error = true)
);
CREATE TABLE student (
	username text PRIMARY KEY REFERENCES payment_profile,
	institution text,
	graduation_date date
);
CREATE TYPE fee_category AS ENUM ('membership', 'storage', 'consumable');
CREATE TABLE fee (
	id serial PRIMARY KEY,
	category fee_category NOT NULL,
	identifier text NOT NULL,
	amount real NOT NULL,
	description text,
	UNIQUE (category, identifier)
);
COPY fee (category, identifier, amount) FROM STDIN;
membership	regular	50.0
membership	student	30.0
\.
CREATE TABLE invoice (
	id serial PRIMARY KEY,
	username text NOT NULL REFERENCES member,
	-- null values are one-time-only bills
	date date NOT NULL DEFAULT now(),
	recurring interval DEFAULT '1 M',
	end_date date,
	name text,
	amount real,
	fee integer REFERENCES fee,
	-- XOR (amount, fee) on null value
	CHECK ((amount IS NULL AND fee IS NOT NULL) OR
		(amount IS NOT NULL AND fee IS NULL))
);
CREATE TABLE txn_scheduler_log (
	time timestamp(0) PRIMARY KEY DEFAULT now()
);
CREATE TABLE transaction (
	-- Beanstream value
	id integer PRIMARY KEY,
	username text NOT NULL REFERENCES member,
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
