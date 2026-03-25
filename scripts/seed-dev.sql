-- =============================================================================
-- SukGebeya Minimal Development Seed Data
-- =============================================================================
-- This file contains minimal seed data for development environment.
-- Run with: make seed-dev
--
-- ⚠️  WARNING: Do NOT run this in production!
-- =============================================================================
-- Test Accounts:
--   Admin:    admin@sukgebeya.com / Admin123!
--   Buyer:    buyer@test.com / Password123!
--   Merchant: merchant@test.com / Password123!
-- =============================================================================

BEGIN;

-- Clear existing data (respects foreign key constraints)
DELETE FROM inventory;
DELETE FROM product_variants;
DELETE FROM product_images;
DELETE FROM products;
DELETE FROM categories;
DELETE FROM merchant_profiles;
DELETE FROM notes;
DELETE FROM users;

-- =============================================================================
-- 1. USERS
-- =============================================================================

-- Password hashes generated with Argon2id (m=65536, t=3, p=2)
-- Admin: Admin123!
-- Others: Password123!

INSERT INTO users (id, email, phone, password_hash, role, full_name, kyc_status, is_active, created_at, updated_at) VALUES
-- Admin user
('00000000-0000-0000-0000-000000000001', 'admin@sukgebeya.com', '+251911000001', '$argon2id$v=19$m=65536,t=3,p=2$Qrih+fLjrQKK23OvZGJwjg$xXhxm0mysb/mhuME/jsb8JubTyqvihUkXJ7N3QZo/XE', 'admin', 'Admin User', 'verified', true, NOW(), NOW()),

-- Test buyer
('00000000-0000-0000-0000-000000000002', 'buyer@test.com', '+251911000002', '$argon2id$v=19$m=65536,t=3,p=2$yM9p79vfcIUiL5uc/mgiPg$1cPlXZApvW0cNWRFTmXBSSRLy2wNUOhVJAklOD6UOi4', 'buyer', 'Test Buyer', 'verified', true, NOW(), NOW()),

-- Test merchant
('00000000-0000-0000-0000-000000000003', 'merchant@test.com', '+251911000003', '$argon2id$v=19$m=65536,t=3,p=2$55oFK0ySTh0jPw9hJ2S81A$/pO807C13YRT799wfgMzeTEP8Aa8Gip8oB5beQxc+IM', 'merchant', 'Test Merchant', 'verified', true, NOW(), NOW());

-- =============================================================================
-- 2. CATEGORIES
-- =============================================================================

INSERT INTO categories (id, name, slug, is_active, sort_order, created_at) VALUES
('10000000-0000-0000-0000-000000000001', 'Clothing & Fashion', 'clothing-fashion', true, 1, NOW()),
('10000000-0000-0000-0000-000000000002', 'Food & Beverages', 'food-beverages', true, 2, NOW()),
('10000000-0000-0000-0000-000000000003', 'Electronics', 'electronics', true, 3, NOW());

-- =============================================================================
-- 3. MERCHANT PROFILES
-- =============================================================================

INSERT INTO merchant_profiles (id, user_id, shop_name, shop_slug, shop_description, contact_phone, contact_email, address, city, blnk_balance_id, is_verified, total_sales_cents, created_at, updated_at) VALUES
('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000003', 'Test Fashion Shop', 'test-fashion-shop', 'Test shop for development', '+251911000003', 'merchant@test.com', 'Bole, Addis Ababa', 'Addis Ababa', NULL, true, 0, NOW(), NOW());

-- =============================================================================
-- 4. PRODUCTS
-- =============================================================================

INSERT INTO products (id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at) VALUES
('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'Test Product 1', 'test-product-1', 'A test product for development', 'active', false, NOW(), NOW()),
('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'Test Product 2', 'test-product-2', 'Another test product', 'active', false, NOW(), NOW());

-- =============================================================================
-- 5. PRODUCT VARIANTS
-- =============================================================================

INSERT INTO product_variants (id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at) VALUES
('50000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'TEST-SKU-001', 'Standard', 50000, NULL, 'ETB', '{}', true, NOW(), NOW()),
('50000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002', 'TEST-SKU-002', 'Standard', 75000, 85000, 'ETB', '{}', true, NOW(), NOW());

COMMIT;

-- =============================================================================
-- Summary
-- =============================================================================
-- Users: 3 (1 admin, 1 buyer, 1 merchant)
-- Categories: 3
-- Merchant Profiles: 1
-- Products: 2 (both active)
-- Product Variants: 2
-- =============================================================================