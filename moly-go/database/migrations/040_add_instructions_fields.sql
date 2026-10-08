-- Add instructions fields to AboutMe and SystemContext
-- Date: October 8, 2026
-- Purpose: Explicit user and system instructions (preset and updatable)

-- AboutMe: Add UserInstructions (how user wants to be understood)
ALTER TABLE about_me ADD COLUMN user_instructions TEXT DEFAULT NULL;
-- JSON array: ["I learn best through examples", "Validate my feelings first"]

-- SystemContext: Add SystemInstructions (how system should behave with this user)
ALTER TABLE system_context ADD COLUMN system_instructions TEXT DEFAULT NULL;
-- JSON array: ["Be Socratic", "Ask questions", "Challenge my assumptions"]

-- Create indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_about_me_user_instructions ON about_me(user_id)
WHERE user_instructions IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_system_context_system_instructions ON system_context(user_id)
WHERE system_instructions IS NOT NULL;
