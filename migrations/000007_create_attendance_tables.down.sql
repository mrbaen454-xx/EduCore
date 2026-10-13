-- Migration 000007 Rollback: Drop attendances and class_sessions tables

DROP INDEX IF EXISTS idx_attendances_status;
DROP INDEX IF EXISTS idx_attendances_enrollment_id;
DROP INDEX IF EXISTS idx_attendances_student_id;
DROP INDEX IF EXISTS idx_attendances_session_id;
DROP TABLE IF EXISTS attendances;

DROP INDEX IF EXISTS idx_class_sessions_lecturer_id;
DROP INDEX IF EXISTS idx_class_sessions_status;
DROP INDEX IF EXISTS idx_class_sessions_date;
DROP INDEX IF EXISTS idx_class_sessions_class_id;
DROP TABLE IF EXISTS class_sessions;
