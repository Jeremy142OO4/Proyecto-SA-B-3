-- Los resultados RPC incluyen el cliente y el token, por lo que pueden
-- superar los 255 caracteres de la definición inicial.
ALTER TABLE processed_messages
    ALTER COLUMN result_reference TYPE TEXT;
