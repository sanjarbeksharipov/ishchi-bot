-- Add previous_state column to registration_drafts table
ALTER TABLE registration_drafts 
ADD COLUMN previous_state VARCHAR(50) NOT NULL DEFAULT 'reg_idle';
