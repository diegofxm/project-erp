ALTER TABLE numbering_ranges
    ALTER COLUMN resolution_number SET NOT NULL,
    ALTER COLUMN resolution_date   SET NOT NULL,
    ALTER COLUMN valid_from        SET NOT NULL,
    ALTER COLUMN valid_to          SET NOT NULL;
