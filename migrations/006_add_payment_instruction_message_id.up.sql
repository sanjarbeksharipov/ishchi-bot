-- Migration: Add payment_instruction_message_id to job_bookings
-- This field stores the message ID of the bot's payment instruction message
-- so it can be edited/deleted when the booking expires

ALTER TABLE job_bookings 
ADD COLUMN payment_instruction_message_id BIGINT;

-- Add comment for clarity
COMMENT ON COLUMN job_bookings.payment_instruction_message_id IS 'Message ID of bot payment instruction (for editing on expiry)';
COMMENT ON COLUMN job_bookings.payment_receipt_message_id IS 'Message ID of user payment receipt screenshot';
