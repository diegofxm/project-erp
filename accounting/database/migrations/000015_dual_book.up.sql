-- Doble libro contable: PCGA (Decreto 2649) vs. NIIF (IFRS).
-- book = 'BOTH' indica que el asiento aplica a ambos libros (la mayoría).
-- book = 'PCGA' es exclusivo del libro local colombiano.
-- book = 'NIIF' es exclusivo del libro IFRS (ajustes de transición, NIIF 16, etc.).

ALTER TABLE accounting.journal_entries
    ADD COLUMN IF NOT EXISTS book VARCHAR(10) NOT NULL DEFAULT 'BOTH';

ALTER TABLE accounting.journal_entries
    ADD CONSTRAINT journal_entries_book_check
    CHECK (book IN ('PCGA', 'NIIF', 'BOTH'));

-- Índice parcial: solo los asientos que NO son BOTH necesitan filtrado especial.
CREATE INDEX IF NOT EXISTS journal_entries_book_idx
    ON accounting.journal_entries (company_id, book)
    WHERE book != 'BOTH';
