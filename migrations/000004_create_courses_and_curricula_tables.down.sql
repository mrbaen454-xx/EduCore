-- Migration 000004 Down: Drop courses and curricula tables in reverse dependency order

DROP TABLE IF EXISTS course_prerequisites;
DROP TABLE IF EXISTS curriculum_courses;
DROP TABLE IF EXISTS curricula;
DROP TABLE IF EXISTS courses;
