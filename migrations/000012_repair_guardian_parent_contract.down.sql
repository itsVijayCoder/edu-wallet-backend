-- Only remove the field introduced by this migration. The guardian user link
-- and student_guardians table are owned by migrations 000011 and 000005.

ALTER TABLE guardians
    DROP COLUMN IF EXISTS opt_in_whatsapp;
