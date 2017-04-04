
DROP SCHEMA IF EXISTS makerspace CASCADE;
CREATE SCHEMA makerspace;
ALTER DATABASE makerspace SET search_path TO makerspace, pg_catalog;

CREATE TABLE member (
	id serial PRIMARY KEY,
	username text NOT NULL UNIQUE,
	name text NOT NULL,
	password_key character(64),
	password_salt character(64) UNIQUE,
	-- NULL indicates unverified e-mail
	-- TODO: e-mail uniqueness requires case-insensitive check
	email text UNIQUE,
	avatar_url text,
	agreed_to_terms boolean NOT NULL DEFAULT false,
	registered timestamp(0) with time zone NOT NULL DEFAULT now(),
	gratuitous boolean NOT NULL DEFAULT false,
	-- NULL indicates not approved
	approved_at timestamp(0) with time zone,
	approved_by integer REFERENCES member,
	CHECK (CASE WHEN approved_at IS NOT NULL THEN approved_by IS NOT NULL END)
);
CREATE TABLE email_verification_token (
	member integer PRIMARY KEY REFERENCES member,
	email text NOT NULL,
	token character(64) NOT NULL,
	time timestamp(0) with time zone NOT NULL DEFAULT now()
);
CREATE TABLE reset_password_token (
	member integer PRIMARY KEY REFERENCES member,
	token character(64) NOT NULL,
	time timestamp(0) with time zone NOT NULL DEFAULT now()
);
CREATE TYPE admin_privilege AS ENUM (
	'approve-member',
	'modify-member',
	'revoke-member',
	'do-transactions');
CREATE TABLE administrator (
	member integer PRIMARY KEY REFERENCES member,
	privileges admin_privilege[]
);
CREATE TABLE student (
	member integer PRIMARY KEY REFERENCES member,
	institution text,
	student_email text,
	graduation_date date
);
CREATE TABLE session_http (
	token character(64) PRIMARY KEY,
	member integer NOT NULL REFERENCES member,
	sign_in_time timestamp(0) NOT NULL DEFAULT now(),
	last_seen timestamp(0) NOT NULL DEFAULT now(),
	expires timestamp(0)
);
CREATE TABLE payment_profile (
	member integer PRIMARY KEY REFERENCES member,
	id text UNIQUE,
	-- NULL or 0 value implies profile is valid
	error integer DEFAULT 1
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
	member integer NOT NULL REFERENCES member,
	created timestamp(0) with time zone NOT NULL DEFAULT now(),
	-- NULL start_date indicates pending approval
	start_date date DEFAULT now(),
	-- Defaults to username when NULL
	paid_by integer NOT NULL REFERENCES payment_profile,
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
	id serial PRIMARY KEY,
	start_time timestamp(0) NOT NULL DEFAULT now(),
	end_time timestamp(0),
	interval interval NOT NULL,
	error text,
	txn_todo integer NOT NULL,
	txn_attempts integer,
	txn_approved integer,
	UNIQUE (start_time, interval)
);
CREATE TABLE transaction (
	-- Beanstream value
	id integer PRIMARY KEY,
	profile integer NOT NULL REFERENCES payment_profile,
	approved boolean NOT NULL,
	time timestamp(0) NOT NULL DEFAULT now(),
	amount real NOT NULL,
	order_id text,
	comment text,
	card character(4),
	ip_address text,
	invoice integer NOT NULL REFERENCES invoice,
	logged integer REFERENCES txn_scheduler_log,
	CHECK (CASE WHEN amount IS NULL THEN invoice IS NOT NULL END)
);
CREATE TABLE missed_payment (
	invoice integer NOT NULL REFERENCES invoice,
	time timestamp(0) NOT NULL DEFAULT now(),
	transaction integer REFERENCES transaction,
	logged integer REFERENCES txn_scheduler_log,
	PRIMARY KEY (invoice, time)
);
-- TODO: multiple members sharing storage
CREATE TABLE storage (
	number integer NOT NULL,
	fee integer NOT NULL REFERENCES fee,
	available boolean NOT NULL DEFAULT true,
	size real,
	invoice integer REFERENCES invoice,
	PRIMARY KEY (number, fee)
);
CREATE TABLE storage_waitlist (
	time timestamp(0) NOT NULL DEFAULT now(),
	identifier integer NOT NULL REFERENCES fee,
	member integer NOT NULL REFERENCES member,
	-- NULL signifies waiting for any number
	number integer,
	PRIMARY KEY (time, identifier)
);
-- storage values
--	Hall lockers
INSERT INTO storage (number, fee)
SELECT	generate_series(1,12), id
FROM	fee
WHERE	category = 'storage' AND identifier = 'hall-locker';
--	Bathroom lockers
INSERT INTO storage (number, fee)
SELECT generate_series(1,11), id
FROM	fee
WHERE	category = 'storage' AND identifier = 'bathroom-locker';
	-- Bathroom lockers 7 and 8 are reserved for VITP cleaners
	UPDATE storage
	SET available = false
	WHERE number IN (7, 8)
		AND fee = (SELECT id FROM fee
			WHERE category = 'storage' AND identifier = 'bathroom-locker');
--	Wall storage
INSERT INTO storage (number, fee, size)
SELECT generate_subscripts(a, 1), id, unnest(a)
FROM (
	SELECT id, ARRAY[2.5,3.5,3,5,4,5,4,4,4,5.5] AS a
	FROM fee
	WHERE category = 'storage' AND identifier = 'wall'
) f;
	-- Storage locations 1 and 2 are owned by makerspace for now
	UPDATE storage
	SET available = false
	WHERE number IN (1, 2)
		AND fee = (SELECT id FROM fee
			WHERE category = 'storage' AND identifier = 'wall');
-- /end storage values
