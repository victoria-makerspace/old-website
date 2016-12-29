--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.5
-- Dumped by pg_dump version 9.5.5

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
-- Name: email; Type: TABLE; Schema: makerspace; Owner: -
--

CREATE TABLE email (
    address text NOT NULL,
    member text NOT NULL
);


--
-- Name: member; Type: TABLE; Schema: makerspace; Owner: -
--

CREATE TABLE member (
    username text NOT NULL,
    name text NOT NULL,
    primary_email text
);


--
-- Name: storage; Type: TABLE; Schema: makerspace; Owner: -
--

CREATE TABLE storage (
    location text NOT NULL,
    number integer NOT NULL,
    member text,
    description text
);


--
-- Name: email_pkey; Type: CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY email
    ADD CONSTRAINT email_pkey PRIMARY KEY (address);


--
-- Name: member_pkey; Type: CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY member
    ADD CONSTRAINT member_pkey PRIMARY KEY (username);


--
-- Name: storage_pkey; Type: CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY storage
    ADD CONSTRAINT storage_pkey PRIMARY KEY (location, number);


--
-- Name: email_member_fkey; Type: FK CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY email
    ADD CONSTRAINT email_member_fkey FOREIGN KEY (member) REFERENCES member(username);


--
-- Name: member_primary_email_fkey; Type: FK CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY member
    ADD CONSTRAINT member_primary_email_fkey FOREIGN KEY (primary_email) REFERENCES email(address);


--
-- Name: storage_member_fkey; Type: FK CONSTRAINT; Schema: makerspace; Owner: -
--

ALTER TABLE ONLY storage
    ADD CONSTRAINT storage_member_fkey FOREIGN KEY (member) REFERENCES member(username);


--
-- PostgreSQL database dump complete
--

