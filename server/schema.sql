CREATE TABLE IF NOT EXISTS accounts (
    accountId INT PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    gjp2 VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
    fileName VARCHAR(255) PRIMARY KEY,
    levelId INT NOT NULL,
    levelName VARCHAR(255) NOT NULL,
    accountId INT NOT NULL,
    FOREIGN KEY(accountId) REFERENCES accounts(accountId)
);
