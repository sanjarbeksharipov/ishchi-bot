-- Migration: Drop jobs table
DROP TABLE IF EXISTS jobs CASCADE;
DROP SEQUENCE IF EXISTS job_order_number_seq;
