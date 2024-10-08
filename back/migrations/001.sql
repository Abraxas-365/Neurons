-- Create auth_user Table
CREATE TABLE auth_user (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    google_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create user_session Table
CREATE TABLE user_session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create table for Users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    auth_user_id TEXT NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('teacher', 'student')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create table for Classrooms
CREATE TABLE classrooms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    teacher_id INTEGER NOT NULL REFERENCES users(id),
    available_neurons INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create relationship table between Users (Students) and Classrooms
CREATE TABLE users_classrooms (
    user_id INTEGER NOT NULL REFERENCES users(id),
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    neurons INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, classroom_id)
);

-- Create table for Neuron transaction records
CREATE TABLE neuron_transactions (
    id SERIAL PRIMARY KEY,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    amount INTEGER NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('assignment', 'return')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes to improve query performance
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_classrooms_teacher_id ON classrooms(teacher_id);
CREATE INDEX idx_users_classrooms_user_id ON users_classrooms(user_id);
CREATE INDEX idx_users_classrooms_classroom_id ON users_classrooms(classroom_id);
CREATE INDEX idx_neuron_transactions_classroom_id ON neuron_transactions(classroom_id);
CREATE INDEX idx_neuron_transactions_user_id ON neuron_transactions(user_id);
