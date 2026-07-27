CREATE EXTENSION IF NOT EXISTS citext;

CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext      NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE stores (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text        NOT NULL,
    slug        text        NOT NULL UNIQUE,
    description text        NOT NULL DEFAULT '',
    currency    char(3)     NOT NULL DEFAULT 'USD',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX stores_owner_id_idx ON stores (owner_id);

CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE products (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id    uuid           NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    name        text           NOT NULL,
    description text           NOT NULL DEFAULT '',
    price_cents integer        NOT NULL CHECK (price_cents >= 0),
    status      product_status NOT NULL DEFAULT 'draft',
    created_at  timestamptz    NOT NULL DEFAULT now(),
    updated_at  timestamptz    NOT NULL DEFAULT now()
);

-- The storefront always filters by store and status, so index the pair.
CREATE INDEX products_store_status_idx ON products (store_id, status);

CREATE TRIGGER products_set_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_images (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id uuid        NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        text        NOT NULL,
    alt_text   text        NOT NULL DEFAULT '',
    position   smallint    NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX product_images_product_idx ON product_images (product_id, position);

CREATE TABLE carts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id   uuid        NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    token      text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER carts_set_updated_at
    BEFORE UPDATE ON carts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cart_items (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id    uuid    NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_id uuid    NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    quantity   integer NOT NULL CHECK (quantity > 0),
    -- One row per product per cart: adding twice bumps the quantity.
    UNIQUE (cart_id, product_id)
);

CREATE TYPE order_status AS ENUM ('pending', 'paid', 'fulfilled', 'cancelled');

CREATE TABLE orders (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT, not CASCADE: a store with order history cannot be deleted.
    store_id          uuid         NOT NULL REFERENCES stores (id) ON DELETE RESTRICT,
    email             citext       NOT NULL,
    status            order_status NOT NULL DEFAULT 'pending',
    total_cents       integer      NOT NULL CHECK (total_cents >= 0),
    currency          char(3)      NOT NULL,
    stripe_session_id text UNIQUE,
    created_at        timestamptz  NOT NULL DEFAULT now(),
    paid_at           timestamptz
);

CREATE INDEX orders_store_created_idx ON orders (store_id, created_at DESC);

CREATE TABLE order_items (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         uuid    NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    -- Kept for reporting, but nullable: deleting a product must not erase history.
    product_id       uuid REFERENCES products (id) ON DELETE SET NULL,
    -- Name and price are copied at purchase time. Editing the product later
    -- must never change what a past order says the customer bought and paid.
    product_name     text    NOT NULL,
    unit_price_cents integer NOT NULL CHECK (unit_price_cents >= 0),
    quantity         integer NOT NULL CHECK (quantity > 0)
);

CREATE INDEX order_items_order_idx ON order_items (order_id);
