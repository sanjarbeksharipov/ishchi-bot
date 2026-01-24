-- Migration: Remove payment_instruction_message_id from job_bookings

ALTER TABLE job_bookings 
DROP COLUMN IF EXISTS payment_instruction_message_id;
