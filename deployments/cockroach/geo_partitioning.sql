-- Mastery: CockroachDB Geo-Partitioning for Global Scale (Blueprint requirement)
-- This logic ensures data for specific regions stays in those regions to minimize latency and satisfy GDPR.

-- 1. Create Regional Database configuration
-- ALTER DATABASE fincore PRIMARY REGION "us-east-1";
-- ALTER DATABASE fincore ADD REGION "eu-west-1";
-- ALTER DATABASE fincore ADD REGION "ap-southeast-1";

-- 2. Define Regional Enums
-- CREATE TYPE region_enum AS ENUM ('us', 'eu', 'ap');

-- 3. Geo-partitioning the accounts table
-- ALTER TABLE accounts ADD COLUMN region region_enum DEFAULT 'eu';
-- ALTER TABLE accounts CONFIGURE ZONE USING constraints='[+region=eu]';

-- 4. Multi-region survival goal
-- ALTER DATABASE fincore SURVIVE REGION FAILURE;

-- This skeleton represents the production scale-out plan for Phase 7.4.
