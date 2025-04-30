-- Up migration

-- Create users_list table if it doesn't exist
CREATE TABLE IF NOT EXISTS users_list (
                                          id SERIAL PRIMARY KEY,
                                          email VARCHAR NOT NULL,
                                          currency_pair VARCHAR(255) NOT NULL,
                                          price INT NOT NULL DEFAULT 9,
                                          chat_id BIGINT NOT NULL,
                                          paid BOOLEAN DEFAULT FALSE
);

-- Create payments table if it doesn't exist
CREATE TABLE IF NOT EXISTS payments (
                                        id VARCHAR PRIMARY KEY DEFAULT gen_random_uuid(),
                                        user_id BIGINT REFERENCES users_list(id),
                                        status VARCHAR NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
                                        session_id VARCHAR NOT NULL UNIQUE,
                                        amount BIGINT NOT NULL DEFAULT 0,
                                        created_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
                                        completed_at BIGINT
);

-- Down migration

-- DROP TABLE IF EXISTS payments;
-- DROP TABLE IF EXISTS users_list;