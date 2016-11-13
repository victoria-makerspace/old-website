--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.4
-- Dumped by pg_dump version 9.5.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: makerspace; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA makerspace;


SET search_path = makerspace, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: member; Type: TABLE; Schema: makerspace; Owner: -
--

CREATE TABLE member (
    username text NOT NULL,
    name text NOT NULL,
    email text NOT NULL
);


--
-- Name: member_pkey; Type: CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY member
    ADD CONSTRAINT member_pkey PRIMARY KEY (username);


--
-- PostgreSQL database dump complete
--

