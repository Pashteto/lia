-- Bounds brute force on the 6-digit verification code. Paired with raising
-- verificationCodeTTL to 24h: the attempt cap, not the clock, limits guessing.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_attempts int NOT NULL DEFAULT 0;
