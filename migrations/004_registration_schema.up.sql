-- Migration: Add registration tables
-- Creates registration_drafts and registered_users tables

-- Table for storing registration drafts (incomplete registrations)
CREATE TABLE IF NOT EXISTS registration_drafts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL DEFAULT 'reg_public_offer',
    full_name VARCHAR(255),
    phone VARCHAR(50),
    age INT,
    weight INT,
    height INT,
    passport_photo_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Table for storing fully registered users
CREATE TABLE IF NOT EXISTS registered_users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    age INT NOT NULL,
    weight INT NOT NULL,
    height INT NOT NULL,
    passport_photo_id VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_registration_drafts_user_id ON registration_drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_registration_drafts_state ON registration_drafts(state);
CREATE INDEX IF NOT EXISTS idx_registered_users_user_id ON registered_users(user_id);
CREATE INDEX IF NOT EXISTS idx_registered_users_phone ON registered_users(phone);
CREATE INDEX IF NOT EXISTS idx_registered_users_is_active ON registered_users(is_active);

-- Trigger to automatically update updated_at on registration_drafts updates
CREATE TRIGGER update_registration_drafts_updated_at BEFORE UPDATE ON registration_drafts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to automatically update updated_at on registered_users updates
CREATE TRIGGER update_registered_users_updated_at BEFORE UPDATE ON registered_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
