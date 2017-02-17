
DROP SCHEMA IF EXISTS makerspace CASCADE;
CREATE SCHEMA makerspace;
ALTER DATABASE makerspace SET search_path TO makerspace, pg_catalog;

CREATE TABLE member (
	username text PRIMARY KEY,
	name text NOT NULL,
	password_key character(64),
	password_salt character(64) UNIQUE,
	email text NOT NULL UNIQUE,
	email_validated boolean NOT NULL DEFAULT false,
	agreed_to_terms boolean NOT NULL DEFAULT false,
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
CREATE TABLE student (
	username text PRIMARY KEY REFERENCES member,
	institution text,
	student_email text,
	graduation_date date
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
	-- NULL value implies profile is valid
	invalid_error payment_profile_error
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
-- fee values
COPY fee (category, identifier, amount, description) FROM STDIN;
membership	regular	50.0	Membership
membership	student	30.0	Membership (student)
storage	hall-locker	5.0	Hall locker
storage	bathroom-locker	5.0	Bathroom locker
\.
-- Wall storage is $5/lineal foot, so the corresponding invoice should multiply
--	by this number.
COPY fee (category, identifier, amount, description) FROM STDIN;
storage	wall	5.0	Wall storage
\.
-- Corporate membership is case-by-case, only to be registered from the admin
--	panel
COPY fee (category, identifier, description, recurring) FROM STDIN;
membership	corporate	Membership (corporate)	\N
\.
-- /end fee values
CREATE TABLE invoice (
	id serial PRIMARY KEY,
	username text NOT NULL REFERENCES member,
	date date NOT NULL DEFAULT now(),
	-- Defaults to username when NULL
	paid_by text REFERENCES member,
	end_date date,
	-- description, amount default to fee values when NULL
	description text,
	amount real,
	fee integer REFERENCES fee,
	-- Defaults to fee.recurring when NULL
	recurring interval,
	CHECK (CASE WHEN amount IS NULL THEN fee IS NOT NULL END)
);
CREATE TABLE txn_scheduler_log (
	time timestamp(0) PRIMARY KEY DEFAULT now()
);
CREATE TABLE transaction (
	-- Beanstream value
	id integer PRIMARY KEY,
	profile text NOT NULL REFERENCES payment_profile,
	approved boolean NOT NULL,
	time timestamp(0) NOT NULL DEFAULT now(),
	amount real NOT NULL,
	order_id text,
	comment text,
	card character(4),
	ip_address text,
	invoice integer REFERENCES invoice,
	logged timestamp(0) REFERENCES txn_scheduler_log,
	CHECK (CASE WHEN amount IS NULL THEN invoice IS NOT NULL END)
);
CREATE TABLE missed_payment (
	invoice integer NOT NULL REFERENCES invoice,
	date date NOT NULL DEFAULT now(),
	transaction integer REFERENCES transaction,
	logged timestamp(0) REFERENCES txn_scheduler_log
);
-- TODO: multiple members sharing storage
CREATE TABLE storage (
	number integer NOT NULL,
	fee integer NOT NULL REFERENCES fee,
	size real,
	invoice integer REFERENCES invoice,
	PRIMARY KEY (number, fee)
);
-- storage values
--	Hall lockers
INSERT INTO storage
SELECT	generate_series(1,12), id
FROM	fee
WHERE	category = 'storage' AND identifier = 'hall-locker';
--	Bathroom lockers
INSERT INTO storage
SELECT generate_series(1,11), id
FROM	fee
WHERE	category = 'storage' AND identifier = 'bathroom-locker';
	-- Bathroom lockers 7 and 8 are reserved for VITP cleaners
	DELETE FROM storage
	WHERE number IN (7, 8)
		AND fee = (SELECT id FROM fee
			WHERE category = 'storage' AND identifier = 'bathroom-locker');
--	Wall storage
INSERT INTO storage
SELECT generate_subscripts(a, 1), id, unnest(a)
FROM (
	SELECT id, ARRAY[2.5,3.5,3,5,4,5,4,4,4,5.5] AS a
	FROM fee
	WHERE category = 'storage' AND identifier = 'wall'
) f;
-- /end storage values
