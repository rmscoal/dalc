CREATE TYPE task_status AS ENUM ('SCHEDULED', 'COMPLETED', 'FAILED');

CREATE TABLE tasks (
  id SERIAL PRIMARY KEY,
  status task_status ,
  expression VARCHAR(150),
  result BIGINT
);
