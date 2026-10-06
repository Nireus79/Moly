-- Migration 018: Drop login_codes table
-- Reason: Incomplete code-based authentication feature removed
-- The incomplete auth handlers (handleGenerateAuthCode, handleCodeBasedLogin, handleValidateAuthCode)
-- have been removed. The login_codes table was only used by these handlers and is no longer needed.
-- The system uses the complete userAuthServer auth system instead.

DROP TABLE IF EXISTS login_codes;
