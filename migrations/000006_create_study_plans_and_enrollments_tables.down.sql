-- Migration 000006 Rollback: Drop enrollments and study_plans tables

DROP INDEX IF EXISTS idx_enrollments_status;
DROP INDEX IF EXISTS idx_enrollments_class_id;
DROP INDEX IF EXISTS idx_enrollments_student_id;
DROP INDEX IF EXISTS idx_enrollments_study_plan_id;
DROP INDEX IF EXISTS uq_enrollments_active_study_plan_class;
DROP INDEX IF EXISTS uq_enrollments_active_student_class;
DROP TABLE IF EXISTS enrollments;

DROP INDEX IF EXISTS idx_study_plans_status;
DROP INDEX IF EXISTS idx_study_plans_period_id;
DROP INDEX IF EXISTS idx_study_plans_student_id;
DROP TABLE IF EXISTS study_plans;
