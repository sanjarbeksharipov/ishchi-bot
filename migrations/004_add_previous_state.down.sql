-- Remove previous_state column from registration_drafts table
ALTER TABLE registration_drafts 
DROP COLUMN previous_state;
