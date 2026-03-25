-- =============================================================================
-- SukGebeya Minimal Development Seed Data
-- =============================================================================
-- This file contains minimal seed data for development environment.
-- Run with: make seed-dev
--
-- ⚠️  WARNING: Do NOT run this in production!
-- =============================================================================
-- Test Accounts:
--   Admin: admin@sukgebeya.com / Admin123!
-- =============================================================================

BEGIN;

-- Clear existing data
DELETE FROM inventory;
DELETE FROM product_variants;
DELETE FROM product_images;
DELETE FROM products;
DELETE FROM categories;
DELETE FROM merchant_profiles;
DELETE FROM notes;
DELETE FROM users;

-- =============================================================================
-- 1. ADMIN USER
-- =============================================================================

-- Password: Admin123! (Argon2id, m=65536, t=3, p=2)
INSERT INTO users (id, email, phone, password_hash, role, full_name, kyc_status, is_active, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000001', 'admin@sukgebeya.com', '+251911000001', '$argon2id$v=19$m=65536,t=3,p=2$Qrih+fLjrQKK23OvZGJwjg$xXhxm0mysb/mhuME/jsb8JubTyqvihUkXJ7N3QZo/XE', 'admin', 'Admin User', 'verified', true, NOW(), NOW());

-- =============================================================================
-- 2. CATEGORIES
-- =============================================================================

INSERT INTO categories (id, name, slug, is_active, sort_order, created_at) VALUES
('10000000-0000-0000-0000-000000000001', 'Clothing & Fashion', 'clothing-fashion', true, 1, NOW()),
('10000000-0000-0000-0000-000000000002', 'Food & Beverages', 'food-beverages', true, 2, NOW()),
('10000000-0000-0000-0000-000000000003', 'Electronics', 'electronics', true, 3, NOW()),
('10000000-0000-0000-0000-000000000004', 'Crafts & Handmade', 'crafts-handmade', true, 4, NOW()),
('10000000-0000-0000-0000-000000000005', 'Beauty & Personal Care', 'beauty-personal-care', true, 5, NOW()),
('10000000-0000-0000-0000-000000000006', 'Home & Living', 'home-living', true, 6, NOW()),
('10000000-0000-0000-0000-000000000007', 'Services', 'services', true, 7, NOW()),
('10000000-0000-0000-0000-000000000008', 'Other', 'other', true, 8, NOW());

COMMIT;