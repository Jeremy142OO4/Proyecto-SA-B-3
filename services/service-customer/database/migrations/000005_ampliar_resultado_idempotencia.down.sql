-- Reversión de la ampliación aplicada en la migración 000005.
ALTER TABLE processed_messages
    ALTER COLUMN result_reference TYPE VARCHAR(255);
