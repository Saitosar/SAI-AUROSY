-- Remove sandbox demo robots
DELETE FROM robots WHERE id IN ('sandbox-r1', 'sandbox-r2');

-- Remove sandbox tenant
DELETE FROM tenants WHERE id = 'sandbox';
