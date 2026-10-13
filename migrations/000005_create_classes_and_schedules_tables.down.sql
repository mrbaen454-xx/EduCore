-- Rollback Migration 000005: Academic Periods, Course Offerings, Classes, Class Lecturers, and Class Schedules

DROP TABLE IF EXISTS class_schedules CASCADE;
DROP TABLE IF EXISTS class_lecturers CASCADE;
DROP TABLE IF EXISTS classes CASCADE;
DROP TABLE IF EXISTS course_offerings CASCADE;
DROP TABLE IF EXISTS academic_periods CASCADE;
