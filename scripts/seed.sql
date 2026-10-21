-- =============================================================================
-- SukGebeya Development Seed Data
-- =============================================================================
-- This file contains test data for development environment.
-- Run with: make seed
--
-- ⚠️  WARNING: Do NOT run this in production!
-- =============================================================================
-- Test Accounts:
--   Admin:    admin@sukgebeya.com / Admin123!
--   Buyer:    buyer@test.com / Password123!
--   Merchant: merchant@test.com / Password123!
--   Merchant: merchant2@test.com / Password123!
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
('00000000-0000-0000-0000-000000000001', 'admin@test.com', '+251911000001', '$argon2id$v=19$m=65536,t=3,p=2$Qrih+fLjrQKK23OvZGJwjg$xXhxm0mysb/mhuME/jsb8JubTyqvihUkXJ7N3QZo/XE', 'admin', 'Admin User', 'verified', true, NOW(), NOW()),

-- Test buyer
('00000000-0000-0000-0000-000000000002', 'buyer@test.com', '+251911000002', '$argon2id$v=19$m=65536,t=3,p=2$yM9p79vfcIUiL5uc/mgiPg$1cPlXZApvW0cNWRFTmXBSSRLy2wNUOhVJAklOD6UOi4', 'buyer', 'Test Buyer', 'verified', true, NOW(), NOW()),

-- Test merchant 1
('00000000-0000-0000-0000-000000000003', 'merchant@test.com', '+251911000003', '$argon2id$v=19$m=65536,t=3,p=2$55oFK0ySTh0jPw9hJ2S81A$/pO807C13YRT799wfgMzeTEP8Aa8Gip8oB5beQxc+IM', 'merchant', 'Test Merchant', 'verified', true, NOW(), NOW()),

-- Test merchant 2
('00000000-0000-0000-0000-000000000004', 'merchant2@test.com', '+251911000004', '$argon2id$v=19$m=65536,t=3,p=2$8a4UTHK9qTSpgZv2iY5TZQ$fR09+8u6qFoslhoaJmp3RD/I/CPNWSt6MjeBXF9bHGg', 'merchant', 'Another Merchant', 'verified', true, NOW(), NOW());

-- =============================================================================
-- 2. CATEGORIES
-- =============================================================================

INSERT INTO categories (id, name, slug, icon_url, is_active, sort_order, created_at) VALUES
('10000000-0000-0000-0000-000000000001', 'Clothing & Fashion', 'clothing-fashion', NULL, true, 1, NOW()),
('10000000-0000-0000-0000-000000000002', 'Food & Beverages', 'food-beverages', NULL, true, 2, NOW()),
('10000000-0000-0000-0000-000000000003', 'Electronics', 'electronics', NULL, true, 3, NOW()),
('10000000-0000-0000-0000-000000000004', 'Crafts & Handmade', 'crafts-handmade', NULL, true, 4, NOW()),
('10000000-0000-0000-0000-000000000005', 'Beauty & Personal Care', 'beauty-personal-care', NULL, true, 5, NOW()),
('10000000-0000-0000-0000-000000000006', 'Home & Living', 'home-living', NULL, true, 6, NOW()),
('10000000-0000-0000-0000-000000000007', 'Services', 'services', NULL, true, 7, NOW()),
('10000000-0000-0000-0000-000000000008', 'Other', 'other', NULL, true, 8, NOW());

-- =============================================================================
-- 3. MERCHANT PROFILES
-- =============================================================================

INSERT INTO merchant_profiles (id, user_id, shop_name, shop_slug, shop_description, contact_phone, contact_email, address, city, blnk_balance_id, is_verified, total_sales_cents, created_at, updated_at) VALUES
-- Merchant 1: Fashion Shop
('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000003', 'Ethiopian Fashion Hub', 'ethiopian-fashion-hub', 'Traditional and modern Ethiopian clothing', '+251911000003', 'merchant@test.com', 'Bole, Addis Ababa', 'Addis Ababa', NULL, true, 0, NOW(), NOW()),

-- Merchant 2: Coffee Shop
('20000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000004', 'Highland Coffee Co.', 'highland-coffee', 'Premium Ethiopian coffee sourced from Yirgacheffe', '+251911000004', 'merchant2@test.com', 'Piazza, Addis Ababa', 'Addis Ababa', NULL, false, 0, NOW(), NOW());

-- =============================================================================
-- 4. PRODUCTS
-- =============================================================================

