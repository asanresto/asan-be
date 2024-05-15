CREATE TABLE chat_rooms (
    "id" bigserial PRIMARY KEY
);

CREATE TABLE user_chat_rooms (
    "room_id" bigint NOT NULL REFERENCES chat_rooms(id),
    "user_id" bigint NOT NULL REFERENCES users(id),
    PRIMARY KEY (room_id, user_id)
);