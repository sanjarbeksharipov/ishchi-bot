-- Create user_violations table
CREATE TABLE user_violations (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    violation_type VARCHAR(50) NOT NULL,
    booking_id BIGINT REFERENCES job_bookings(id) ON DELETE SET NULL,
    admin_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_violations_user_id ON user_violations(user_id);
CREATE INDEX idx_user_violations_created_at ON user_violations(created_at);

-- Create blocked_users table
CREATE TABLE blocked_users (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    blocked_until TIMESTAMP, -- NULL means permanent block
    total_violations INT NOT NULL DEFAULT 0,
    blocked_by_admin_id BIGINT,
    reason TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blocked_users_blocked_until ON blocked_users(blocked_until);
