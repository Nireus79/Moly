-- Add WHAT context to contacts table
-- Purpose: Link extracted intentions/goals to specific contacts
-- This extends "Who is Who" to include what user wants/needs with each contact
-- Date: October 8, 2026

-- ContactRole: What role does this contact play in user's intentions?
ALTER TABLE contacts ADD COLUMN contact_role TEXT DEFAULT NULL;
-- Example: "trusted advisor", "supporter", "decision-maker"

-- InvolvedInIntentions: What intentions involve this contact?
ALTER TABLE contacts ADD COLUMN involved_intentions TEXT DEFAULT NULL;
-- JSON array: ["ask for advice", "get feedback", "share news"]

-- PastSuccesses: What has this contact successfully helped with?
ALTER TABLE contacts ADD COLUMN past_successes TEXT DEFAULT NULL;
-- JSON array: ["helped with presentation", "gave career advice"]

-- Dependencies: What must be true for this contact to help?
ALTER TABLE contacts ADD COLUMN dependencies TEXT DEFAULT NULL;
-- JSON array: ["available this week", "needs context", "free evening"]

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_contacts_contact_role ON contacts(user_id, contact_role)
WHERE contact_role IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_involved_intentions ON contacts(user_id)
WHERE involved_intentions IS NOT NULL;
