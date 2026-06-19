-- +goose Up

/*
* Users
*/
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    email CITEXT NOT NULL,
    email_verified_at TIMESTAMP WITH TIME ZONE,

    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    avatar_url TEXT NOT NULL DEFAULT '',

    language VARCHAR(2) NOT NULL DEFAULT 'en',
    password TEXT
);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

/*
* User Legal Document Acceptances
*/
CREATE TABLE IF NOT EXISTS user_legal_document_acceptances (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    document_type VARCHAR(255) NOT NULL,
    document_version VARCHAR(255) NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_legal_document_acceptances_user_document_version 
    ON user_legal_document_acceptances(user_id, document_type, document_version);

/*
* User Identities
*/
CREATE TABLE IF NOT EXISTS user_identities (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    provider VARCHAR(10) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identities_user_provider
    ON user_identities(user_id, provider);

/*
* Admin Access
*/
CREATE TABLE IF NOT EXISTS admin_access (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

/*
* User Refresh Tokens
*/
CREATE TABLE IF NOT EXISTS user_refresh_tokens (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,

    ip INET NOT NULL,
    user_agent TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    impersonated_by UUID REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT chk_refresh_expires_after_created CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_refresh_tokens_hash 
    ON user_refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_user 
    ON user_refresh_tokens(user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_impersonated_by 
    ON user_refresh_tokens(impersonated_by)
    WHERE impersonated_by IS NOT NULL;

/*
* User Password Reset Tokens
*/
CREATE TABLE IF NOT EXISTS user_password_reset_tokens (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    
    token_hash TEXT NOT NULL,
    
    CONSTRAINT chk_reset_expires_after_created CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_password_reset_tokens_hash 
    ON user_password_reset_tokens(token_hash);

/*
* User Email Verification Tokens
*/
CREATE TABLE IF NOT EXISTS user_email_verification_tokens (
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    
    token_hash TEXT NOT NULL,
    
    CONSTRAINT chk_email_verification_expires_after_created CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email_verification_tokens_hash 
    ON user_email_verification_tokens(token_hash);

-- +goose Down

DROP TABLE IF EXISTS user_email_verification_tokens;
DROP TABLE IF EXISTS user_password_reset_tokens;
DROP TABLE IF EXISTS user_refresh_tokens;
DROP TABLE IF EXISTS admin_access;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS user_legal_document_acceptances;
DROP TABLE IF EXISTS users;