INSERT INTO products (id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at) VALUES
-- Products for Merchant 1 (Fashion)
('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'Traditional Habesha Kemis', 'traditional-habesha-kemis', 'Beautiful hand-woven traditional Ethiopian dress with intricate embroidery', 'active', false, NOW(), NOW()),
('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'Modern Ethiopian Netela', 'modern-ethiopian-netela', 'Elegant hand-spun cotton shawl with modern patterns', 'active', false, NOW(), NOW()),

-- Products for Merchant 2 (Coffee)
('30000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'Yirgacheffe Coffee Beans', 'yirgacheffe-coffee-beans', 'Single-origin Arabica coffee beans from Yirgacheffe region', 'active', false, NOW(), NOW()),
('30000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'Sidamo Ground Coffee', 'sidamo-ground-coffee', 'Freshly ground coffee from Sidamo region, 250g pack', 'draft', false, NOW(), NOW());

-- =============================================================================
-- 5. PRODUCT IMAGES
-- =============================================================================

INSERT INTO product_images (id, product_id, url, alt_text, sort_order, is_primary) VALUES
-- Product 1 images
('40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'https://placehold.co/600x600?text=Habesha+Kemis+1', 'Traditional Habesha Kemis front view', 0, true),
('40000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000001', 'https://placehold.co/600x600?text=Habesha+Kemis+2', 'Traditional Habesha Kemis back view', 1, false),

-- Product 2 images
('40000000-0000-0000-0000-000000000003', '30000000-0000-0000-0000-000000000002', 'https://placehold.co/600x600?text=Netela+1', 'Modern Ethiopian Netela', 0, true),

-- Product 3 images
('40000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000003', 'https://placehold.co/600x600?text=Yirgacheffe+Coffee', 'Yirgacheffe Coffee Beans', 0, true),

-- Product 4 images
('40000000-0000-0000-0000-000000000005', '30000000-0000-0000-0000-000000000004', 'https://placehold.co/600x600?text=Sidamo+Coffee', 'Sidamo Ground Coffee', 0, true);

-- =============================================================================
-- 6. PRODUCT VARIANTS
-- =============================================================================

INSERT INTO product_variants (id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at) VALUES
-- Product 1 variants (clothing sizes)
('50000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'HK-S-001', 'Small', 850000, 950000, 'ETB', '{"size": "S"}', true, NOW(), NOW()),
('50000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000001', 'HK-M-001', 'Medium', 850000, 950000, 'ETB', '{"size": "M"}', true, NOW(), NOW()),
('50000000-0000-0000-0000-000000000003', '30000000-0000-0000-0000-000000000001', 'HK-L-001', 'Large', 850000, 950000, 'ETB', '{"size": "L"}', true, NOW(), NOW()),

-- Product 2 variants
('50000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000002', 'NE-001', 'Standard', 250000, NULL, 'ETB', '{"size": "standard"}', true, NOW(), NOW()),

-- Product 3 variants (coffee sizes)
('50000000-0000-0000-0000-000000000005', '30000000-0000-0000-0000-000000000003', 'YC-250', '250g', 450000, 500000, 'ETB', '{"weight": "250g"}', true, NOW(), NOW()),
('50000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000003', 'YC-500', '500g', 850000, 950000, 'ETB', '{"weight": "500g"}', true, NOW(), NOW()),
('50000000-0000-0000-0000-000000000007', '30000000-0000-0000-0000-000000000003', 'YC-1000', '1kg', 1600000, 1800000, 'ETB', '{"weight": "1kg"}', true, NOW(), NOW()),

-- Product 4 variants
('50000000-0000-0000-0000-000000000008', '30000000-0000-0000-0000-000000000004', 'SG-250', '250g', 350000, NULL, 'ETB', '{"weight": "250g"}', false, NOW(), NOW());

-- =============================================================================
-- 7. INVENTORY
-- =============================================================================

INSERT INTO inventory (id, variant_id, quantity, reserved, updated_at) VALUES
-- Product 1 inventory
('60000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', 10, 0, NOW()),
('60000000-0000-0000-0000-000000000002', '50000000-0000-0000-0000-000000000002', 15, 0, NOW()),
('60000000-0000-0000-0000-000000000003', '50000000-0000-0000-0000-000000000003', 8, 0, NOW()),

-- Product 2 inventory
('60000000-0000-0000-0000-000000000004', '50000000-0000-0000-0000-000000000004', 20, 0, NOW()),

-- Product 3 inventory (coffee)
('60000000-0000-0000-0000-000000000005', '50000000-0000-0000-0000-000000000005', 50, 0, NOW()),
('60000000-0000-0000-0000-000000000006', '50000000-0000-0000-0000-000000000006', 30, 0, NOW()),
('60000000-0000-0000-0000-000000000007', '50000000-0000-0000-0000-000000000007', 20, 0, NOW()),

-- Product 4 inventory (draft product - no stock)
('60000000-0000-0000-0000-000000000008', '50000000-0000-0000-0000-000000000008', 0, 0, NOW());

COMMIT;

-- =============================================================================
-- Summary
-- =============================================================================
-- Users: 4 (1 admin, 1 buyer, 2 merchants)
-- Categories: 8
-- Merchant Profiles: 2
-- Products: 4 (3 active, 1 draft)
-- Product Images: 5
-- Product Variants: 8
-- Inventory: 8 records
-- =============================================================================
-- Test accounts have already confirmed their phones; shop 1 charges 100 ETB delivery, free above 2,000 ETB.
-- Seeded merchants have no Blnk wallet yet: run POST /api/v1/admin/merchants/{id}/wallet for each.
UPDATE users SET phone_verified_at = NOW() WHERE email LIKE '%@test.com';
UPDATE merchant_profiles SET delivery_fee_cents = 10000, free_delivery_threshold_cents = 200000
WHERE id = '20000000-0000-0000-0000-000000000001';
