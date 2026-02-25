CREATE TABLE emails (
                        id SERIAL PRIMARY KEY,
                        to_email TEXT NOT NULL,
                        subject TEXT NOT NULL,
                        template TEXT NOT NULL,
                        data JSONB NOT NULL,
                        status TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT NOW()
);