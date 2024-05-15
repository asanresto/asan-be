CREATE TYPE status AS ENUM ('active');

CREATE TABLE users (
    "id" bigserial PRIMARY KEY,
    "name" text,
    "email" text UNIQUE,
    "hashed_password" text,
    "avatar_url" text,
    "status" status
);

CREATE TABLE refresh_tokens (
    "token" text PRIMARY KEY,
    "user_id" bigint NOT NULL REFERENCES users(id),
    "expiry_time" timestamp
);

CREATE TABLE verification_tokens (
    "token" text PRIMARY KEY,
    "user_id" bigint NOT NULL REFERENCES users(id),
    "expiry_time" timestamp
);

CREATE TABLE forgot_password_tokens (
    "token" text PRIMARY KEY,
    "user_id" bigint NOT NULL REFERENCES users(id),
    "expiry_time" timestamp
);

CREATE TABLE outlets (
    "id" bigserial PRIMARY KEY,
    "name" text,
    "address" text
);

CREATE TABLE products (
    "id" bigserial PRIMARY KEY,
    "name" text,
    "price" real,
    "description" text,
    "status" status
);

CREATE TABLE product_images (
    "product_id" bigint NOT NULL REFERENCES products(id),
    "url" text
);

CREATE TABLE stocks (
    "product_id" bigint NOT NULL REFERENCES products(id),
    "outlet_id" bigint NOT NULL REFERENCES outlets(id),
    "amount" integer
);

CREATE TABLE orders (
    "id" bigserial PRIMARY KEY,
    "user_id" bigint NOT NULL REFERENCES users(id),
    "total" real,
    "status" status
);

CREATE TABLE order_items (
    "id" bigserial PRIMARY KEY,
    "order_id" bigint NOT NULL REFERENCES orders(id),
    "user_id" bigint NOT NULL REFERENCES users(id),
    "product_id" bigint NOT NULL REFERENCES products(id),
    "product_name" text,
    "product_price" real,
    "quantity" smallint
);

CREATE TABLE permissions (
    "id" bigserial PRIMARY KEY,
    "name" text
);

CREATE TABLE roles (
    "id" bigserial PRIMARY KEY,
    "name" text
);

CREATE TABLE user_permissions (
    "user_id" bigint NOT NULL REFERENCES users(id),
    "permission_id" bigint NOT NULL REFERENCES permissions(id)
);

CREATE TABLE user_roles (
    "user_id" bigint NOT NULL REFERENCES users(id),
    "role_id" bigint NOT NULL REFERENCES roles(id)
);

CREATE TABLE role_permissions (
    "role_id" bigint NOT NULL REFERENCES roles(id),
    "permission_id" bigint NOT NULL REFERENCES permissions(id)
);