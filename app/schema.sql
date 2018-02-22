
CREATE TABLE user (
	id serial PRIMARY KEY,
	username text NOT NULL UNIQUE,
	email text NOT NULL UNIQUE,
	password character(128), -- First half is the key, next half is the salt.
	registered timestamp(0) with time zone NOT NULL DEFAULT now()
);
CREATE TABLE session (
	token character(64) PRIMARY KEY,
	user integer NOT NULL REFERENCES user,
	sign_in_time timestamp(0) with time zone NOT NULL DEFAULT now(),
	last_seen timestamp(0) with time zone NOT NULL DEFAULT now(),
	expires timestamp(0) with time zone
);
